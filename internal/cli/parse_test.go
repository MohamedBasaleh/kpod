package cli

import (
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/action"
)

func TestParseActions(t *testing.T) {
	inv, err := Parse([]string{"dev", "console"})
	if err != nil || inv.Spec.ID != action.Console || inv.Env != "dev" {
		t.Fatalf("%+v %v", inv, err)
	}
	inv, err = Parse([]string{"dev", "c"})
	if err != nil || inv.Spec.ID != action.Console {
		t.Fatalf("%+v %v", inv, err)
	}
	inv, err = Parse([]string{"dev", "logs", "api"})
	if err != nil || inv.Spec.ID != action.Logs || inv.PodQuery != "api" {
		t.Fatalf("%+v %v", inv, err)
	}
	inv, err = Parse([]string{"staging", "bash"})
	if err != nil || inv.Spec.ID != action.Shell {
		t.Fatalf("%+v %v", inv, err)
	}
	inv, err = Parse([]string{"dev", "sh"})
	if err != nil || inv.ShellPref != "sh" {
		t.Fatalf("%+v %v", inv, err)
	}
}

func TestParseUnknownAction(t *testing.T) {
	_, err := Parse([]string{"dev", "explode"})
	if err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Fatalf("%v", err)
	}
}

func TestParseCopyAndBrowse(t *testing.T) {
	inv, err := Parse([]string{"dev", "download", "/tmp/a", "./b"})
	if err != nil || inv.PodQuery != "" || len(inv.Extra) != 2 {
		t.Fatalf("%+v %v", inv, err)
	}
	inv, err = Parse([]string{"dev", "download", "api", "/tmp/a", "./b"})
	if err != nil || inv.PodQuery != "api" {
		t.Fatalf("%+v %v", inv, err)
	}
	inv, err = Parse([]string{"dev"})
	if err != nil || !inv.BrowseOnly {
		t.Fatalf("%+v %v", inv, err)
	}
	inv, err = Parse([]string{"dev", "pods"})
	if err != nil || !inv.OpenPods {
		t.Fatalf("%+v %v", inv, err)
	}
}
