package tui

import (
	"errors"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/discover"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	"github.com/MohamedBasaleh/kpod/internal/state"
	tea "github.com/charmbracelet/bubbletea"
)

type discoverMsg struct {
	result     discover.Result
	state      state.State
	stateDirty bool
	err        error
}

type discoverSavedMsg struct {
	cfg       config.Config
	result    discover.Result
	err       error
	hint      string
	remaining []onboardNS
}

func runDiscoveryCmd(svc *kube.Service, kubeErr error, paths config.Paths, cfg config.Config, refresh bool) tea.Cmd {
	return func() tea.Msg {
		if kubeErr != nil {
			return discoverMsg{err: kubeErr, result: discover.Result{NewNamespaces: map[string][]string{}, ContextsErr: kubeErr}}
		}
		if svc == nil {
			return discoverMsg{err: kube.ErrKubectlNotFound, result: discover.Result{NewNamespaces: map[string][]string{}, ContextsErr: kube.ErrKubectlNotFound}}
		}
		st, err := state.NewRepository(paths).LoadOrDefault()
		if err != nil {
			return discoverMsg{err: err, result: discover.Result{NewNamespaces: map[string][]string{}}}
		}
		ctx, cancel := timeoutCtx()
		defer cancel()
		out := discover.Run(ctx, svc, cfg, st, discover.Options{Refresh: refresh})
		if out.StateDirty {
			_ = state.NewRepository(paths).Mutate(func(cur *state.State) error {
				cur.DiscoveryReady = out.State.DiscoveryReady
				cur.SeenContexts = out.State.SeenContexts
				cur.SeenNamespaces = out.State.SeenNamespaces
				return nil
			})
		}
		return discoverMsg{result: out.Result, state: out.State, stateDirty: out.StateDirty, err: out.Result.ContextsErr}
	}
}

func persistConfigCmd(paths config.Paths, mutate func(config.Config) (config.Config, error), seen func(*state.State), result discover.Result, hint string, remaining []onboardNS) tea.Cmd {
	return func() tea.Msg {
		repo := config.NewRepository(paths)
		cfg, err := repo.Load()
		if err != nil {
			return discoverSavedMsg{err: err, result: result, remaining: remaining}
		}
		cfg, err = mutate(cfg)
		if err != nil {
			return discoverSavedMsg{err: err, result: result, remaining: remaining}
		}
		if err := repo.Save(cfg); err != nil {
			return discoverSavedMsg{err: err, result: result, remaining: remaining}
		}
		if seen != nil {
			_ = discover.PersistSeen(state.NewRepository(paths), seen)
		}
		return discoverSavedMsg{cfg: cfg, result: result, hint: hint, remaining: remaining}
	}
}

func (m mainModel) discoverViews() bool {
	switch m.current.view {
	case viewUpdates, viewDiscoverItem, viewDiscoverNSPick, viewDiscoverAlias:
		return true
	default:
		return false
	}
}

func (m mainModel) visibleUpdates() discover.Result {
	return discover.Visible(m.disc, m.discSession)
}

func (m mainModel) showDiscoveryBadge() bool {
	if !m.cfg.Settings.Notifications {
		return false
	}
	if !m.discReady || m.discLoading {
		return false
	}
	return !m.visibleUpdates().Empty()
}

func (m mainModel) onDiscover(msg discoverMsg) (tea.Model, tea.Cmd) {
	m.discLoading = false
	m.discReady = true
	if msg.result.NewNamespaces == nil {
		msg.result.NewNamespaces = map[string][]string{}
	}
	m.disc = msg.result
	m.discErr = msg.err
	if m.discCursor >= len(m.visibleUpdates().Items()) {
		m.discCursor = 0
	}
	return m, nil
}

func (m mainModel) onDiscoverSaved(msg discoverSavedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = onboard.UserMessage(msg.err)
		return m, nil
	}
	m.cfg = msg.cfg
	m.envs = msg.cfg.Environments()
	m.disc = msg.result
	m.status = msg.hint
	if len(msg.remaining) > 0 {
		return m.startAlias(msg.remaining)
	}
	m.discPending = nil
	for m.current.view == viewDiscoverAlias || m.current.view == viewDiscoverNSPick || m.current.view == viewDiscoverItem {
		if !m.pop() {
			break
		}
	}
	if m.current.view != viewUpdates {
		m.current = navFrame{view: viewUpdates}
	}
	if m.visibleUpdates().Empty() {
		m.pop()
	}
	return m, nil
}

