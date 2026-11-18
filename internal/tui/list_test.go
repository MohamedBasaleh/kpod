package tui

import "testing"

func TestChecklistToggleAndMove(t *testing.T) {
	c := newChecklist([]string{"Nawaqis-Dev", "Nawaqis-Prod"})
	c.toggle()
	if !c.isSelected("Nawaqis-Dev") {
		t.Fatal("expected selected")
	}
	c.move(1)
	c.toggle()
	got := c.selectedItems()
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
	c.move(-10)
	if c.current() != "Nawaqis-Dev" {
		t.Fatalf("cursor %q", c.current())
	}
}

func TestChecklistViewTruncates(t *testing.T) {
	long := "arn:aws:eks:eu-central-1:133136949499:cluster/Nawaqis-Development"
	c := newChecklist([]string{long})
	view := c.view(24, 5)
	if len([]rune(view)) > 80 && !containsEllipsis(view) {
		t.Fatalf("expected truncation: %q", view)
	}
}

func containsEllipsis(s string) bool {
	for _, r := range s {
		if r == '…' {
			return true
		}
	}
	return false
}
