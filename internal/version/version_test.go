package version

import "strings"

import "testing"

func TestDisplayDefault(t *testing.T) {
	if Display() != "2.0.0-dev" {
		t.Fatalf("%s", Display())
	}
	if Line() != "kpod 2.0.0-dev" {
		t.Fatalf("%s", Line())
	}
}

func TestDisplayInjected(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })
	Version = "v2.0.0"
	if Display() != "2.0.0" {
		t.Fatalf("%s", Display())
	}
	if !strings.Contains(Verbose(), "commit") {
		t.Fatalf("%s", Verbose())
	}
}
