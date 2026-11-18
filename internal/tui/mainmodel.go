package tui

import (
	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/discover"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/state"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type dashView int

const (
	viewDashboard dashView = iota
	viewEnv
	viewPods
	viewPodDetail
	viewContexts
	viewContextDetail
	viewNsPickContext
	viewNamespaces
	viewNsForbidden
	viewError
	viewContainerSelect
	viewConfirm
	viewPrompt
	viewLogsMenu
	viewOutput
	viewShortcuts
	viewShortcutAddEnv
	viewShortcutAddAction
	viewShortcutID
	viewShortcutGen
	viewShortcutRC
	viewShortcutRemove
	viewUpdates
	viewDiscoverItem
	viewDiscoverNSPick
	viewDiscoverAlias
)

type promptKind int

const (
	promptNone promptKind = iota
	promptRunner
	promptLocalPort
	promptRemotePort
	promptLocalPath
	promptRemotePath
)

type navFrame struct {
	view        dashView
	kubeContext string
	envAlias    string
	podName     string
}

type kubeCtxRow struct {
	Name  string
	Label string
}

type nsRow struct {
	Name string
	Env  string
}

type mainModel struct {
	width, height int
	paths         config.Paths
	kube          *kube.Service
	kubeErr       error
	cfg           config.Config
	debug         bool
	spinner       spinner.Model
	search        textinput.Model
	searching     bool
	showHelp      bool

	current navFrame
	stack   []navFrame

	envs        []config.EnvironmentLookup
	envCursor   int
	lastEnv     string
	podCount    *int
	podErr      error
	podsLoading bool
	pods        []kube.Pod
	podsAlias   string
	podCursor   int

	ctxRows    []kubeCtxRow
	ctxCursor  int
	ctxLoading bool
	ctxErr     error

	nsRows    []nsRow
	nsCursor  int
	nsLoading bool
	nsErr     error
	nsContext string

	status string

	acts             *action.Service
	actionCursor     int
	containerCursor  int
	confirmCursor    int
	logsMenuCursor   int
	promptKind       promptKind
	promptInput      textinput.Model
	actEntry         action.MenuEntry
	actContainer     string
	actRunner        string
	actLocalPort     string
	actRemotePort    string
	actLocalPath     string
	actRemotePath    string
	actDeploy        string
	outputTitle      string
	outputBody       string
	vp               viewport.Model
	sessionContainer map[string]string
	pendingRefresh   bool
	pendingDeleted   bool

	welcomeN       int
	welcomeDone    bool
	welcomeStarted bool

	scMenu         menu
	scAddEnvCursor int
	scAddActCursor int
	scGen          []shortcutCandidate
	scGenCursor    int
	scRCInstall    bool

	disc        discover.Result
	discReady   bool
	discLoading bool
	discErr     error
	discSession discover.Session
	discCursor  int
	discCheck   checklist
	discPending []onboardNS
	discAliasIn textinput.Model
}

type onboardNS struct {
	Context   string
	Namespace string
}

type shortcutCandidate struct {
	ID       string
	Env      string
	Action   string
	Selected bool
	Label    string
}

func newMainModel(paths config.Paths, svc *kube.Service, kubeErr error, cfg config.Config, debug bool, open *config.EnvironmentLookup, openPods bool) mainModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	in := textinput.New()
	in.CharLimit = 64
	in.Width = 32
	in.Prompt = "Search: "
	pin := textinput.New()
	pin.CharLimit = 256
	pin.Width = 48
	pin.Prompt = ""
	m := mainModel{
		paths:            paths,
		kube:             svc,
		kubeErr:          kubeErr,
		cfg:              cfg,
		debug:            debug,
		spinner:          sp,
		search:           in,
		envs:             cfg.Environments(),
		current:          navFrame{view: viewDashboard},
		acts:             action.New(svc),
		sessionContainer: map[string]string{},
		promptInput:      pin,
		confirmCursor:    1,
		vp:               viewport.New(80, 20),
		disc:             discover.Result{NewNamespaces: map[string][]string{}},
		discSession:      discover.NewSession(),
		discLoading:      true,
	}
	ain := textinput.New()
	ain.CharLimit = 64
	ain.Width = 32
	ain.Prompt = "Environment name: "
	m.discAliasIn = ain
	if m.skipWelcomeAnim() {
		m.finishWelcome()
	}
	st, err := state.NewRepository(paths).LoadOrDefault()
	if err == nil {
		m.lastEnv = st.LastEnvironment
	}
	m.envCursor = indexOfAlias(m.envs, m.lastEnv)
	if open != nil {
		m.current = navFrame{view: viewEnv, envAlias: open.Alias, kubeContext: open.KubeContext}
		m.envCursor = indexOfAlias(m.envs, open.Alias)
		m.podsLoading = true
		if openPods {
			m.current = navFrame{view: viewPods, envAlias: open.Alias, kubeContext: open.KubeContext}
		}
	}
	return m
}

