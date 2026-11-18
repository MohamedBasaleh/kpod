package tui

import (
	"testing"
	"time"

	"github.com/MohamedBasaleh/kpod/internal/kube"
)

func samplePods() []kube.Pod {
	return []kube.Pod{
		{Name: "worker-foo", Status: "CrashLoopBackOff", Ready: 0, Total: 1, Restarts: 8, Age: 15 * time.Minute, Node: "n1", Containers: []kube.Container{{Name: "worker"}}},
		{Name: "dev-core-api-def456", Status: "Running", Ready: 1, Total: 1, Restarts: 1, Age: 25 * time.Hour, Node: "n2", Containers: []kube.Container{{Name: "api"}}},
		{Name: "dev-core-api-abc123", Status: "Running", Ready: 1, Total: 1, Restarts: 0, Age: 2 * time.Hour, Node: "n2", Containers: []kube.Container{{Name: "api"}}},
		{Name: "job-old", Status: "Completed", Ready: 0, Total: 1, Restarts: 0, Age: 3 * 24 * time.Hour, Containers: []kube.Container{{Name: "job"}}},
		{Name: "wait-0", Status: "Pending", Ready: 0, Total: 1, Restarts: 0, Age: 32 * time.Second, Containers: []kube.Container{{Name: "web"}}},
		{Name: "pull-0", Status: "ImagePullBackOff", Ready: 0, Total: 1, Restarts: 0, Age: 8 * time.Minute, Containers: []kube.Container{{Name: "web"}}},
		{Name: "gone-0", Status: "Terminating", Ready: 0, Total: 1, Restarts: 0, Age: time.Minute, Terminating: true, Containers: []kube.Container{{Name: "web"}}},
	}
}

func TestFormatAge(t *testing.T) {
	cases := map[time.Duration]string{
		32 * time.Second:   "32s",
		8 * time.Minute:    "8m",
		2 * time.Hour:      "2h",
		3 * 24 * time.Hour: "3d",
		90 * time.Second:   "1m",
		0:                  "0s",
	}
	for d, want := range cases {
		if got := formatAge(d); got != want {
			t.Fatalf("%v: got %s want %s", d, got, want)
		}
	}
}

func TestSortPodsRunningFirstThenName(t *testing.T) {
	got := sortPods(samplePods())
	if got[0].Name != "dev-core-api-abc123" || got[1].Name != "dev-core-api-def456" {
		t.Fatalf("running first by name: %+v", names(got))
	}
	if got[0].Status != "Running" || got[1].Status != "Running" {
		t.Fatalf("%+v", names(got))
	}
	for i := 2; i < len(got); i++ {
		if got[i].Status == "Running" {
			t.Fatalf("running after non-running: %v", names(got))
		}
	}
}

func TestReadyAndRestarts(t *testing.T) {
	p := kube.Pod{Ready: 1, Total: 2, Restarts: 9}
	if readyString(p) != "1/2" {
		t.Fatal(readyString(p))
	}
	if p.Restarts != 9 {
		t.Fatal("pod-level restart total from Phase 3")
	}
}

func TestAppContainersExcludeInit(t *testing.T) {
	p := kube.Pod{Containers: []kube.Container{
		{Name: "init-db", Init: true},
		{Name: "api"},
		{Name: "sidecar"},
	}}
	apps := appContainers(p)
	if len(apps) != 2 || apps[0].Name != "api" || apps[1].Name != "sidecar" {
		t.Fatalf("%+v", apps)
	}
	if len(initContainers(p)) != 1 {
		t.Fatal("init")
	}
}

func TestNarrowColumnsHideNode(t *testing.T) {
	wide := podColumns(120)
	if !wide.showNode || wide.name < 10 {
		t.Fatalf("%+v", wide)
	}
	narrow := podColumns(50)
	if narrow.showNode || narrow.showRestarts {
		t.Fatalf("%+v", narrow)
	}
}

func names(pods []kube.Pod) []string {
	out := make([]string, len(pods))
	for i, p := range pods {
		out[i] = p.Name
	}
	return out
}

func TestIndexPodByNameFallback(t *testing.T) {
	pods := sortPods(samplePods())
	if indexPodByName(pods, "dev-core-api-def456") == 0 {
		t.Fatal("should find def456")
	}
	if indexPodByName(pods, "missing") != 0 {
		t.Fatal("missing falls back to first")
	}
}

func TestLongNamePreservedInternally(t *testing.T) {
	long := "dev-core-api-586c8c88dc-9p5xs-this-is-a-very-long-generated-name"
	p := kube.Pod{Name: long, Status: "Running"}
	if ellipsize(p.Name, 20) == long {
		t.Fatal("expected visual truncate")
	}
	if p.Name != long {
		t.Fatal("full name must remain on the model")
	}
}
