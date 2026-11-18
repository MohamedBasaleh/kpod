package kube

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCacheHitSkipsSecondCall(t *testing.T) {
	now := time.Now()
	r := &fakeRunner{stdout: "Nawaqis-Dev\n"}
	svc := newTestService(t, r, WithNow(func() time.Time { return now }), WithCacheTTL(CacheTTL{
		Contexts:   30 * time.Second,
		Namespaces: 20 * time.Second,
		Pods:       4 * time.Second,
	}))
	if _, err := svc.ListContexts(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListContexts(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.callCount() != 1 {
		t.Fatalf("calls %d", r.callCount())
	}
}

func TestCacheExpiryReexecutes(t *testing.T) {
	now := time.Now()
	r := &fakeRunner{stdout: "Nawaqis-Dev\n"}
	svc := newTestService(t, r, WithNow(func() time.Time { return now }), WithCacheTTL(CacheTTL{Contexts: 30 * time.Second}))
	if _, err := svc.ListContexts(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(31 * time.Second)
	if _, err := svc.ListContexts(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.callCount() != 2 {
		t.Fatalf("calls %d", r.callCount())
	}
}

func TestCacheKeysDoNotCollide(t *testing.T) {
	r := &fakeRunner{
		handle: func(_ context.Context, _ string, args []string) (Result, error) {
			stdout := `{"items":[]}`
			if len(args) > 0 && args[0] == "config" {
				stdout = "Nawaqis-Dev\n"
			}
			return Result{Name: "kubectl", Args: args, Stdout: []byte(stdout)}, nil
		},
	}
	svc := newTestService(t, r)
	if _, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListNamespaces(context.Background(), "Nawaqis-Prod"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListPods(context.Background(), "Nawaqis-Dev", "dev-core"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListPods(context.Background(), "Nawaqis-Dev", "staging-core"); err != nil {
		t.Fatal(err)
	}
	if r.callCount() != 4 {
		t.Fatalf("calls %d", r.callCount())
	}
	if _, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev"); err != nil {
		t.Fatal(err)
	}
	if r.callCount() != 4 {
		t.Fatalf("cache miss after hit, calls %d", r.callCount())
	}
}

func TestForceRefreshBypassesCache(t *testing.T) {
	r := &fakeRunner{stdout: "Nawaqis-Dev\n"}
	svc := newTestService(t, r)
	if _, err := svc.ListContexts(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListContexts(context.Background(), Refresh()); err != nil {
		t.Fatal(err)
	}
	if r.callCount() != 2 {
		t.Fatalf("calls %d", r.callCount())
	}
}

func TestFailuresAreNotCachedAsSuccess(t *testing.T) {
	r := &fakeRunner{exit: 1, stderr: `Error from server (Forbidden): namespaces is forbidden`}
	svc := newTestService(t, r)
	if _, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev"); err == nil {
		t.Fatal("expected error")
	}
	r.exit = 0
	r.stderr = ""
	r.stdout = `{"items":[{"metadata":{"name":"dev-core"}}]}`
	got, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
	if r.callCount() != 2 {
		t.Fatalf("calls %d", r.callCount())
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	r := &fakeRunner{
		handle: func(_ context.Context, _ string, args []string) (Result, error) {
			time.Sleep(5 * time.Millisecond)
			return Result{Name: "kubectl", Args: args, Stdout: []byte("Nawaqis-Dev\n")}, nil
		},
	}
	svc := newTestService(t, r)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.ListContexts(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestCommandConstructionNeverUseContext(t *testing.T) {
	r := &fakeRunner{
		handle: func(_ context.Context, _ string, args []string) (Result, error) {
			stdout := `{"items":[]}`
			if len(args) > 0 && args[0] == "config" {
				stdout = "Nawaqis-Dev\n"
			}
			if len(args) > 0 && args[len(args)-1] == "pods" && args[len(args)-2] == "get" && contains(args, "auth") {
				stdout = "yes\n"
			} else if contains(args, "auth") {
				stdout = "yes\n"
			}
			return Result{Name: "kubectl", Args: args, Stdout: []byte(stdout)}, nil
		},
	}
	svc := newTestService(t, r)
	ctx := context.Background()
	if _, err := svc.ListContexts(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListNamespaces(ctx, "Nawaqis-Dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CanAccessNamespace(ctx, "Nawaqis-Dev", "dev-core"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListPods(ctx, "Nawaqis-Dev", "dev-core"); err != nil {
		t.Fatal(err)
	}
	assertNoUseContext(t, r)
	nsArgs := r.allArgs()[1]
	requireArgPair(t, nsArgs, "--context", "Nawaqis-Dev")
	podArgs := r.allArgs()[3]
	requireArgPair(t, podArgs, "--context", "Nawaqis-Dev")
	requireArgPair(t, podArgs, "-n", "dev-core")
}

func contains(args []string, v string) bool {
	for _, a := range args {
		if a == v {
			return true
		}
	}
	return false
}
