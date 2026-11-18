package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 {
		m.width = 80
	}
	var body string
	switch m.screen {
	case screenWelcome:
		body = m.viewWelcome()
	case screenKubectlWait, screenContextsWait, screenNamespacesWait, screenAccessWait, screenSaveWait, screenMigrateWait:
		body = m.viewLoading()
	case screenFatal:
		body = styleErr.Render(m.fatal) + "\n\n" + styleHelp.Render("Press Enter to exit")
	case screenMigrate:
		body = m.viewMigrate()
	case screenMigrateView:
		body = m.viewMigrateMappings()
	case screenMigrateDone:
		body = styleOK.Render("✓ "+m.doneMsg) + "\n\n" + styleHelp.Render("Press Enter to exit")
	case screenContexts:
		body = m.viewContexts()
	case screenNamespaces:
		body = m.viewNamespaces()
	case screenForbidden:
		body = m.viewForbidden()
	case screenManualNS:
		body = m.viewManualNS()
	case screenAccessDenied:
		body = m.viewDenied()
	case screenAlias:
		body = m.viewAlias()
	case screenReview:
		body = m.viewReview()
	case screenDone:
		body = styleOK.Render("✓ "+m.doneMsg) + "\n\n" + styleHelp.Render("Press Enter to continue")
	}
	if m.status != "" && m.screen != screenKubectlWait && m.screen != screenContextsWait &&
		m.screen != screenNamespacesWait && m.screen != screenAccessWait &&
		m.screen != screenSaveWait && m.screen != screenMigrateWait {
		ok := strings.Contains(m.status, "accessible") || strings.HasPrefix(m.status, "✓")
		styled := m.status
		if ok {
			styled = styleOK.Render("✓ " + strings.TrimPrefix(m.status, "✓ "))
		} else if m.screen != screenAccessDenied {
			styled = styleErr.Render(m.status)
		} else {
			styled = styleErr.Render(m.status)
		}
		body = body + "\n\n" + styled
	}
	return renderFrame(m.width, body)
}

func renderFrame(width int, body string) string {
	if width <= 0 {
		width = 80
	}
	// Leave one column empty so a full-width last cell does not wrap and
	// clip the first character of every line in the alt screen.
	if width > 1 {
		width--
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(body)
}

func (m model) viewWelcome() string {
	box := styleBox.Render(styleTitle.Render("kpod") + "\n" + styleMuted.Render("Kubernetes access made simple"))
	return box + "\n\nWelcome to kpod.\n\nLet's set up your Kubernetes environments.\n\n" +
		styleHelp.Render("Press Enter to continue")
}

func (m model) viewLoading() string {
	label := m.status
	if label == "" {
		label = "Working..."
	}
	return m.spinner.View() + " " + label
}

func (m model) viewMigrate() string {
	n := len(m.preview.Mappings)
	var b strings.Builder
	b.WriteString(styleTitle.Render("Existing kpod configuration found") + "\n\n")
	if m.previewErr != nil {
		b.WriteString(styleErr.Render(m.previewErr.Error()) + "\n\n")
	} else {
		b.WriteString(fmt.Sprintf("%d environments:\n\n", n))
		b.WriteString(m.mappingList(8))
		b.WriteString("\n\n")
	}
	b.WriteString(m.migrateMenu.view())
	b.WriteString("\n\n" + styleHelp.Render("j/k  enter  q"))
	return b.String()
}

func (m model) viewMigrateMappings() string {
	return styleTitle.Render("Current mappings") + "\n\n" + m.mappingList(20) + "\n\n" +
		styleHelp.Render("Enter or esc to go back")
}

func (m model) mappingList(limit int) string {
	var b strings.Builder
	for i, mp := range m.preview.Mappings {
		if i >= limit {
			b.WriteString(styleMuted.Render(fmt.Sprintf("  … %d more", len(m.preview.Mappings)-limit)))
			break
		}
		line := fmt.Sprintf("%-12s  %s / %s", mp.Alias, ellipsize(mp.KubeContext, 28), mp.Namespace)
		b.WriteString("  " + line)
		if i < len(m.preview.Mappings)-1 && i < limit-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m model) viewContexts() string {
	title := styleTitle.Render("Select Kubernetes contexts")
	intro := "Choose the clusters you use. Space selects, Enter continues."
	if m.help != "" {
		intro = m.help
	}
	head := title
	if n := m.contexts.len(); n > 0 {
		head = styleMuted.Render(fmt.Sprintf("%d contexts found", n)) + "\n\n" + title
	}
	return head + "\n\n" + intro + "\n\n" +
		m.contexts.view(m.listWidth(), m.listHeight()) + "\n\n" +
		styleHelp.Render("j/k  space  enter  q")
}

func (m model) viewNamespaces() string {
	return styleTitle.Render(ellipsize(m.current, m.listWidth())) + "\n\n" +
		"Select namespaces to use with kpod\n\n" +
		styleMuted.Render(m.help) + "\n\n" +
		m.namespaces.view(m.listWidth(), m.listHeight()) + "\n\n" +
		styleHelp.Render("j/k  space  enter  esc  q")
}

func (m model) viewForbidden() string {
	return styleTitle.Render(ellipsize(m.current, m.listWidth())) + "\n\n" +
		"Your Kubernetes account cannot list namespaces.\n\n" +
		"You can still add a namespace you already have access to.\n\n" +
		m.forbiddenMenu.view() + "\n\n" +
		styleHelp.Render("j/k  enter  esc  q")
}

func (m model) viewManualNS() string {
	return styleTitle.Render(ellipsize(m.current, m.listWidth())) + "\n\n" +
		"Namespace:\n" + m.input.View() + "\n\n" +
		styleHelp.Render("enter to check access  esc back")
}

func (m model) viewDenied() string {
	return styleTitle.Render(ellipsize(m.current, m.listWidth())) + "\n\n" +
		m.deniedMenu.view() + "\n\n" +
		styleHelp.Render("j/k  enter")
}

func (m model) viewAlias() string {
	ns, ctx := "", ""
	if m.pendingI >= 0 && m.pendingI < len(m.pending) {
		ns = m.pending[m.pendingI].Namespace
		ctx = ellipsize(m.pending[m.pendingI].KubeContext, 40)
	}
	step := fmt.Sprintf("Environment %d of %d", m.pendingI+1, len(m.pending))
	return styleTitle.Render(step) + "\n\n" +
		styleMuted.Render(ctx) + "\n\n" +
		"Namespace:\n" + ns + "\n\n" +
		"Environment name:\n" + m.input.View() + "\n\n" +
		styleMuted.Render("The suggestion is editable. Names must be unique.") + "\n\n" +
		styleHelp.Render("enter  esc")
}

func (m model) viewReview() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Review environments") + "\n\n")
	for _, line := range m.draft.ReviewLines() {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("Enter saves  esc back  q quit"))
	return b.String()
}
