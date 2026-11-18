package resolve

import (
	"fmt"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

// NotFoundError is returned when an environment alias does not exist.
type NotFoundError struct {
	Query     string
	Available []string
	Suggest   string
}

func (e NotFoundError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Environment %q was not found.\n", e.Query)
	if len(e.Available) == 0 {
		b.WriteString("\nNo environments are configured.")
		return b.String()
	}
	b.WriteString("\nAvailable environments:\n\n")
	for _, alias := range e.Available {
		fmt.Fprintf(&b, "  %s\n", alias)
	}
	if e.Suggest != "" {
		fmt.Fprintf(&b, "\nDid you mean %q?\n", e.Suggest)
	}
	return b.String()
}

// Lookup resolves a globally unique environment alias.
func Lookup(cfg config.Config, alias string) (config.EnvironmentLookup, error) {
	alias = strings.TrimSpace(alias)
	if env, ok := cfg.LookupEnvironment(alias); ok {
		return env, nil
	}
	available := cfg.Aliases()
	return config.EnvironmentLookup{}, NotFoundError{
		Query:     alias,
		Available: available,
		Suggest:   suggest(alias, available),
	}
}

func suggest(query string, aliases []string) string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || len(aliases) == 0 {
		return ""
	}
	best := ""
	bestDist := 3
	for _, alias := range aliases {
		lower := strings.ToLower(alias)
		if strings.HasPrefix(lower, query) || strings.HasPrefix(query, lower) {
			return alias
		}
		d := levenshtein(query, lower)
		if d < bestDist {
			bestDist = d
			best = alias
		}
	}
	if bestDist <= 2 {
		return best
	}
	return ""
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min(del, ins, sub)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}
