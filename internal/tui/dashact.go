package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	tea "github.com/charmbracelet/bubbletea"
)

type captureMsg struct {
	title   string
	body    string
	err     error
	refresh bool
	deleted bool
}

type execReadyMsg struct {
	args    []string
	title   string
	refresh bool
	deleted bool
	err     error
}

type execDoneMsg struct {
	err     error
	refresh bool
	deleted bool
}

type ownerMsg struct {
	name string
	err  error
}

func (m mainModel) selectedPod() (kube.Pod, bool) {
	for _, p := range m.pods {
		if p.Name == m.current.podName {
			return p, true
		}
	}
	return kube.Pod{}, false
}

func (m mainModel) actionViews() bool {
	switch m.current.view {
	case viewPodDetail, viewContainerSelect, viewConfirm, viewPrompt, viewLogsMenu, viewOutput:
		return true
	default:
		return false
	}
}

func (m mainModel) handleActionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.current.view == viewOutput {
		return m.handleOutputKey(msg)
	}
	if m.current.view == viewPrompt {
		return m.handlePromptKey(msg)
	}
	switch msg.String() {
	case "up":
		m.moveAction(-1)
		return m, nil
	case "down", "j":
		m.moveAction(1)
		return m, nil
	case "enter":
		return m.activateAction()
	case "esc":
		if m.pop() {
			return m, nil
		}
		return m, nil
	}
	if m.current.view == viewPodDetail {
		switch msg.String() {
		case "c":
			return m.startShortcut(action.Console, false)
		case "b":
			return m.startShortcut(action.Shell, false)
		case "l":
			return m.startShortcut(action.Logs, true)
		case "d":
			return m.startShortcut(action.Describe, false)
		}
	}
	if m.current.view == viewConfirm {
		switch msg.String() {
		case "y":
			m.confirmCursor = 0
			return m.activateAction()
		case "n":
			m.pop()
			return m, nil
		}
	}
	return m, nil
}

func (m mainModel) handleOutputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pop()
		return m, nil
	case "g":
		m.vp.GotoTop()
		return m, nil
	case "G":
		m.vp.GotoBottom()
		return m, nil
	case "pgup":
		m.vp.ViewUp()
		return m, nil
	case "pgdown":
		m.vp.ViewDown()
		return m, nil
	case "up":
		m.vp.LineUp(1)
		return m, nil
	case "down", "j":
		m.vp.LineDown(1)
		return m, nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m mainModel) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pop()
		return m, nil
	case "enter":
		val := strings.TrimSpace(m.promptInput.Value())
		switch m.promptKind {
		case promptRunner:
			if val == "" {
				m.status = "Cancelled."
				m.pop()
				return m, nil
			}
			m.actRunner = val
			return m.executePrepared()
		case promptLocalPort:
			m.actLocalPort = val
			return m.beginPrompt(promptRemotePort, "Remote port")
		case promptRemotePort:
			m.actRemotePort = val
			return m.executePrepared()
		case promptRemotePath:
			m.actRemotePath = val
			if m.actEntry.Spec.ID == action.Download {
				return m.beginPrompt(promptLocalPath, "Local destination")
			}
			return m.executePrepared()
		case promptLocalPath:
			m.actLocalPath = val
			if m.actEntry.Spec.ID == action.Upload && m.actRemotePath == "" {
				return m.beginPrompt(promptRemotePath, "Remote path (in pod)")
			}
			return m.executePrepared()
		}
	}
	var cmd tea.Cmd
	m.promptInput, cmd = m.promptInput.Update(msg)
	return m, cmd
}

func (m *mainModel) moveAction(delta int) {
	switch m.current.view {
	case viewPodDetail:
		items := action.Menu()
		m.actionCursor = clamp(m.actionCursor+delta, 0, len(items)-1)
	case viewContainerSelect:
		pod, ok := m.selectedPod()
		if !ok {
			return
		}
		names := action.AppContainerNames(pod)
		m.containerCursor = clamp(m.containerCursor+delta, 0, len(names)-1)
	case viewConfirm:
		m.confirmCursor = clamp(m.confirmCursor+delta, 0, 1)
	case viewLogsMenu:
		m.logsMenuCursor = clamp(m.logsMenuCursor+delta, 0, len(action.LogsChoices())-1)
	}
}

