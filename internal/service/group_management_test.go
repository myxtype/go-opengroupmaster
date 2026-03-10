package service

import (
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
)

func TestBeginAdminSyncRespectsCooldown(t *testing.T) {
	s := &Service{
		adminSyncAt:    map[int64]time.Time{},
		adminSyncEvery: 3 * time.Minute,
	}
	const tgGroupID int64 = 1001

	if !s.beginAdminSync(tgGroupID, false) {
		t.Fatalf("expected first non-force sync to begin")
	}
	if s.beginAdminSync(tgGroupID, false) {
		t.Fatalf("expected second non-force sync to be blocked by cooldown")
	}
}

func TestBeginAdminSyncForceBypassesCooldown(t *testing.T) {
	s := &Service{
		adminSyncAt:    map[int64]time.Time{},
		adminSyncEvery: 3 * time.Minute,
	}
	const tgGroupID int64 = 1001

	if !s.beginAdminSync(tgGroupID, false) {
		t.Fatalf("expected first non-force sync to begin")
	}
	if !s.beginAdminSync(tgGroupID, true) {
		t.Fatalf("expected force sync to bypass cooldown")
	}
	if s.beginAdminSync(tgGroupID, false) {
		t.Fatalf("expected non-force sync to remain in cooldown after force sync")
	}
}

func TestIsGroupAdminChatMember(t *testing.T) {
	if !isGroupAdminChatMember(models.ChatMember{Type: models.ChatMemberTypeOwner}) {
		t.Fatalf("expected owner to be treated as group admin")
	}
	if !isGroupAdminChatMember(models.ChatMember{Type: models.ChatMemberTypeAdministrator}) {
		t.Fatalf("expected administrator to be treated as group admin")
	}
	if isGroupAdminChatMember(models.ChatMember{Type: models.ChatMemberTypeMember}) {
		t.Fatalf("expected member to be treated as non-admin")
	}
}
