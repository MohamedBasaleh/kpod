package uninstall

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	"github.com/MohamedBasaleh/kpod/internal/shortcut"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func setupHome(t *testing.T) (home, bin, cfgDir string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KPOD_CONFIG_DIR", filepath.Join(home, ".config", "kpod"))
	t.Setenv("KPOD_INSTALL_DIR", filepath.Join(home, ".local", "bin"))
	bin = filepath.Join(home, ".local", "bin", "kpod")
	cfgDir = filepath.Join(home, ".config", "kpod")
	writeFile(t, bin, "#!/bin/sh\necho kpod\n")
	writeFile(t, filepath.Join(cfgDir, "config.yaml"), "version: 2\ncontexts: {}\n")
	writeFile(t, filepath.Join(cfgDir, "state.yaml"), "version: 1\n")
	writeFile(t, filepath.Join(cfgDir, "shell", "aliases.zsh"), "# kpod\n")
	writeFile(t, filepath.Join(cfgDir, "contexts.v1.bak"), "legacy\n")
	writeFile(t, filepath.Join(home, ".kube", "config"), "apiVersion: v1\nkind: Config\n")
	writeFile(t, filepath.Join(home, ".aws", "credentials"), "aws\n")
	return home, bin, cfgDir
}

func opts(t *testing.T, home, bin, cfgDir string, purge, yes bool, in string) (Options, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	errw := &bytes.Buffer{}
	return Options{
		Home:           home,
		Paths:          config.NewPaths(cfgDir),
		Purge:          purge,
		Yes:            yes,
		ExplicitConfig: true,
		ExecPath:       bin,
		LookPath:       func(string) (string, error) { return bin, nil },
		In:             strings.NewReader(in),
		Out:            out,
		Err:            errw,
	}, out
}

func TestUninstallRemovesBinaryOnly(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	o, _ := opts(t, home, bin, cfgDir, false, true, "")
	res, err := Execute(o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Fatal("binary should be removed")
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "config.yaml")); err != nil {
		t.Fatal("config should be preserved")
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "shell", "aliases.zsh")); err != nil {
		t.Fatal("shortcuts should be preserved")
	}
	if len(res.Removed) != 1 {
		t.Fatalf("removed %v", res.Removed)
	}
}

func TestUninstallDefaultConfirmationIsNo(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	o, out := opts(t, home, bin, cfgDir, false, false, "\n")
	if _, err := Execute(o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatal("binary must remain when confirmation is declined")
	}
	if !strings.Contains(out.String(), "Cancelled.") {
		t.Fatalf("%s", out.String())
	}
}

func TestUninstallYesBypassesConfirmation(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	o, out := opts(t, home, bin, cfgDir, false, true, "")
	if _, err := Execute(o); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Continue?") {
		t.Fatalf("prompt should be skipped: %s", out.String())
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Fatal("expected removal")
	}
}

func TestUninstallNDeclines(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	o, _ := opts(t, home, bin, cfgDir, false, false, "n\n")
	if _, err := Execute(o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatal("must keep binary")
	}
}

func TestPurgeRemovesConfigAndShortcuts(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	o, _ := opts(t, home, bin, cfgDir, true, true, "")
	res, err := Execute(o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Fatal("binary")
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "config.yaml")); !os.IsNotExist(err) {
		t.Fatal("config.yaml")
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "shell", "aliases.zsh")); !os.IsNotExist(err) {
		t.Fatal("shortcuts")
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "contexts.v1.bak")); !os.IsNotExist(err) {
		t.Fatal("migration backup")
	}
	if !res.Fresh {
		t.Fatal("expected first-run state")
	}
	if onboard.Detect(config.NewPaths(cfgDir)) != onboard.ModeFresh {
		t.Fatal("Detect")
	}
}

