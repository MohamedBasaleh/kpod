package onboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

func TestDetectConfigured(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	cfg := config.Default()
	cfg.Contexts["Nawaqis-Dev"] = config.Context{
		Environments: map[string]config.Environment{
			"dev": {Namespace: "dev-core"},
		},
	}
	if err := config.NewRepository(paths).Save(cfg); err != nil {
		t.Fatal(err)
	}
	if Detect(paths) != ModeConfigured {
		t.Fatal("expected configured")
	}
}

func TestDetectMigrate(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	if err := os.WriteFile(filepath.Join(dir, "contexts"), []byte("dev|Nawaqis-Dev|dev-core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if Detect(paths) != ModeMigrate {
		t.Fatal("expected migrate")
	}
}

func TestDetectFresh(t *testing.T) {
	if Detect(config.NewPaths(t.TempDir())) != ModeFresh {
		t.Fatal("expected fresh")
	}
}

func TestDetectConfiguredWinsOverLegacy(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	if err := os.WriteFile(filepath.Join(dir, "contexts"), []byte("dev|Nawaqis-Dev|dev-core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Contexts["Nawaqis-Dev"] = config.Context{
		Environments: map[string]config.Environment{
			"staging": {Namespace: "staging-core"},
		},
	}
	if err := config.NewRepository(paths).Save(cfg); err != nil {
		t.Fatal(err)
	}
	if Detect(paths) != ModeConfigured {
		t.Fatal("v2 must win")
	}
}
