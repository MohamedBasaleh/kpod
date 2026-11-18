package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/discover"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/charmbracelet/lipgloss"
)

func (m mainModel) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.showHelp {
		return m.header() + "\n\n" + m.viewHelp()
	}
	var body string
	switch m.current.view {
	case viewDashboard:
		body = m.viewDashboard()
	case viewEnv:
		body = m.viewEnv()
	case viewPods:
		body = m.viewPods()
	case viewPodDetail:
		body = m.viewPodDetail()
	case viewContexts, viewNsPickContext:
		body = m.viewContexts()
	case viewContextDetail:
		body = m.viewContextDetail()
	case viewNamespaces:
		body = m.viewNamespaces()
	case viewNsForbidden:
		body = m.viewNsForbidden()
	case viewError:
		body = styleErr.Render(m.status)
	case viewContainerSelect:
		body = m.viewContainerSelect()
	case viewConfirm:
		body = m.viewConfirm()
	case viewPrompt:
		body = m.viewPrompt()
	case viewLogsMenu:
		body = m.viewLogsMenu()
	case viewOutput:
		body = m.viewOutput()
	case viewShortcuts:
		body = m.viewShortcuts()
	case viewShortcutAddEnv:
		body = m.viewShortcutAddEnv()
	case viewShortcutAddAction:
		body = m.viewShortcutAddAction()
	case viewShortcutID:
		body = m.viewShortcutID()
	case viewShortcutGen:
		body = m.viewShortcutGen()
	case viewShortcutRC:
		body = m.viewShortcutRC()
	case viewShortcutRemove:
		body = m.viewShortcutRemove()
	case viewUpdates:
		body = m.viewUpdates()
	case viewDiscoverItem:
		body = m.viewDiscoverItem()
	case viewDiscoverNSPick:
		body = m.viewDiscoverNSPick()
	case viewDiscoverAlias:
		body = m.viewDiscoverAlias()
	}
	if m.searching {
		body += "\n\n" + m.search.View()
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(m.header() + "\n\n" + body)
}

func (m mainModel) viewDashboard() string {
	var b strings.Builder
	b.WriteString(m.viewDiscoveryBanner())
	b.WriteString(styleTitle.Render("Environments") + "\n\n")
	items := m.filteredEnvs()
	if len(m.envs) == 0 {
		b.WriteString("No environments are configured.\n\nRun setup to add a Kubernetes environment.\n\n")
		b.WriteString(styleHelp.Render("enter/s  setup   u updates   q quit"))
		return b.String()
	}
	if len(items) == 0 {
		b.WriteString(styleMuted.Render("No environments match this search.") + "\n")
	} else {
		aw, nw, cw := 14, 16, 18
		if m.width > 60 {
			cw = m.width - aw - nw - 12
			if cw < 12 {
				cw = 12
			}
		}
		b.WriteString(styleMuted.Render("  "+pad("ENVIRONMENT", aw)+"  "+pad("NAMESPACE", nw)+"  "+pad("CONTEXT", cw)) + "\n")
		for i, env := range items {
			line := pad(env.Alias, aw) + "  " + pad(env.Namespace, nw) + "  " + pad(env.KubeContext, cw)
			b.WriteString(row(i == m.envCursor, line) + "\n")
		}
	}
	b.WriteString("\n" + styleHelp.Render("enter open   / search   n namespaces   k contexts   u updates   s settings   ? help   q quit"))
	return b.String()
}

func (m mainModel) viewDiscoveryBanner() string {
	if !m.cfg.Settings.Notifications {
		return ""
	}
	if m.discReady && m.discErr != nil && m.visibleUpdates().Empty() {
		return styleMuted.Render("Unable to check Kubernetes updates.") + "\n\n"
	}
	if !m.showDiscoveryBadge() {
		return ""
	}
	vis := m.visibleUpdates()
	line := vis.Badge()
	if line == "" {
		return ""
	}
	return styleWarn.Render(line+"   [u] view") + "\n\n"
}

