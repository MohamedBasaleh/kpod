package resolve

import (
	"errors"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

func testConfig() config.Config {
	cfg := config.Default()
	cfg.Contexts["Nawaqis-Dev"] = config.Context{
		Environments: map[string]config.Environment{
			"dev":     {Namespace: "dev-core"},
			"staging": {Namespace: "staging-core"},
		},
	}
	return cfg
}

func TestLookupSuccess(t *testing.T) {
	env, err := Lookup(testConfig(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	if env.KubeContext != "Nawaqis-Dev" || env.Namespace != "dev-core" {
		t.Fatalf("%+v", env)
	}
}

func TestLookupUnknown(t *testing.T) {
	_, err := Lookup(testConfig(), "deev")
	var nf NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, `Environment "deev" was not found`) {
		t.Fatalf("%q", msg)
	}
	if !strings.Contains(msg, "dev") || !strings.Contains(msg, "staging") {
		t.Fatalf("%q", msg)
	}
	if !strings.Contains(msg, `Did you mean "dev"`) {
		t.Fatalf("expected suggestion: %q", msg)
	}
}
