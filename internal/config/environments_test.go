package config

import "testing"

func TestEnvironmentsSortedByAlias(t *testing.T) {
	cfg := Default()
	cfg.Contexts["Nawaqis-Prod"] = Context{
		Environments: map[string]Environment{"production": {Namespace: "prod-core"}},
	}
	cfg.Contexts["Nawaqis-Dev"] = Context{
		Environments: map[string]Environment{
			"staging": {Namespace: "staging-core"},
			"dev":     {Namespace: "dev-core"},
		},
	}
	got := cfg.Environments()
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Alias != "dev" || got[1].Alias != "production" || got[2].Alias != "staging" {
		t.Fatalf("%+v", got)
	}
	if got[0].KubeContext != "Nawaqis-Dev" || got[0].Namespace != "dev-core" {
		t.Fatalf("%+v", got[0])
	}
}

func TestAliasForNamespace(t *testing.T) {
	cfg := Default()
	cfg.Contexts["Nawaqis-Dev"] = Context{
		Environments: map[string]Environment{"dev": {Namespace: "dev-core"}},
	}
	if cfg.AliasForNamespace("Nawaqis-Dev", "dev-core") != "dev" {
		t.Fatal("expected mapping")
	}
	if cfg.AliasForNamespace("Nawaqis-Dev", "kube-system") != "" {
		t.Fatal("namespace is not an environment")
	}
}