func (m mainModel) startShortcut(id action.ID, logsMenu bool) (tea.Model, tea.Cmd) {
	items := action.Menu()
	for i, item := range items {
		if item.Spec.ID == id && item.LogsMenu == logsMenu && item.Logs != action.LogsPrevious {
			m.actionCursor = i
			return m.beginEntry(item)
		}
	}
	return m, nil
}

func (m mainModel) activateAction() (tea.Model, tea.Cmd) {
	switch m.current.view {
	case viewPodDetail:
		items := action.Menu()
		if m.actionCursor >= len(items) {
			return m, nil
		}
		return m.beginEntry(items[m.actionCursor])
	case viewContainerSelect:
		pod, ok := m.selectedPod()
		if !ok {
			return m, nil
		}
		names := action.AppContainerNames(pod)
		if m.containerCursor >= len(names) {
			return m, nil
		}
		m.actContainer = names[m.containerCursor]
		m.sessionContainer[pod.Name] = m.actContainer
		m.pop()
		return m.afterContainer()
	case viewLogsMenu:
		choices := action.LogsChoices()
		if m.logsMenuCursor >= len(choices) {
			return m, nil
		}
		m.actEntry.Logs = choices[m.logsMenuCursor].Mode
		m.actEntry.LogsMenu = false
		m.pop()
		return m.afterContainer()
	case viewConfirm:
		if m.confirmCursor != 0 {
			m.pop()
			return m, nil
		}
		return m.executePrepared()
	}
	return m, nil
}

func (m mainModel) beginEntry(entry action.MenuEntry) (tea.Model, tea.Cmd) {
	pod, ok := m.selectedPod()
	if !ok {
		m.status = "Pod is no longer in the list."
		return m, nil
	}
	m.actEntry = entry
	m.actRunner, m.actLocalPort, m.actRemotePort = "", "", ""
	m.actLocalPath, m.actRemotePath, m.actDeploy = "", "", ""
	if entry.Spec.NeedsRunning && !action.IsRunning(pod) {
		m.status = fmt.Sprintf("%s needs a running container (pod is %s).", entry.Label, pod.Status)
		return m, nil
	}
	if remembered := m.sessionContainer[pod.Name]; remembered != "" {
		m.actContainer = remembered
	}
	if entry.Spec.NeedsContainer {
		name, err := action.PickContainer(pod, m.actContainer)
		if err != nil && m.actContainer != "" {
			m.actContainer = ""
			name, err = action.PickContainer(pod, "")
		}
		switch {
		case err == nil:
			m.actContainer = name
		case errors.Is(err, action.ErrNeedContainer):
			m.push(navFrame{view: viewContainerSelect, envAlias: m.current.envAlias, kubeContext: m.current.kubeContext, podName: m.current.podName})
			m.containerCursor = 0
			return m, nil
		default:
			m.status = err.Error()
			return m, nil
		}
	}
	return m.afterContainer()
}

func (m mainModel) afterContainer() (tea.Model, tea.Cmd) {
	entry := m.actEntry
	if entry.LogsMenu {
		m.push(navFrame{view: viewLogsMenu, envAlias: m.current.envAlias, kubeContext: m.current.kubeContext, podName: m.current.podName})
		m.logsMenuCursor = 0
		return m, nil
	}
	switch entry.Spec.ID {
	case action.Runner:
		return m.beginPrompt(promptRunner, "Runner expression")
	case action.PortForward:
		return m.beginPrompt(promptLocalPort, "Local port")
	case action.Download:
		return m.beginPrompt(promptRemotePath, "Remote path (in pod)")
	case action.Upload:
		return m.beginPrompt(promptLocalPath, "Local source")
	case action.Restart:
		return m, m.loadOwnerCmd()
	case action.Delete:
		m.push(navFrame{view: viewConfirm, envAlias: m.current.envAlias, kubeContext: m.current.kubeContext, podName: m.current.podName})
		m.confirmCursor = 1
		return m, nil
	}
	return m.executePrepared()
}

