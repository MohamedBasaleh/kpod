package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionZshAndBash(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"completion", "zsh"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	zsh := buf.String()
	if !strings.Contains(zsh, "compdef") && !strings.Contains(zsh, "_kpod") {
		t.Fatalf("zsh completion: %s", zsh[:min(len(zsh), 200)])
	}

	buf.Reset()
	cmd = newRootCommand()
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"completion", "bash"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	bash := buf.String()
	if !strings.Contains(bash, "_kpod") && !strings.Contains(bash, "complete") {
		t.Fatalf("bash completion: %s", bash[:min(len(bash), 200)])
	}
}

func TestCompleteEnvironmentsFromConfig(t *testing.T) {
	dir := t.TempDir()
	body := []byte("version: 2\ncontexts:\n  my-company-dev:\n    environments:\n      dev:\n        namespace: dev-core\n      staging:\n        namespace: staging-core\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	got, _ := completeRoot(dir, nil, "st")
	if len(got) != 1 || got[0] != "staging" {
		t.Fatalf("%v", got)
	}
	acts, _ := completeRoot(dir, []string{"dev"}, "con")
	if len(acts) == 0 {
		t.Fatal("expected action completions")
	}
}

func TestHelpListsCompletion(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "completion") {
		t.Fatalf("%s", buf.String())
	}
}
