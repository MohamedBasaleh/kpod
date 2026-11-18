package onboard

import "testing"

func TestSuggestAliasHyphenPrefix(t *testing.T) {
	got := SuggestAlias("dev-core", nil)
	if got != "dev" {
		t.Fatalf("got %q", got)
	}
}

func TestSuggestAliasCollisionUsesSuffix(t *testing.T) {
	taken := map[string]struct{}{"dev": {}}
	got := SuggestAlias("dev-inventory", taken)
	if got != "inventory" {
		t.Fatalf("got %q", got)
	}
}

func TestSuggestAliasCollisionUsesNamespace(t *testing.T) {
	taken := map[string]struct{}{"dev": {}, "core": {}}
	got := SuggestAlias("dev-core", taken)
	if got != "dev-core" {
		t.Fatalf("got %q", got)
	}
}

func TestSuggestAliasPlain(t *testing.T) {
	got := SuggestAlias("argocd", nil)
	if got != "argocd" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateAlias(t *testing.T) {
	if err := ValidateAlias("  ", nil); err == nil {
		t.Fatal("empty")
	}
	if err := ValidateAlias("dev core", nil); err == nil {
		t.Fatal("spaces")
	}
	if err := ValidateAlias("dev", map[string]struct{}{"dev": {}}); err == nil {
		t.Fatal("duplicate")
	}
	if err := ValidateAlias("dev", nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlias("dev;rm", nil); err == nil {
		t.Fatal("metacharacters")
	}
}
