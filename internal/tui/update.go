package tui

import (
	"errors"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		return m.handleKey(msg)
	case kubectlOKMsg:
		m.screen = screenContextsWait
		m.status = "Discovering Kubernetes contexts..."
		return m, tea.Batch(m.spinner.Tick, loadContextsCmd(m.kube))
	case kubectlErrMsg:
		m.screen = screenFatal
		m.fatal = onboard.UserMessage(msg.err)
		return m, nil
	case contextsMsg:
		return m.onContexts(msg)
	case namespacesMsg:
		return m.onNamespaces(msg)
	case accessMsg:
		return m.onAccess(msg)
	case migratedMsg:
		return m.onMigrated(msg)
	case savedMsg:
		if msg.err != nil {
			m.screen = screenReview
			m.status = onboard.UserMessage(msg.err)
			return m, nil
		}
		return m, func() tea.Msg { return onboardCompleteMsg{} }
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenManualNS, screenAlias:
		return m.handleInputKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		if m.screen == screenDone || m.screen == screenMigrateDone || m.screen == screenFatal {
			return m, tea.Quit
		}
		if m.screen != screenKubectlWait && m.screen != screenContextsWait && m.screen != screenNamespacesWait &&
			m.screen != screenAccessWait && m.screen != screenSaveWait && m.screen != screenMigrateWait {
			return m, tea.Quit
		}
	case "esc":
		return m.handleEsc()
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case " ":
		return m.handleSpace()
	case "enter":
		return m.handleEnter()
	}
	return m, nil
}

func (m model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m.handleEsc()
	case "enter":
		return m.handleEnter()
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *model) moveCursor(delta int) {
	switch m.screen {
	case screenMigrate:
		m.migrateMenu.move(delta)
	case screenContexts:
		m.contexts.move(delta)
	case screenNamespaces:
		m.namespaces.move(delta)
	case screenForbidden:
		m.forbiddenMenu.move(delta)
	case screenAccessDenied:
		m.deniedMenu.move(delta)
	}
}

func (m model) handleSpace() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenContexts:
		m.contexts.toggle()
		m.status = ""
	case screenNamespaces:
		m.namespaces.toggle()
		m.status = ""
	}
	return m, nil
}

func (m model) handleEsc() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenMigrateView:
		m.screen = screenMigrate
	case screenContexts:
		return m, tea.Quit
	case screenNamespaces:
		m.screen = screenContexts
	case screenForbidden, screenManualNS, screenAccessDenied:
		m.screen = screenNamespaces
		if m.namespaces.len() == 0 {
			m.screen = screenContexts
		}
	case screenAlias:
		m.screen = screenNamespaces
		if m.namespaces.len() == 0 {
			m.screen = screenContexts
		}
	case screenReview:
		m.screen = screenAlias
		m.prepareAliasInput()
	case screenFatal, screenDone, screenMigrateDone, screenWelcome:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenWelcome:
		m.screen = screenKubectlWait
		m.status = "Checking kubectl..."
		return m, tea.Batch(m.spinner.Tick, validateKubectlCmd(m.kube, m.kubeErr))
	case screenFatal, screenDone, screenMigrateDone:
		return m, tea.Quit
	case screenMigrate:
		return m.onMigrateMenu()
	case screenMigrateView:
		m.screen = screenMigrate
	case screenContexts:
		return m.confirmContexts()
	case screenNamespaces:
		return m.confirmNamespaces()
	case screenForbidden:
		return m.onForbiddenMenu()
	case screenManualNS:
		return m.submitManualNS()
	case screenAccessDenied:
		return m.onDeniedMenu()
	case screenAlias:
		return m.submitAlias()
	case screenReview:
		cfg, err := m.draft.ToConfig()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.screen = screenSaveWait
		m.status = "Saving configuration..."
		return m, tea.Batch(m.spinner.Tick, saveConfigCmd(m.paths, cfg))
	}
	return m, nil
}

func (m model) onMigrateMenu() (tea.Model, tea.Cmd) {
	switch m.migrateMenu.current() {
	case "Migrate to kpod v2":
		m.screen = screenMigrateWait
		m.status = "Migrating configuration..."
		return m, tea.Batch(m.spinner.Tick, migrateCmd(m.paths))
	case "View mappings":
		m.screen = screenMigrateView
	default:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) onMigrated(msg migratedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.screen = screenMigrate
		m.status = onboard.UserMessage(msg.err)
		return m, nil
	}
	if msg.result.Skipped != "" {
		m.screen = screenMigrate
		m.status = msg.result.Skipped
		return m, nil
	}
	return m, func() tea.Msg { return onboardCompleteMsg{} }
}

func (m model) onContexts(msg contextsMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.screen = screenFatal
		m.fatal = onboard.UserMessage(msg.err)
		return m, nil
	}
	if len(msg.list) == 0 {
		m.screen = screenFatal
		m.fatal = "No Kubernetes contexts were found in your kubeconfig."
		return m, nil
	}
	names := make([]string, 0, len(msg.list))
	for _, c := range msg.list {
		names = append(names, c.Name)
	}
	m.contexts = newChecklist(names)
	m.status = ""
	m.help = "Space toggles selection. Enter continues."
	m.screen = screenContexts
	return m, nil
}

