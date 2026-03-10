package handler

import (
	"strings"
	"testing"
)

func TestGroupHelpTextContainsSyncCommand(t *testing.T) {
	if !strings.Contains(groupHelpText(), "/sync - 立即同步群管理员权限（管理员，无冷却）") {
		t.Fatalf("expected group help to include /sync command")
	}
}

func TestPrivateHelpTextContainsSyncCommand(t *testing.T) {
	if !strings.Contains(privateHelpText(), "/sync - 立即同步群管理员权限（管理员，无冷却）") {
		t.Fatalf("expected private help to include /sync command")
	}
}
