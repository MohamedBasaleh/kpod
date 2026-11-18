package tui

import "strings"

func matches(query string, parts ...string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	for _, part := range parts {
		if strings.Contains(strings.ToLower(part), q) {
			return true
		}
	}
	return false
}
