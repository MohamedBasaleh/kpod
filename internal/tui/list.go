package tui

import "strings"

type checklist struct {
	items    []string
	selected map[string]bool
	cursor   int
	offset   int
}

func newChecklist(items []string) checklist {
	return checklist{
		items:    append([]string{}, items...),
		selected: map[string]bool{},
	}
}

func (c *checklist) len() int { return len(c.items) }

func (c *checklist) current() string {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return ""
	}
	return c.items[c.cursor]
}

func (c *checklist) move(delta int) {
	if len(c.items) == 0 {
		return
	}
	c.cursor += delta
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor >= len(c.items) {
		c.cursor = len(c.items) - 1
	}
}

func (c *checklist) toggle() {
	item := c.current()
	if item == "" {
		return
	}
	c.selected[item] = !c.selected[item]
}

func (c *checklist) setSelected(item string, on bool) {
	c.selected[item] = on
}

func (c *checklist) isSelected(item string) bool {
	return c.selected[item]
}

func (c *checklist) selectedItems() []string {
	out := make([]string, 0, len(c.items))
	for _, item := range c.items {
		if c.selected[item] {
			out = append(out, item)
		}
	}
	return out
}

func (c *checklist) view(width, height int) string {
	if len(c.items) == 0 {
		return styleMuted.Render("Nothing to show.")
	}
	if height < 1 {
		height = 8
	}
	if c.cursor < c.offset {
		c.offset = c.cursor
	}
	if c.cursor >= c.offset+height {
		c.offset = c.cursor - height + 1
	}
	var b strings.Builder
	end := c.offset + height
	if end > len(c.items) {
		end = len(c.items)
	}
	nameWidth := width - 10
	if nameWidth < 8 {
		nameWidth = 8
	}
	for i := c.offset; i < end; i++ {
		item := c.items[i]
		mark := "[ ]"
		if c.selected[item] {
			mark = "[x]"
		}
		prefix := "  "
		line := prefix + mark + " " + ellipsize(item, nameWidth)
		if i == c.cursor {
			line = styleCursor.Render("❯ " + mark + " " + ellipsize(item, nameWidth))
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

type menu struct {
	items  []string
	cursor int
}

func (m *menu) move(delta int) {
	if len(m.items) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
}

func (m menu) current() string {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return ""
	}
	return m.items[m.cursor]
}

func (m menu) view() string {
	var b strings.Builder
	for i, item := range m.items {
		prefix := "  "
		if i == m.cursor {
			prefix = "❯ "
			b.WriteString(styleCursor.Render(prefix + item))
		} else {
			b.WriteString(prefix + item)
		}
		if i < len(m.items)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
