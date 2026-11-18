package shortcut

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/config"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func ValidateID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("shortcut name is required")
	}
	if strings.HasPrefix(id, "k-") {
		return fmt.Errorf("store the shortcut without the k- prefix (got %q)", id)
	}
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid shortcut %q (use letters, digits, _ or -)", id)
	}
	return nil
}

func CommandName(id string) string {
	return "k-" + id
}

func CanonicalAction(name string) (action.ID, error) {
	spec, err := action.Resolve(name)
	if err != nil {
		return "", fmt.Errorf("unknown action %q", name)
	}
	return spec.ID, nil
}

func Prepare(id string, sc config.Shortcut) (string, config.Shortcut, error) {
	id = strings.TrimSpace(id)
	if err := ValidateID(id); err != nil {
		return "", sc, err
	}
	if strings.TrimSpace(sc.Environment) == "" {
		return "", sc, fmt.Errorf("environment is required")
	}
	act, err := CanonicalAction(sc.Action)
	if err != nil {
		return "", sc, err
	}
	sc.Action = string(act)
	return id, sc, nil
}

func CheckRefs(cfg config.Config, id string, sc config.Shortcut) error {
	if _, ok := cfg.LookupEnvironment(sc.Environment); !ok {
		return fmt.Errorf("k-%s references missing environment %q", id, sc.Environment)
	}
	if _, err := CanonicalAction(sc.Action); err != nil {
		return fmt.Errorf("k-%s references unknown action %q", id, sc.Action)
	}
	return nil
}

func SortedIDs(cfg config.Config) []string {
	ids := make([]string, 0, len(cfg.Shortcuts))
	for id := range cfg.Shortcuts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func Invocation(sc config.Shortcut) string {
	return "kpod " + sc.Environment + " " + sc.Action
}
