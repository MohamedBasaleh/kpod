package tui

import (
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestOpenShortcutsFromDashboard(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.width = 80
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = next.(mainModel)
	if m.current.view != viewShortcuts {
		t.Fatalf("%+v", m.current)
	}
	view := m.View()
	if !strings.Contains(view, "Shortcuts") || !strings.Contains(view, "Add shortcut") {
		t.Fatalf("%s", view)
	}
}

func TestShortcutAddCancelWritesNothing(t *testing.T) {
	dir := t.TempDir()
	m := newMainModel(config.NewPaths(dir), nil, nil, testCfg(), false, nil, false)
	next, _ := m.openShortcuts()
	m = next.(mainModel)
	next, _ = m.activateShortcut() // Add shortcut is first
	m = next.(mainModel)
	if m.current.view != viewShortcutAddEnv {
		t.Fatalf("%+v", m.current)
	}
	next, _ = m.handleShortcutKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(mainModel)
	if m.current.view != viewShortcuts {
		t.Fatalf("%+v", m.current)
	}
	cfg, err := config.NewRepository(config.NewPaths(dir)).Load()
	if err == nil && len(cfg.Shortcuts) > 0 {
		t.Fatal("cancel must not write shortcuts")
	}
}

func TestShortcutListAndRemove(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg()
	cfg.Shortcuts = map[string]config.Shortcut{"dc": {Environment: "dev", Action: "console"}}
	m := newMainModel(config.NewPaths(dir), nil, nil, cfg, false, nil, false)
	m.width = 80
	next, _ := m.openShortcuts()
	m = next.(mainModel)
	if !strings.Contains(m.View(), "k-dc") || !strings.Contains(m.View(), "kpod dev console") {
		t.Fatalf("%s", m.View())
	}
	m.scMenu.cursor = indexOfMenu(m.scMenu.items, "Remove shortcut")
	next, _ = m.activateShortcut()
	m = next.(mainModel)
	if m.current.view != viewShortcutRemove {
		t.Fatalf("%+v", m.current)
	}
	next, _ = m.handleShortcutKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(mainModel)
	if _, ok := m.cfg.Shortcuts["dc"]; !ok {
		t.Fatal("esc must not remove")
	}
}

func TestShortcutGenerateSelection(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	next, _ := m.openShortcuts()
	m = next.(mainModel)
	m.scMenu.cursor = indexOfMenu(m.scMenu.items, "Generate recommended shortcuts")
	next, _ = m.activateShortcut()
	m = next.(mainModel)
	if m.current.view != viewShortcutGen || len(m.scGen) == 0 {
		t.Fatalf("%+v %d", m.current, len(m.scGen))
	}
	on := 0
	for _, r := range m.scGen {
		if r.Selected {
			on++
		}
	}
	if on != 4 {
		t.Fatalf("defaults %d", on)
	}
	next, _ = m.handleShortcutKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(mainModel)
	if len(m.cfg.Shortcuts) != 0 {
		t.Fatal("cancel generate")
	}
}

func indexOfMenu(items []string, name string) int {
	for i, s := range items {
		if s == name {
			return i
		}
	}
	return 0
}
