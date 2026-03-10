package service

import (
	"testing"
	"unicode/utf8"
)

func TestBuildChineseCaptchaOptionsUsesConfiguredLength(t *testing.T) {
	answer := "中"

	options := buildChineseCaptchaOptions(answer)

	if len(options) != captchaOptionCount {
		t.Fatalf("expected %d options, got %d", captchaOptionCount, len(options))
	}
	found := false
	for _, option := range options {
		if utf8.RuneCountInString(option) != chineseCaptchaLength {
			t.Fatalf("expected option %q to have %d chinese characters", option, chineseCaptchaLength)
		}
		if option == answer {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected options to contain answer %q", answer)
	}
}

func TestRandomChineseCaptchaTextUsesConfiguredLength(t *testing.T) {
	text := randomChineseCaptchaText(0)
	if utf8.RuneCountInString(text) != chineseCaptchaLength {
		t.Fatalf("expected %d chinese characters, got %q", chineseCaptchaLength, text)
	}
}
