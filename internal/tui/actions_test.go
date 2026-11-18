package tui

import (
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	tea "github.com/charmbracelet/bubbletea"
)

func actionModel(t *testing.T, pod kube.Pod) mainModel {
	t.Helper()
	m := envModel(t)
	m.current.view = viewPodDetail
	m.current.podName = pod.Name
	m.current.envAlias = "dev"
	m.pods = []kube.Pod{pod}
	m.width = 100
	return m
}

func TestActionMenuReplacesStub(t *testing.T) {
	m := actionModel(t, kube.Pod{Name: "api-0", Status: "Running", Containers: []kube.Container{{Name: "api"}}})
	view := m.View()
	if strings.Contains(view, "Actions arrive in Phase 7") {
		t.Fatal(view)
	}
	for _, want := range []string{"Pod Actions", "Connect", "Observe", "Manage", "Transfer", "Network", "Console", "Delete Pod"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in %s", want, view)
		}
	}
}

func TestContainerSelectorForMulti(t *testing.T) {
	m := actionModel(t, kube.Pod{
		Name:   "api-0",
		Status: "Running",
		Containers: []kube.Container{
			{Name: "init", Init: true},
			{Name: "api"},
			{Name: "envoy"},
		},
	})
	var shell action.MenuEntry
	for _, e := range action.Menu() {
		if e.Spec.ID == action.Shell {
			shell = e
			break
		}
	}
	next, _ := m.beginEntry(shell)
	m = next.(mainModel)
	if m.current.view != viewContainerSelect {
		t.Fatalf("%+v", m.current)
	}
	view := m.View()
	if !strings.Contains(view, "api") || !strings.Contains(view, "envoy") {
		t.Fatalf("%s", view)
	}
	if strings.Contains(view, "init") {
		t.Fatal("init container listed")
	}
	next, _ = m.handleActionKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(mainModel)
	if m.current.view != viewPodDetail {
		t.Fatalf("esc should return to actions %+v", m.current)
	}
}

func TestSingleContainerAutoSelect(t *testing.T) {
	m := actionModel(t, kube.Pod{
		Name:       "api-0",
		Status:     "Running",
		Containers: []kube.Container{{Name: "api"}},
	})
	var console action.MenuEntry
	for _, e := range action.Menu() {
		if e.Spec.ID == action.Console {
			console = e
			break
		}
	}
	next, _ := m.beginEntry(console)
	m = next.(mainModel)
	if m.current.view == viewContainerSelect {
		t.Fatal("should auto-select")
	}
	if m.actContainer != "api" {
		t.Fatalf("%s", m.actContainer)
	}
}

func TestDeleteConfirmDefaultsNo(t *testing.T) {
	m := actionModel(t, kube.Pod{Name: "api-0", Status: "Running", Containers: []kube.Container{{Name: "api"}}})
	var del action.MenuEntry
	for _, e := range action.Menu() {
		if e.Spec.ID == action.Delete {
			del = e
			break
		}
	}
	next, _ := m.beginEntry(del)
	m = next.(mainModel)
	if m.current.view != viewConfirm {
		t.Fatalf("%+v", m.current)
	}
	if m.confirmCursor != 1 {
		t.Fatalf("default must be No, cursor %d", m.confirmCursor)
	}
	view := m.View()
	if !strings.Contains(view, "Delete pod?") || !strings.Contains(view, "api-0") {
		t.Fatalf("%s", view)
	}
	next, cmd := m.activateAction()
	m = next.(mainModel)
	if cmd != nil {
		t.Fatal("Enter on No must not delete")
	}
	if m.current.view != viewPodDetail {
		t.Fatalf("cancel %+v", m.current)
	}
}

func TestExecDoneRestoresActions(t *testing.T) {
	m := actionModel(t, kube.Pod{Name: "api-0", Status: "Running", Containers: []kube.Container{{Name: "api"}}})
	m.push(navFrame{view: viewPrompt, envAlias: "dev", podName: "api-0"})
	next, _ := m.Update(execDoneMsg{})
	m = next.(mainModel)
	if m.current.view != viewPodDetail {
		t.Fatalf("%+v", m.current)
	}
}

func TestOutputEscBack(t *testing.T) {
	m := actionModel(t, kube.Pod{Name: "api-0", Status: "Running"})
	m.push(navFrame{view: viewOutput, envAlias: "dev", podName: "api-0"})
	m.outputBody = "hello"
	m.vp.SetContent("hello")
	next, _ := m.handleActionKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(mainModel)
	if m.current.view != viewPodDetail {
		t.Fatalf("%+v", m.current)
	}
}

func TestNeedsRunningExplanation(t *testing.T) {
	m := actionModel(t, kube.Pod{Name: "job-0", Status: "Completed", Containers: []kube.Container{{Name: "job"}}})
	var console action.MenuEntry
	for _, e := range action.Menu() {
		if e.Spec.ID == action.Console {
			console = e
			break
		}
	}
	next, _ := m.beginEntry(console)
	m = next.(mainModel)
	if m.current.view != viewPodDetail {
		t.Fatal()
	}
	if !strings.Contains(m.status, "running") {
		t.Fatalf("%s", m.status)
	}
}
