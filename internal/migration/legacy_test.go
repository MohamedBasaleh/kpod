package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

func TestParseLegacyGroupsByContext(t *testing.T) {
	text := "dev|Nawaqis-Dev|dev-core\nstaging|Nawaqis-Dev|staging-core\n"
	cfg, err := ParseLegacy(text)
	if err != nil {
		t.Fatal(err)
	}
	envs := cfg.Contexts["Nawaqis-Dev"].Environments
	if envs["dev"].Namespace != "dev-core" {
		t.Fatalf("%+v", envs)
	}
	if envs["staging"].Namespace != "staging-core" {
		t.Fatalf("%+v", envs)
	}
}

func TestParseLegacyMultipleContexts(t *testing.T) {
	text := "dev|Nawaqis-Dev|dev-core\nprod|Nawaqis-Prod|prod-core\n"
	cfg, err := ParseLegacy(text)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Contexts["Nawaqis-Dev"].Environments["dev"].Namespace != "dev-core" {
		t.Fatal("dev")
	}
	if cfg.Contexts["Nawaqis-Prod"].Environments["prod"].Namespace != "prod-core" {
		t.Fatal("prod")
	}
}

func TestParseLegacyIgnoresBlankAndComments(t *testing.T) {
	text := "# header\n\ndev|Nawaqis-Dev|dev-core\n\n# footer\n"
	cfg, err := ParseLegacy(text)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Contexts["Nawaqis-Dev"].Environments["dev"].Namespace != "dev-core" {
		t.Fatalf("%+v", cfg)
	}
}

func TestParseLegacyTrailingNewline(t *testing.T) {
	text := "dev|Nawaqis-Dev|dev-core\n"
	if _, err := ParseLegacy(text); err != nil {
		t.Fatal(err)
	}
}

func TestParseLegacyMalformedRow(t *testing.T) {
	_, err := ParseLegacy("dev|Nawaqis-Dev|dev-core\nnot-a-mapping\n")
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "expected alias|context|namespace") {
		t.Fatalf("got %v", err)
	}
}

func TestParseLegacyEmptyFields(t *testing.T) {
	_, err := ParseLegacy("dev||dev-core\n")
	if err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("got %v", err)
	}
}

func TestParseLegacyDuplicateAlias(t *testing.T) {
	_, err := ParseLegacy("dev|ctx-a|ns-a\ndev|ctx-b|ns-b\n")
	if err == nil || !strings.Contains(err.Error(), `duplicate environment alias "dev"`) {
		t.Fatalf("got %v", err)
	}
}

func TestMigrateWritesConfigAndBackup(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	legacy := "# maps\ndev|Nawaqis-Dev|dev-core\nstaging|Nawaqis-Dev|staging-core\n"
	if err := os.WriteFile(paths.LegacyContextsFile(), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Migrate(paths)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Migrated {
		t.Fatalf("skipped: %s", result.Skipped)
	}
	if _, err := os.Stat(paths.LegacyBackupFile()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.LegacyContextsFile()); err != nil {
		t.Fatal("original file was removed")
	}
	original, err := os.ReadFile(paths.LegacyContextsFile())
	if err != nil {
		t.Fatal(err)
	}
	if string(original) != legacy {
		t.Fatal("original file content changed")
	}
	got, err := config.NewRepository(paths).Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Contexts["Nawaqis-Dev"].Environments["dev"].Namespace != "dev-core" {
		t.Fatalf("%+v", got)
	}
	if got.Contexts["Nawaqis-Dev"].Environments["staging"].Namespace != "staging-core" {
		t.Fatalf("%+v", got)
	}
}

func TestMigrateDoesNotOverwriteExistingV2(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	existing := config.Default()
	existing.Contexts["Keep"] = config.Context{
		Environments: map[string]config.Environment{"prod": {Namespace: "prod-core"}},
	}
	if err := config.NewRepository(paths).Save(existing); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.LegacyContextsFile(), []byte("dev|Nawaqis-Dev|dev-core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Migrate(paths)
	if err != nil {
		t.Fatal(err)
	}
	if result.Migrated || result.Skipped == "" {
		t.Fatalf("%+v", result)
	}
	got, err := config.NewRepository(paths).Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Contexts["Keep"]; !ok {
		t.Fatal("v2 config was overwritten")
	}
	if _, ok := got.Contexts["Nawaqis-Dev"]; ok {
		t.Fatal("legacy data leaked into existing v2 config")
	}
}

func TestMigrateMissingLegacy(t *testing.T) {
	result, err := Migrate(config.NewPaths(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if result.Migrated {
		t.Fatal("expected skip")
	}
}

func TestMigrateDoesNotDeleteOriginalOnParseError(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	if err := os.WriteFile(paths.LegacyContextsFile(), []byte("bad-line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Migrate(paths)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := os.Stat(paths.LegacyContextsFile()); err != nil {
		t.Fatal("original deleted")
	}
	if _, err := os.Stat(paths.ConfigFile()); !os.IsNotExist(err) {
		t.Fatal("config.yaml should not exist after failed migration")
	}
}

func TestMigrateARNContextName(t *testing.T) {
	arn := "arn:aws:eks:eu-central-1:851725235050:cluster/Nawaqis-dev"
	cfg, err := ParseLegacy("old|" + arn + "|staging-core\n")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Contexts[arn].Environments["old"].Namespace != "staging-core" {
		t.Fatalf("%+v", cfg.Contexts)
	}
}

func TestBackupNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	if err := os.WriteFile(paths.LegacyContextsFile(), []byte("dev|Nawaqis-Dev|dev-core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(paths); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.LegacyBackupFile(), []byte("first-backup\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Remove v2 so a second migrate would run, but keep bak.
	if err := os.Remove(paths.ConfigFile()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.LegacyContextsFile(), []byte("staging|Nawaqis-Dev|staging-core\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(paths); err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(paths.LegacyBackupFile())
	if err != nil {
		t.Fatal(err)
	}
	if string(bak) != "first-backup\n" {
		t.Fatalf("backup overwritten: %q", bak)
	}
}

func TestMigrateRealisticMultiContext(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	legacy := "dev|Context-A|dev-core\nstaging|Context-A|staging-core\nproduction|Context-B|prod-core\n"
	if err := os.WriteFile(paths.LegacyContextsFile(), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Migrate(paths)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Migrated {
		t.Fatalf("%+v", result)
	}
	got, err := config.NewRepository(paths).Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Contexts["Context-A"].Environments["dev"].Namespace != "dev-core" {
		t.Fatal("dev")
	}
	if got.Contexts["Context-A"].Environments["staging"].Namespace != "staging-core" {
		t.Fatal("staging")
	}
	if got.Contexts["Context-B"].Environments["production"].Namespace != "prod-core" {
		t.Fatal("production")
	}
	if _, err := os.Stat(paths.LegacyBackupFile()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.LegacyContextsFile()); err != nil {
		t.Fatal("original removed")
	}
	if got.Settings.Notifications != true || got.Settings.ShellShortcuts {
		t.Fatalf("unexpected settings %+v", got.Settings)
	}
	if len(got.Shortcuts) != 0 {
		t.Fatal("shortcuts should be empty after migration")
	}
}

func TestHasLegacyFile(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	if HasLegacyFile(paths) {
		t.Fatal("expected false")
	}
	if err := os.WriteFile(filepath.Join(dir, "contexts"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if !HasLegacyFile(paths) {
		t.Fatal("expected true")
	}
}
