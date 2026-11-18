package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	tea "github.com/charmbracelet/bubbletea"
)

func nsBrowser(t *testing.T) (mainModel, string) {
	t.Helper()
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	cfg := testCfg()
	if err := config.NewRepository(paths).Save(cfg); err != nil {
		t.Fatal(err)
	}
	kubeconfig := filepath.Join(dir, "kubeconfig")
	if err := os.WriteFile(kubeconfig, []byte("kind: Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newMainModel(paths, nil, nil, cfg, false, nil, false)
	m.width = 80
	m.finishWelcome()
	m.current = navFrame{view: viewNamespaces, kubeContext: "Nawaqis-Dev"}
	m.nsContext = "Nawaqis-Dev"
	m.nsRows = []nsRow{
		{Name: "dev-core", Env: "dev"},
		{Name: "dev-inventory", Env: ""},
		{Name: "jobs", Env: ""},
		{Name: "staging-core", Env: "staging"},
	}
	m.nsCursor = 1
	return m, kubeconfig
}

func TestMappedNamespaceEnterOpensEnvironment(t *testing.T) {
	m, _ := nsBrowser(t)
	m.nsCursor = 0
	next, _ := m.open()
	m = next.(mainModel)
	if m.current.view != viewEnv || m.current.envAlias != "dev" {
		t.Fatalf("%+v", m.current)
	}
	if strings.Contains(m.status, "is a namespace, not a kpod environment") {
		t.Fatal(m.status)
	}
}

func TestUnmappedNamespaceEnterOpensAddFlow(t *testing.T) {
	m, _ := nsBrowser(t)
	next, _ := m.open()
	m = next.(mainModel)
	if m.current.view != viewNsAdd {
		t.Fatalf("view=%v", m.current.view)
	}
	if m.nsAddContext != "Nawaqis-Dev" || m.nsAddNamespace != "dev-inventory" {
		t.Fatalf("%s %s", m.nsAddContext, m.nsAddNamespace)
	}
	if !strings.Contains(m.View(), "Add namespace to kpod") {
		t.Fatalf("%s", m.View())
	}
	if strings.Contains(m.status, "is a namespace, not a kpod environment") {
		t.Fatal(m.status)
	}
}

func TestNsAddSuggestedAlias(t *testing.T) {
	m, _ := nsBrowser(t)
	next, _ := m.startNsAdd()
	m = next.(mainModel)
	got := strings.TrimSpace(m.nsAddIn.Value())
	if got != "inventory" {
		t.Fatalf("got %q want inventory", got)
	}
	if onboard.SuggestAlias("dev-inventory", m.cfg.TakenAliases()) != "inventory" {
		t.Fatal(got)
	}
}

func TestNsAddEditedAliasAccepted(t *testing.T) {
	m, kubeconfig := nsBrowser(t)
	next, _ := m.startNsAdd()
	m = next.(mainModel)
	m.nsAddIn.SetValue("stock")
	next, _ = m.confirmNsAddAlias()
	m = next.(mainModel)
	if m.current.view != viewNsAddConfirm {
		t.Fatalf("%v", m.current.view)
	}
	next, cmd := m.commitNsAdd()
	if cmd == nil {
		t.Fatal("expected save")
	}
	msg := cmd()
	saved, ok := msg.(nsAddedMsg)
	if !ok || saved.err != nil {
		t.Fatalf("%T %+v", msg, saved)
	}
	next, _ = next.(mainModel).onNsAdded(saved)
	m = next.(mainModel)
	if m.cfg.AliasForNamespace("Nawaqis-Dev", "dev-inventory") != "stock" {
		t.Fatal(m.cfg)
	}
	body, err := os.ReadFile(kubeconfig)
	if err != nil || string(body) != "kind: Config\n" {
		t.Fatalf("kubeconfig mutated: %s %v", body, err)
	}
}

func TestNsAddDuplicateAliasRejected(t *testing.T) {
	m, _ := nsBrowser(t)
	next, _ := m.startNsAdd()
	m = next.(mainModel)
	m.nsAddIn.SetValue("dev")
	next, cmd := m.confirmNsAddAlias()
	m = next.(mainModel)
	if cmd != nil {
		t.Fatal("must not save")
	}
	if m.current.view != viewNsAdd {
		t.Fatalf("%v", m.current.view)
	}
	if !strings.Contains(m.status, "already used") {
		t.Fatalf("%q", m.status)
	}
}

func TestNsAddCancelLeavesConfig(t *testing.T) {
	m, _ := nsBrowser(t)
	before, err := os.ReadFile(m.paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	next, _ := m.startNsAdd()
	m = next.(mainModel)
	next, _ = m.handleNsAddKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(mainModel)
	if m.current.view != viewNamespaces {
		t.Fatalf("%v", m.current.view)
	}
	after, err := os.ReadFile(m.paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("config changed")
	}
}

func TestNsAddPersistsAndRefreshesBrowser(t *testing.T) {
	m, _ := nsBrowser(t)
	next, _ := m.startNsAdd()
	m = next.(mainModel)
	next, _ = m.confirmNsAddAlias()
	m = next.(mainModel)
	if !strings.Contains(m.View(), "Create environment?") {
		t.Fatalf("%s", m.View())
	}
	next, cmd := m.handleNsAddConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("save")
	}
	saved := cmd().(nsAddedMsg)
	if saved.err != nil {
		t.Fatal(saved.err)
	}
	next, _ = next.(mainModel).onNsAdded(saved)
	m = next.(mainModel)
	if m.current.view != viewNamespaces {
		t.Fatalf("%v", m.current.view)
	}
	if m.cfg.AliasForNamespace("Nawaqis-Dev", "dev-inventory") != "inventory" {
		t.Fatal("not persisted in memory")
	}
	loaded, err := config.NewRepository(m.paths).Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AliasForNamespace("Nawaqis-Dev", "dev-inventory") != "inventory" {
		t.Fatal("not on disk")
	}
	found := false
	for i, row := range m.nsRows {
		if row.Name == "dev-inventory" {
			found = true
			if row.Env != "inventory" {
				t.Fatalf("column %q", row.Env)
			}
			if m.nsCursor != i {
				t.Fatalf("selection %d want %d", m.nsCursor, i)
			}
		}
	}
	if !found {
		t.Fatal("row missing")
	}
	if strings.Contains(m.View(), "is a namespace, not a kpod environment") {
		t.Fatal(m.View())
	}
}

func TestNsAddSameContextMultipleNamespaces(t *testing.T) {
	m, _ := nsBrowser(t)
	next, _ := m.startNsAdd()
	m = next.(mainModel)
	next, _ = m.confirmNsAddAlias()
	m = next.(mainModel)
	next, cmd := m.commitNsAdd()
	saved := cmd().(nsAddedMsg)
	next, _ = next.(mainModel).onNsAdded(saved)
	m = next.(mainModel)
	m.nsCursor = 2
	next, _ = m.startNsAdd()
	m = next.(mainModel)
	if m.nsAddNamespace != "jobs" {
		t.Fatalf("%s", m.nsAddNamespace)
	}
	next, _ = m.confirmNsAddAlias()
	m = next.(mainModel)
	next, cmd = m.commitNsAdd()
	saved = cmd().(nsAddedMsg)
	if saved.err != nil {
		t.Fatal(saved.err)
	}
	next, _ = next.(mainModel).onNsAdded(saved)
	m = next.(mainModel)
	if m.cfg.AliasForNamespace("Nawaqis-Dev", "dev-inventory") != "inventory" {
		t.Fatal("inventory")
	}
	if m.cfg.AliasForNamespace("Nawaqis-Dev", "jobs") != "jobs" {
		t.Fatal("jobs")
	}
}

func TestNsAddAKeyOnMappedOpens(t *testing.T) {
	m, _ := nsBrowser(t)
	m.nsCursor = 0
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = next.(mainModel)
	if m.current.view != viewEnv || m.current.envAlias != "dev" {
		t.Fatalf("%+v", m.current)
	}
}

func TestNsHelpNarrow(t *testing.T) {
	m, _ := nsBrowser(t)
	m.width = 40
	if m.nsHelp() != "enter  a  /  r  esc" {
		t.Fatal(m.nsHelp())
	}
	m.width = 80
	if !strings.Contains(m.nsHelp(), "enter open/add") {
		t.Fatal(m.nsHelp())
	}
}