func TestPurgeNeverTouchesKubeOrAWS(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	kube := filepath.Join(home, ".kube", "config")
	aws := filepath.Join(home, ".aws", "credentials")
	beforeKube, _ := os.ReadFile(kube)
	beforeAWS, _ := os.ReadFile(aws)
	o, out := opts(t, home, bin, cfgDir, true, true, "")
	if _, err := Execute(o); err != nil {
		t.Fatal(err)
	}
	afterKube, err := os.ReadFile(kube)
	if err != nil {
		t.Fatal("kubeconfig must exist")
	}
	afterAWS, err := os.ReadFile(aws)
	if err != nil {
		t.Fatal("aws credentials must exist")
	}
	if string(afterKube) != string(beforeKube) || string(afterAWS) != string(beforeAWS) {
		t.Fatal("protected files changed")
	}
	if !strings.Contains(out.String(), "~/.kube/config") {
		t.Fatalf("plan should mention kubeconfig protection:\n%s", out.String())
	}
}

func TestMissingBinaryIsClean(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	if err := os.Remove(bin); err != nil {
		t.Fatal(err)
	}
	o, out := opts(t, home, bin, cfgDir, false, true, "")
	o.LookPath = func(string) (string, error) { return "", os.ErrNotExist }
	if _, err := Execute(o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "config.yaml")); err != nil {
		t.Fatal("config")
	}
	if !strings.Contains(out.String(), "nothing in known") && !strings.Contains(out.String(), "Nothing to remove") {
		t.Fatalf("%s", out.String())
	}
}

func TestMissingConfigDirOnPurgeIsClean(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	if err := os.RemoveAll(cfgDir); err != nil {
		t.Fatal(err)
	}
	o, _ := opts(t, home, bin, cfgDir, true, true, "")
	res, err := Execute(o)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Fatal("binary")
	}
	if !res.Fresh {
		t.Fatal("fresh")
	}
}

func TestDoesNotDeleteUnknownBinaries(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	decoy := filepath.Join(home, "tools", "kpod")
	writeFile(t, decoy, "decoy\n")
	other := filepath.Join(home, "bin", "kpod-v1")
	writeFile(t, other, "v1\n")
	o, out := opts(t, home, bin, cfgDir, false, true, "")
	o.LookPath = func(string) (string, error) { return decoy, nil }
	if _, err := Execute(o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(decoy); err != nil {
		t.Fatal("decoy must remain")
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatal("kpod-v1 must remain")
	}
	if !strings.Contains(out.String(), "Also found, not removed") {
		t.Fatalf("%s", out.String())
	}
}

func TestPurgeReportsShellRCWithoutEditing(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	rc := filepath.Join(home, ".zshrc")
	alias := filepath.Join(cfgDir, "shell", "aliases.zsh")
	writeFile(t, rc, "export PATH=1\n"+shortcut.ManagedBlock(alias))
	o, out := opts(t, home, bin, cfgDir, true, true, "")
	if _, err := Execute(o); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "# kpod shortcuts") {
		t.Fatal("rc must not be rewritten")
	}
	if !strings.Contains(out.String(), ".zshrc") || !strings.Contains(out.String(), "will not edit") {
		t.Fatalf("%s", out.String())
	}
}

func TestFirstRunStateAfterPurge(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	o, _ := opts(t, home, bin, cfgDir, true, true, "")
	if _, err := Execute(o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "config.yaml")); !os.IsNotExist(err) {
		t.Fatal("config.yaml must be gone")
	}
	if onboard.Detect(config.NewPaths(cfgDir)) != onboard.ModeFresh {
		t.Fatal("expected ModeFresh so kpod opens onboarding")
	}
}

func TestPreviewRefusesHomePurge(t *testing.T) {
	home, _, _ := setupHome(t)
	_, err := Preview(Options{Home: home, Paths: config.NewPaths(home), Purge: true})
	if err == nil {
		t.Fatal("expected refuse")
	}
}

func TestYConfirms(t *testing.T) {
	home, bin, cfgDir := setupHome(t)
	o, _ := opts(t, home, bin, cfgDir, false, false, "y\n")
	if _, err := Execute(o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Fatal("confirmed uninstall should remove binary")
	}
}
