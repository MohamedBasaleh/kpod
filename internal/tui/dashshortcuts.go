package tui

import (
	"fmt"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/shortcut"
	tea "github.com/charmbracelet/bubbletea"
)

func shortcutMenuItems(cfg config.Config) []string {
	items := []string{"Add shortcut", "Generate recommended shortcuts", "Regenerate shell file"}
	if cfg.Settings.ShellShortcuts {
		items = append(items, "Disable shortcuts")
	} else {
		items = append(items, "Enable shortcuts")
	}
	if len(cfg.Shortcuts) > 0 {
		items = append(items, "Remove shortcut")
	}
	return append(items, "Back")
}

func shortcutActionChoices() []action.Spec {
	var safe, danger []action.Spec
	for _, s := range action.All() {
		if s.Destructive {
			danger = append(danger, s)
			continue
		}
		safe = append(safe, s)
	}
	return append(safe, danger...)
}

func (m mainModel) openShortcuts() (tea.Model, tea.Cmd) {
	m.push(navFrame{view: viewShortcuts})
	m.scMenu = menu{items: shortcutMenuItems(m.cfg)}
	return m, nil
}

func (m mainModel) shortcutViews() bool {
	switch m.current.view {
	case viewShortcuts, viewShortcutAddEnv, viewShortcutAddAction, viewShortcutID, viewShortcutGen, viewShortcutRC, viewShortcutRemove:
		return true
	default:
		return false
	}
}

func (m *mainModel) moveShortcut(delta int) {
	switch m.current.view {
	case viewShortcuts:
		m.scMenu.move(delta)
	case viewShortcutAddEnv:
		m.scAddEnvCursor = clamp(m.scAddEnvCursor+delta, 0, len(m.envs)-1)
	case viewShortcutRemove:
		m.scAddEnvCursor = clamp(m.scAddEnvCursor+delta, 0, len(shortcut.SortedIDs(m.cfg))-1)
	case viewShortcutAddAction:
		m.scAddActCursor = clamp(m.scAddActCursor+delta, 0, len(shortcutActionChoices())-1)
	case viewShortcutGen:
		m.scGenCursor = clamp(m.scGenCursor+delta, 0, len(m.scGen)-1)
	case viewShortcutRC:
		m.confirmCursor = clamp(m.confirmCursor+delta, 0, 1)
	}
}

func (m mainModel) handleShortcutKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.current.view == viewShortcutID {
		return m.handleShortcutIDKey(msg)
	}
	switch msg.String() {
	case "up":
		m.moveShortcut(-1)
		return m, nil
	case "down", "j":
		m.moveShortcut(1)
		return m, nil
	case "esc":
		if m.pop() {
			return m, nil
		}
		return m, nil
	case " ":
		if m.current.view == viewShortcutGen && m.scGenCursor >= 0 && m.scGenCursor < len(m.scGen) {
			m.scGen[m.scGenCursor].Selected = !m.scGen[m.scGenCursor].Selected
		}
		return m, nil
	case "enter":
		return m.activateShortcut()
	}
	return m, nil
}

func (m mainModel) handleShortcutIDKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pop()
		return m, nil
	case "enter":
		id := strings.TrimSpace(m.promptInput.Value())
		return m.saveNewShortcut(id)
	}
	var cmd tea.Cmd
	m.promptInput, cmd = m.promptInput.Update(msg)
	return m, cmd
}

