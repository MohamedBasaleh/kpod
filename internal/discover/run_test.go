package discover

import (
	"context"
	"errors"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/state"
)

type fakeCluster struct {
	contexts   []string
	ctxErr     error
	namespaces map[string][]string
	nsErr      map[string]error
	ctxCalls   int
	nsCalls    []string
}

func (f *fakeCluster) ListContexts(ctx context.Context, opts ...kube.ListOption) ([]kube.Context, error) {
	f.ctxCalls++
	if f.ctxErr != nil {
		return nil, f.ctxErr
	}
	out := make([]kube.Context, 0, len(f.contexts))
	for _, n := range f.contexts {
		out = append(out, kube.Context{Name: n})
	}
	return out, nil
}

func (f *fakeCluster) ListNamespaces(ctx context.Context, kubeContext string, opts ...kube.ListOption) ([]kube.Namespace, error) {
	f.nsCalls = append(f.nsCalls, kubeContext)
	if err := f.nsErr[kubeContext]; err != nil {
		return nil, err
	}
	raw := f.namespaces[kubeContext]
	out := make([]kube.Namespace, 0, len(raw))
	for _, n := range raw {
		out = append(out, kube.Namespace{Name: n})
	}
	return out, nil
}

func readyState() state.State {
	st := state.Default()
	st.DiscoveryReady = true
	st.SeenContexts = []string{"Nawaqis-Dev"}
	st.SeenNamespaces = map[string][]string{
		"Nawaqis-Dev": {"dev-core", "staging-core"},
	}
	return st
}

func TestRunNoChanges(t *testing.T) {
	cfg := testCfg()
	cluster := &fakeCluster{
		contexts: []string{"Nawaqis-Dev"},
		namespaces: map[string][]string{
			"Nawaqis-Dev": {"dev-core", "staging-core"},
		},
	}
	out := Run(context.Background(), cluster, cfg, readyState(), Options{})
	if !out.Result.Empty() {
		t.Fatalf("%+v", out.Result)
	}
	if out.StateDirty {
		t.Fatal("no baseline expected")
	}
}

func TestRunNewContextAndNamespaces(t *testing.T) {
	cfg := testCfg()
	cluster := &fakeCluster{
		contexts: []string{"Nawaqis-Dev", "company-production"},
		namespaces: map[string][]string{
			"Nawaqis-Dev": {"dev-core", "staging-core", "jobs", "monitoring"},
		},
	}
	out := Run(context.Background(), cluster, cfg, readyState(), Options{})
	if len(out.Result.NewContexts) != 1 || out.Result.NewContexts[0] != "company-production" {
		t.Fatalf("%v", out.Result.NewContexts)
	}
	got := out.Result.NewNamespaces["Nawaqis-Dev"]
	if len(got) != 2 || got[0] != "jobs" || got[1] != "monitoring" {
		t.Fatalf("%v", got)
	}
	if state.Contains(out.State.SeenContexts, "company-production") {
		t.Fatal("new context must not be marked seen before the user acts")
	}
	if state.Contains(out.State.SeenNamespaces["Nawaqis-Dev"], "jobs") {
		t.Fatal("new namespace must not be marked seen before the user acts")
	}
}

func TestRunDoesNotQueryUnconfiguredContexts(t *testing.T) {
	cfg := testCfg()
	cluster := &fakeCluster{
		contexts: []string{"Nawaqis-Dev", "company-production"},
		namespaces: map[string][]string{
			"Nawaqis-Dev":        {"dev-core"},
			"company-production": {"prod"},
		},
	}
	st := readyState()
	_ = Run(context.Background(), cluster, cfg, st, Options{})
	for _, name := range cluster.nsCalls {
		if name == "company-production" {
			t.Fatal("must not eagerly list namespaces for unconfigured contexts")
		}
	}
}

func TestRunFirstBaselineIsQuiet(t *testing.T) {
	cfg := testCfg()
	cluster := &fakeCluster{
		contexts: []string{"Nawaqis-Dev", "company-production"},
		namespaces: map[string][]string{
			"Nawaqis-Dev": {"dev-core", "staging-core", "jobs"},
		},
	}
	out := Run(context.Background(), cluster, cfg, state.Default(), Options{})
	if !out.Result.Empty() {
		t.Fatalf("first inventory must baseline, not notify: %+v", out.Result)
	}
	if !out.StateDirty || !out.State.DiscoveryReady {
		t.Fatal("expected persisted baseline")
	}
	if !state.Contains(out.State.SeenContexts, "company-production") {
		t.Fatal("visible onboarding contexts should become seen")
	}
	if !state.Contains(out.State.SeenNamespaces["Nawaqis-Dev"], "jobs") {
		t.Fatal("observed namespaces should become seen on first baseline")
	}
}

