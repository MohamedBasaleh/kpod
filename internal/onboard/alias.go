package onboard

import (
	"fmt"
	"strings"
	"unicode"
)

// SuggestAlias returns an editable default environment name for a namespace.
// The namespace itself is the default so the mapping stays transparent.
// A shorter form is used only when it is an obvious, lossless prefix of a
// hyphenated name (for example dev-core → dev) and does not collide.
func SuggestAlias(namespace string, taken map[string]struct{}) string {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return ""
	}
	for _, candidate := range aliasCandidates(ns) {
		if available(candidate, taken) {
			return candidate
		}
	}
	return uniquify(ns, taken)
}

func aliasCandidates(ns string) []string {
	var out []string
	seen := map[string]struct{}{}
	add := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(hyphenPrefix(ns))
	add(hyphenSuffix(ns))
	add(ns)
	return out
}

func hyphenPrefix(namespace string) string {
	i := strings.IndexByte(namespace, '-')
	if i <= 0 {
		return ""
	}
	prefix := namespace[:i]
	if !isSimpleAlias(prefix) {
		return ""
	}
	return prefix
}

func hyphenSuffix(namespace string) string {
	i := strings.LastIndexByte(namespace, '-')
	if i <= 0 || i == len(namespace)-1 {
		return ""
	}
	suffix := namespace[i+1:]
	if !isSimpleAlias(suffix) {
		return ""
	}
	return suffix
}

func isSimpleAlias(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func available(alias string, taken map[string]struct{}) bool {
	_, ok := taken[alias]
	return !ok
}

func uniquify(base string, taken map[string]struct{}) string {
	if available(base, taken) {
		return base
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if available(candidate, taken) {
			return candidate
		}
	}
	return base
}

// ValidateAlias checks a candidate environment name against global uniqueness.
func ValidateAlias(alias string, taken map[string]struct{}) error {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return fmt.Errorf("environment name must not be empty")
	}
	if strings.ContainsAny(alias, " \t") {
		return fmt.Errorf("environment name must not contain spaces")
	}
	if strings.ContainsAny(alias, ";&|`$()<>\\\"'") {
		return fmt.Errorf("environment name contains unsupported characters")
	}
	if _, ok := taken[alias]; ok {
		return fmt.Errorf("environment name %q is already used", alias)
	}
	return nil
}