func (m mainModel) activateShortcut() (tea.Model, tea.Cmd) {
	switch m.current.view {
	case viewShortcuts:
		switch m.scMenu.current() {
		case "Add shortcut":
			m.push(navFrame{view: viewShortcutAddEnv})
			m.scAddEnvCursor = 0
			return m, nil
		case "Generate recommended shortcuts":
			m.push(navFrame{view: viewShortcutGen})
			m.scGen = recommendedRows(m.cfg)
			m.scGenCursor = 0
			return m, nil
		case "Regenerate shell file":
			return m.regenerateShortcuts()
		case "Enable shortcuts":
			return m.enableShortcuts()
		case "Disable shortcuts":
			return m.disableShortcuts()
		case "Remove shortcut":
			return m.removeSelectedShortcut()
		case "Back":
			m.pop()
			return m, nil
		}
	case viewShortcutRemove:
		return m.confirmRemoveShortcut()
	case viewShortcutAddEnv:
		if len(m.envs) == 0 {
			return m, nil
		}
		m.push(navFrame{view: viewShortcutAddAction})
		m.scAddActCursor = 0
		return m, nil
	case viewShortcutAddAction:
		env := m.envs[m.scAddEnvCursor]
		act := shortcutActionChoices()[m.scAddActCursor]
		sug := shortcut.Suggest(env.Alias, act.ID, m.cfg, nil)
		m.push(navFrame{view: viewShortcutID})
		m.promptInput.SetValue(sug)
		m.promptInput.Prompt = "Shortcut id: "
		m.promptInput.Placeholder = sug
		m.promptInput.Focus()
		return m, nil
	case viewShortcutGen:
		return m.applyGenerated()
	case viewShortcutRC:
		if m.confirmCursor != 0 {
			m.pop()
			m.status = "Shortcuts enabled. Run kpod shortcut install to add the source line."
			m.scMenu = menu{items: shortcutMenuItems(m.cfg)}
			return m, nil
		}
		svc := shortcut.New(m.paths)
		if err := svc.InstallRC(true); err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.pop()
		m.status = "Shell source line installed."
		m.scMenu = menu{items: shortcutMenuItems(m.cfg)}
		return m, nil
	}
	return m, nil
}

func (m mainModel) saveNewShortcut(id string) (tea.Model, tea.Cmd) {
	env := m.envs[m.scAddEnvCursor]
	act := shortcutActionChoices()[m.scAddActCursor]
	svc := shortcut.New(m.paths)
	cfg, err := svc.Add(m.cfg, id, env.Alias, string(act.ID))
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if err := svc.Save(cfg); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.cfg = cfg
	if cfg.Settings.ShellShortcuts {
		_ = svc.Generate(cfg)
	}
	for m.current.view != viewShortcuts && m.pop() {
	}
	m.status = "Added " + shortcut.CommandName(id)
	m.scMenu = menu{items: shortcutMenuItems(m.cfg)}
	return m, nil
}

func (m mainModel) applyGenerated() (tea.Model, tea.Cmd) {
	svc := shortcut.New(m.paths)
	cfg := m.cfg
	var err error
	for _, row := range m.scGen {
		if !row.Selected {
			continue
		}
		cfg, err = svc.Add(cfg, row.ID, row.Env, row.Action)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
	}
	if err := svc.Save(cfg); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.cfg = cfg
	if cfg.Settings.ShellShortcuts {
		_ = svc.Generate(cfg)
	}
	m.pop()
	m.status = "Shortcuts saved."
	m.scMenu = menu{items: shortcutMenuItems(m.cfg)}
	return m, nil
}

func (m mainModel) regenerateShortcuts() (tea.Model, tea.Cmd) {
	svc := shortcut.New(m.paths)
	if err := svc.Generate(m.cfg); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "Regenerated " + svc.Paths.AliasFile(string(svc.Shell))
	return m, nil
}

func (m mainModel) enableShortcuts() (tea.Model, tea.Cmd) {
	svc := shortcut.New(m.paths)
	cfg := svc.Enable(m.cfg)
	if err := svc.Save(cfg); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.cfg = cfg
	if err := svc.Generate(cfg); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.push(navFrame{view: viewShortcutRC})
	m.scRCInstall = true
	m.confirmCursor = 0
	return m, nil
}

func (m mainModel) disableShortcuts() (tea.Model, tea.Cmd) {
	svc := shortcut.New(m.paths)
	cfg := svc.Disable(m.cfg)
	if err := svc.Save(cfg); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.cfg = cfg
	m.scMenu = menu{items: shortcutMenuItems(m.cfg)}
	m.status = "Shortcuts disabled. Definitions were kept."
	return m, nil
}

func (m mainModel) removeSelectedShortcut() (tea.Model, tea.Cmd) {
	ids := shortcut.SortedIDs(m.cfg)
	if len(ids) == 0 {
		return m, nil
	}
	m.push(navFrame{view: viewShortcutRemove})
	m.scAddEnvCursor = 0
	return m, nil
}

func (m mainModel) confirmRemoveShortcut() (tea.Model, tea.Cmd) {
	ids := shortcut.SortedIDs(m.cfg)
	if m.scAddEnvCursor < 0 || m.scAddEnvCursor >= len(ids) {
		return m, nil
	}
	id := ids[m.scAddEnvCursor]
	svc := shortcut.New(m.paths)
	cfg, err := svc.Remove(m.cfg, id)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if err := svc.Save(cfg); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.cfg = cfg
	if cfg.Settings.ShellShortcuts {
		_ = svc.Generate(cfg)
	}
	m.scMenu = menu{items: shortcutMenuItems(m.cfg)}
	m.status = "Removed " + shortcut.CommandName(id)
	m.pop()
	return m, nil
}

