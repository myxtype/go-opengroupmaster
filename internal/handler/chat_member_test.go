package handler

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestIsBotActivatedInChat(t *testing.T) {
	update := &models.ChatMemberUpdated{
		OldChatMember: models.ChatMember{
			Type: models.ChatMemberTypeLeft,
			Left: &models.ChatMemberLeft{User: &models.User{ID: 1}},
		},
		NewChatMember: models.ChatMember{
			Type:   models.ChatMemberTypeMember,
			Member: &models.ChatMemberMember{User: &models.User{ID: 1}},
		},
	}

	if !isBotActivatedInChat(update) {
		t.Fatalf("expected bot activation to be detected")
	}
}

func TestIsBotActivatedInChatIgnoresExistingMembership(t *testing.T) {
	update := &models.ChatMemberUpdated{
		OldChatMember: models.ChatMember{
			Type:          models.ChatMemberTypeAdministrator,
			Administrator: &models.ChatMemberAdministrator{User: models.User{ID: 1}},
		},
		NewChatMember: models.ChatMember{
			Type:          models.ChatMemberTypeAdministrator,
			Administrator: &models.ChatMemberAdministrator{User: models.User{ID: 1}},
		},
	}

	if isBotActivatedInChat(update) {
		t.Fatalf("expected unchanged active membership to be ignored")
	}
}

func TestBotAddedToGroupTextUsesBotName(t *testing.T) {
	got := botAddedToGroupText("MyBot")
	want := "MyBot 已成功加入本群。\n点击下方按钮前往私聊，打开 /start 后即可开始配置群功能。"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestBotAddedToGroupTextFallsBackWhenEmpty(t *testing.T) {
	got := botAddedToGroupText("   ")
	want := "机器人 已成功加入本群。\n点击下方按钮前往私聊，打开 /start 后即可开始配置群功能。"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
