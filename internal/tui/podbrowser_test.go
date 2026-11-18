package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	tea "github.com/charmbracelet/bubbletea"
)

func envModel(t *testing.T) mainModel {
	t.Helper()
	env, _ := testCfg().LookupEnvironment("dev")
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, &env, false)
	m.width = 100
	return m
}

func TestEnvironmentOpensPodBrowser(t *testing.T) {
	m := envModel(t)
	next, _ := m.open()
	m = next.(mainModel)
	if m.current.view != viewPods {
		t.Fatalf("%+v", m.current)
	}
	m.pods = sortPods(samplePods())
	m.podsAlias = "dev"
	m.podsLoading = false
	view := m.View()
	if !strings.Contains(view, "Pods — dev") {
		t.Fatalf("%s", view)
	}
	if !strings.Contains(view, "CrashLoopBackOff") || !strings.Contains(view, "Pending") {
		t.Fatal("all statuses must be shown")
	}
	if strings.Contains(view, "Phase 6") {
		t.Fatal("stub remains")
	}
}

func TestZeroPods(t *testing.T) {
	m := envModel(t)
	m.current.view = viewPods
	m.podsLoading = false
	m.pods = nil
	view := m.View()
	if !strings.Contains(view, "No pods found in dev-core") {
		t.Fatalf("%s", view)
	}
}

func TestPodSearch(t *testing.T) {
	m := envModel(t)
	m.current.view = viewPods
	m.podsLoading = false
	m.pods = sortPods(samplePods())
	m.searching = true
	m.search.SetValue("API")
	got := m.filteredPods()
	if len(got) != 2 {
		t.Fatalf("%d", len(got))
	}
	m.search.SetValue("no-such-pod")
	if len(m.filteredPods()) != 0 {
		t.Fatal("zero matches")
	}
	m.searching = false
	m.search.SetValue("")
	if len(m.filteredPods()) != len(m.pods) {
		t.Fatal("clear search")
	}
}

func TestRefreshKeepsSelection(t *testing.T) {
	m := envModel(t)
	m.current.view = viewPods
	m.current.envAlias = "dev"
	m.pods = sortPods(samplePods())
	m.podCursor = indexPodByName(m.filteredPods(), "dev-core-api-def456")
	next, _ := m.Update(dashPodsMsg{
		alias: "dev",
		list: []kube.Pod{
			{Name: "dev-core-api-def456", Status: "Running", Ready: 1, Total: 1},
			{Name: "dev-core-api-abc123", Status: "Running", Ready: 1, Total: 1},
		},
		refresh: true,
	})
	m = next.(mainModel)
	if m.filteredPods()[m.podCursor].Name != "dev-core-api-def456" {
		t.Fatalf("cursor %d name %s", m.podCursor, selectedPodName(m.filteredPods(), m.podCursor))
	}
}

func TestRefreshMissingPodFallsBack(t *testing.T) {
	m := envModel(t)
	m.current.view = viewPods
	m.current.envAlias = "dev"
	m.pods = sortPods(samplePods())
	m.podCursor = indexPodByName(m.filteredPods(), "worker-foo")
	next, _ := m.Update(dashPodsMsg{
		alias: "dev",
		list: []kube.Pod{
			{Name: "dev-core-api-abc123", Status: "Running"},
		},
		refresh: true,
	})
	m = next.(mainModel)
	if m.filteredPods()[m.podCursor].Name != "dev-core-api-abc123" {
		t.Fatalf("%s", selectedPodName(m.filteredPods(), m.podCursor))
	}
}

func TestRefreshErrorKeepsList(t *testing.T) {
	m := envModel(t)
	m.current.view = viewPods
	m.current.envAlias = "dev"
	m.pods = sortPods(samplePods())
	next, _ := m.Update(dashPodsMsg{alias: "dev", err: kube.ErrTimeout, refresh: true})
	m = next.(mainModel)
	if len(m.pods) == 0 {
		t.Fatal("previous list must remain")
	}
	if m.podErr == nil {
		t.Fatal("expected error")
	}
}