func TestRunUpgradeDoesNotFalsePositiveConfigured(t *testing.T) {
	cfg := testCfg()
	cluster := &fakeCluster{
		contexts: []string{"Nawaqis-Dev"},
		namespaces: map[string][]string{
			"Nawaqis-Dev": {"dev-core", "staging-core"},
		},
	}
	st := state.Default() // Phase 9 upgrade: seen empty, discovery_ready missing
	out := Run(context.Background(), cluster, cfg, st, Options{})
	if !out.Result.Empty() {
		t.Fatalf("%+v", out.Result)
	}
}

func TestRunForbiddenDoesNotWipeSeen(t *testing.T) {
	cfg := testCfg()
	st := readyState()
	cluster := &fakeCluster{
		contexts: []string{"Nawaqis-Dev"},
		nsErr:    map[string]error{"Nawaqis-Dev": kube.ErrForbidden},
	}
	out := Run(context.Background(), cluster, cfg, st, Options{})
	if len(out.Result.Unavailable) != 1 {
		t.Fatalf("%v", out.Result.Unavailable)
	}
	if _, ok := out.Result.NewNamespaces["Nawaqis-Dev"]; ok {
		t.Fatal("forbidden is not zero namespaces")
	}
	if !state.Contains(out.State.SeenNamespaces["Nawaqis-Dev"], "dev-core") {
		t.Fatal("must not wipe seen namespaces")
	}
}

func TestRunUnauthorized(t *testing.T) {
	cfg := testCfg()
	cluster := &fakeCluster{
		contexts: []string{"Nawaqis-Dev"},
		nsErr:    map[string]error{"Nawaqis-Dev": kube.ErrUnauthorized},
	}
	out := Run(context.Background(), cluster, cfg, readyState(), Options{})
	if len(out.Result.Unavailable) != 1 {
		t.Fatalf("%v", out.Result.Unavailable)
	}
}

func TestRunContextUnavailableForNamespaces(t *testing.T) {
	cfg := testCfg()
	cluster := &fakeCluster{
		contexts: []string{"Nawaqis-Dev"},
		nsErr:    map[string]error{"Nawaqis-Dev": kube.ErrContextNotFound},
	}
	out := Run(context.Background(), cluster, cfg, readyState(), Options{})
	if len(out.Result.Unavailable) != 1 {
		t.Fatalf("%v", out.Result.Unavailable)
	}
	if len(out.Result.NewNamespaces) != 0 {
		t.Fatalf("%v", out.Result.NewNamespaces)
	}
}

func TestRunContextListFailure(t *testing.T) {
	cfg := testCfg()
	cluster := &fakeCluster{
		ctxErr: errors.New("no kubeconfig"),
		namespaces: map[string][]string{
			"Nawaqis-Dev": {"dev-core", "staging-core", "jobs"},
		},
	}
	out := Run(context.Background(), cluster, cfg, readyState(), Options{})
	if out.Result.ContextsErr == nil {
		t.Fatal("expected context list error")
	}
	if len(out.Result.NewContexts) != 0 {
		t.Fatalf("%v", out.Result.NewContexts)
	}
	if len(out.Result.NewNamespaces["Nawaqis-Dev"]) != 1 {
		t.Fatalf("namespace discovery should still run for managed contexts: %v", out.Result.NewNamespaces)
	}
}

func TestRunMultipleContextsNamespaces(t *testing.T) {
	cfg := testCfg()
	cfg.Contexts["other"] = config.Context{
		Environments: map[string]config.Environment{"jobs": {Namespace: "jobs"}},
	}
	st := readyState()
	st.SeenContexts = append(st.SeenContexts, "other")
	st.SeenNamespaces["other"] = []string{"jobs"}
	cluster := &fakeCluster{
		contexts: []string{"Nawaqis-Dev", "other"},
		namespaces: map[string][]string{
			"Nawaqis-Dev": {"dev-core", "staging-core"},
			"other":       {"jobs", "monitoring"},
		},
	}
	out := Run(context.Background(), cluster, cfg, st, Options{})
	if len(out.Result.NewNamespaces) != 1 || out.Result.NewNamespaces["other"][0] != "monitoring" {
		t.Fatalf("%v", out.Result.NewNamespaces)
	}
}

func TestRunNilCluster(t *testing.T) {
	out := Run(context.Background(), nil, testCfg(), readyState(), Options{})
	if !errors.Is(out.Result.ContextsErr, kube.ErrKubectlNotFound) {
		t.Fatalf("%v", out.Result.ContextsErr)
	}
}
