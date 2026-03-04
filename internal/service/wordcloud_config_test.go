package service

import "testing"

func TestNormalizeWordCloudConfigAutoPushDisabled(t *testing.T) {
	cfg := normalizeWordCloudConfig(wordCloudConfig{
		PushHour:    -1,
		PushMinute:  35,
		LastPushDay: " 2026-03-04 ",
	})
	if cfg.PushHour != -1 {
		t.Fatalf("want PushHour=-1, got %d", cfg.PushHour)
	}
	if cfg.PushMinute != 0 {
		t.Fatalf("want PushMinute=0 when disabled, got %d", cfg.PushMinute)
	}
	if cfg.LastPushDay != "2026-03-04" {
		t.Fatalf("unexpected LastPushDay: %q", cfg.LastPushDay)
	}
}

func TestNormalizeWordCloudConfigInvalidRange(t *testing.T) {
	cfg := normalizeWordCloudConfig(wordCloudConfig{
		PushHour:   99,
		PushMinute: 99,
	})
	if cfg.PushHour != 18 {
		t.Fatalf("want PushHour=18, got %d", cfg.PushHour)
	}
	if cfg.PushMinute != 0 {
		t.Fatalf("want PushMinute=0, got %d", cfg.PushMinute)
	}
}
