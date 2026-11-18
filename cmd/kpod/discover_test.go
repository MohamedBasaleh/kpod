package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDiscoverCfg(t *testing.T, dir string) {
	t.Helper()
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
}

func TestDiscoverIgnoreContextAndNamespace(t *testing.T) {
	dir := t.TempDir()
	writeDiscoverCfg(t, dir)

	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config-dir", dir, "discover", "ignore-context", "analytics"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Ignored context analytics") {
		t.Fatalf("%s", buf.String())
	}

	buf.Reset()
	cmd = newRootCommand()
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config-dir", dir, "discover", "ignore-namespace", "Nawaqis-Dev", "kube-system"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "analytics") || !strings.Contains(text, "kube-system") {
		t.Fatalf("%s", text)
	}
	if !strings.Contains(text, "dev-core") || !strings.Contains(text, "notifications: true") {
		t.Fatal("unrelated config lost")
	}
}

func TestDiscoverHelp(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"discover", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "newly visible") {
		t.Fatalf("%s", buf.String())
	}
}
