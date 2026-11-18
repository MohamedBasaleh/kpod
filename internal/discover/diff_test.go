package discover

import (
	"errors"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/state"
)

func testCfg() config.Config {
	cfg := config.Default()
	cfg.Contexts["Nawaqis-Dev"] = config.Context{
		Environments: map[string]config.Environment{
			"dev":     {Namespace: "dev-core"},
			"staging": {Namespace: "staging-core"},
		},
	}
	return cfg
}

func TestDiffContextsNoChanges(t *testing.T) {
	cfg := testCfg()
	st := state.Default()
	st.DiscoveryReady = true
	st.SeenContexts = []string{"Nawaqis-Dev", "other"}
	got := DiffContexts([]string{"Nawaqis-Dev", "other"}, cfg, st)
	if len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestDiffContextsOneNew(t *testing.T) {
	cfg := testCfg()
	st := state.Default()
	st.DiscoveryReady = true
	st.SeenContexts = []string{"Nawaqis-Dev"}
	got := DiffContexts([]string{"Nawaqis-Dev", "company-production"}, cfg, st)
	if len(got) != 1 || got[0] != "company-production" {
		t.Fatalf("%v", got)
	}
}

func TestDiffContextsMultipleNew(t *testing.T) {
	cfg := testCfg()
	st := state.Default()
	st.DiscoveryReady = true
	got := DiffContexts([]string{"analytics", "company-production", "Nawaqis-Dev"}, cfg, st)
	if len(got) != 2 || got[0] != "analytics" || got[1] != "company-production" {
		t.Fatalf("%v", got)
	}
}

func TestDiffContextsConfiguredNotNew(t *testing.T) {
	cfg := testCfg()
	st := state.Default()
	st.DiscoveryReady = true
	got := DiffContexts([]string{"Nawaqis-Dev"}, cfg, st)
	if len(got) != 0 {
		t.Fatalf("configured context must not be new: %v", got)
	}
}

func TestDiffContextsIgnoredSuppressed(t *testing.T) {
	cfg := testCfg()
	cfg.IgnoredContexts = []string{"analytics"}
	st := state.Default()
	st.DiscoveryReady = true
	got := DiffContexts([]string{"Nawaqis-Dev", "analytics"}, cfg, st)
	if len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestDiffContextsSeenUnconfiguredNotNew(t *testing.T) {
	cfg := testCfg()
	st := state.Default()
	st.DiscoveryReady = true
	st.SeenContexts = []string{"analytics"}
	got := DiffContexts([]string{"Nawaqis-Dev", "analytics"}, cfg, st)
	if len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestDiffNamespacesOneAndSeveral(t *testing.T) {
	cfg := testCfg()
	st := state.Default()
	st.SeenNamespaces = map[string][]string{
		"Nawaqis-Dev": {"dev-core", "staging-core"},
	}
	got := DiffNamespaces("Nawaqis-Dev", []string{"dev-core", "staging-core", "jobs"}, cfg, st)
	if len(got) != 1 || got[0] != "jobs" {
		t.Fatalf("%v", got)
	}
	got = DiffNamespaces("Nawaqis-Dev", []string{"dev-core", "staging-core", "jobs", "monitoring"}, cfg, st)
	if len(got) != 2 || got[0] != "jobs" || got[1] != "monitoring" {
		t.Fatalf("%v", got)
	}
}

func TestDiffNamespacesIgnoredAndSeenAndConfigured(t *testing.T) {
	cfg := testCfg()
	ctx := cfg.Contexts["Nawaqis-Dev"]
	ctx.IgnoredNamespaces = []string{"kube-system"}
	cfg.Contexts["Nawaqis-Dev"] = ctx
	st := state.Default()
	st.SeenNamespaces = map[string][]string{
		"Nawaqis-Dev": {"dev-core", "staging-core", "jobs"},
	}
	got := DiffNamespaces("Nawaqis-Dev", []string{"dev-core", "staging-core", "jobs", "kube-system", "monitoring"}, cfg, st)
	if len(got) != 1 || got[0] != "monitoring" {
		t.Fatalf("%v", got)
	}
}

func TestDiffNamespacesZero(t *testing.T) {
	cfg := testCfg()
	st := state.Default()
	st.SeenNamespaces = map[string][]string{"Nawaqis-Dev": {}}
	got := DiffNamespaces("Nawaqis-Dev", nil, cfg, st)
	if len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestBaselineDoesNotReportExisting(t *testing.T) {
	cfg := testCfg()
	st := state.Default()
	ApplyContextBaseline(&st, []string{"Nawaqis-Dev", "other"}, cfg)
	if !st.DiscoveryReady {
		t.Fatal("expected discovery ready")
	}
	got := DiffContexts([]string{"Nawaqis-Dev", "other"}, cfg, st)
	if len(got) != 0 {
		t.Fatalf("baseline should consume current contexts: %v", got)
	}
	ApplyNamespaceBaseline(&st, "Nawaqis-Dev", []string{"dev-core", "staging-core", "jobs"}, cfg)
	gotNS := DiffNamespaces("Nawaqis-Dev", []string{"dev-core", "staging-core", "jobs"}, cfg, st)
	if len(gotNS) != 0 {
		t.Fatalf("%v", gotNS)
	}
}

func TestLaterDoesNotPersistIgnore(t *testing.T) {
	s := NewSession()
	s.LaterContext("analytics")
	s.LaterNamespace("Nawaqis-Dev", "jobs")
	r := Result{
		NewContexts: []string{"analytics", "other"},
		NewNamespaces: map[string][]string{
			"Nawaqis-Dev": {"jobs", "monitoring"},
		},
	}
	vis := Visible(r, s)
	if len(vis.NewContexts) != 1 || vis.NewContexts[0] != "other" {
		t.Fatalf("%v", vis.NewContexts)
	}
	if len(vis.NewNamespaces["Nawaqis-Dev"]) != 1 || vis.NewNamespaces["Nawaqis-Dev"][0] != "monitoring" {
		t.Fatalf("%v", vis.NewNamespaces)
	}
	if r.NewContexts[0] != "analytics" {
		t.Fatal("Later must not mutate the stored result used for persistence")
	}
}

func TestFormatEmptyAndNonEmpty(t *testing.T) {
	if Format(Result{NewNamespaces: map[string][]string{}}) != "No new Kubernetes contexts or namespaces.\n" {
		t.Fatal("empty")
	}
	out := Format(Result{
		NewContexts: []string{"company-production"},
		NewNamespaces: map[string][]string{
			"Nawaqis-Dev": {"jobs"},
		},
	})
	if !strings.Contains(out, "company-production") || !strings.Contains(out, "jobs") {
		t.Fatalf("%s", out)
	}
	failed := Format(Result{ContextsErr: errors.New("boom"), NewNamespaces: map[string][]string{}})
	if !strings.Contains(failed, "Unable to list Kubernetes contexts") {
		t.Fatalf("%s", failed)
	}
}