func (m model) confirmContexts() (tea.Model, tea.Cmd) {
	selected := m.contexts.selectedItems()
	if len(selected) == 0 {
		m.status = "Select at least one context."
		return m, nil
	}
	m.draft = onboard.NewDraft()
	for _, name := range selected {
		m.draft.SelectContext(name)
	}
	m.queue = append([]string{}, selected...)
	return m.startNextContext()
}

func (m model) startNextContext() (tea.Model, tea.Cmd) {
	if len(m.queue) == 0 {
		return m.startAliases()
	}
	m.current = m.queue[0]
	m.queue = m.queue[1:]
	m.screen = screenNamespacesWait
	m.status = "Discovering namespaces in " + ellipsize(m.current, 48) + "..."
	return m, tea.Batch(m.spinner.Tick, loadNamespacesCmd(m.kube, m.current))
}

func (m model) onNamespaces(msg namespacesMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		if errors.Is(msg.err, kube.ErrForbidden) {
			m.screen = screenForbidden
			m.forbiddenMenu.cursor = 0
			m.status = ""
			return m, nil
		}
		m.screen = screenFatal
		m.fatal = onboard.UserMessage(msg.err)
		return m, nil
	}
	if len(msg.list) == 0 {
		m.namespaces = newChecklist(nil)
		m.screen = screenNamespaces
		m.status = "No namespaces were returned."
		return m, nil
	}
	names := make([]string, 0, len(msg.list))
	for _, ns := range msg.list {
		names = append(names, ns.Name)
	}
	m.namespaces = newChecklist(names)
	m.screen = screenNamespaces
	m.status = ""
	m.help = "Nothing is selected by default. Space toggles. Enter continues."
	return m, nil
}

func (m model) confirmNamespaces() (tea.Model, tea.Cmd) {
	selected := m.namespaces.selectedItems()
	m.draft.SetNamespaces(m.current, selected)
	return m.startNextContext()
}

func (m model) onForbiddenMenu() (tea.Model, tea.Cmd) {
	switch m.forbiddenMenu.current() {
	case "Add namespace manually":
		return m.openManualNS()
	case "Retry":
		m.queue = append([]string{m.current}, m.queue...)
		return m.startNextContext()
	default:
		return m.startNextContext()
	}
}

func (m model) openManualNS() (tea.Model, tea.Cmd) {
	m.screen = screenManualNS
	m.input.SetValue("")
	m.input.Placeholder = "dev-core"
	m.input.Focus()
	m.status = ""
	return m, nil
}

func (m model) submitManualNS() (tea.Model, tea.Cmd) {
	ns := strings.TrimSpace(m.input.Value())
	if ns == "" {
		m.status = "Enter a namespace name."
		return m, nil
	}
	m.accessNS = ns
	m.screen = screenAccessWait
	m.status = "Checking access to " + ns + "..."
	m.input.Blur()
	return m, tea.Batch(m.spinner.Tick, accessNamespaceCmd(m.kube, m.current, ns))
}

func (m model) onAccess(msg accessMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.screen = screenAccessDenied
		m.status = onboard.UserMessage(msg.err)
		m.deniedMenu.cursor = 0
		return m, nil
	}
	if !msg.allowed {
		m.screen = screenAccessDenied
		m.status = "You do not currently have permission to read pods in " + msg.ns + "."
		m.deniedMenu.cursor = 0
		return m, nil
	}
	m.draft.AddNamespace(m.current, msg.ns)
	m.status = msg.ns + " is accessible"
	m.screen = screenForbidden
	m.forbiddenMenu.cursor = 0
	m.help = "Add another namespace, retry listing, or continue."
	return m, nil
}

func (m model) onDeniedMenu() (tea.Model, tea.Cmd) {
	if m.deniedMenu.current() == "Back" {
		m.screen = screenForbidden
		return m, nil
	}
	return m.openManualNS()
}

func (m model) startAliases() (tea.Model, tea.Cmd) {
	m.pending = m.draft.PendingAliasTargets()
	if len(m.pending) == 0 {
		m.screen = screenContexts
		m.status = "Add at least one namespace before continuing."
		return m, nil
	}
	m.pendingI = 0
	m.screen = screenAlias
	m.prepareAliasInput()
	return m, nil
}

func (m *model) prepareAliasInput() {
	if m.pendingI < 0 || m.pendingI >= len(m.pending) {
		return
	}
	target := m.pending[m.pendingI]
	m.input.SetValue(onboard.SuggestAlias(target.Namespace, m.draft.TakenAliases()))
	m.input.Placeholder = target.Namespace
	m.input.Focus()
	m.status = ""
}

func (m model) submitAlias() (tea.Model, tea.Cmd) {
	if m.pendingI >= len(m.pending) {
		m.screen = screenReview
		return m, nil
	}
	target := m.pending[m.pendingI]
	if err := m.draft.SetMapping(target.KubeContext, target.Namespace, m.input.Value()); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.pendingI++
	if m.pendingI >= len(m.pending) {
		m.input.Blur()
		m.screen = screenReview
		m.status = ""
		return m, nil
	}
	m.prepareAliasInput()
	return m, nil
}