func indexOfAlias(envs []config.EnvironmentLookup, alias string) int {
	if alias == "" {
		return 0
	}
	for i, env := range envs {
		if env.Alias == alias {
			return i
		}
	}
	return 0
}

func (m mainModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)
	if m.current.view == viewDashboard && !m.welcomeDone {
		cmds = append(cmds, welcomeTickCmd())
	}
	if m.current.view == viewEnv || m.current.view == viewPods {
		env, ok := m.lookup(m.current.envAlias)
		if ok {
			cmds = append(cmds, loadDashPodsCmd(m.kube, env, false), saveLastEnvCmd(m.paths, env.Alias))
		}
	}
	cmds = append(cmds, runDiscoveryCmd(m.kube, m.kubeErr, m.paths, m.cfg, false))
	return tea.Batch(cmds...)
}

func (m mainModel) lookup(alias string) (config.EnvironmentLookup, bool) {
	for _, env := range m.envs {
		if env.Alias == alias {
			return env, true
		}
	}
	return config.EnvironmentLookup{}, false
}

func (m mainModel) currentEnv() (config.EnvironmentLookup, bool) {
	if m.current.envAlias != "" {
		return m.lookup(m.current.envAlias)
	}
	if m.envCursor >= 0 && m.envCursor < len(m.filteredEnvs()) {
		return m.filteredEnvs()[m.envCursor], true
	}
	if len(m.envs) > 0 {
		return m.envs[0], true
	}
	return config.EnvironmentLookup{}, false
}

func (m mainModel) filteredEnvs() []config.EnvironmentLookup {
	q := ""
	if m.searching {
		q = m.search.Value()
	}
	out := make([]config.EnvironmentLookup, 0, len(m.envs))
	for _, env := range m.envs {
		if matches(q, env.Alias, env.Namespace, env.KubeContext) {
			out = append(out, env)
		}
	}
	return out
}

func (m mainModel) filteredCtx() []kubeCtxRow {
	q := ""
	if m.searching {
		q = m.search.Value()
	}
	out := make([]kubeCtxRow, 0, len(m.ctxRows))
	for _, row := range m.ctxRows {
		if matches(q, row.Name, row.Label) {
			out = append(out, row)
		}
	}
	return out
}

func (m mainModel) filteredPods() []kube.Pod {
	q := ""
	if m.searching {
		q = m.search.Value()
	}
	out := make([]kube.Pod, 0, len(m.pods))
	for _, p := range m.pods {
		if matches(q, p.Name) {
			out = append(out, p)
		}
	}
	return out
}

func (m mainModel) filteredNS() []nsRow {
	q := ""
	if m.searching {
		q = m.search.Value()
	}
	out := make([]nsRow, 0, len(m.nsRows))
	for _, row := range m.nsRows {
		if matches(q, row.Name, row.Env) {
			out = append(out, row)
		}
	}
	return out
}

func (m *mainModel) push(next navFrame) {
	if m.current.view == viewDashboard && !m.welcomeDone {
		m.finishWelcome()
	}
	m.stack = append(m.stack, m.current)
	m.current = next
	m.searching = false
	m.search.Blur()
	m.search.SetValue("")
	m.showHelp = false
	m.status = ""
}

func (m *mainModel) pop() bool {
	m.searching = false
	m.search.Blur()
	m.search.SetValue("")
	m.showHelp = false
	m.status = ""
	if len(m.stack) == 0 {
		return false
	}
	m.current = m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return true
}

func (m mainModel) scopedContext() string {
	if m.current.kubeContext != "" {
		return m.current.kubeContext
	}
	if env, ok := m.currentEnv(); ok {
		return env.KubeContext
	}
	names := m.cfg.ConfiguredContextNames()
	if len(names) == 1 {
		return names[0]
	}
	return ""
}
