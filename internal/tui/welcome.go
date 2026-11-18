package tui

import (
	"os"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	welcomeFull  = "Welcome to kpod — Kubernetes access made simple"
	welcomeShort = "Welcome to kpod"
	welcomeMini  = "kpod"
	welcomeTick  = 12 * time.Millisecond
	welcomeChunk = 4
)

type welcomeTickMsg struct{}

func noColor() bool {
	return os.Getenv("NO_COLOR") != ""
}

func welcomeCopy(width int) string {
	switch {
	case width < 12:
		return welcomeMini
	case width < 40:
		return welcomeShort
	default:
		return welcomeFull
	}
}

func typePrefix(text string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(text)
	if n >= len(r) {
		return text
	}
	return string(r[:n])
}

func welcomeTickCmd() tea.Cmd {
	return tea.Tick(welcomeTick, func(time.Time) tea.Msg { return welcomeTickMsg{} })
}

func (m mainModel) maybeWelcomeCmd() tea.Cmd {
	if m.current.view != viewDashboard || m.welcomeDone || m.skipWelcomeAnim() {
		return nil
	}
	return welcomeTickCmd()
}

func (m *mainModel) skipWelcomeAnim() bool {
	return noColor()
}

func (m *mainModel) finishWelcome() {
	m.welcomeDone = true
	m.welcomeStarted = true
	copy := welcomeCopy(m.headerWidth())
	m.welcomeN = utf8.RuneCountInString(copy)
}

func (m mainModel) headerWidth() int {
	w := m.width
	if w <= 0 {
		w = 80
	}
	return w
}

func (m *mainModel) advanceWelcome() bool {
	if m.welcomeDone || m.skipWelcomeAnim() {
		m.finishWelcome()
		return false
	}
	if m.current.view != viewDashboard {
		m.finishWelcome()
		return false
	}
	m.welcomeStarted = true
	copy := welcomeCopy(m.headerWidth())
	total := utf8.RuneCountInString(copy)
	m.welcomeN += welcomeChunk
	if m.welcomeN >= total {
		m.finishWelcome()
		return false
	}
	return true
}

func (m mainModel) welcomeVisible() string {
	copy := welcomeCopy(m.headerWidth())
	if m.welcomeDone || m.skipWelcomeAnim() {
		return copy
	}
	return typePrefix(copy, m.welcomeN)
}

func (m mainModel) dashboardHeader() string {
	text := m.welcomeVisible()
	if strings.TrimSpace(text) == "" {
		text = " "
	}
	if m.skipWelcomeAnim() {
		return text
	}
	inner := styleTitle.Render(text)
	if !m.welcomeDone {
		inner = styleMuted.Render(text)
	}
	w := m.headerWidth()
	box := styleWelcome.MaxWidth(w)
	return box.Render(inner)
}

func (m mainModel) header() string {
	if m.current.view == viewDashboard && !m.showHelp {
		return m.dashboardHeader()
	}
	if m.skipWelcomeAnim() {
		return "kpod"
	}
	return styleBox.Render(styleTitle.Render("kpod"))
}
