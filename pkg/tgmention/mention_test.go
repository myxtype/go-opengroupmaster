package tgmention

import (
	"testing"
)

func TestUserLabelPrefersNameOverUsername(t *testing.T) {
	got := UserLabel(UserRef{
		FirstName: "Alice",
		LastName:  "Smith",
		Username:  "alice_smith",
	})
	if got != "A░i░e░S░i░h" {
		t.Fatalf("expected name label, got %q", got)
	}
}

func TestUserLabelMasksLongName(t *testing.T) {
	got := UserLabel(UserRef{
		FirstName: "123445567823123",
	})
	want := "1░3░4░5░7░2░1░3"
	if got != want {
		t.Fatalf("expected masked long name %q, got %q", want, got)
	}
}

func TestUserLabelFallsBackToUsername(t *testing.T) {
	got := UserLabel(UserRef{
		Username: "alice_smith",
	})
	if got != "@a░i░e░s░i░h" {
		t.Fatalf("expected username label, got %q", got)
	}
}
