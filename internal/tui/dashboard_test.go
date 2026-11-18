package tui

import (
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func testCfg() config.Config {
	cfg := config.Default()
	cfg.Contexts["Nawaqis-Dev"] = config.Context{
		Environments: map[string]config.Environment{
			"staging": {Namespace: "staging-core"},
			"dev":     {Namespace: "dev-core"},
		},
	}
	return cfg
}

func TestDashboardListsEnvironmentsSorted(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "dev") || !strings.Contains(view, "staging") {
		t.Fatalf("%s", view)
	}
	dev := strings.Index(view, "dev")
	staging := strings.Index(view, "staging")
	if dev < 0 || staging < 0 || dev > staging {
		t.Fatalf("expected alphabetical order: %s", view)
	}
	if strings.Contains(view, "Dashboard is coming") {
		t.Fatal("placeholder must not show")
	}
}

func TestDashboardSearch(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.searching = true
	m.search.SetValue("sta")
	got := m.filteredEnvs()
	if len(got) != 1 || got[0].Alias != "staging" {
		t.Fatalf("%+v", got)
	}
}

func TestOpenEnvironmentPushesAndPops(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.envCursor = 0
	next, _ := m.open()
	m = next.(mainModel)
	if m.current.view != viewEnv || m.current.envAlias != "dev" {
		t.Fatalf("%+v", m.current)
	}
	if !m.pop() || m.current.view != viewDashboard {
		t.Fatalf("pop %+v", m.current)
	}
}

func TestUnknownNamespaceIsNotEnvironment(t *testing.T) {
	cfg := testCfg()
	if cfg.AliasForNamespace("Nawaqis-Dev", "kube-system") != "" {
		t.Fatal("kube-system must not be an environment")
	}
}

func TestEscFromDirectEnvironmentGoesToDashboard(t *testing.T) {
	env, _ := testCfg().LookupEnvironment("dev")
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, &env, false)
	next, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(mainModel)
	if cmd != nil {
		t.Fatal("should not quit")
	}
	if m.current.view != viewDashboard {
		t.Fatalf("%+v", m.current)
	}
}

func TestDirectEnvironmentOpen(t *testing.T) {
	env, _ := testCfg().LookupEnvironment("staging")
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, &env, false)
	m.width = 80
	if m.current.view != viewEnv || m.current.envAlias != "staging" {
		t.Fatalf("%+v", m.current)
	}
	view := m.View()
	if !strings.Contains(view, "staging-core") || !strings.Contains(view, "Nawaqis-Dev") {
		t.Fatalf("%s", view)
	}
}

func TestHelpKey(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = next.(mainModel)
	if !m.showHelp {
		t.Fatal("expected help")
	}
	view := m.View()
	if !strings.Contains(view, "Navigation") {
		t.Fatalf("%s", view)
	}
}

func TestEmptyEnvironments(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, config.Default(), false, nil, false)
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "No environments are configured") {
		t.Fatalf("%s", view)
	}
}

func TestForbiddenNamespaceView(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.current = navFrame{view: viewNsForbidden, kubeContext: "Nawaqis-Dev"}
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "cannot list namespaces") {
		t.Fatalf("%s", view)
	}
	if !strings.Contains(view, "dev-core") {
		t.Fatalf("configured envs should remain visible: %s", view)
	}
}
