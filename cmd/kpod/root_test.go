package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "kpod 2.0.0-dev") {
		t.Fatalf("got %q", out)
	}
}

func TestVersionCommandVerbose(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version", "--verbose"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "kpod 2.0.0-dev") || !strings.Contains(out, "commit") {
		t.Fatalf("got %q", out)
	}
}

func TestHelpFlag(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Fast, interactive Kubernetes access") && !strings.Contains(out, "Kubernetes developer CLI") {
		t.Fatalf("got %q", out)
	}
	if strings.Contains(out, "Phase ") {
		t.Fatalf("help must not mention phases: %q", out)
	}
	if strings.Contains(out, "  debug ") {
		t.Fatalf("debug command should be hidden: %q", out)
	}
	if !strings.Contains(out, "kpod dev console") || !strings.Contains(out, "kpod staging logs api") {
		t.Fatalf("help should mention direct actions: %q", out)
	}
}

func TestUnknownEnvironment(t *testing.T) {
	dir := t.TempDir()
	cfg := "version: 2\ncontexts:\n  Nawaqis-Dev:\n    environments:\n      dev:\n        namespace: dev-core\n      staging:\n        namespace: staging-core\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--config-dir", dir, "deev"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, `Environment "deev" was not found`) {
		t.Fatalf("%q", msg)
	}
	if !strings.Contains(msg, "dev") || !strings.Contains(msg, "staging") {
		t.Fatalf("%q", msg)
	}
}

func TestUnknownAction(t *testing.T) {
	dir := t.TempDir()
	cfg := "version: 2\ncontexts:\n  Nawaqis-Dev:\n    environments:\n      dev:\n        namespace: dev-core\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand()
	cmd.SetArgs([]string{"--config-dir", dir, "dev", "not-an-action"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Fatalf("got %v", err)
	}
}

func TestFreshUserRequiresTTY(t *testing.T) {
	dir := t.TempDir()
	cmd := newRootCommand()
	cmd.SetArgs([]string{"--config-dir", dir})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("got %v", err)
	}
}

func TestMalformedConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("version: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand()
	cmd.SetArgs([]string{"--config-dir", dir})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "malformed YAML") && !strings.Contains(err.Error(), "could not load") {
		t.Fatalf("got %v", err)
	}
}

func TestConfiguredRequiresTTY(t *testing.T) {
	dir := t.TempDir()
	cfg := "version: 2\ncontexts:\n  Nawaqis-Dev:\n    environments:\n      dev:\n        namespace: dev-core\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand()
	cmd.SetArgs([]string{"--config-dir", dir})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("got %v", err)
	}
}

func TestMigrateCommand(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "contexts")
	if err := os.WriteFile(legacy, []byte("dev|Nawaqis-Dev|dev-core\nstaging|Nawaqis-Dev|staging-core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config-dir", dir, "migrate"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "migrated") {
		t.Fatalf("got %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "contexts.v1.bak")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatal("legacy file deleted")
	}
}
