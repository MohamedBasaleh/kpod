package action

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/kube"
)

type fakeRunner struct {
	mu       sync.Mutex
	calls    [][]string
	attached [][]string
	handle   func(args []string) (kube.Result, error)
	attach   func(args []string) error
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (kube.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, append([]string(nil), args...))
	h := f.handle
	f.mu.Unlock()
	if h != nil {
		return h(args)
	}
	return kube.Result{Name: name, Args: args, ExitCode: 0}, nil
}

func (f *fakeRunner) RunAttached(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, name string, args ...string) error {
	f.mu.Lock()
	f.attached = append(f.attached, append([]string(nil), args...))
	fn := f.attach
	f.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if fn != nil {
		return fn(args)
	}
	return nil
}

func (f *fakeRunner) all() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.calls...)
}

func newAct(t *testing.T, r *fakeRunner) *Service {
	t.Helper()
	svc, err := kube.New(kube.WithRunner(r), kube.WithKubectl("kubectl"))
	if err != nil {
		t.Fatal(err)
	}
	return New(svc)
}

func TestShellFallback(t *testing.T) {
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		cmd := args[len(args)-3]
		switch cmd {
		case "bash":
			return kube.Result{Args: args, ExitCode: 1, Stderr: []byte(`exec: "bash": executable file not found in $PATH`)}, nil
		case "sh":
			return kube.Result{Args: args, ExitCode: 0}, nil
		default:
			return kube.Result{Args: args, ExitCode: 1, Stderr: []byte("no")}, nil
		}
	}}
	s := newAct(t, r)
	req := req(Shell, "api-0", "api")
	sh, err := s.ResolveShell(context.Background(), req)
	if err != nil || sh != "sh" {
		t.Fatalf("got %q %v", sh, err)
	}
}

func TestShellFallbackAsh(t *testing.T) {
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		cmd := args[len(args)-3]
		if cmd == "ash" {
			return kube.Result{Args: args, ExitCode: 0}, nil
		}
		return kube.Result{Args: args, ExitCode: 1, Stderr: []byte(`executable file not found`)}, nil
	}}
	s := newAct(t, r)
	sh, err := s.ResolveShell(context.Background(), req(Shell, "api-0", "api"))
	if err != nil || sh != "ash" {
		t.Fatalf("got %q %v", sh, err)
	}
}

func TestShellNoFallbackOnForbidden(t *testing.T) {
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		return kube.Result{Args: args, ExitCode: 1, Stderr: []byte(`Error from server (Forbidden): pods "api-0" is forbidden`)}, nil
	}}
	s := newAct(t, r)
	_, err := s.ResolveShell(context.Background(), req(Shell, "api-0", "api"))
	if !errors.Is(err, kube.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
	if len(r.all()) != 1 {
		t.Fatalf("should not probe next shell, calls %d", len(r.all()))
	}
}

func TestMetricsUnavailable(t *testing.T) {
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		return kube.Result{Args: args, ExitCode: 1, Stderr: []byte("error: Metrics API not available")}, nil
	}}
	s := newAct(t, r)
	_, err := s.Capture(context.Background(), req(Metrics, "api-0", ""))
	if !errors.Is(err, kube.ErrMetricsUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestPreviousLogsUnavailable(t *testing.T) {
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		return kube.Result{Args: args, ExitCode: 1, Stderr: []byte("Error from server (BadRequest): previous terminated container api in pod api-0 not found")}, nil
	}}
	s := newAct(t, r)
	req := req(Logs, "api-0", "api")
	req.Logs = LogsPrevious
	_, err := s.Capture(context.Background(), req)
	if !errors.Is(err, kube.ErrPreviousLogsUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestPortForwardCancel(t *testing.T) {
	r := &fakeRunner{}
	s := newAct(t, r)
	req := req(PortForward, "api-0", "")
	req.LocalPort, req.RemotePort = 3000, 8080
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := s.Attach(ctx, req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if len(r.attached) != 1 || !containsSeq(r.attached[0], "port-forward", "pod/api-0", "3000:8080") {
		t.Fatalf("%q", r.attached)
	}
}

func TestKubectlArgsNeverUseContext(t *testing.T) {
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		if containsArg(args, "bash") || containsArg(args, "-c") {
			return kube.Result{Args: args, ExitCode: 0}, nil
		}
		return kube.Result{Args: args, ExitCode: 0, Stdout: []byte(`{"metadata":{"ownerReferences":[{"kind":"ReplicaSet","name":"rs1"}]}}`)}, nil
	}}
	s := newAct(t, r)
	req := req(Describe, "api-0", "")
	args, err := s.KubectlArgs(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "config use-context") {
		t.Fatal(joined)
	}
}

func TestShellBashSuccessArgs(t *testing.T) {
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		return kube.Result{Args: args, ExitCode: 0}, nil
	}}
	s := newAct(t, r)
	args, err := s.KubectlArgs(context.Background(), req(Shell, "api-0", "api"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(args, "exec", "-it", "api-0", "-c", "api", "--", "bash") {
		t.Fatalf("%q", args)
	}
}
