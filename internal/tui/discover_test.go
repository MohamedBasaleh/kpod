package tui

import (
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/discover"
	"github.com/MohamedBasaleh/kpod/internal/state"
	tea "github.com/charmbracelet/bubbletea"
)

func sampleDiscovery() discover.Result {
	return discover.Result{
		NewContexts: []string{"company-production"},
		NewNamespaces: map[string][]string{
			"Nawaqis-Dev": {"jobs", "monitoring"},
		},
	}
}

func readyModel(t *testing.T, cfg config.Config, disc discover.Result) mainModel {
	t.Helper()
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, cfg, false, nil, false)
	m.width = 80
	m.finishWelcome()
	m.disc = disc
	m.discReady = true
	m.discLoading = false
	m.discErr = nil
	return m
}

func TestDiscoveryBadgeAndHiddenWhenDisabled(t *testing.T) {
	cfg := testCfg()
	m := readyModel(t, cfg, sampleDiscovery())
	view := m.View()
	if !strings.Contains(view, "Kubernetes updates available") && !strings.Contains(view, "new") {
		t.Fatalf("expected badge: %s", view)
	}
	cfg.Settings.Notifications = false
	m.cfg = cfg
	view = m.View()
	if strings.Contains(view, "Kubernetes updates available") || strings.Contains(view, "[u] view") {
		t.Fatalf("notifications off must hide badge: %s", view)
	}
}

func TestNoBadgeWhenEmpty(t *testing.T) {
	m := readyModel(t, testCfg(), discover.Result{NewNamespaces: map[string][]string{}})
	if m.showDiscoveryBadge() {
		t.Fatal("empty discovery")
	}
	if strings.Contains(m.View(), "[u] view") {
		t.Fatalf("%s", m.View())
	}
}

func TestOpenUpdatesScreen(t *testing.T) {
	m := readyModel(t, testCfg(), sampleDiscovery())
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = next.(mainModel)
	if m.current.view != viewUpdates {
		t.Fatalf("%+v", m.current)
	}
	view := m.View()
	if !strings.Contains(view, "company-production") || !strings.Contains(view, "jobs") {
		t.Fatalf("%s", view)
	}
}

func TestDiscoverLaterSessionOnly(t *testing.T) {
	m := readyModel(t, testCfg(), sampleDiscovery())
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = next.(mainModel)
	next, _ = m.handleDiscoverKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = next.(mainModel)
	if m.cfg.ContextIgnored("company-production") {
		t.Fatal("Later must not ignore")
	}
	if !m.discSession.ContextDeferred("company-production") {
		t.Fatal("expected session later")
	}
	vis := m.visibleUpdates()
	if vis.ContextCount() != 0 {
		t.Fatalf("%v", vis.NewContexts)
	}
	if vis.NamespaceCount() != 2 {
		t.Fatalf("%v", vis.NewNamespaces)
	}
}

func TestDiscoverEscWritesNothing(t *testing.T) {
	dir := t.TempDir()
	repo := config.NewRepository(config.NewPaths(dir))
	cfg := testCfg()
	if err := repo.Save(cfg); err != nil {
		t.Fatal(err)
	}
	m := newMainModel(config.NewPaths(dir), nil, nil, cfg, false, nil, false)
	m.width = 80
	m.finishWelcome()
	m.disc = sampleDiscovery()
	m.discReady = true
	m.discLoading = false
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = next.(mainModel)
	next, _ = m.handleDiscoverKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(mainModel)
	if m.current.view != viewDashboard {
		t.Fatalf("%+v", m.current)
	}
	got, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.IgnoredContexts) != 0 {
		t.Fatalf("%v", got.IgnoredContexts)
	}
}

