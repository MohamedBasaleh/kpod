package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeShortcutCfg(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	body := "version: 2\ncontexts:\n  Nawaqis-Dev:\n    environments:\n      dev:\n        namespace: dev-core\n      staging:\n        namespace: staging-core\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestShortcutAddList(t *testing.T) {
	dir := writeShortcutCfg(t)
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config-dir", dir, "shortcut", "add", "dc", "dev", "c"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	cmd = newRootCommand()
	buf = &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config-dir", dir, "shortcut", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "k-dc") || !strings.Contains(out, "kpod dev console") {
		t.Fatalf("%s", out)
	}
}

func TestShortcutInstallYesTempHome(t *testing.T) {
	dir := writeShortcutCfg(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	cmd := newRootCommand()
	cmd.SetArgs([]string{"--config-dir", dir, "shortcut", "add", "sb", "staging", "bash"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	cmd = newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config-dir", dir, "shortcut", "install", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "aliases.zsh") {
		t.Fatalf("%s", body)
	}
	cmd = newRootCommand()
	cmd.SetArgs([]string{"--config-dir", dir, "shortcut", "install", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	body, _ = os.ReadFile(filepath.Join(home, ".zshrc"))
	if strings.Count(string(body), "source ") != 1 {
		t.Fatalf("not idempotent: %s", body)
	}
}
