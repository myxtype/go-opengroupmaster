package tgmention

import (
	"strings"
	"testing"
)

func TestUserLabelPrefersNameOverUsername(t *testing.T) {
	got := UserLabel(UserRef{
		FirstName: "Alice",
		LastName:  "Smith",
		Username:  "alice_smith",
	})
	if got != "Alice Smith" {
		t.Fatalf("expected name label, got %q", got)
	}
}

func TestUserLabelMasksLongName(t *testing.T) {
	got := UserLabel(UserRef{
		FirstName: strings.Repeat("a", maxDisplayNameRunes+5),
	})
	want := strings.Repeat("a", maxDisplayNameRunes-1) + nameMaskSuffix
	if got != want {
		t.Fatalf("expected masked long name %q, got %q", want, got)
	}
	if len([]rune(got)) != maxDisplayNameRunes {
		t.Fatalf("expected %d runes, got %d", maxDisplayNameRunes, len([]rune(got)))
	}
}

func TestUserLabelFallsBackToUsername(t *testing.T) {
	got := UserLabel(UserRef{
		Username: "alice_smith",
	})
	if got != "@alice_smith" {
		t.Fatalf("expected username label, got %q", got)
	}
}
