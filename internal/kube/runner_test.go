package kube

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestExecRunnerSuccess(t *testing.T) {
	res, err := ExecRunner{}.Run(context.Background(), "/bin/echo", "-n", "ok")
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit %d", res.ExitCode)
	}
	if string(res.Stdout) != "ok" {
		t.Fatalf("stdout %q", res.Stdout)
	}
}

func TestExecRunnerNonZeroExit(t *testing.T) {
	res, err := ExecRunner{}.Run(context.Background(), "/bin/sh", "-c", "echo fail >&2; exit 3")
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 3 {
		t.Fatalf("exit %d", res.ExitCode)
	}
	if !strings.Contains(string(res.Stderr), "fail") {
		t.Fatalf("stderr %q", res.Stderr)
	}
}

func TestExecRunnerCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, err := ExecRunner{}.Run(ctx, "/bin/sleep", "5")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestExecRunnerTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := ExecRunner{}.Run(ctx, "/bin/sleep", "5")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}

func TestExecRunnerMissingBinary(t *testing.T) {
	_, err := ExecRunner{}.Run(context.Background(), "kpod-kubectl-does-not-exist-xyz")
	if !errors.Is(err, ErrKubectlNotFound) {
		t.Fatalf("got %v", err)
	}
}