func (m mainModel) beginPrompt(kind promptKind, label string) (tea.Model, tea.Cmd) {
	m.push(navFrame{view: viewPrompt, envAlias: m.current.envAlias, kubeContext: m.current.kubeContext, podName: m.current.podName})
	m.promptKind = kind
	m.promptInput.SetValue("")
	m.promptInput.Placeholder = label
	m.promptInput.Prompt = label + ": "
	m.promptInput.Focus()
	return m, nil
}

func (m mainModel) buildRequest() (action.Request, error) {
	env, ok := m.lookup(m.current.envAlias)
	if !ok {
		return action.Request{}, fmt.Errorf("environment is no longer configured")
	}
	req := action.Request{
		Spec:       m.actEntry.Spec,
		Env:        env,
		Pod:        m.current.podName,
		Container:  m.actContainer,
		Logs:       m.actEntry.Logs,
		RunnerCode: m.actRunner,
		LocalPath:  m.actLocalPath,
		RemotePath: m.actRemotePath,
	}
	if m.actLocalPort != "" {
		n, err := strconv.Atoi(m.actLocalPort)
		if err != nil {
			return req, fmt.Errorf("local port must be a number")
		}
		req.LocalPort = n
	}
	if m.actRemotePort != "" {
		n, err := strconv.Atoi(m.actRemotePort)
		if err != nil {
			return req, fmt.Errorf("remote port must be a number")
		}
		req.RemotePort = n
	}
	if m.current.view == viewConfirm && m.confirmCursor == 0 {
		req.Confirmed = true
	}
	return req, nil
}

func (m mainModel) executePrepared() (tea.Model, tea.Cmd) {
	req, err := m.buildRequest()
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if req.Spec.ID == action.PortForward {
		m.status = fmt.Sprintf("localhost:%d → pod:%d", req.LocalPort, req.RemotePort)
	}
	return m, m.runActionCmd(req)
}

func (m mainModel) runActionCmd(req action.Request) tea.Cmd {
	return func() tea.Msg {
		if m.acts == nil || m.kube == nil {
			return captureMsg{err: kube.ErrKubectlNotFound}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		refresh := req.Spec.ID == action.Restart || req.Spec.ID == action.Delete
		deleted := req.Spec.ID == action.Delete
		if action.Attached(req) {
			if req.Spec.ID == action.PortForward || req.Spec.ID == action.Logs {
				ctx = context.Background()
			}
			args, err := m.acts.KubectlArgs(ctx, req)
			if err != nil {
				return execReadyMsg{err: err}
			}
			return execReadyMsg{args: args, title: req.Spec.Label, refresh: refresh, deleted: deleted}
		}
		if req.Spec.ID == action.Restart || req.Spec.ID == action.Rollout {
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
		}
		res, err := m.acts.Capture(ctx, req)
		body := string(res.Stdout)
		if body == "" {
			body = string(res.Stderr)
		}
		return captureMsg{title: req.Spec.Label, body: body, err: err, refresh: refresh, deleted: deleted}
	}
}

func (m mainModel) loadOwnerCmd() tea.Cmd {
	return func() tea.Msg {
		if m.kube == nil {
			return ownerMsg{err: kube.ErrKubectlNotFound}
		}
		env, ok := m.lookup(m.current.envAlias)
		if !ok {
			return ownerMsg{err: fmt.Errorf("environment is no longer configured")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		name, err := m.kube.OwnerDeployment(ctx, env.KubeContext, env.Namespace, m.current.podName)
		return ownerMsg{name: name, err: err}
	}
}

func (m mainModel) onOwner(msg ownerMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = friendlyActionError(msg.err)
		return m, nil
	}
	m.actDeploy = msg.name
	m.push(navFrame{view: viewConfirm, envAlias: m.current.envAlias, kubeContext: m.current.kubeContext, podName: m.current.podName})
	m.confirmCursor = 1
	return m, nil
}

func (m mainModel) onCapture(msg captureMsg) (tea.Model, tea.Cmd) {
	if msg.deleted && msg.err == nil {
		return m.afterDelete()
	}
	if msg.err != nil && msg.body == "" {
		m.status = friendlyActionError(msg.err)
		return m, nil
	}
	body := msg.body
	if msg.err != nil && body == "" {
		body = msg.err.Error()
	} else if msg.err != nil {
		body = friendlyActionError(msg.err) + "\n\n" + body
	}
	if action.PreferPager([]byte(body)) {
		if cmd := action.PagerCommand(body); cmd != nil {
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return execDoneMsg{err: err, refresh: msg.refresh}
			})
		}
	}
	m.push(navFrame{view: viewOutput, envAlias: m.current.envAlias, kubeContext: m.current.kubeContext, podName: m.current.podName})
	m.outputTitle = msg.title
	m.outputBody = body
	m.vp.Width = maxInt(m.width-2, 20)
	m.vp.Height = maxInt(m.height-10, 8)
	m.vp.SetContent(body)
	m.vp.GotoTop()
	if msg.refresh {
		return m, m.refreshPodsCmd()
	}
	return m, nil
}

