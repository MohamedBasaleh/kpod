package action

import (
	"errors"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/kube"
)

func sample() []kube.Pod {
	return []kube.Pod{
		{Name: "dev-core-api-abc", Status: "Running"},
		{Name: "dev-core-api-def", Status: "Running"},
		{Name: "worker-1", Status: "CrashLoopBackOff"},
		{Name: "job-old", Status: "Completed"},
	}
}

func TestMatchPodsNoneOneMany(t *testing.T) {
	if got := MatchPods(sample(), "nope", true); len(got) != 0 {
		t.Fatal(got)
	}
	one := MatchPods(sample(), "worker", true)
	if len(one) != 1 || one[0].Name != "worker-1" {
		t.Fatalf("%+v", one)
	}
	many := MatchPods(sample(), "api", true)
	if len(many) != 2 {
		t.Fatalf("%d", len(many))
	}
}

func TestMatchPodsPreferRunningButKeepNonRunning(t *testing.T) {
	onlyStopped := MatchPods(sample(), "job", true)
	if len(onlyStopped) != 1 || onlyStopped[0].Status != "Completed" {
		t.Fatalf("%+v", onlyStopped)
	}
	all := MatchPods(sample(), "", false)
	if len(all) != 4 {
		t.Fatal(len(all))
	}
}

func TestPickContainerSingleMultiInit(t *testing.T) {
	single := kube.Pod{Containers: []kube.Container{{Name: "init", Init: true}, {Name: "api"}}}
	got, err := PickContainer(single, "")
	if err != nil || got != "api" {
		t.Fatalf("%s %v", got, err)
	}
	multi := kube.Pod{Containers: []kube.Container{{Name: "api"}, {Name: "envoy"}}}
	_, err = PickContainer(multi, "")
	if !errors.Is(err, ErrNeedContainer) {
		t.Fatalf("%v", err)
	}
	got, err = PickContainer(multi, "envoy")
	if err != nil || got != "envoy" {
		t.Fatalf("%s %v", got, err)
	}
	_, err = PickContainer(multi, "init")
	if err == nil {
		t.Fatal("init is not an app container")
	}
}