func (m mainModel) viewUpdates() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Kubernetes updates") + "\n\n")
	if m.discLoading && !m.discReady {
		b.WriteString(m.spinner.View() + " Checking Kubernetes...\n")
		return b.String()
	}
	if m.discErr != nil && m.visibleUpdates().Empty() {
		b.WriteString(styleMuted.Render("Unable to check Kubernetes updates.") + "\n")
		if m.debug {
			b.WriteString(styleMuted.Render(m.discErr.Error()) + "\n")
		}
		b.WriteString("\n" + styleHelp.Render("r retry   esc back"))
		return b.String()
	}
	vis := m.visibleUpdates()
	if vis.Empty() {
		b.WriteString("No new Kubernetes contexts or namespaces.\n\n")
		b.WriteString(styleHelp.Render("r refresh   esc back"))
		return b.String()
	}
	items := vis.Items()
	if vis.ContextCount() > 0 {
		b.WriteString(styleMuted.Render("New contexts") + "\n")
		for i, item := range items {
			if item.Kind != discover.KindContext {
				continue
			}
			b.WriteString(row(i == m.discCursor, item.Context) + "\n")
		}
		b.WriteString("\n")
	}
	if vis.NamespaceCount() > 0 {
		b.WriteString(styleMuted.Render("New namespaces") + "\n")
		lastCtx := ""
		for i, item := range items {
			if item.Kind != discover.KindNamespace {
				continue
			}
			if item.Context != lastCtx {
				b.WriteString("  " + item.Context + "\n")
				lastCtx = item.Context
			}
			b.WriteString(row(i == m.discCursor, "  "+item.Namespace) + "\n")
		}
	}
	if m.status != "" {
		b.WriteString("\n" + styleMuted.Render(m.status) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter view   a add   i ignore   l later   r refresh   esc back"))
	return b.String()
}

func (m mainModel) viewDiscoverItem() string {
	item, ok := m.selectedDiscoverItem()
	if !ok {
		return styleMuted.Render("Nothing selected.") + "\n\n" + styleHelp.Render("esc back")
	}
	var b strings.Builder
	if item.Kind == discover.KindContext {
		b.WriteString(styleTitle.Render(item.Context) + "\n\n")
		b.WriteString("New Kubernetes context.\nNothing is added until you choose Add.\n\n")
		opts := []string{"Add to kpod", "Ignore", "Later"}
		for i, o := range opts {
			b.WriteString(row(i == m.confirmCursor, o) + "\n")
		}
	} else {
		b.WriteString(styleTitle.Render(item.Namespace) + "\n\n")
		b.WriteString(styleMuted.Render(item.Context) + "\n\n")
		b.WriteString("New namespace. Creating an environment is optional.\n\n")
		opts := []string{"Create environment", "Ignore namespace", "Later"}
		for i, o := range opts {
			b.WriteString(row(i == m.confirmCursor, o) + "\n")
		}
	}
	if m.status != "" {
		b.WriteString("\n" + styleMuted.Render(m.status) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter choose   a add   i ignore   l later   esc back"))
	return b.String()
}

func (m mainModel) viewDiscoverNSPick() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Namespaces — "+ellipsize(m.current.kubeContext, 40)) + "\n\n")
	if m.nsLoading {
		b.WriteString(m.spinner.View() + " Loading namespaces...")
		return b.String()
	}
	if m.nsErr != nil {
		b.WriteString(styleMuted.Render(m.status) + "\n\n" + styleHelp.Render("esc back"))
		return b.String()
	}
	b.WriteString(m.discCheck.view(m.width, 12) + "\n")
	if m.status != "" {
		b.WriteString(styleMuted.Render(m.status) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("space toggle   enter continue   esc back"))
	return b.String()
}

func (m mainModel) viewDiscoverAlias() string {
	target := onboardNS{}
	if len(m.discPending) > 0 {
		target = m.discPending[0]
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render("Environment name") + "\n\n")
	b.WriteString(styleMuted.Render(target.Context+" / "+target.Namespace) + "\n\n")
	b.WriteString(m.discAliasIn.View() + "\n")
	if m.status != "" {
		b.WriteString("\n" + styleErr.Render(m.status) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter save   esc cancel"))
	return b.String()
}

func (m mainModel) viewEnv() string {
	env, ok := m.lookup(m.current.envAlias)
	if !ok {
		return styleErr.Render("Environment is no longer configured.")
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render(env.Alias) + "\n\n")
	b.WriteString("Context\n  " + env.KubeContext + "\n\n")
	b.WriteString("Namespace\n  " + env.Namespace + "\n\n")
	b.WriteString("Pods\n  ")
	switch {
	case m.podsLoading:
		b.WriteString(m.spinner.View())
	case m.podErr != nil:
		b.WriteString(styleErr.Render("unavailable"))
		if m.debug {
			b.WriteString("\n\n" + styleMuted.Render(kubeContextError(m.podErr, env.KubeContext, true)))
		} else {
			b.WriteString("\n\n" + styleMuted.Render(kubeContextError(m.podErr, env.KubeContext, false)))
		}
	case m.podCount != nil:
		b.WriteString(strconv.Itoa(*m.podCount))
	default:
		b.WriteString(styleMuted.Render("—"))
	}
	b.WriteString("\n\n" + styleHelp.Render("enter/p  Browse pods   n namespaces   k contexts   r refresh   esc back   q quit"))
	return b.String()
}

func (m mainModel) viewPods() string {
	env, _ := m.lookup(m.current.envAlias)
	var b strings.Builder
	b.WriteString(styleTitle.Render("Pods — "+env.Alias) + "\n")
	b.WriteString(styleMuted.Render(env.KubeContext+" / "+env.Namespace) + "\n\n")
	if m.podsLoading && len(m.pods) == 0 {
		b.WriteString(m.spinner.View() + " Loading pods...")
		return b.String()
	}
	if m.podErr != nil && len(m.pods) == 0 {
		b.WriteString(styleErr.Render(kubePodError(m.podErr, env.KubeContext, env.Namespace, m.debug)))
		b.WriteString("\n\n" + styleHelp.Render("r retry   esc back"))
		return b.String()
	}
	items := m.filteredPods()
	if len(m.pods) == 0 {
		b.WriteString(fmt.Sprintf("No pods found in %s.\n\n", env.Namespace))
		b.WriteString(styleHelp.Render("[r] Refresh   [esc] Back"))
		return b.String()
	}
	if len(items) == 0 {
		b.WriteString(styleMuted.Render("No pods match this search.") + "\n")
	} else {
		cols := podColumns(m.width)
		header := pad("NAME", cols.name) + "  " + pad("READY", cols.ready) + "  " + pad("STATUS", cols.status)
		if cols.showRestarts {
			header += "  " + pad("RESTARTS", cols.restarts)
		}
		if cols.showAge {
			header += "  " + pad("AGE", cols.age)
		}
		if cols.showNode {
			header += "  " + pad("NODE", cols.node)
		}
		b.WriteString(styleMuted.Render("  "+header) + "\n")
		for i, p := range items {
			status := statusStyle(p.Status).Render(pad(p.Status, cols.status))
			line := pad(p.Name, cols.name) + "  " + pad(readyString(p), cols.ready) + "  " + status
			if cols.showRestarts {
				line += "  " + pad(strconv.Itoa(p.Restarts), cols.restarts)
			}
			if cols.showAge {
				line += "  " + pad(formatAge(p.Age), cols.age)
			}
			if cols.showNode {
				line += "  " + pad(p.Node, cols.node)
			}
			b.WriteString(row(i == m.podCursor, line) + "\n")
		}
	}
	if m.podsLoading {
		b.WriteString("\n" + m.spinner.View() + " Refreshing pods...")
	}
	b.WriteString("\n" + styleHelp.Render("enter open   / search   r refresh   esc back   q quit"))
	return b.String()
}

func (m mainModel) viewPodDetail() string {
	env, _ := m.lookup(m.current.envAlias)
	pod, found := m.selectedPod()
	if !found {
		return styleErr.Render("Pod is no longer in the list.") + "\n\n" + styleHelp.Render("esc back")
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render(ellipsize(pod.Name, m.width)) + "\n\n")
	b.WriteString(m.renderPodInfo(env.Alias, env.KubeContext, env.Namespace, pod))
	b.WriteString("\n" + styleTitle.Render("Pod Actions") + "\n")
	items := action.Menu()
	var last action.Group
	for i, item := range items {
		if item.Group != last {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(styleMuted.Render(string(item.Group)) + "\n")
			last = item.Group
		}
		b.WriteString(row(i == m.actionCursor, item.Label) + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + styleMuted.Render(m.status) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter run   c console  b shell  l logs  d describe   esc back"))
	return b.String()
}

func (m mainModel) renderPodInfo(alias, kubeContext, namespace string, pod kube.Pod) string {
	node := pod.Node
	if node == "" {
		node = "—"
	}
	envFields := []kvField{
		{Label: "Environment", Value: alias},
		{Label: "Context", Value: kubeContext},
		{Label: "Namespace", Value: namespace},
		{Label: "Node", Value: node},
	}
	statFields := []kvField{
		{Label: "Status", Value: statusPlain(pod.Status), Paint: statusPaint(pod.Status)},
		{Label: "Ready", Value: readyString(pod)},
		{Label: "Restarts", Value: strconv.Itoa(pod.Restarts)},
	}
	width := m.width
	if width <= 0 {
		width = 80
	}
	var b strings.Builder
	if kvLayoutFor(width).Columns == 2 {
		b.WriteString(renderKVGrid(envFields, statFields, width))
	} else {
		stacked := []kvField{envFields[0], envFields[1], envFields[2], statFields[0], statFields[1], statFields[2], envFields[3]}
		b.WriteString(renderKVGrid(stacked, nil, width))
	}
	apps := make([]string, 0, len(pod.Containers))
	inits := make([]string, 0, len(pod.Containers))
	for _, c := range pod.Containers {
		if c.Init {
			inits = append(inits, c.Name)
			continue
		}
		apps = append(apps, c.Name)
	}
	extra := []kvField{{Label: "Containers", Value: joinNames(apps)}}
	if len(inits) > 0 {
		extra = append(extra, kvField{Label: "Init", Value: joinNames(inits)})
	}
	b.WriteString(renderKVGrid(extra, nil, width))
	return b.String()
}

func (m mainModel) viewContainerSelect() string {
	pod, _ := m.selectedPod()
	names := action.AppContainerNames(pod)
	var b strings.Builder
	b.WriteString(styleTitle.Render("Select container") + "\n\n")
	for i, n := range names {
		b.WriteString(row(i == m.containerCursor, n) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter select   esc back"))
	return b.String()
}

func (m mainModel) viewLogsMenu() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Logs") + "\n\n")
	for i, c := range action.LogsChoices() {
		b.WriteString(row(i == m.logsMenuCursor, c.Label) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter run   esc back"))
	return b.String()
}

func (m mainModel) viewConfirm() string {
	env, _ := m.lookup(m.current.envAlias)
	pod, _ := m.selectedPod()
	var b strings.Builder
	if m.actEntry.Spec.ID == action.Delete {
		b.WriteString(styleTitle.Render("Delete pod?") + "\n\n")
		b.WriteString(pod.Name + "\n\n")
		b.WriteString("Environment: " + env.Alias + "\n")
		b.WriteString("Context: " + env.KubeContext + "\n")
		b.WriteString("Namespace: " + env.Namespace + "\n\n")
	} else {
		b.WriteString(styleTitle.Render("Restart deployment?") + "\n\n")
		b.WriteString("Environment: " + env.Alias + "\n")
		b.WriteString("Namespace: " + env.Namespace + "\n")
		b.WriteString("Deployment: " + m.actDeploy + "\n\n")
	}
	b.WriteString(row(m.confirmCursor == 0, "Yes") + "\n")
	b.WriteString(row(m.confirmCursor == 1, "No") + "\n")
	b.WriteString("\n" + styleHelp.Render("enter confirms selection (default No)   esc cancel"))
	return b.String()
}

func (m mainModel) viewPrompt() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render(m.actEntry.Label) + "\n\n")
	if m.actEntry.Spec.ID == action.PortForward && m.actLocalPort != "" && m.promptKind == promptRemotePort {
		b.WriteString(styleMuted.Render("localhost:"+m.actLocalPort+" → pod:?") + "\n\n")
	}
	b.WriteString(m.promptInput.View() + "\n\n")
	b.WriteString(styleHelp.Render("enter continue   esc cancel"))
	return b.String()
}

func (m mainModel) viewOutput() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render(m.outputTitle) + "\n\n")
	b.WriteString(m.vp.View())
	b.WriteString("\n\n" + styleHelp.Render("↑/↓ PgUp/PgDn g/G   esc back"))
	return b.String()
}

func (m mainModel) viewContexts() string {
	title := "Kubernetes Contexts"
	if m.current.view == viewNsPickContext {
		title = "Namespaces\n\nSelect a context"
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render(title) + "\n\n")
	if m.ctxLoading {
		b.WriteString(m.spinner.View() + " Loading contexts...")
		return b.String()
	}
	if m.ctxErr != nil {
		b.WriteString(styleErr.Render(kubeContextError(m.ctxErr, "the cluster", m.debug)))
		b.WriteString("\n\n" + styleHelp.Render("r retry   esc back"))
		return b.String()
	}
	items := m.filteredCtx()
	if len(items) == 0 {
		b.WriteString(styleMuted.Render("No contexts match.") + "\n")
	} else {
		nameW := 28
		if m.width > 50 {
			nameW = m.width - 24
		}
		for i, rowItem := range items {
			line := pad(rowItem.Name, nameW) + "  " + pad(rowItem.Label, 18)
			b.WriteString(row(i == m.ctxCursor, line) + "\n")
		}
	}
	b.WriteString("\n" + styleHelp.Render("enter open   / search   r refresh   esc back"))
	return b.String()
}

func (m mainModel) viewContextDetail() string {
	name := m.current.kubeContext
	envs := m.cfg.EnvironmentsForContext(name)
	var b strings.Builder
	b.WriteString(styleTitle.Render(ellipsize(name, m.width-4)) + "\n\n")
	b.WriteString("Configured environments\n\n")
	if len(envs) == 0 {
		b.WriteString(styleMuted.Render("  not configured") + "\n")
	} else {
		for _, env := range envs {
			fmt.Fprintf(&b, "  %-12s  %s\n", env.Alias, env.Namespace)
		}
	}
	b.WriteString("\nNamespaces\n  Browse →\n\n")
	b.WriteString(styleHelp.Render("enter namespaces   esc back"))
	return b.String()
}

func (m mainModel) viewNamespaces() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Namespaces — "+ellipsize(m.current.kubeContext, 40)) + "\n\n")
	if m.nsLoading {
		b.WriteString(m.spinner.View() + " Loading namespaces...")
		return b.String()
	}
	if m.nsErr != nil {
		b.WriteString(styleErr.Render(kubeContextError(m.nsErr, m.current.kubeContext, m.debug)))
		b.WriteString("\n\n" + styleHelp.Render("r retry   esc back"))
		return b.String()
	}
	items := m.filteredNS()
	if len(items) == 0 && m.status != "" {
		b.WriteString(m.status + "\n")
	} else {
		b.WriteString(styleMuted.Render("  "+pad("NAMESPACE", 20)+"  ENVIRONMENT") + "\n")
		for i, ns := range items {
			env := ns.Env
			if env == "" {
				env = "—"
			}
			line := pad(ns.Name, 20) + "  " + env
			b.WriteString(row(i == m.nsCursor, line) + "\n")
		}
	}
	if m.status != "" && len(items) > 0 {
		b.WriteString("\n" + styleMuted.Render(m.status) + "\n")
	}
	b.WriteString("\n" + styleHelp.Render("enter open env   / search   r refresh   esc back"))
	return b.String()
}

func (m mainModel) viewNsForbidden() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Namespaces — "+ellipsize(m.current.kubeContext, 40)) + "\n\n")
	b.WriteString("Your Kubernetes account cannot list namespaces.\n\n")
	b.WriteString("Configured environments are still available:\n\n")
	envs := m.cfg.EnvironmentsForContext(m.current.kubeContext)
	if len(envs) == 0 {
		b.WriteString(styleMuted.Render("  none") + "\n")
	} else {
		for _, env := range envs {
			fmt.Fprintf(&b, "  %-12s  %s\n", env.Alias, env.Namespace)
		}
	}
	b.WriteString("\nYou can continue using these environments.\n\n")
	b.WriteString(styleHelp.Render("r retry   esc back"))
	return b.String()
}

func (m mainModel) viewHelp() string {
	if m.current.view == viewPods || m.current.view == viewPodDetail {
		return styleTitle.Render("Pod Browser") + `

↑/↓        Move
Enter      Open pod
/          Search
r          Refresh
Esc        Back
q          Quit
`
	}
	return styleTitle.Render("Navigation") + `

↑/↓ j      Move
Enter      Open
Esc        Back
/          Search
r          Refresh
n          Namespaces
k          Contexts
u          Kubernetes updates
s          Settings
?          Help
q          Quit
`
}
