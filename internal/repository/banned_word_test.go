package repository

import (
	"path/filepath"
	"testing"
)

func TestDeleteBannedWordsByWord(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "repo.db")
	repo, err := New("sqlite://"+dbPath, true)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}

	groupID := uint(1)
	if err := repo.CreateBannedWord(groupID, "他妈的"); err != nil {
		t.Fatalf("create banned word failed: %v", err)
	}
	if err := repo.CreateBannedWord(groupID, "尼玛"); err != nil {
		t.Fatalf("create banned word failed: %v", err)
	}
	if err := repo.CreateBannedWord(groupID, "尼玛"); err != nil {
		t.Fatalf("create banned word failed: %v", err)
	}

	deleted, err := repo.DeleteBannedWordsByWord(groupID, "尼玛")
	if err != nil {
		t.Fatalf("delete banned words by word failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("want deleted=2, got %d", deleted)
	}

	words, err := repo.GetBannedWords(groupID)
	if err != nil {
		t.Fatalf("list banned words failed: %v", err)
	}
	if len(words) != 1 || words[0].Word != "他妈的" {
		t.Fatalf("unexpected words: %+v", words)
	}

	deleted, err = repo.DeleteBannedWordsByWord(groupID, "不存在")
	if err != nil {
		t.Fatalf("delete non-existent banned words failed: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("want deleted=0 for non-existent word, got %d", deleted)
	}
}
