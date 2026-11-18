package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorTitle  = lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#93C5FD"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colorOK     = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#86EFAC"}
	colorWarn   = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FCD34D"}
	colorErr    = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FCA5A5"}
	colorAccent = lipgloss.AdaptiveColor{Light: "#0F766E", Dark: "#5EEAD4"}
	colorBorder = lipgloss.AdaptiveColor{Light: "#93C5FD", Dark: "#334155"}

	styleTitle = lipgloss.NewStyle().Foreground(colorTitle).Bold(true)
	styleMuted = lipgloss.NewStyle().Foreground(colorMuted)
	styleOK    = lipgloss.NewStyle().Foreground(colorOK)
	styleWarn  = lipgloss.NewStyle().Foreground(colorWarn)
	styleErr   = lipgloss.NewStyle().Foreground(colorErr)
	styleHelp  = lipgloss.NewStyle().Foreground(colorMuted)
	styleBox   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 4).
			Align(lipgloss.Center)
	styleWelcome = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 2).
			Align(lipgloss.Center)
	styleCursor = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
)

func ellipsize(s string, max int) string {
	if max <= 1 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
