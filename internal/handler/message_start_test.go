package handler

import "testing"

func TestStartWelcomeTextUsesBotName(t *testing.T) {
	got := startWelcomeText("MyBot")
	want := "欢迎使用 MyBot。\n请通过按钮管理群组。"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestStartWelcomeTextFallsBackWhenEmpty(t *testing.T) {
	got := startWelcomeText("   ")
	want := "欢迎使用 机器人。\n请通过按钮管理群组。"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
