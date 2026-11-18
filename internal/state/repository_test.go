package state

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

func TestSaveLoad(t *testing.T) {
	repo := NewRepository(config.NewPaths(t.TempDir()))
	s := Default()
	s.LastEnvironment = "dev"
	s.SeenContexts = []string{"Nawaqis-Dev"}
	s.SeenNamespaces = map[string][]string{
		"Nawaqis-Dev": {"dev-core", "staging-core"},
	}
	if err := repo.Save(s); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != Version {
		t.Fatalf("version %d", got.Version)
	}
	if got.LastEnvironment != "dev" {
		t.Fatalf("last %q", got.LastEnvironment)
	}
	if len(got.SeenContexts) != 1 || got.SeenContexts[0] != "Nawaqis-Dev" {
		t.Fatalf("seen contexts %+v", got.SeenContexts)
	}
	if got.SeenNamespaces["Nawaqis-Dev"][1] != "staging-core" {
		t.Fatalf("seen ns %+v", got.SeenNamespaces)
	}
}

func TestLoadOlderStateWithoutDiscoveryReady(t *testing.T) {
	dir := t.TempDir()
	body := []byte("version: 1\nlast_environment: dev\nseen_contexts:\n  - my-company-dev\nseen_namespaces:\n  my-company-dev:\n    - dev-core\n")
	if err := os.WriteFile(filepath.Join(dir, "state.yaml"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := NewRepository(config.NewPaths(dir)).Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.DiscoveryReady {
		t.Fatal("missing discovery_ready must stay false")
	}
	if got.LastEnvironment != "dev" || got.SeenContexts[0] != "my-company-dev" {
		t.Fatalf("%+v", got)
	}
}

func TestLoadOrDefaultMissing(t *testing.T) {
	repo := NewRepository(config.NewPaths(t.TempDir()))
	got, err := repo.LoadOrDefault()
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != Version {
		t.Fatalf("version %d", got.Version)
	}
	if got.SeenContexts == nil || got.SeenNamespaces == nil {
		t.Fatal("expected initialized empty slices/maps")
	}
}

func TestLoadMissingIsErrNotExist(t *testing.T) {
	_, err := NewRepository(config.NewPaths(t.TempDir())).Load()
	if !errors.Is(err, ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}

func TestMutatePreservesFields(t *testing.T) {
	repo := NewRepository(config.NewPaths(t.TempDir()))
	s := Default()
	s.LastEnvironment = "dev"
	s.SeenContexts = []string{"Nawaqis-Dev"}
	if err := repo.Save(s); err != nil {
		t.Fatal(err)
	}
	if err := repo.Mutate(func(cur *State) error {
		MarkContextSeen(cur, "other")
		cur.DiscoveryReady = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.LastEnvironment != "dev" {
		t.Fatalf("last env lost: %q", got.LastEnvironment)
	}
	if !Contains(got.SeenContexts, "Nawaqis-Dev") || !Contains(got.SeenContexts, "other") {
		t.Fatalf("%v", got.SeenContexts)
	}
	if !got.DiscoveryReady {
		t.Fatal("expected discovery ready")
	}
}

func TestUnsupportedVersion(t *testing.T) {
	s := Default()
	s.Version = 9
	err := NewRepository(config.NewPaths(t.TempDir())).Save(s)
	if err == nil || !strings.Contains(err.Error(), "unsupported state version") {
		t.Fatalf("got %v", err)
	}
}
