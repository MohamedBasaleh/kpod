package tui

import (
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/discover"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	"github.com/MohamedBasaleh/kpod/internal/state"
	tea "github.com/charmbracelet/bubbletea"
)

type nsAddedMsg struct {
	cfg       config.Config
	alias     string
	namespace string
	context   string
	err       error
}

func (m mainModel) nsAddViews() bool {
	return m.current.view == viewNsAdd || m.current.view == viewNsAddConfirm
}

func (m mainModel) selectedNS() (nsRow, bool) {
	items := m.filteredNS()
	if len(items) == 0 {
		return nsRow{}, false
	}
	if m.nsCursor >= len(items) {
		m.nsCursor = 0
	}
	return items[m.nsCursor], true
}

func (m mainModel) startNsAdd() (tea.Model, tea.Cmd) {
	row, ok := m.selectedNS()
	if !ok {
		return m, nil
	}
	if row.Env != "" {
		env, found := m.lookup(row.Env)
		if !found {
			return m, nil
		}
		return m.openMappedNamespace(env)
	}
	ctx := m.current.kubeContext
	if ctx == "" {
		ctx = m.nsContext
	}
	if ctx == "" {
		m.status = "No Kubernetes context selected."
		return m, nil
	}
	m.nsAddContext = ctx
	m.nsAddNamespace = row.Name
	m.nsAddIn.SetValue(onboard.SuggestAlias(row.Name, m.cfg.TakenAliases()))
	m.nsAddIn.Focus()
	m.status = ""
	m.push(navFrame{view: viewNsAdd, kubeContext: ctx})
	return m, nil
}

func (m mainModel) openMappedNamespace(env config.EnvironmentLookup) (tea.Model, tea.Cmd) {
	m.push(navFrame{view: viewEnv, envAlias: env.Alias, kubeContext: env.KubeContext})
	m.podsLoading = true
	return m, tea.Batch(loadDashPodsCmd(m.kube, env, false), saveLastEnvCmd(m.paths, env.Alias), m.spinner.Tick)
}

func (m mainModel) handleNsAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.current.view == viewNsAddConfirm {
		return m.handleNsAddConfirmKey(msg)
	}
	switch msg.String() {
	case "esc":
		m.nsAddIn.Blur()
		if m.pop() {
			return m, nil
		}
		return m, nil
	case "enter":
		return m.confirmNsAddAlias()
	}
	var cmd tea.Cmd
	m.nsAddIn, cmd = m.nsAddIn.Update(msg)
	return m, cmd
}

func (m mainModel) confirmNsAddAlias() (tea.Model, tea.Cmd) {
	alias := strings.TrimSpace(m.nsAddIn.Value())
	if err := onboard.ValidateAlias(alias, m.cfg.TakenAliases()); err != nil {
		m.status = err.Error()
		return m, nil
	}
	if existing := m.cfg.AliasForNamespace(m.nsAddContext, m.nsAddNamespace); existing != "" {
		m.status = "namespace is already environment " + existing
		return m, nil
	}
	m.nsAddIn.Blur()
	m.status = ""
	m.push(navFrame{view: viewNsAddConfirm, kubeContext: m.nsAddContext})
	return m, nil
}

func (m mainModel) handleNsAddConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		if m.pop() {
			m.nsAddIn.Focus()
			return m, nil
		}
		return m, nil
	case "enter", "y", "Y":
		return m.commitNsAdd()
	}
	return m, nil
}

func (m mainModel) commitNsAdd() (tea.Model, tea.Cmd) {
	ctxName := m.nsAddContext
	ns := m.nsAddNamespace
	alias := strings.TrimSpace(m.nsAddIn.Value())
	if err := onboard.ValidateAlias(alias, m.cfg.TakenAliases()); err != nil {
		m.status = err.Error()
		if m.pop() {
			m.nsAddIn.Focus()
		}
		return m, nil
	}
	paths := m.paths
	return m, func() tea.Msg {
		repo := config.NewRepository(paths)
		cfg, err := repo.Load()
		if err != nil {
			return nsAddedMsg{err: err}
		}
		cfg, err = config.AddEnvironment(cfg, ctxName, ns, alias)
		if err != nil {
			return nsAddedMsg{err: err}
		}
		if err := repo.Save(cfg); err != nil {
			return nsAddedMsg{err: err}
		}
		_ = discover.PersistSeen(state.NewRepository(paths), func(st *state.State) {
			state.MarkContextSeen(st, ctxName)
			state.MarkNamespaceSeen(st, ctxName, ns)
		})
		return nsAddedMsg{cfg: cfg, alias: alias, namespace: ns, context: ctxName}
	}
}

func (m mainModel) onNsAdded(msg nsAddedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = onboard.UserMessage(msg.err)
		if m.current.view == viewNsAddConfirm {
			m.pop()
			m.nsAddIn.Focus()
		}
		return m, nil
	}
	m.cfg = msg.cfg
	m.envs = msg.cfg.Environments()
	for i, row := range m.nsRows {
		if row.Name == msg.namespace {
			m.nsRows[i].Env = msg.alias
		}
	}
	for m.current.view == viewNsAddConfirm || m.current.view == viewNsAdd {
		if !m.pop() {
			break
		}
	}
	m.current.view = viewNamespaces
	m.current.kubeContext = msg.context
	items := m.filteredNS()
	for i, row := range items {
		if row.Name == msg.namespace {
			m.nsCursor = i
			break
		}
	}
	m.status = "Added " + msg.alias + "."
	return m, nil
}