func (m mainModel) onExecReady(msg execReadyMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = friendlyActionError(msg.err)
		return m, nil
	}
	if m.kube == nil {
		m.status = friendlyActionError(kube.ErrKubectlNotFound)
		return m, nil
	}
	cmd := m.kube.Command(msg.args...)
	m.pendingRefresh = msg.refresh
	m.pendingDeleted = msg.deleted
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execDoneMsg{err: err, refresh: msg.refresh, deleted: msg.deleted}
	})
}

func (m mainModel) onExecDone(msg execDoneMsg) (tea.Model, tea.Cmd) {
	if msg.deleted && msg.err == nil {
		return m.afterDelete()
	}
	if msg.err != nil {
		m.status = friendlyActionError(msg.err)
	}
	for m.current.view == viewPrompt || m.current.view == viewConfirm || m.current.view == viewLogsMenu || m.current.view == viewContainerSelect {
		if !m.pop() {
			break
		}
	}
	if msg.refresh {
		return m, m.refreshPodsCmd()
	}
	return m, nil
}

func (m mainModel) afterDelete() (tea.Model, tea.Cmd) {
	for m.current.view != viewPods && m.pop() {
	}
	m.current.view = viewPods
	m.status = "Pod deleted."
	env, ok := m.lookup(m.current.envAlias)
	if ok && m.kube != nil {
		m.kube.InvalidatePods(env.KubeContext, env.Namespace)
	}
	return m, m.refreshPodsCmd()
}

func (m mainModel) refreshPodsCmd() tea.Cmd {
	env, ok := m.lookup(m.current.envAlias)
	if !ok {
		return nil
	}
	m.podsLoading = true
	return loadDashPodsCmd(m.kube, env, true)
}

func friendlyActionError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, kube.ErrMetricsUnavailable):
		return "Pod metrics unavailable. Is metrics-server installed on this cluster?"
	case errors.Is(err, kube.ErrDeploymentNotFound):
		return "This pod is not managed by a Deployment."
	case errors.Is(err, kube.ErrNoRunningContainer):
		return "No running container is available for this action."
	case errors.Is(err, kube.ErrContainerNotFound):
		return "Container was not found."
	case errors.Is(err, kube.ErrPreviousLogsUnavailable):
		return "Previous logs are not available for this container."
	case errors.Is(err, kube.ErrNeedsConfirm):
		return "Confirmation is required."
	case errors.Is(err, context.Canceled):
		return ""
	default:
		return err.Error()
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
