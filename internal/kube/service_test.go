package kube

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestNewWithoutKubectl(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := New()
	if !errors.Is(err, ErrKubectlNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestEmptyContextRejected(t *testing.T) {
	r := &fakeRunner{stdout: `{"items":[]}`}
	svc := newTestService(t, r)
	if _, err := svc.ListNamespaces(context.Background(), "  "); err == nil {
		t.Fatal("expected error")
	}
	if r.callCount() != 0 {
		t.Fatalf("should not call kubectl, calls %d", r.callCount())
	}
}

func TestValidateKubectl(t *testing.T) {
	r := &fakeRunner{stdout: "Client Version: v1.31.0\n"}
	svc := newTestService(t, r)
	if err := svc.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	args := r.lastArgs()
	if len(args) < 2 || args[0] != "version" || args[1] != "--client" {
		t.Fatalf("args %q", args)
	}
	assertNoUseContext(t, r)
}

func TestLookPathUsedWhenPresent(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	svc := newTestService(t, &fakeRunner{stdout: "ok"})
	if svc.kubectl != "kubectl" {
		t.Fatalf("kubectl name %s", svc.kubectl)
	}
}
