package tui

import "strings"

func pad(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) > width {
		return ellipsize(s, width)
	}
	return s + strings.Repeat(" ", width-len(r))
}

func row(cursor bool, cells ...string) string {
	line := "  " + strings.Join(cells, "  ")
	if cursor {
		return styleCursor.Render("❯ " + strings.Join(cells, "  "))
	}
	return line
}
