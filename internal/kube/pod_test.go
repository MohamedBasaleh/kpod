package kube

import (
	"context"
	"errors"
	"testing"
	"time"
)

func podList(items string) string {
	return `{"items":[` + items + `]}`
}

func TestListPodsMultipleAndContainers(t *testing.T) {
	now := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	body := podList(`{
		"metadata": {"name": "api-0", "namespace": "dev-core", "creationTimestamp": "2024-01-01T00:00:00Z"},
		"spec": {"nodeName": "ip-10-0-1-2", "containers": [{"name": "web"}, {"name": "sidecar"}]},
		"status": {
			"phase": "Running",
			"containerStatuses": [
				{"name": "web", "ready": true, "restartCount": 2, "state": {"running": {"startedAt": "2024-01-01T00:01:00Z"}}},
				{"name": "sidecar", "ready": true, "restartCount": 1, "state": {"running": {"startedAt": "2024-01-01T00:01:00Z"}}}
			]
		}
	}`)
	r := &fakeRunner{stdout: body}
	svc := newTestService(t, r, WithNow(func() time.Time { return now }))
	got, err := svc.ListPods(context.Background(), "Nawaqis-Dev", "dev-core")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len %d", len(got))
	}
	p := got[0]
	if p.Name != "api-0" || p.Status != "Running" || p.Ready != 2 || p.Total != 2 || p.Restarts != 3 {
		t.Fatalf("pod %+v", p)
	}
	if p.Node != "ip-10-0-1-2" {
		t.Fatalf("node %q", p.Node)
	}
	if len(p.Containers) != 2 {
		t.Fatalf("containers %+v", p.Containers)
	}
	requireArgPair(t, r.lastArgs(), "--context", "Nawaqis-Dev")
	requireArgPair(t, r.lastArgs(), "-n", "dev-core")
	assertNoUseContext(t, r)
}

func TestListPodsZero(t *testing.T) {
	r := &fakeRunner{stdout: `{"items":[]}`}
	svc := newTestService(t, r)
	got, err := svc.ListPods(context.Background(), "Nawaqis-Dev", "dev-core")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestListPodsPendingMissingNode(t *testing.T) {
	body := podList(`{
		"metadata": {"name": "wait-0", "namespace": "dev-core", "creationTimestamp": "2024-01-01T00:00:00Z"},
		"spec": {"containers": [{"name": "web"}]},
		"status": {"phase": "Pending"}
	}`)
	r := &fakeRunner{stdout: body}
	svc := newTestService(t, r)
	got, err := svc.ListPods(context.Background(), "Nawaqis-Dev", "dev-core")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Status != "Pending" || got[0].Node != "" || got[0].Ready != 0 {
		t.Fatalf("%+v", got[0])
	}
}

func TestListPodsCompleted(t *testing.T) {
	body := podList(`{
		"metadata": {"name": "job-0", "namespace": "dev-core", "creationTimestamp": "2024-01-01T00:00:00Z"},
		"spec": {"nodeName": "n1", "containers": [{"name": "job"}]},
		"status": {
			"phase": "Succeeded",
			"containerStatuses": [
				{"name": "job", "ready": false, "restartCount": 0, "state": {"terminated": {"reason": "Completed", "exitCode": 0}}}
			]
		}
	}`)
	r := &fakeRunner{stdout: body}
	svc := newTestService(t, r)
	got, err := svc.ListPods(context.Background(), "Nawaqis-Dev", "dev-core")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Status != "Completed" {
		t.Fatalf("status %s", got[0].Status)
	}
}

func TestListPodsCrashLoopBackOff(t *testing.T) {
	body := podList(`{
		"metadata": {"name": "api-1", "namespace": "dev-core", "creationTimestamp": "2024-01-01T00:00:00Z"},
		"spec": {"nodeName": "n1", "containers": [{"name": "web"}]},
		"status": {
			"phase": "Running",
			"containerStatuses": [
				{"name": "web", "ready": false, "restartCount": 12, "state": {"waiting": {"reason": "CrashLoopBackOff"}}}
			]
		}
	}`)
	r := &fakeRunner{stdout: body}
	svc := newTestService(t, r)
	got, err := svc.ListPods(context.Background(), "Nawaqis-Dev", "dev-core")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Status != "CrashLoopBackOff" || got[0].Phase != "Running" || got[0].Restarts != 12 {
		t.Fatalf("%+v", got[0])
	}
}

func TestListPodsImagePullBackOff(t *testing.T) {
	body := podList(`{
		"metadata": {"name": "api-2", "namespace": "dev-core", "creationTimestamp": "2024-01-01T00:00:00Z"},
		"spec": {"nodeName": "n1", "containers": [{"name": "web"}]},
		"status": {
			"phase": "Pending",
			"containerStatuses": [
				{"name": "web", "ready": false, "restartCount": 0, "state": {"waiting": {"reason": "ImagePullBackOff"}}}
			]
		}
	}`)
	r := &fakeRunner{stdout: body}
	svc := newTestService(t, r)
	got, err := svc.ListPods(context.Background(), "Nawaqis-Dev", "dev-core")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Status != "ImagePullBackOff" {
		t.Fatalf("status %s", got[0].Status)
	}
}

func TestListPodsMalformed(t *testing.T) {
	r := &fakeRunner{stdout: "not-json"}
	svc := newTestService(t, r)
	_, err := svc.ListPods(context.Background(), "Nawaqis-Dev", "dev-core")
	if !errors.Is(err, ErrMalformedOutput) {
		t.Fatalf("got %v", err)
	}
}

func TestListPodsARNContext(t *testing.T) {
	arn := "arn:aws:eks:eu-central-1:133136949499:cluster/Nawaqis-Development"
	r := &fakeRunner{stdout: `{"items":[]}`}
	svc := newTestService(t, r)
	if _, err := svc.ListPods(context.Background(), arn, "dev-core"); err != nil {
		t.Fatal(err)
	}
	requireArgPair(t, r.lastArgs(), "--context", arn)
	requireArgPair(t, r.lastArgs(), "-n", "dev-core")
	assertNoUseContext(t, r)
}
