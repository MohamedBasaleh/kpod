package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var ansiSeq = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiSeq.ReplaceAllString(s, "")
}

func contextPickerAt(t *testing.T, width int) model {
	t.Helper()
	m := newModel(config.NewPaths(t.TempDir()), onboard.ModeFresh, nil, nil)
	m.width = width
	m.height = 24
	m.screen = screenContexts
	m.help = "Space toggles selection. Enter continues."
	m.contexts = newChecklist([]string{
		"arn:aws:eks:eu-central-1:851725235060:cluster/Nawaqis-dev",
		"dev",
		"prod",
		"staging",
		"arn:aws:eks:eu-central-1:851725235060:cluster/Nawaqis-prod",
	})
	return m
}

func TestContextPickerShowsFullCopy(t *testing.T) {
	view := stripANSI(contextPickerAt(t, 80).View())
	for _, want := range []string{
		"Select Kubernetes contexts",
		"5 contexts found",
		"Space toggles selection. Enter continues.",
		"dev",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in:\n%s", want, view)
		}
	}
}

func TestContextPickerCopyAtWidths(t *testing.T) {
	phrases := []string{
		"Select Kubernetes contexts",
		"Space toggles selection. Enter continues.",
		"5 contexts found",
	}
	for _, width := range []int{40, 64, 80, 120, 160, 200} {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			m := contextPickerAt(t, width)
			view := m.View()
			plain := stripANSI(view)
			frame := width - 1
			if width <= 1 {
				frame = width
			}
			for _, line := range strings.Split(view, "\n") {
				if lipgloss.Width(line) > frame {
					t.Fatalf("line %q is %d cells (max %d)", line, lipgloss.Width(line), frame)
				}
			}
			for _, phrase := range phrases {
				assertPhraseVisible(t, width, plain, phrase)
			}
			if strings.Contains(plain, "elect Kubernetes contexts") && !strings.Contains(plain, "Select Kubernetes contexts") {
				t.Fatalf("clipped title:\n%s", plain)
			}
			if strings.Contains(plain, "pace toggles") && !strings.Contains(plain, "Space toggles") {
				t.Fatalf("clipped help:\n%s", plain)
			}
		})
	}
}

func assertPhraseVisible(t *testing.T, width int, plain, phrase string) {
	t.Helper()
	frame := width - 1
	if width <= 1 {
		frame = width
	}
	if len(phrase) < frame && !strings.Contains(plain, phrase) {
		t.Fatalf("width %d missing %q in:\n%s", width, phrase, plain)
	}
	first := strings.Fields(phrase)[0]
	if !strings.Contains(plain, first) {
		t.Fatalf("width %d missing first word %q of %q in:\n%s", width, first, phrase, plain)
	}
}

func TestRenderFrameKeepsFirstCharacter(t *testing.T) {
	body := "Select Kubernetes contexts\nSpace toggles selection. Enter continues.\n5 contexts found"
	for _, width := range []int{40, 64, 80, 120, 160, 200} {
		got := stripANSI(renderFrame(width, body))
		if !strings.Contains(got, "Select") {
			t.Fatalf("width %d clipped title:\n%s", width, got)
		}
		if !strings.Contains(got, "Space") {
			t.Fatalf("width %d clipped help:\n%s", width, got)
		}
		if !strings.Contains(got, "5 contexts found") {
			t.Fatalf("width %d clipped count:\n%s", width, got)
		}
		frame := width - 1
		if width <= 1 {
			frame = width
		}
		for _, line := range strings.Split(got, "\n") {
			if lipgloss.Width(line) > frame {
				t.Fatalf("width %d line too wide (%d): %q", width, lipgloss.Width(line), line)
			}
		}
	}
}

func TestContextPickerKeys(t *testing.T) {
	m := contextPickerAt(t, 80)
	if m.screen != screenContexts {
		t.Fatalf("setup: screen=%v", m.screen)
	}
	if m.contexts.len() != 5 {
		t.Fatalf("setup: items=%d", m.contexts.len())
	}
	next, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if cmd != nil {
		t.Fatalf("j should not quit: %v", cmd)
	}
	m = next.(model)
	if m.contexts.cursor != 1 {
		t.Fatalf("j should move down, cursor=%d screen=%v key=%q", m.contexts.cursor, m.screen, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}.String())
	}
	next, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = next.(model)
	if m.contexts.cursor != 0 {
		t.Fatalf("k should move up, cursor=%d", m.contexts.cursor)
	}
	next, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = next.(model)
	if !m.contexts.isSelected("arn:aws:eks:eu-central-1:851725235060:cluster/Nawaqis-dev") {
		t.Fatal("space should select the current context")
	}
	if m.screen != screenContexts {
		t.Fatalf("space must not leave onboarding picker: %v", m.screen)
	}
}

func TestContextPickerNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	view := stripANSI(contextPickerAt(t, 80).View())
	if !strings.Contains(view, "Select Kubernetes contexts") {
		t.Fatalf("%s", view)
	}
	if !strings.Contains(view, "5 contexts found") {
		t.Fatalf("%s", view)
	}
}

func TestDashboardStillRendersAtWidths(t *testing.T) {
	for _, width := range []int{40, 64, 80, 120, 160} {
		m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, nil, false)
		m.width = width
		m.finishWelcome()
		view := stripANSI(m.View())
		if !strings.Contains(view, "dev") || !strings.Contains(view, "staging") {
			t.Fatalf("width %d missing environments:\n%s", width, view)
		}
	}
}
