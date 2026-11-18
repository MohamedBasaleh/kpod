package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	repo := NewRepository(NewPaths(t.TempDir()))
	cfg := Default()
	cfg.Contexts["Nawaqis-Dev"] = Context{
		Environments: map[string]Environment{
			"dev":     {Namespace: "dev-core"},
			"staging": {Namespace: "staging-core"},
		},
	}
	arn := "arn:aws:eks:eu-central-1:133136949499:cluster/Nawaqis-Development"
	cfg.Contexts[arn] = Context{
		Environments: map[string]Environment{
			"prod": {Namespace: "prod-core"},
		},
	}
	if err := repo.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != Version {
		t.Fatalf("version %d", got.Version)
	}
	if got.Contexts["Nawaqis-Dev"].Environments["dev"].Namespace != "dev-core" {
		t.Fatalf("dev mapping: %+v", got.Contexts["Nawaqis-Dev"])
	}
	if got.Contexts["Nawaqis-Dev"].Environments["staging"].Namespace != "staging-core" {
		t.Fatalf("staging mapping: %+v", got.Contexts["Nawaqis-Dev"])
	}
	if got.Contexts[arn].Environments["prod"].Namespace != "prod-core" {
		t.Fatalf("arn mapping: %+v", got.Contexts[arn])
	}
	if !got.Settings.Notifications {
		t.Fatal("notifications default lost")
	}
}

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`version: 2
contexts:
  Nawaqis-Dev:
    environments:
      dev:
        namespace: dev-core
      staging:
        namespace: staging-core
ignored_contexts: []
shortcuts: {}
settings:
  notifications: true
  shell_shortcuts: false
`)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := NewRepository(NewPaths(dir)).Load()
	if err != nil {
		t.Fatal(err)
	}
	env, ok := got.LookupEnvironment("staging")
	if !ok || env.KubeContext != "Nawaqis-Dev" || env.Namespace != "staging-core" {
		t.Fatalf("lookup: %+v ok=%v", env, ok)
	}
}

func TestLoadOlderConfigWithoutIgnoredNamespaces(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`version: 2
contexts:
  my-company-dev:
    environments:
      dev:
        namespace: dev-core
ignored_contexts: []
shortcuts: {}
settings:
  notifications: true
  shell_shortcuts: false
`)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := NewRepository(NewPaths(dir)).Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx := got.Contexts["my-company-dev"]
	if ctx.IgnoredNamespaces == nil {
		t.Fatal("ignored_namespaces should normalize to empty")
	}
	if got.AliasForNamespace("my-company-dev", "dev-core") != "dev" {
		t.Fatal("mapping lost")
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := NewRepository(NewPaths(t.TempDir())).Load()
	if err != ErrNotExist {
		t.Fatalf("got %v", err)
	}
}

func TestMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("version: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewRepository(NewPaths(dir)).Load()
	if err == nil || !strings.Contains(err.Error(), "malformed YAML") {
		t.Fatalf("got %v", err)
	}
}

func TestUnsupportedVersion(t *testing.T) {
	cfg := Default()
	cfg.Version = 99
	err := NewRepository(NewPaths(t.TempDir())).Save(cfg)
	if err == nil || !strings.Contains(err.Error(), "unsupported config version") {
		t.Fatalf("got %v", err)
	}
}

func TestDuplicateAliasAcrossContexts(t *testing.T) {
	cfg := Default()
	cfg.Contexts["ctx-a"] = Context{Environments: map[string]Environment{"dev": {Namespace: "a"}}}
	cfg.Contexts["ctx-b"] = Context{Environments: map[string]Environment{"dev": {Namespace: "b"}}}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), `duplicate environment alias "dev"`) {
		t.Fatalf("got %v", err)
	}
}

func TestEmptyAliasAndNamespace(t *testing.T) {
	cfg := Default()
	cfg.Contexts["ctx"] = Context{Environments: map[string]Environment{"": {Namespace: "ns"}}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected empty alias error")
	}
	cfg = Default()
	cfg.Contexts["ctx"] = Context{Environments: map[string]Environment{"dev": {Namespace: ""}}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected empty namespace error")
	}
	cfg = Default()
	cfg.Contexts[""] = Context{Environments: map[string]Environment{"dev": {Namespace: "ns"}}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected empty context error")
	}
}

func TestInvalidSaveDoesNotChangeExistingFile(t *testing.T) {
	repo := NewRepository(NewPaths(t.TempDir()))
	ok := Default()
	ok.Contexts["Nawaqis-Dev"] = Context{Environments: map[string]Environment{"dev": {Namespace: "dev-core"}}}
	if err := repo.Save(ok); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(repo.Paths().ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	bad := ok
	bad.Contexts["other"] = Context{Environments: map[string]Environment{"dev": {Namespace: "x"}}}
	if err := repo.Save(bad); err == nil {
		t.Fatal("expected duplicate alias error")
	}
	after, err := os.ReadFile(repo.Paths().ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("existing config was modified after failed save")
	}
}

func TestDefaultPathsHonorsEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KPOD_CONFIG_DIR", dir)
	paths, err := DefaultPaths()
	if err != nil {
		t.Fatal(err)
	}
	if paths.Dir != dir {
		t.Fatalf("got %s", paths.Dir)
	}
	if filepath.Base(paths.ConfigFile()) != "config.yaml" {
		t.Fatalf("config file %s", paths.ConfigFile())
	}
}