func (m mainModel) openUpdates() (tea.Model, tea.Cmd) {
	m.push(navFrame{view: viewUpdates})
	m.discCursor = 0
	if m.discReady && !m.discLoading {
		return m, nil
	}
	m.discLoading = true
	return m, tea.Batch(m.spinner.Tick, runDiscoveryCmd(m.kube, m.kubeErr, m.paths, m.cfg, false))
}

func (m mainModel) handleDiscoverKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.current.view == viewDiscoverAlias {
		return m.handleDiscoverAliasKey(msg)
	}
	switch msg.String() {
	case "up":
		m.moveDiscover(-1)
		return m, nil
	case "down", "j":
		m.moveDiscover(1)
		return m, nil
	case "esc":
		if m.pop() {
			return m, m.maybeWelcomeCmd()
		}
		return m, nil
	case "r":
		m.discLoading = true
		m.status = "Checking Kubernetes updates..."
		return m, tea.Batch(m.spinner.Tick, runDiscoveryCmd(m.kube, m.kubeErr, m.paths, m.cfg, true))
	case "enter":
		if m.current.view == viewUpdates {
			return m.openDiscoverItem()
		}
		if m.current.view == viewDiscoverItem {
			return m.activateDiscoverAction()
		}
		if m.current.view == viewDiscoverNSPick {
			return m.confirmDiscoverNamespaces()
		}
	case " ":
		if m.current.view == viewDiscoverNSPick {
			m.discCheck.toggle()
			return m, nil
		}
	case "a":
		return m.discoverAdd()
	case "i":
		return m.discoverIgnore()
	case "l":
		return m.discoverLater()
	}
	return m, nil
}

func (m *mainModel) moveDiscover(delta int) {
	switch m.current.view {
	case viewUpdates:
		items := m.visibleUpdates().Items()
		m.discCursor = clamp(m.discCursor+delta, 0, len(items)-1)
	case viewDiscoverItem:
		m.confirmCursor = clamp(m.confirmCursor+delta, 0, 2)
	case viewDiscoverNSPick:
		m.discCheck.move(delta)
	}
}

func (m mainModel) selectedDiscoverItem() (discover.Item, bool) {
	items := m.visibleUpdates().Items()
	if len(items) == 0 {
		return discover.Item{}, false
	}
	if m.discCursor >= len(items) {
		m.discCursor = 0
	}
	return items[m.discCursor], true
}

func (m mainModel) openDiscoverItem() (tea.Model, tea.Cmd) {
	item, ok := m.selectedDiscoverItem()
	if !ok {
		return m, nil
	}
	m.push(navFrame{view: viewDiscoverItem, kubeContext: item.Context})
	m.confirmCursor = 0
	return m, nil
}

func (m mainModel) discoverAdd() (tea.Model, tea.Cmd) {
	if m.current.view == viewUpdates || m.current.view == viewDiscoverItem {
		return m.startDiscoverAdd()
	}
	return m, nil
}

func (m mainModel) startDiscoverAdd() (tea.Model, tea.Cmd) {
	item, ok := m.selectedDiscoverItem()
	if !ok {
		return m, nil
	}
	if item.Kind == discover.KindNamespace {
		return m.startAlias([]onboardNS{{Context: item.Context, Namespace: item.Namespace}})
	}
	m.push(navFrame{view: viewDiscoverNSPick, kubeContext: item.Context})
	m.nsContext = item.Context
	m.nsLoading = true
	m.status = "Loading namespaces..."
	return m, tea.Batch(m.spinner.Tick, loadDashNamespacesCmd(m.kube, item.Context, false))
}

func (m mainModel) onDiscoverNamespaces(msg dashNamespacesMsg) (tea.Model, tea.Cmd) {
	if m.current.view != viewDiscoverNSPick {
		return m.onNamespacesLoaded(msg)
	}
	m.nsLoading = false
	if msg.err != nil {
		m.nsErr = msg.err
		if errors.Is(msg.err, kube.ErrForbidden) {
			m.status = "Namespace discovery unavailable for this context."
		} else {
			m.status = kubeContextError(msg.err, msg.kubeContext, m.debug)
		}
		return m, nil
	}
	m.nsErr = nil
	var names []string
	for _, ns := range msg.list {
		if m.cfg.AliasForNamespace(msg.kubeContext, ns.Name) != "" {
			continue
		}
		if m.cfg.NamespaceIgnored(msg.kubeContext, ns.Name) {
			continue
		}
		names = append(names, ns.Name)
	}
	m.discCheck = newChecklist(names)
	if len(names) == 0 {
		m.status = "No unconfigured namespaces in this context."
	} else {
		m.status = ""
	}
	return m, nil
}