func TestDiscoverIgnoreContextPersists(t *testing.T) {
	dir := t.TempDir()
	repo := config.NewRepository(config.NewPaths(dir))
	cfg := testCfg()
	if err := repo.Save(cfg); err != nil {
		t.Fatal(err)
	}
	m := newMainModel(config.NewPaths(dir), nil, nil, cfg, false, nil, false)
	m.finishWelcome()
	m.disc = sampleDiscovery()
	m.discReady = true
	m.discLoading = false
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = next.(mainModel)
	next, c := m.discoverIgnore()
	m = next.(mainModel)
	if c == nil {
		t.Fatal("expected persist cmd")
	}
	msg := c()
	saved, ok := msg.(discoverSavedMsg)
	if !ok {
		t.Fatalf("%T", msg)
	}
	if saved.err != nil {
		t.Fatal(saved.err)
	}
	next, _ = m.onDiscoverSaved(saved)
	m = next.(mainModel)
	if !m.cfg.ContextIgnored("company-production") {
		t.Fatal("expected ignored")
	}
	got, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !got.ContextIgnored("company-production") {
		t.Fatal("config not saved")
	}
	if got.AliasForNamespace("Nawaqis-Dev", "dev-core") != "dev" {
		t.Fatal("existing env lost")
	}
}

func TestDiscoverAddNamespacePreservesConfig(t *testing.T) {
	dir := t.TempDir()
	repo := config.NewRepository(config.NewPaths(dir))
	cfg := testCfg()
	cfg.Shortcuts["dc"] = config.Shortcut{Environment: "dev", Action: "console"}
	if err := repo.Save(cfg); err != nil {
		t.Fatal(err)
	}
	m := newMainModel(config.NewPaths(dir), nil, nil, cfg, false, nil, false)
	m.finishWelcome()
	m.disc = sampleDiscovery()
	m.discReady = true
	m.discLoading = false
	m.discCursor = 1 // jobs
	m.discPending = []onboardNS{{Context: "Nawaqis-Dev", Namespace: "jobs"}}
	m.discAliasIn.SetValue("jobs")
	next, c := m.commitDiscoverAlias()
	m = next.(mainModel)
	if c == nil {
		t.Fatal("expected save")
	}
	msg := c()
	saved := msg.(discoverSavedMsg)
	if saved.err != nil {
		t.Fatal(saved.err)
	}
	next, _ = m.onDiscoverSaved(saved)
	m = next.(mainModel)
	if m.cfg.AliasForNamespace("Nawaqis-Dev", "jobs") != "jobs" {
		t.Fatal("mapping missing")
	}
	if m.cfg.Shortcuts["dc"].Action != "console" {
		t.Fatal("shortcut lost")
	}
	if len(m.cfg.Shortcuts) != 1 {
		t.Fatal("shortcuts must not be auto-generated")
	}
}

func TestDiscoverDuplicateAliasRejected(t *testing.T) {
	m := readyModel(t, testCfg(), sampleDiscovery())
	m.discPending = []onboardNS{{Context: "Nawaqis-Dev", Namespace: "jobs"}}
	m.discAliasIn.SetValue("dev")
	next, c := m.commitDiscoverAlias()
	m = next.(mainModel)
	if c != nil {
		t.Fatal("must not save")
	}
	if m.status == "" {
		t.Fatal("expected validation error")
	}
}

func TestManualUpdatesWhenNotificationsDisabled(t *testing.T) {
	cfg := testCfg()
	cfg.Settings.Notifications = false
	m := readyModel(t, cfg, sampleDiscovery())
	if m.showDiscoveryBadge() {
		t.Fatal("badge")
	}
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = next.(mainModel)
	if m.current.view != viewUpdates {
		t.Fatalf("%+v", m.current)
	}
	if !strings.Contains(m.View(), "company-production") {
		t.Fatalf("%s", m.View())
	}
}

func TestSessionDedupDashboard(t *testing.T) {
	m := readyModel(t, testCfg(), sampleDiscovery())
	m.discSession.LaterContext("company-production")
	m.discSession.LaterNamespace("Nawaqis-Dev", "jobs")
	m.discSession.LaterNamespace("Nawaqis-Dev", "monitoring")
	if m.showDiscoveryBadge() {
		t.Fatal("session later should hide badge")
	}
}

func TestStateSeenDistinctFromIgnored(t *testing.T) {
	st := state.Default()
	state.MarkContextSeen(&st, "analytics")
	if !state.Contains(st.SeenContexts, "analytics") {
		t.Fatal("seen")
	}
	cfg := testCfg()
	if cfg.ContextIgnored("analytics") {
		t.Fatal("seen is not ignored")
	}
}

func TestDashboardInitDoesNotBlockOnDiscovery(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.width = 80
	m.finishWelcome()
	view := m.View()
	if !strings.Contains(view, "Environments") {
		t.Fatalf("%s", view)
	}
}
