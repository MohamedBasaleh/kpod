package shortcut

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

func TestInsertRCIdempotent(t *testing.T) {
	alias := "/tmp/kpod-test/shell/aliases.zsh"
	first, changed := InsertManagedBlock("export PATH=/usr/bin\n", alias)
	if !changed || !strings.Contains(first, managedComment) || !strings.Contains(first, "export PATH=/usr/bin") {
		t.Fatalf("%q", first)
	}
	second, changed := InsertManagedBlock(first, alias)
	if changed || second != first {
		t.Fatal("repeated insert must be idempotent")
	}
	count := strings.Count(second, "source ")
	if count != 1 {
		t.Fatalf("source lines %d", count)
	}
}

func TestRemoveRCPreservesUnrelated(t *testing.T) {
	alias := "/tmp/kpod-test/shell/aliases.zsh"
	orig := "export EDITOR=vim\n\n# kpod shortcuts\nsource \"/tmp/kpod-test/shell/aliases.zsh\"\n\nalias ll=ls\n"
	next, changed := RemoveManagedBlock(orig, alias)
	if !changed {
		t.Fatal("expected change")
	}
	if !strings.Contains(next, "export EDITOR=vim") || !strings.Contains(next, "alias ll=ls") {
		t.Fatalf("%q", next)
	}
	if strings.Contains(next, "kpod shortcuts") || strings.Contains(next, "aliases.zsh") {
		t.Fatalf("%q", next)
	}
}

func TestInstallRequiresConfirm(t *testing.T) {
	s := Service{Paths: config.NewPaths(t.TempDir()), Home: t.TempDir(), Shell: ShellZsh}
	if err := s.InstallRC(false); err == nil {
		t.Fatal("must not write without confirmation")
	}
	if _, err := os.Stat(filepath.Join(s.Home, ".zshrc")); !os.IsNotExist(err) {
		t.Fatal("rc must not exist")
	}
}

func TestApplyRCInstallAndUninstall(t *testing.T) {
	home := t.TempDir()
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("echo hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "shell", "aliases.zsh")
	if err := ApplyRC(rc, alias, true); err != nil {
		t.Fatal(err)
	}
	if err := ApplyRC(rc, alias, true); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(rc)
	if strings.Count(string(body), "source ") != 1 {
		t.Fatalf("%s", body)
	}
	if !strings.Contains(string(body), "echo hello") {
		t.Fatal("unrelated lost")
	}
	if err := ApplyRC(rc, alias, false); err != nil {
		t.Fatal(err)
	}
	body, _ = os.ReadFile(rc)
	if strings.Contains(string(body), "kpod") {
		t.Fatalf("%s", body)
	}
	if !strings.Contains(string(body), "echo hello") {
		t.Fatal("unrelated lost on remove")
	}
}

func TestCommandCollision(t *testing.T) {
	cfg := testCfg()
	err := Collision(cfg, "sc", func(name string) bool { return name == "k-sc" })
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("%v", err)
	}
}

func TestStatusReportsMissingEnv(t *testing.T) {
	cfg := testCfg()
	cfg.Shortcuts["sc"] = config.Shortcut{Environment: "gone", Action: "console"}
	s := Service{Paths: config.NewPaths(t.TempDir()), Home: t.TempDir(), Shell: ShellZsh}
	st := s.Status(cfg)
	if len(st.Problems) != 1 || !strings.Contains(st.Problems[0], "missing environment") {
		t.Fatalf("%v", st.Problems)
	}
}