func (m mainModel) confirmDiscoverNamespaces() (tea.Model, tea.Cmd) {
	selected := m.discCheck.selectedItems()
	if len(selected) == 0 {
		m.status = "Select at least one namespace, or press esc to cancel."
		return m, nil
	}
	pending := make([]onboardNS, 0, len(selected))
	for _, ns := range selected {
		pending = append(pending, onboardNS{Context: m.current.kubeContext, Namespace: ns})
	}
	return m.startAlias(pending)
}

func (m mainModel) startAlias(pending []onboardNS) (tea.Model, tea.Cmd) {
	if len(pending) == 0 {
		return m, nil
	}
	m.discPending = pending
	if m.current.view != viewDiscoverAlias {
		m.push(navFrame{view: viewDiscoverAlias, kubeContext: pending[0].Context})
	} else {
		m.current.kubeContext = pending[0].Context
	}
	taken := m.cfg.TakenAliases()
	m.discAliasIn.SetValue(onboard.SuggestAlias(pending[0].Namespace, taken))
	m.discAliasIn.Focus()
	m.status = ""
	return m, nil
}

func (m mainModel) handleDiscoverAliasKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.pop() {
			return m, nil
		}
		return m, nil
	case "enter":
		return m.commitDiscoverAlias()
	}
	var cmd tea.Cmd
	m.discAliasIn, cmd = m.discAliasIn.Update(msg)
	return m, cmd
}

func (m mainModel) commitDiscoverAlias() (tea.Model, tea.Cmd) {
	if len(m.discPending) == 0 {
		return m, nil
	}
	target := m.discPending[0]
	alias := strings.TrimSpace(m.discAliasIn.Value())
	if err := onboard.ValidateAlias(alias, m.cfg.TakenAliases()); err != nil {
		m.status = err.Error()
		return m, nil
	}
	remaining := append([]onboardNS{}, m.discPending[1:]...)
	result := m.disc.WithoutNamespace(target.Context, target.Namespace)
	result = result.WithoutContext(target.Context)
	hint := "Added " + alias + ". Shell shortcuts are not generated automatically."
	return m, persistConfigCmd(m.paths, func(cfg config.Config) (config.Config, error) {
		return config.AddEnvironment(cfg, target.Context, target.Namespace, alias)
	}, func(st *state.State) {
		state.MarkContextSeen(st, target.Context)
		state.MarkNamespaceSeen(st, target.Context, target.Namespace)
	}, result, hint, remaining)
}

func (m mainModel) discoverIgnore() (tea.Model, tea.Cmd) {
	item, ok := m.selectedDiscoverItem()
	if !ok {
		return m, nil
	}
	if item.Kind == discover.KindContext {
		result := m.disc.WithoutContext(item.Context)
		return m, persistConfigCmd(m.paths, func(cfg config.Config) (config.Config, error) {
			return config.IgnoreContext(cfg, item.Context), nil
		}, func(st *state.State) {
			state.MarkContextSeen(st, item.Context)
		}, result, "Ignored context "+item.Context+".", nil)
	}
	result := m.disc.WithoutNamespace(item.Context, item.Namespace)
	return m, persistConfigCmd(m.paths, func(cfg config.Config) (config.Config, error) {
		return config.IgnoreNamespace(cfg, item.Context, item.Namespace), nil
	}, func(st *state.State) {
		state.MarkNamespaceSeen(st, item.Context, item.Namespace)
	}, result, "Ignored namespace "+item.Namespace+".", nil)
}

func (m mainModel) discoverLater() (tea.Model, tea.Cmd) {
	item, ok := m.selectedDiscoverItem()
	if !ok {
		return m, nil
	}
	if item.Kind == discover.KindContext {
		m.discSession.LaterContext(item.Context)
	} else {
		m.discSession.LaterNamespace(item.Context, item.Namespace)
	}
	if m.current.view == viewDiscoverItem {
		m.pop()
	}
	items := m.visibleUpdates().Items()
	if m.discCursor >= len(items) {
		m.discCursor = 0
	}
	if len(items) == 0 && m.current.view == viewUpdates {
		m.pop()
	}
	m.status = "Remind me next time."
	return m, nil
}

func (m mainModel) activateDiscoverAction() (tea.Model, tea.Cmd) {
	switch m.confirmCursor {
	case 1:
		return m.discoverIgnore()
	case 2:
		return m.discoverLater()
	default:
		return m.startDiscoverAdd()
	}
}