func TestPodDetail(t *testing.T) {
	m := envModel(t)
	m.current.view = viewPods
	m.pods = []kube.Pod{{
		Name:     "multi-0",
		Status:   "Running",
		Ready:    1,
		Total:    2,
		Restarts: 3,
		Node:     "ip-10-0-1-2",
		Age:      2 * time.Hour,
		Containers: []kube.Container{
			{Name: "init-db", Init: true},
			{Name: "api"},
			{Name: "sidecar"},
		},
	}}
	m.podCursor = 0
	next, _ := m.open()
	m = next.(mainModel)
	if m.current.view != viewPodDetail || m.current.podName != "multi-0" {
		t.Fatalf("%+v", m.current)
	}
	view := m.View()
	for _, want := range []string{"multi-0", "dev", "dev-core", "Running", "Console", "Shell", "Logs", "Describe"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in %s", want, view)
		}
	}
	if !m.pop() || m.current.view != viewPods {
		t.Fatal("detail should return to list")
	}
}

func TestPodNavEscAndHelp(t *testing.T) {
	m := envModel(t)
	next, _ := m.open()
	m = next.(mainModel)
	if m.current.view != viewPods {
		t.Fatal()
	}
	next, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = next.(mainModel)
	if !strings.Contains(m.View(), "Pod Browser") {
		t.Fatalf("%s", m.View())
	}
	m.showHelp = false
	next, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(mainModel)
	if cmd != nil || m.current.view != viewEnv {
		t.Fatalf("esc to env %+v %v", m.current, cmd)
	}
}

func TestPodErrors(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{kube.ErrUnauthorized, "Authentication failed"},
		{kube.ErrForbidden, "You do not have permission to list pods in dev-core"},
		{kube.ErrClusterUnavailable, "Unable to reach Nawaqis-Dev"},
		{kube.ErrTimeout, "did not respond in time"},
		{kube.ErrMalformedOutput, "could not read"},
	}
	for _, tc := range cases {
		msg := kubePodError(tc.err, "Nawaqis-Dev", "dev-core", false)
		if !strings.Contains(msg, tc.want) {
			t.Fatalf("%v: %q", tc.err, msg)
		}
		if strings.Contains(msg, "exit status") {
			t.Fatalf("raw: %q", msg)
		}
	}
}

func TestKDoesNotMoveInPodBrowser(t *testing.T) {
	m := envModel(t)
	m.current.view = viewPods
	m.pods = sortPods(samplePods())
	m.podCursor = 2
	next, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = next.(mainModel)
	if cmd != nil {
		t.Fatal("k must not open contexts from pod browser")
	}
	if m.podCursor != 2 {
		t.Fatalf("k must not move up, cursor %d", m.podCursor)
	}
}

func TestDirectOpenPods(t *testing.T) {
	env, _ := testCfg().LookupEnvironment("dev")
	m := newMainModel(config.NewPaths(t.TempDir()), nil, nil, testCfg(), false, &env, true)
	if m.current.view != viewPods {
		t.Fatalf("%+v", m.current)
	}
}

func TestQQuitsPodBrowser(t *testing.T) {
	m := envModel(t)
	m.current.view = viewPods
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected quit")
	}
}

func TestOpenPodsReusesLoadedList(t *testing.T) {
	m := envModel(t)
	m.pods = sortPods(samplePods())
	m.podsAlias = "dev"
	m.podsLoading = false
	next, cmd := m.openPods()
	m = next.(mainModel)
	if cmd != nil {
		t.Fatal("should reuse cached list without a new kubectl command")
	}
	if m.current.view != viewPods {
		t.Fatal()
	}
}

func TestPodErrorView(t *testing.T) {
	m := envModel(t)
	m.current.view = viewPods
	m.podsLoading = false
	m.podErr = kube.ErrForbidden
	view := m.View()
	if !strings.Contains(view, "permission to list pods") {
		t.Fatalf("%s", view)
	}
}
