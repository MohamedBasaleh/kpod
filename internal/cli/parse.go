package cli

import (
	"fmt"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/action"
)

type Invocation struct {
	Env        string
	Action     string
	Spec       action.Spec
	PodQuery   string
	Extra      []string
	ShellPref  string
	OpenPods   bool
	BrowseOnly bool
}

func Parse(args []string) (Invocation, error) {
	var inv Invocation
	if len(args) == 0 {
		inv.BrowseOnly = true
		return inv, nil
	}
	inv.Env = args[0]
	if len(args) == 1 {
		inv.BrowseOnly = true
		return inv, nil
	}
	if args[1] == "pods" {
		inv.OpenPods = true
		inv.BrowseOnly = true
		return inv, nil
	}
	spec, err := action.Resolve(args[1])
	if err != nil {
		return inv, fmt.Errorf("unknown action %q\n\nTry: console, shell, logs, describe, yaml, events, metrics, rollout, restart, delete, download, upload, port-forward, dbconsole, runner", args[1])
	}
	inv.Action = args[1]
	inv.Spec = spec
	if strings.EqualFold(args[1], "sh") {
		inv.ShellPref = "sh"
	}
	rest := args[2:]
	if spec.ID == action.Download || spec.ID == action.Upload {
		inv.PodQuery, inv.Extra = splitCopyArgs(rest)
		return inv, nil
	}
	if len(rest) == 0 {
		return inv, nil
	}
	if spec.ID == action.PortForward && looksPort(rest[0]) {
		inv.Extra = rest
		return inv, nil
	}
	if spec.ID == action.Runner {
		inv.PodQuery = rest[0]
		if action.LooksLikePath(rest[0]) || looksPort(rest[0]) {
			inv.PodQuery = ""
			inv.Extra = rest
		} else if len(rest) > 1 {
			inv.Extra = rest[1:]
		} else {
			// single token may be code or pod filter; treat as pod if it has no spaces and looks like a name
			inv.PodQuery = rest[0]
			inv.Extra = nil
		}
		if len(rest) == 1 && (strings.Contains(rest[0], " ") || strings.Contains(rest[0], "(") || strings.Contains(rest[0], ".")) {
			inv.PodQuery = ""
			inv.Extra = rest
		}
		return inv, nil
	}
	inv.PodQuery = rest[0]
	if len(rest) > 1 {
		inv.Extra = rest[1:]
	}
	return inv, nil
}

func splitCopyArgs(rest []string) (string, []string) {
	if len(rest) == 0 {
		return "", nil
	}
	if len(rest) >= 2 && action.LooksLikePath(rest[0]) {
		return "", rest
	}
	if len(rest) >= 3 {
		return rest[0], rest[1:]
	}
	if len(rest) == 2 && !action.LooksLikePath(rest[0]) {
		return rest[0], rest[1:]
	}
	return "", rest
}

func looksPort(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
