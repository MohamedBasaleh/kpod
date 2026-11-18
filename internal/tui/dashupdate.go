package tui

import (
	"errors"
	"strconv"

	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.vp.Width = maxInt(m.width-2, 20)
		m.vp.Height = maxInt(m.height-10, 8)
		return m, nil
	case welcomeTickMsg:
		if m.advanceWelcome() {
			return m, welcomeTickCmd()
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case dashContextsMsg:
		return m.onContextsLoaded(msg)
	case dashNamespacesMsg:
		return m.onNamespacesLoaded(msg)
	case dashPodsMsg:
		return m.onPodsLoaded(msg)
	case lastEnvSavedMsg:
		return m, nil
	case discoverMsg:
		return m.onDiscover(msg)
	case discoverSavedMsg:
		return m.onDiscoverSaved(msg)
	case ownerMsg:
		return m.onOwner(msg)
	case captureMsg:
		return m.onCapture(msg)
	case execReadyMsg:
		return m.onExecReady(msg)
	case execDoneMsg:
		return m.onExecDone(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m mainModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.welcomeDone && m.current.view == viewDashboard {
		m.finishWelcome()
	}
	if m.showHelp {
		switch msg.String() {
		case "?", "esc", "q", "enter":
			m.showHelp = false
		}
		return m, nil
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}
	if m.discoverViews() && msg.String() != "ctrl+c" && msg.String() != "q" && msg.String() != "?" {
		return m.handleDiscoverKey(msg)
	}
	if m.shortcutViews() && msg.String() != "ctrl+c" && msg.String() != "q" && msg.String() != "?" {
		return m.handleShortcutKey(msg)
	}
	if m.actionViews() && msg.String() != "ctrl+c" && msg.String() != "q" && msg.String() != "?" {
		return m.handleActionKey(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		return m, tea.Quit
	case "?":
		m.showHelp = true
		return m, nil
	case "esc":
		if m.pop() {
			return m, m.maybeWelcomeCmd()
		}
		if m.current.view != viewDashboard {
			m.current = navFrame{view: viewDashboard}
			m.searching = false
			m.showHelp = false
			m.status = ""
			return m, m.maybeWelcomeCmd()
		}
		return m, tea.Quit
	case "/":
		if m.current.view == viewDashboard || m.current.view == viewContexts || m.current.view == viewNamespaces || m.current.view == viewNsPickContext || m.current.view == viewPods {
			m.searching = true
			m.search.SetValue("")
			m.search.Focus()
			return m, nil
		}
	case "r":
		return m.refresh()
	case "u":
		if m.current.view == viewDashboard {
			return m.openUpdates()
		}
	case "n":
		if m.current.view == viewDashboard || m.current.view == viewEnv {
			return m.openNamespaces()
		}
	case "k":
		if m.current.view == viewDashboard || m.current.view == viewEnv {
			return m.openContexts()
		}
		return m, nil
	case "up":
		m.move(-1)
		return m, nil
	case "down", "j":
		m.move(1)
		return m, nil
	case "enter":
		return m.open()
	case "p":
		if m.current.view == viewEnv {
			return m.open()
		}
	case "s":
		if m.current.view == viewDashboard && len(m.envs) == 0 {
			return m, func() tea.Msg { return startSetupMsg{} }
		}
		if m.current.view == viewDashboard {
			return m.openShortcuts()
		}
	}
	return m, nil
}

func (m mainModel) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.searching = false
		m.search.Blur()
		m.search.SetValue("")
		return m, nil
	case "enter":
		m.searching = false
		m.search.Blur()
		return m.open()
	case "up":
		m.move(-1)
		return m, nil
	case "down":
		m.move(1)
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.envCursor, m.ctxCursor, m.nsCursor, m.podCursor = 0, 0, 0, 0
	return m, cmd
}

func (m *mainModel) move(delta int) {
	switch m.current.view {
	case viewDashboard:
		items := m.filteredEnvs()
		m.envCursor = clamp(m.envCursor+delta, 0, len(items)-1)
	case viewContexts, viewNsPickContext:
		items := m.filteredCtx()
		m.ctxCursor = clamp(m.ctxCursor+delta, 0, len(items)-1)
	case viewNamespaces:
		items := m.filteredNS()
		m.nsCursor = clamp(m.nsCursor+delta, 0, len(items)-1)
	case viewPods:
		items := m.filteredPods()
		m.podCursor = clamp(m.podCursor+delta, 0, len(items)-1)
	}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return 0
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m mainModel) open() (tea.Model, tea.Cmd) {
	switch m.current.view {
	case viewDashboard:
		items := m.filteredEnvs()
		if len(items) == 0 {
			if len(m.envs) == 0 {
				return m, func() tea.Msg { return startSetupMsg{} }
			}
			return m, nil
		}
		if m.envCursor >= len(items) {
			m.envCursor = 0
		}
		env := items[m.envCursor]
		m.push(navFrame{view: viewEnv, envAlias: env.Alias, kubeContext: env.KubeContext})
		m.podCount = nil
		m.podErr = nil
		m.podsLoading = true
		return m, tea.Batch(loadDashPodsCmd(m.kube, env, false), saveLastEnvCmd(m.paths, env.Alias), m.spinner.Tick)
	case viewEnv:
		return m.openPods()
	case viewPods:
		items := m.filteredPods()
		if len(items) == 0 {
			return m, nil
		}
		if m.podCursor >= len(items) {
			m.podCursor = 0
		}
		pod := items[m.podCursor]
		m.push(navFrame{view: viewPodDetail, envAlias: m.current.envAlias, kubeContext: m.current.kubeContext, podName: pod.Name})
		return m, nil
	case viewPodDetail:
		return m.activateAction()
	case viewContexts:
		items := m.filteredCtx()
		if len(items) == 0 {
			return m, nil
		}
		row := items[m.ctxCursor]
		m.push(navFrame{view: viewContextDetail, kubeContext: row.Name})
		return m, nil
	case viewContextDetail:
		return m.openNamespacesFor(m.current.kubeContext)
	case viewNsPickContext:
		items := m.filteredCtx()
		if len(items) == 0 {
			return m, nil
		}
		return m.openNamespacesFor(items[m.ctxCursor].Name)
	case viewNamespaces:
		items := m.filteredNS()
		if len(items) == 0 {
			return m, nil
		}
		row := items[m.nsCursor]
		if row.Env == "" {
			m.status = row.Name + " is a namespace, not a kpod environment."
			return m, nil
		}
		env, ok := m.lookup(row.Env)
		if !ok {
			return m, nil
		}
		m.push(navFrame{view: viewEnv, envAlias: env.Alias, kubeContext: env.KubeContext})
		m.podsLoading = true
		return m, tea.Batch(loadDashPodsCmd(m.kube, env, false), saveLastEnvCmd(m.paths, env.Alias), m.spinner.Tick)
	}
	return m, nil
}

func (m mainModel) openContexts() (tea.Model, tea.Cmd) {
	m.push(navFrame{view: viewContexts})
	m.ctxLoading = true
	m.ctxErr = nil
	m.status = "Loading contexts..."
	return m, tea.Batch(m.spinner.Tick, loadDashContextsCmd(m.kube, false))
}

func (m mainModel) openNamespaces() (tea.Model, tea.Cmd) {
	ctxName := m.scopedContext()
	if ctxName == "" {
		m.push(navFrame{view: viewNsPickContext})
		m.ctxLoading = true
		m.ctxErr = nil
		m.status = "Loading contexts..."
		return m, tea.Batch(m.spinner.Tick, loadDashContextsCmd(m.kube, false))
	}
	return m.openNamespacesFor(ctxName)
}

func (m mainModel) openNamespacesFor(kubeContext string) (tea.Model, tea.Cmd) {
	m.push(navFrame{view: viewNamespaces, kubeContext: kubeContext})
	m.nsContext = kubeContext
	m.nsLoading = true
	m.nsErr = nil
	m.nsRows = nil
	m.status = "Loading namespaces..."
	return m, tea.Batch(m.spinner.Tick, loadDashNamespacesCmd(m.kube, kubeContext, false))
}

func (m mainModel) refresh() (tea.Model, tea.Cmd) {
	switch m.current.view {
	case viewDashboard, viewUpdates:
		m.discLoading = true
		return m, tea.Batch(m.spinner.Tick, runDiscoveryCmd(m.kube, m.kubeErr, m.paths, m.cfg, true))
	case viewContexts, viewNsPickContext:
		m.ctxLoading = true
		m.ctxErr = nil
		m.status = "Loading contexts..."
		return m, tea.Batch(m.spinner.Tick, loadDashContextsCmd(m.kube, true))
	case viewNamespaces, viewNsForbidden:
		if m.current.kubeContext == "" {
			return m, nil
		}
		m.current.view = viewNamespaces
		m.nsLoading = true
		m.nsErr = nil
		m.status = "Loading namespaces..."
		return m, tea.Batch(m.spinner.Tick, loadDashNamespacesCmd(m.kube, m.current.kubeContext, true))
	case viewEnv, viewPods:
		env, ok := m.lookup(m.current.envAlias)
		if !ok {
			return m, nil
		}
		m.podsLoading = true
		if m.current.view == viewEnv {
			m.podErr = nil
		}
		return m, tea.Batch(m.spinner.Tick, loadDashPodsCmd(m.kube, env, true))
	}
	return m, nil
}

func (m mainModel) openPods() (tea.Model, tea.Cmd) {
	env, ok := m.lookup(m.current.envAlias)
	if !ok {
		return m, nil
	}
	m.push(navFrame{view: viewPods, envAlias: env.Alias, kubeContext: env.KubeContext})
	if m.podsAlias == env.Alias && (len(m.pods) > 0 || m.podErr != nil) && !m.podsLoading {
		return m, nil
	}
	m.podsLoading = true
	m.podErr = nil
	return m, tea.Batch(m.spinner.Tick, loadDashPodsCmd(m.kube, env, false))
}

func (m mainModel) onPodsLoaded(msg dashPodsMsg) (tea.Model, tea.Cmd) {
	if m.current.envAlias != "" && msg.alias != m.current.envAlias {
		return m, nil
	}
	m.podsLoading = false
	if msg.err != nil {
		m.podErr = msg.err
		if msg.refresh && len(m.pods) > 0 {
			m.status = kubePodError(msg.err, m.current.kubeContext, m.namespaceFor(msg.alias), m.debug)
			count := len(m.pods)
			m.podCount = &count
			return m, nil
		}
		m.podCount = nil
		if !msg.refresh {
			m.pods = nil
			m.podsAlias = msg.alias
		}
		return m, nil
	}
	keep := selectedPodName(m.filteredPods(), m.podCursor)
	m.pods = sortPods(msg.list)
	m.podsAlias = msg.alias
	m.podErr = nil
	m.status = ""
	count := len(m.pods)
	m.podCount = &count
	filtered := m.filteredPods()
	m.podCursor = indexPodByName(filtered, keep)
	return m, nil
}

func (m mainModel) namespaceFor(alias string) string {
	env, ok := m.lookup(alias)
	if !ok {
		return ""
	}
	return env.Namespace
}

func (m mainModel) onContextsLoaded(msg dashContextsMsg) (tea.Model, tea.Cmd) {
	m.ctxLoading = false
	m.status = ""
	if msg.err != nil {
		m.ctxErr = msg.err
		return m, nil
	}
	m.ctxErr = nil
	rows := make([]kubeCtxRow, 0, len(msg.list))
	for _, c := range msg.list {
		n := len(m.cfg.EnvironmentsForContext(c.Name))
		label := "not configured"
		if n == 1 {
			label = "1 environment"
		} else if n > 1 {
			label = strconv.Itoa(n) + " environments"
		}
		rows = append(rows, kubeCtxRow{Name: c.Name, Label: label})
	}
	m.ctxRows = rows
	m.ctxCursor = 0
	return m, nil
}

func (m mainModel) onNamespacesLoaded(msg dashNamespacesMsg) (tea.Model, tea.Cmd) {
	if m.current.view == viewDiscoverNSPick {
		return m.onDiscoverNamespaces(msg)
	}
	if msg.kubeContext != m.current.kubeContext && m.current.view != viewNsForbidden {
		return m, nil
	}
	m.nsLoading = false
	m.status = ""
	if msg.err != nil {
		if errors.Is(msg.err, kube.ErrForbidden) {
			m.current.view = viewNsForbidden
			m.nsErr = msg.err
			return m, nil
		}
		m.nsErr = msg.err
		return m, nil
	}
	m.nsErr = nil
	rows := make([]nsRow, 0, len(msg.list))
	for _, ns := range msg.list {
		alias := m.cfg.AliasForNamespace(msg.kubeContext, ns.Name)
		rows = append(rows, nsRow{Name: ns.Name, Env: alias})
	}
	m.nsRows = rows
	m.nsCursor = 0
	if len(rows) == 0 {
		m.status = "No namespaces were returned for this context."
	}
	return m, nil
}
