package action

import "testing"

func TestResolveCanonicalAndAliases(t *testing.T) {
	cases := map[string]ID{
		"console": "console", "c": Console, "rails": Console,
		"dbconsole": DBConsole, "runner": Runner,
		"shell": Shell, "bash": Shell, "sh": Shell, "exec": Shell,
		"logs": Logs, "log": Logs, "l": Logs,
		"describe": Describe, "yaml": YAML, "events": Events, "metrics": Metrics, "top": Metrics,
		"rollout": Rollout, "restart": Restart, "delete": Delete,
		"download": Download, "dl": Download, "get": Download,
		"upload": Upload, "ul": Upload, "put": Upload,
		"port-forward": PortForward, "pf": PortForward,
	}
	for name, id := range cases {
		spec, err := Resolve(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if spec.ID != id {
			t.Fatalf("%s: got %s want %s", name, spec.ID, id)
		}
	}
}

func TestResolveUnknown(t *testing.T) {
	if _, err := Resolve("explode"); err == nil {
		t.Fatal("expected error")
	}
}
