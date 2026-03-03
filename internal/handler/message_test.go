package handler

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestSkipIncomingMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  *tgbotapi.Message
		want bool
	}{
		{
			name: "nil message",
			msg:  nil,
			want: true,
		},
		{
			name: "bot user message",
			msg: &tgbotapi.Message{
				From: &tgbotapi.User{IsBot: true},
			},
			want: true,
		},
		{
			name: "normal user message",
			msg: &tgbotapi.Message{
				From: &tgbotapi.User{ID: 1001},
			},
			want: false,
		},
		{
			name: "sender chat message",
			msg: &tgbotapi.Message{
				SenderChat: &tgbotapi.Chat{ID: -100200300},
			},
			want: false,
		},
		{
			name: "missing identities",
			msg:  &tgbotapi.Message{},
			want: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := skipIncomingMessage(tc.msg); got != tc.want {
				t.Fatalf("skipIncomingMessage()=%v, want=%v", got, tc.want)
			}
		})
	}
}
