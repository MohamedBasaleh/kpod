package onboard

import (
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

func TestDraftToConfigGroupsByContext(t *testing.T) {
	d := NewDraft()
	d.SelectContext("Nawaqis-Dev")
	d.AddNamespace("Nawaqis-Dev", "dev-core")
	d.AddNamespace("Nawaqis-Dev", "staging-core")
	if err := d.SetMapping("Nawaqis-Dev", "dev-core", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := d.SetMapping("Nawaqis-Dev", "staging-core", "staging"); err != nil {
		t.Fatal(err)
	}
	cfg, err := d.ToConfig()
	if err != nil {
		t.Fatal(err)
	}
	envs := cfg.Contexts["Nawaqis-Dev"].Environments
	if envs["dev"].Namespace != "dev-core" || envs["staging"].Namespace != "staging-core" {
		t.Fatalf("%+v", envs)
	}
}

func TestDraftDuplicateAliasRejected(t *testing.T) {
	d := NewDraft()
	if err := d.SetMapping("Nawaqis-Dev", "dev-core", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := d.SetMapping("Nawaqis-Prod", "prod-core", "dev"); err == nil {
		t.Fatal("expected duplicate alias error")
	}
}

func TestDraftToggleContextClearsNamespaces(t *testing.T) {
	d := NewDraft()
	d.ToggleContext("Nawaqis-Dev")
	d.AddNamespace("Nawaqis-Dev", "dev-core")
	_ = d.SetMapping("Nawaqis-Dev", "dev-core", "dev")
	d.ToggleContext("Nawaqis-Dev")
	if d.ContextSelected("Nawaqis-Dev") {
		t.Fatal("should be unselected")
	}
	if d.HasAnyMapping() {
		t.Fatal("mappings should be cleared")
	}
}

func TestDraftPendingAliasTargets(t *testing.T) {
	d := NewDraft()
	d.SelectContext("Nawaqis-Dev")
	d.AddNamespace("Nawaqis-Dev", "dev-core")
	d.AddNamespace("Nawaqis-Dev", "staging-core")
	_ = d.SetMapping("Nawaqis-Dev", "dev-core", "dev")
	pending := d.PendingAliasTargets()
	if len(pending) != 1 || pending[0].Namespace != "staging-core" {
		t.Fatalf("%+v", pending)
	}
}

func TestDraftToConfigRequiresMapping(t *testing.T) {
	d := NewDraft()
	if _, err := d.ToConfig(); err == nil {
		t.Fatal("expected error")
	}
}

func TestDraftValidateUsesConfigRules(t *testing.T) {
	d := NewDraft()
	_ = d.SetMapping("Nawaqis-Dev", "dev-core", "dev")
	cfg, err := d.ToConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := config.Validate(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestReviewLines(t *testing.T) {
	d := NewDraft()
	_ = d.SetMapping("Nawaqis-Dev", "dev-core", "dev")
	_ = d.SetMapping("Nawaqis-Dev", "staging-core", "staging")
	lines := d.ReviewLines()
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "dev") || !strings.Contains(joined, "dev-core") {
		t.Fatalf("%q", joined)
	}
}
