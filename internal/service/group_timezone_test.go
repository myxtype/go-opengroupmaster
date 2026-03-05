package service

import (
	"io"
	"log"
	"path/filepath"
	"testing"

	"supervisor/internal/config"
	"supervisor/internal/repository"

	"github.com/go-telegram/bot/models"
)

func TestParseUTCOffset(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "plus hour", raw: "+8", want: 8 * 60},
		{name: "minus hour", raw: "-5", want: -5 * 60},
		{name: "utc prefix", raw: "UTC+8:30", want: 8*60 + 30},
		{name: "empty", raw: "", wantErr: true},
		{name: "range overflow", raw: "+15", wantErr: true},
		{name: "invalid format", raw: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUTCOffset(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("want %d, got %d", tt.want, got)
			}
		})
	}
}

func TestGroupTimezone_DefaultAndNightModeConsistency(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "group-timezone.db")
	repo, err := repository.New("sqlite://"+dbPath, true)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	svc := New(repo, log.New(io.Discard, "", 0), &config.Config{})

	group, err := repo.UpsertGroup(&models.Chat{ID: -10001, Title: "时区测试群"})
	if err != nil {
		t.Fatalf("upsert group failed: %v", err)
	}

	tzView, err := svc.GroupTimezoneViewByTGGroupID(group.TGGroupID)
	if err != nil {
		t.Fatalf("load group timezone view failed: %v", err)
	}
	if tzView.OffsetMinutes != 8*60 || tzView.TimezoneText != "UTC+8" {
		t.Fatalf("unexpected default timezone view: %+v", tzView)
	}

	tz, err := svc.SetGroupTimezoneByTGGroupID(group.TGGroupID, "-5:30")
	if err != nil {
		t.Fatalf("set group timezone failed: %v", err)
	}
	if tz != "UTC-5:30" {
		t.Fatalf("want timezone UTC-5:30, got %s", tz)
	}

	nightView, err := svc.NightModeViewByTGGroupID(group.TGGroupID)
	if err != nil {
		t.Fatalf("load night mode view failed: %v", err)
	}
	if nightView.TimezoneText != "UTC-5:30" {
		t.Fatalf("night mode should read group timezone, got %s", nightView.TimezoneText)
	}
}
