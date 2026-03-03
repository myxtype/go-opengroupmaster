package service

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestAntiSpamViolation_BlockExternalReply(t *testing.T) {
	cfg := defaultAntiSpamConfig()
	cfg.BlockExternalReply = true

	msg := &models.Message{
		ExternalReply: &models.ExternalReplyInfo{},
	}

	blocked, code, label := antiSpamViolation(msg, cfg)
	if !blocked {
		t.Fatalf("expected blocked=true")
	}
	if code != "external_reply" {
		t.Fatalf("expected reason code external_reply, got %s", code)
	}
	if label != "外部回复" {
		t.Fatalf("expected label 外部回复, got %s", label)
	}
}

func TestAntiSpamViolation_AllowExternalReplyWhenDisabled(t *testing.T) {
	cfg := defaultAntiSpamConfig()
	cfg.BlockExternalReply = false

	msg := &models.Message{
		ExternalReply: &models.ExternalReplyInfo{},
	}

	blocked, _, _ := antiSpamViolation(msg, cfg)
	if blocked {
		t.Fatalf("expected blocked=false when BlockExternalReply disabled")
	}
}
