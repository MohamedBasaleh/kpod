package kube

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestOwnerDeploymentWalk(t *testing.T) {
	r := &fakeRunner{handle: func(ctx context.Context, name string, args []string) (Result, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "get pod api-0"):
			return Result{Stdout: []byte(`{"metadata":{"ownerReferences":[{"kind":"ReplicaSet","name":"api-rs"}]}}`)}, nil
		case strings.Contains(joined, "get rs api-rs"):
			return Result{Stdout: []byte(`{"metadata":{"ownerReferences":[{"kind":"Deployment","name":"api"}]}}`)}, nil
		default:
			return Result{ExitCode: 1, Stderr: []byte("unexpected " + joined)}, nil
		}
	}}
	svc := newTestService(t, r)
	got, err := svc.OwnerDeployment(context.Background(), "Nawaqis-Dev", "dev-core", "api-0")
	if err != nil {
		t.Fatal(err)
	}
	if got != "api" {
		t.Fatalf("got %q", got)
	}
	assertNoUseContext(t, r)
	for _, args := range r.allArgs() {
		requireArgPair(t, args, "--context", "Nawaqis-Dev")
		requireArgPair(t, args, "-n", "dev-core")
	}
}

func TestOwnerDeploymentNoReplicaSet(t *testing.T) {
	r := &fakeRunner{stdout: `{"metadata":{"ownerReferences":[{"kind":"Job","name":"job-1"}]}}`}
	svc := newTestService(t, r)
	_, err := svc.OwnerDeployment(context.Background(), "ctx", "ns", "pod-1")
	if !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestOwnerDeploymentMalformedJSON(t *testing.T) {
	r := &fakeRunner{stdout: `{not json`}
	svc := newTestService(t, r)
	_, err := svc.OwnerDeployment(context.Background(), "ctx", "ns", "pod-1")
	if !errors.Is(err, ErrMalformedOutput) {
		t.Fatalf("got %v", err)
	}
}
