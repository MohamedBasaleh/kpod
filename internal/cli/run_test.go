package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
)

type fakeRunner struct {
	calls    [][]string
	attached [][]string
	handle   func(args []string) (kube.Result, error)
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (kube.Result, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if f.handle != nil {
		return f.handle(args)
	}
	return kube.Result{Name: name, Args: args}, nil
}

func (f *fakeRunner) RunAttached(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, name string, args ...string) error {
	f.attached = append(f.attached, append([]string(nil), args...))
	return nil
}

func writeCfg(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := "version: 2\ncontexts:\n  Nawaqis-Dev:\n    environments:\n      dev:\n        namespace: dev-core\n      staging:\n        namespace: staging-core\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func item(name, phase string) string {
	running := `{"running":{}}`
	if phase != "Running" {
		running = `{"terminated":{"reason":"` + phase + `"}}`
	}
	return `{"metadata":{"name":"` + name + `","namespace":"dev-core","creationTimestamp":"2026-01-01T00:00:00Z"},"spec":{"containers":[{"name":"api"}]},"status":{"phase":"` + phase + `","containerStatuses":[{"name":"api","ready":true,"state":` + running + `}]}}`
}

func listJSON(items ...string) string {
	return `{"items":[` + strings.Join(items, ",") + `]}`
}

func TestRunUnknownEnv(t *testing.T) {
	dir := writeCfg(t)
	inv, _ := Parse([]string{"deev", "console"})
	err := Run(Options{Paths: config.NewPaths(dir), Inv: inv})
	if err == nil || !strings.Contains(err.Error(), "deev") {
		t.Fatalf("%v", err)
	}
}

func TestRunConsoleFastPath(t *testing.T) {
	dir := writeCfg(t)
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		if strings.Contains(strings.Join(args, " "), "get pods") {
			return kube.Result{Stdout: []byte(listJSON(item("dev-core-api-1", "Running")))}, nil
		}
		return kube.Result{}, nil
	}}
	svc, err := kube.New(kube.WithRunner(r), kube.WithKubectl("kubectl"))
	if err != nil {
		t.Fatal(err)
	}
	inv, _ := Parse([]string{"dev", "c"})
	if err := Run(Options{Paths: config.NewPaths(dir), Inv: inv, Kube: svc}); err != nil {
		t.Fatal(err)
	}
	if len(r.attached) != 1 {
		t.Fatalf("attached %d", len(r.attached))
	}
	args := r.attached[0]
	if !contains(args, "--context") || !contains(args, "Nawaqis-Dev") || !contains(args, "-n") {
		t.Fatalf("%q", args)
	}
	if contains(args, "use-context") {
		t.Fatal(args)
	}
	if !contains(args, "rails") || !contains(args, "console") {
		t.Fatalf("expected rails console: %q", args)
	}
}

func TestRunLogsPodFilter(t *testing.T) {
	dir := writeCfg(t)
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		if strings.Contains(strings.Join(args, " "), "get pods") {
			return kube.Result{Stdout: []byte(listJSON(
				item("dev-core-api-1", "Running"),
				item("dev-core-worker-1", "Running"),
			))}, nil
		}
		return kube.Result{}, nil
	}}
	svc, _ := kube.New(kube.WithRunner(r), kube.WithKubectl("kubectl"))
	inv, _ := Parse([]string{"dev", "logs", "api"})
	if err := Run(Options{Paths: config.NewPaths(dir), Inv: inv, Kube: svc}); err != nil {
		t.Fatal(err)
	}
	if len(r.attached) != 1 || !contains(r.attached[0], "dev-core-api-1") || !contains(r.attached[0], "-f") {
		t.Fatalf("%q", r.attached)
	}
}

func TestRunNoMatch(t *testing.T) {
	dir := writeCfg(t)
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		return kube.Result{Stdout: []byte(listJSON(item("dev-core-api-1", "Running")))}, nil
	}}
	svc, _ := kube.New(kube.WithRunner(r), kube.WithKubectl("kubectl"))
	inv, _ := Parse([]string{"dev", "logs", "zzz"})
	err := Run(Options{Paths: config.NewPaths(dir), Inv: inv, Kube: svc})
	if err == nil || !strings.Contains(err.Error(), "no pods matching") {
		t.Fatalf("%v", err)
	}
}

func TestRunMultipleNoTTY(t *testing.T) {
	dir := writeCfg(t)
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		return kube.Result{Stdout: []byte(listJSON(
			item("dev-core-api-1", "Running"),
			item("dev-core-api-2", "Running"),
		))}, nil
	}}
	svc, _ := kube.New(kube.WithRunner(r), kube.WithKubectl("kubectl"))
	inv, _ := Parse([]string{"dev", "logs", "api"})
	err := Run(Options{Paths: config.NewPaths(dir), Inv: inv, Kube: svc})
	if err == nil || !strings.Contains(err.Error(), "multiple pods match") {
		t.Fatalf("%v", err)
	}
}

func TestRunMultipleSelector(t *testing.T) {
	dir := writeCfg(t)
	r := &fakeRunner{handle: func(args []string) (kube.Result, error) {
		if strings.Contains(strings.Join(args, " "), "get pods") {
			return kube.Result{Stdout: []byte(listJSON(
				item("dev-core-api-1", "Running"),
				item("dev-core-api-2", "Running"),
			))}, nil
		}
		return kube.Result{Stdout: []byte("ok")}, nil
	}}
	svc, _ := kube.New(kube.WithRunner(r), kube.WithKubectl("kubectl"))
	inv, _ := Parse([]string{"dev", "describe", "api"})
	var out bytes.Buffer
	err := Run(Options{
		Paths: config.NewPaths(dir), Inv: inv, Kube: svc, Out: &out,
		SelectFn: func(label string, names []string) (string, error) {
			return "dev-core-api-2", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range r.calls {
		if contains(c, "describe") && contains(c, "dev-core-api-2") {
			found = true
		}
	}
	if !found {
		t.Fatalf("calls %+v", r.calls)
	}
}

func contains(args []string, s string) bool {
	for _, a := range args {
		if a == s {
			return true
		}
	}
	return false
}
