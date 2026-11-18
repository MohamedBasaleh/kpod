package tui

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/MohamedBasaleh/kpod/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func withColor(t *testing.T) {
	t.Helper()
	orig, ok := os.LookupEnv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	t.Cleanup(func() {
		if ok {
			os.Setenv("NO_COLOR", orig)
		}
	})
}

func TestWelcomeCopyFitsWidth(t *testing.T) {
	if welcomeCopy(80) != welcomeFull {
		t.Fatal(welcomeCopy(80))
	}
	if welcomeCopy(24) != welcomeShort {
		t.Fatal(welcomeCopy(24))
	}
	if welcomeCopy(8) != welcomeMini {
		t.Fatal(welcomeCopy(8))
	}
}

func TestTypePrefix(t *testing.T) {
	if typePrefix(welcomeFull, 0) != "" {
		t.Fatal("empty")
	}
	got := typePrefix(welcomeFull, 7)
	if got != "Welcome" {
		t.Fatalf("%q", got)
	}
	if typePrefix(welcomeFull, 500) != welcomeFull {
		t.Fatal("cap")
	}
}

func TestDashboardWelcomeShowsEnvironmentsDuringAnim(t *testing.T) {
	withColor(t)
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.width = 80
	m.welcomeN = 8
	view := m.View()
	if !strings.Contains(view, "dev") || !strings.Contains(view, "staging") {
		t.Fatalf("animation must not hide the list:\n%s", view)
	}
	if strings.Contains(view, welcomeFull) {
		t.Fatal("partial type should not show the full line")
	}
}

func TestWelcomeCompletesOnce(t *testing.T) {
	withColor(t)
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.width = 80
	total := utf8.RuneCountInString(welcomeFull)
	steps := 0
	for !m.welcomeDone {
		steps++
		if !m.advanceWelcome() {
			break
		}
		if steps > 50 {
			t.Fatal("animation did not finish quickly")
		}
	}
	if m.welcomeN < total {
		t.Fatalf("n=%d total=%d", m.welcomeN, total)
	}
	view := m.View()
	if !strings.Contains(view, welcomeFull) {
		t.Fatalf("%s", view)
	}
}

func TestWelcomeSkippedOnReturnToDashboard(t *testing.T) {
	withColor(t)
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.width = 80
	m.finishWelcome()
	next, _ := m.open()
	m = next.(mainModel)
	if m.current.view != viewEnv {
		t.Fatal(m.current)
	}
	m.pop()
	if m.current.view != viewDashboard {
		t.Fatal(m.current)
	}
	if !m.welcomeDone {
		t.Fatal("must not restart")
	}
	if cmd := m.maybeWelcomeCmd(); cmd != nil {
		t.Fatal("no further ticks after first play")
	}
	if !strings.Contains(m.View(), welcomeFull) {
		t.Fatalf("%s", m.View())
	}
}

func TestWelcomeKeyDoesNotBlockNavigation(t *testing.T) {
	withColor(t)
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.width = 80
	m.envCursor = 0
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(mainModel)
	if !m.welcomeDone {
		t.Fatal("key should snap the header")
	}
	if m.current.view != viewEnv {
		t.Fatalf("enter must still open env: %+v", m.current)
	}
}

func TestWelcomeRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.width = 80
	if !m.welcomeDone {
		t.Fatal("NO_COLOR must skip animation")
	}
	view := m.View()
	if !strings.Contains(view, welcomeFull) {
		t.Fatalf("%s", view)
	}
	if strings.Contains(view, "╭") || strings.Contains(view, "╮") {
		t.Fatalf("NO_COLOR should not use a decorative box: %s", view)
	}
}

func TestWelcomeResizeUsesShorterCopy(t *testing.T) {
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	m.finishWelcome()
	m.width = 24
	view := m.View()
	if !strings.Contains(view, welcomeShort) {
		t.Fatalf("%s", view)
	}
	if strings.Contains(view, "Kubernetes access") {
		t.Fatal("narrow terminal should drop the tagline")
	}
}

func TestWelcomeInitStartsTickOnDashboard(t *testing.T) {
	withColor(t)
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected tick")
	}
}
