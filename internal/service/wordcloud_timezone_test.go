package service

import (
	"io"
	"log"
	"path/filepath"
	"testing"
	"time"

	"supervisor/internal/config"
	"supervisor/internal/repository"

	"github.com/go-telegram/bot/models"
)

func TestWordCloudReadyToPush_UsesGroupTimezone(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wordcloud-timezone.db")
	repo, err := repository.New("sqlite://"+dbPath, true)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	svc := New(repo, log.New(io.Discard, "", 0), &config.Config{})

	group, err := repo.UpsertGroup(&models.Chat{ID: -10001, Title: "词云时区群"})
	if err != nil {
		t.Fatalf("upsert group failed: %v", err)
	}
	if _, err := svc.SetWordCloudEnabledByTGGroupID(group.TGGroupID, true); err != nil {
		t.Fatalf("enable word cloud failed: %v", err)
	}
	if _, _, err := svc.SetWordCloudPushTimeByTGGroupID(group.TGGroupID, "09:30"); err != nil {
		t.Fatalf("set push time failed: %v", err)
	}
	if _, err := svc.SetGroupTimezoneByTGGroupID(group.TGGroupID, "+8"); err != nil {
		t.Fatalf("set group timezone failed: %v", err)
	}

	group, err = repo.FindGroupByTGID(group.TGGroupID)
	if err != nil {
		t.Fatalf("reload group failed: %v", err)
	}

	loc := timezoneLocation(8 * 60)
	now := time.Date(2026, time.March, 5, 9, 30, 0, 0, loc).UTC()
	ready, err := svc.wordCloudReadyToPush(group, now)
	if err != nil {
		t.Fatalf("wordCloudReadyToPush failed: %v", err)
	}
	if !ready {
		t.Fatalf("want ready=true at configured local time")
	}

	// Same UTC minute but local time no longer matches after +1 minute.
	ready, err = svc.wordCloudReadyToPush(group, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("wordCloudReadyToPush+1m failed: %v", err)
	}
	if ready {
		t.Fatalf("want ready=false when local minute does not match push time")
	}

	dayKey := wordCloudDayKey(now, 8*60)
	if err := svc.markWordCloudPushed(group.ID, dayKey); err != nil {
		t.Fatalf("markWordCloudPushed failed: %v", err)
	}
	ready, err = svc.wordCloudReadyToPush(group, now)
	if err != nil {
		t.Fatalf("wordCloudReadyToPush after mark failed: %v", err)
	}
	if ready {
		t.Fatalf("want ready=false after same-day push mark")
	}
}
