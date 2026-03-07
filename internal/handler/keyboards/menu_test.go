package keyboards

import "testing"

func TestGroupOnboardingKeyboardUsesPrivateStartLink(t *testing.T) {
	markup := GroupOnboardingKeyboard("demo_bot")
	if len(markup.InlineKeyboard) != 1 || len(markup.InlineKeyboard[0]) != 1 {
		t.Fatalf("unexpected keyboard layout: %#v", markup.InlineKeyboard)
	}
	button := markup.InlineKeyboard[0][0]
	want := "https://t.me/demo_bot?start=group_onboarding"
	if button.URL != want {
		t.Fatalf("want %q, got %q", want, button.URL)
	}
}
