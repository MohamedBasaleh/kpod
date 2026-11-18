package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

type kvField struct {
	Label string
	Value string
	Paint func(string) string
}

type kvLayout struct {
	Columns int
	LabelW  int
	Gap     int
	Gutter  int
}

func kvLayoutFor(width int) kvLayout {
	if width < 1 {
		width = 80
	}
	switch {
	case width >= 88:
		return kvLayout{Columns: 2, LabelW: 12, Gap: 3, Gutter: 4}
	case width >= 64:
		return kvLayout{Columns: 2, LabelW: 11, Gap: 2, Gutter: 2}
	default:
		return kvLayout{Columns: 1, LabelW: 12, Gap: 3, Gutter: 0}
	}
}

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

func formatKV(f kvField, labelW, gap, valueW int) string {
	if valueW < 1 {
		valueW = 1
	}
	lab := pad(f.Label, labelW)
	vis := ellipsize(f.Value, valueW)
	padN := valueW - runeLen(vis)
	if padN < 0 {
		padN = 0
	}
	shown := vis
	if f.Paint != nil {
		shown = f.Paint(vis)
	}
	return lab + strings.Repeat(" ", gap) + shown + strings.Repeat(" ", padN)
}

func columnWidth(width int, layout kvLayout) int {
	if layout.Columns < 2 {
		return width
	}
	w := (width - layout.Gutter) / 2
	if w < layout.LabelW+layout.Gap+6 {
		return width
	}
	return w
}

func valueWidth(colW int, layout kvLayout) int {
	w := colW - layout.LabelW - layout.Gap
	if w < 4 {
		return 4
	}
	return w
}

func renderKVGrid(left, right []kvField, width int) string {
	layout := kvLayoutFor(width)
	colW := columnWidth(width, layout)
	if layout.Columns == 2 && colW >= width {
		layout.Columns = 1
		colW = width
	}
	vw := valueWidth(colW, layout)
	if layout.Columns < 2 {
		return renderKVStack(appendKV(left, right), layout, vw)
	}
	var b strings.Builder
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	for i := 0; i < n; i++ {
		var lf, rf kvField
		if i < len(left) {
			lf = left[i]
		}
		if i < len(right) {
			rf = right[i]
		}
		leftCol := strings.Repeat(" ", colW)
		if lf.Label != "" || lf.Value != "" {
			leftCol = padVisual(formatKV(lf, layout.LabelW, layout.Gap, vw), colW)
		}
		rightCol := ""
		if rf.Label != "" || rf.Value != "" {
			rightCol = formatKV(rf, layout.LabelW, layout.Gap, vw)
		}
		line := strings.TrimRight(leftCol+strings.Repeat(" ", layout.Gutter)+rightCol, " ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func renderKVStack(fields []kvField, layout kvLayout, valueW int) string {
	var b strings.Builder
	for _, f := range fields {
		if f.Label == "" && f.Value == "" {
			continue
		}
		b.WriteString(strings.TrimRight(formatKV(f, layout.LabelW, layout.Gap, valueW), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

func padVisual(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func appendKV(left, right []kvField) []kvField {
	out := make([]kvField, 0, len(left)+len(right))
	out = append(out, left...)
	out = append(out, right...)
	return out
}

func joinNames(names []string) string {
	if len(names) == 0 {
		return "—"
	}
	return strings.Join(names, ", ")
}

func statusPlain(status string) string {
	if status == "" {
		return "—"
	}
	if noColor() {
		return status
	}
	return "● " + status
}

func statusPaint(status string) func(string) string {
	if noColor() {
		return nil
	}
	return func(visible string) string {
		return statusStyle(status).Render(visible)
	}
}
