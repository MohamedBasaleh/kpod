package onboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

func TestLoadLegacyPreview(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	body := "dev|Nawaqis-Dev|dev-core\nstaging|Nawaqis-Dev|staging-core\n"
	if err := os.WriteFile(filepath.Join(dir, "contexts"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := LoadLegacyPreview(paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Mappings) != 2 {
		t.Fatalf("%+v", preview.Mappings)
	}
	if preview.Mappings[0].Alias != "dev" || preview.Mappings[0].Namespace != "dev-core" {
		t.Fatalf("%+v", preview.Mappings[0])
	}
}
