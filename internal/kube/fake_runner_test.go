package kube

import (
	"context"
	"strings"
	"sync"
	"testing"
)

type runCall struct {
	name string
	args []string
}

type fakeRunner struct {
	mu     sync.Mutex
	calls  []runCall
	handle func(ctx context.Context, name string, args []string) (Result, error)
	stdout string
	stderr string
	exit   int
	err    error
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, runCall{name: name, args: append([]string(nil), args...)})
	handle := f.handle
	stdout, stderr, exit, runErr := f.stdout, f.stderr, f.exit, f.err
	f.mu.Unlock()

	if handle != nil {
		return handle(ctx, name, args)
	}
	return Result{
		Name:     name,
		Args:     append([]string(nil), args...),
		Stdout:   []byte(stdout),
		Stderr:   []byte(stderr),
		ExitCode: exit,
	}, runErr
}

func (f *fakeRunner) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeRunner) lastArgs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return nil
	}
	return f.calls[len(f.calls)-1].args
}

func (f *fakeRunner) allArgs() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]string, len(f.calls))
	for i, c := range f.calls {
		out[i] = append([]string(nil), c.args...)
	}
	return out
}

func newTestService(t *testing.T, r *fakeRunner, opts ...Option) *Service {
	t.Helper()
	all := []Option{WithRunner(r), WithKubectl("kubectl")}
	all = append(all, opts...)
	s, err := New(all...)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func assertNoUseContext(t *testing.T, r *fakeRunner) {
	t.Helper()
	for _, args := range r.allArgs() {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "config use-context") || hasUseContext(args) {
			t.Fatalf("command used config use-context: %q", args)
		}
	}
}

func hasUseContext(args []string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "config" && args[i+1] == "use-context" {
			return true
		}
	}
	return false
}

func requireArgPair(t *testing.T, args []string, flag, value string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return
		}
	}
	t.Fatalf("missing %s %s in %q", flag, value, args)
}