func recommendedRows(cfg config.Config) []shortcutCandidate {
	var out []shortcutCandidate
	for _, c := range shortcut.Recommend(cfg) {
		out = append(out, shortcutCandidate{
			ID:       c.ID,
			Env:      c.Env,
			Action:   string(c.Action),
			Selected: c.DefaultOn,
			Label:    fmt.Sprintf("k-%s   %s %s", c.ID, c.Env, c.Action),
		})
	}
	return out
}

func (m mainModel) viewShortcuts() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Shortcuts") + "\n\n")
	ids := shortcut.SortedIDs(m.cfg)
	if len(ids) == 0 {
		b.WriteString(styleMuted.Render("No shortcuts yet.") + "\n\n")
	} else {
		b.WriteString(styleMuted.Render("  "+pad("NAME", 10)+"  COMMAND") + "\n")
		for _, id := range ids {
			sc := m.cfg.Shortcuts[id]
			line := pad(shortcut.CommandName(id), 10) + "  " + shortcut.Invocation(sc)
			if err := shortcut.CheckRefs(m.cfg, id, sc); err != nil {
				line += "  " + styleWarn.Render("missing ref")
			}
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
	}
	enabled := "disabled"
	if m.cfg.Settings.ShellShortcuts {
		enabled = "enabled"
	}
	b.WriteString(styleMuted.Render("Integration: "+enabled) + "\n\n")
	b.WriteString(m.scMenu.view())
	if m.status != "" {
		b.WriteString("\n\n" + styleMuted.Render(m.status))
	}
	b.WriteString("\n\n" + styleHelp.Render("enter select   esc back"))
	return b.String()
}

func (m mainModel) viewShortcutAddEnv() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Add shortcut") + "\n\nSelect environment\n\n")
	for i, env := range m.envs {
		b.WriteString(row(i == m.scAddEnvCursor, env.Alias) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter continue   esc back"))
	return b.String()
}

func (m mainModel) viewShortcutAddAction() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Add shortcut") + "\n\nSelect action\n\n")
	for i, spec := range shortcutActionChoices() {
		label := spec.Label
		if spec.Destructive {
			label += "  (confirmation still required)"
		}
		b.WriteString(row(i == m.scAddActCursor, label) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter continue   esc back"))
	return b.String()
}

func (m mainModel) viewShortcutID() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Add shortcut") + "\n\n")
	b.WriteString("Command will be k-<id>\n\n")
	b.WriteString(m.promptInput.View() + "\n")
	if m.status != "" {
		b.WriteString("\n" + styleMuted.Render(m.status) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter save   esc cancel"))
	return b.String()
}

func (m mainModel) viewShortcutGen() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Recommended shortcuts") + "\n\n")
	for i, row := range m.scGen {
		mark := "[ ]"
		if row.Selected {
			mark = "[x]"
		}
		line := mark + "  " + row.Label
		b.WriteString(rowLine(i == m.scGenCursor, line) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("space toggle   enter save   esc cancel"))
	return b.String()
}

func rowLine(cursor bool, line string) string {
	if cursor {
		return styleCursor.Render("❯ " + line)
	}
	return "  " + line
}

func (m mainModel) viewShortcutRC() string {
	svc := shortcut.New(m.paths)
	_, _, snippet := shortcut.InstallPlan(m.paths, svc.Home, svc.Shell)
	var b strings.Builder
	b.WriteString(styleTitle.Render("Install shell source?") + "\n\n")
	fmt.Fprintf(&b, "kpod needs to add one line to %s:\n\n%s\n", shortcut.RCPath(svc.Home, svc.Shell), snippet)
	b.WriteString(row(m.confirmCursor == 0, "Yes") + "\n")
	b.WriteString(row(m.confirmCursor == 1, "No") + "\n")
	b.WriteString("\n" + styleHelp.Render("enter confirms selection   esc cancel"))
	return b.String()
}

func (m mainModel) viewShortcutRemove() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Remove shortcut") + "\n\n")
	ids := shortcut.SortedIDs(m.cfg)
	for i, id := range ids {
		sc := m.cfg.Shortcuts[id]
		b.WriteString(row(i == m.scAddEnvCursor, shortcut.CommandName(id)+"  "+shortcut.Invocation(sc)) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter remove   esc cancel"))
	return b.String()
}
