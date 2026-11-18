package config

import "testing"

func TestAddEnvironmentPreservesUnrelated(t *testing.T) {
	cfg := Default()
	cfg.IgnoredContexts = []string{"skip-me"}
	cfg.Shortcuts["dc"] = Shortcut{Environment: "dev", Action: "console"}
	cfg.Settings.ShellShortcuts = true
	cfg.Contexts["Nawaqis-Dev"] = Context{
		Environments: map[string]Environment{"dev": {Namespace: "dev-core"}},
	}
	got, err := AddEnvironment(cfg, "Nawaqis-Dev", "jobs", "jobs")
	if err != nil {
		t.Fatal(err)
	}
	if got.AliasForNamespace("Nawaqis-Dev", "jobs") != "jobs" {
		t.Fatal("expected new mapping")
	}
	if got.AliasForNamespace("Nawaqis-Dev", "dev-core") != "dev" {
		t.Fatal("existing mapping lost")
	}
	if got.Shortcuts["dc"].Environment != "dev" || !got.Settings.ShellShortcuts {
		t.Fatalf("settings/shortcuts lost: %+v", got)
	}
	if got.IgnoredContexts[0] != "skip-me" {
		t.Fatal("ignored contexts lost")
	}
	if _, ok := cfg.Contexts["Nawaqis-Dev"].Environments["jobs"]; ok {
		t.Fatal("original config must not be mutated")
	}
}

func TestAddEnvironmentDuplicateAliasRejected(t *testing.T) {
	cfg := Default()
	cfg.Contexts["Nawaqis-Dev"] = Context{
		Environments: map[string]Environment{"dev": {Namespace: "dev-core"}},
	}
	_, err := AddEnvironment(cfg, "other", "jobs", "dev")
	if err == nil {
		t.Fatal("expected duplicate alias error")
	}
}

func TestAddEnvironmentFromNewNamespaceAndContext(t *testing.T) {
	cfg := Default()
	cfg.Contexts["Nawaqis-Dev"] = Context{
		Environments: map[string]Environment{"dev": {Namespace: "dev-core"}},
	}
	got, err := AddEnvironment(cfg, "company-production", "prod-core", "prod")
	if err != nil {
		t.Fatal(err)
	}
	if got.AliasForNamespace("company-production", "prod-core") != "prod" {
		t.Fatal("expected new context mapping")
	}
	if len(got.Contexts["Nawaqis-Dev"].Environments) != 1 {
		t.Fatal("existing context changed")
	}
}

func TestIgnoreContextAndNamespace(t *testing.T) {
	cfg := Default()
	cfg.Contexts["Nawaqis-Dev"] = Context{
		Environments: map[string]Environment{"dev": {Namespace: "dev-core"}},
	}
	cfg = IgnoreContext(cfg, "analytics")
	cfg = IgnoreContext(cfg, "analytics")
	if len(cfg.IgnoredContexts) != 1 || cfg.IgnoredContexts[0] != "analytics" {
		t.Fatalf("%v", cfg.IgnoredContexts)
	}
	if !cfg.ContextIgnored("analytics") {
		t.Fatal("expected ignored context")
	}
	cfg = IgnoreNamespace(cfg, "Nawaqis-Dev", "kube-system")
	if !cfg.NamespaceIgnored("Nawaqis-Dev", "kube-system") {
		t.Fatal("expected ignored namespace")
	}
	if cfg.AliasForNamespace("Nawaqis-Dev", "dev-core") != "dev" {
		t.Fatal("environments must remain")
	}
}

func TestCloneIndependent(t *testing.T) {
	cfg := Default()
	cfg.Contexts["Nawaqis-Dev"] = Context{
		Environments: map[string]Environment{"dev": {Namespace: "dev-core"}},
	}
	got := Clone(cfg)
	got.Contexts["Nawaqis-Dev"].Environments["staging"] = Environment{Namespace: "staging-core"}
	if _, ok := cfg.Contexts["Nawaqis-Dev"].Environments["staging"]; ok {
		t.Fatal("clone leaked")
	}
}
