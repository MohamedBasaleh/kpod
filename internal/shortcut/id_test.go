package shortcut

import (
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/config"
)

func testCfg() config.Config {
	cfg := config.Default()
	cfg.Contexts["Nawaqis-Dev"] = config.Context{
		Environments: map[string]config.Environment{
			"dev":     {Namespace: "dev-core"},
			"staging": {Namespace: "staging-core"},
		},
	}
	return cfg
}

func TestPrepareValidCanonical(t *testing.T) {
	id, sc, err := Prepare("dc", config.Shortcut{Environment: "dev", Action: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "dc" || sc.Action != string(action.Console) || sc.Environment != "dev" {
		t.Fatalf("%s %+v", id, sc)
	}
}

func TestPrepareUnknownEnvOnCheck(t *testing.T) {
	cfg := testCfg()
	_, sc, err := Prepare("xx", config.Shortcut{Environment: "nope", Action: "console"})
	if err != nil {
		t.Fatal(err)
	}
	err = CheckRefs(cfg, "xx", sc)
	if err == nil || !strings.Contains(err.Error(), `missing environment "nope"`) {
		t.Fatalf("%v", err)
	}
}

func TestPrepareUnknownAction(t *testing.T) {
	_, _, err := Prepare("xx", config.Shortcut{Environment: "dev", Action: "explode"})
	if err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Fatalf("%v", err)
	}
}

func TestPrepareInvalidID(t *testing.T) {
	cases := []string{"", "k-dc", "d c", "d;c", "d$c"}
	for _, id := range cases {
		if _, _, err := Prepare(id, config.Shortcut{Environment: "dev", Action: "console"}); err == nil {
			t.Fatalf("expected invalid %q", id)
		}
	}
}

func TestDuplicateShortcut(t *testing.T) {
	cfg := testCfg()
	s := Service{Exists: func(string) bool { return false }}
	cfg, err := s.Add(cfg, "dc", "dev", "console")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Add(cfg, "dc", "staging", "console")
	if err == nil || !strings.Contains(err.Error(), "already a kpod shortcut") {
		t.Fatalf("%v", err)
	}
}
