package tui

import "testing"

func TestMatchesSubstring(t *testing.T) {
	if !matches("STA", "staging", "staging-core") {
		t.Fatal("expected match")
	}
	if matches("prod", "dev", "dev-core") {
		t.Fatal("unexpected match")
	}
}
