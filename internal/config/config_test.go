package config

import "testing"

func TestLoad_DefaultRunModePolling(t *testing.T) {
	t.Setenv("BOT_TOKEN", "test-token")
	t.Setenv("BOT_RUN_MODE", "")
	t.Setenv("WEBHOOK_URL", "")
	t.Setenv("WEBHOOK_LISTEN_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BotRunMode != "polling" {
		t.Fatalf("BotRunMode = %q, want polling", cfg.BotRunMode)
	}
	if cfg.WebhookListenAddr != ":8080" {
		t.Fatalf("WebhookListenAddr = %q, want :8080", cfg.WebhookListenAddr)
	}
}

func TestLoad_WebhookModeRequiresURL(t *testing.T) {
	t.Setenv("BOT_TOKEN", "test-token")
	t.Setenv("BOT_RUN_MODE", "webhook")
	t.Setenv("WEBHOOK_URL", "")
	t.Setenv("WEBHOOK_LISTEN_ADDR", ":8080")

	_, err := Load()
	if err == nil {
		t.Fatalf("Load() error = nil, want error")
	}
}

func TestLoad_WebhookModeWithValidURL(t *testing.T) {
	t.Setenv("BOT_TOKEN", "test-token")
	t.Setenv("BOT_RUN_MODE", "webhook")
	t.Setenv("WEBHOOK_URL", "https://example.com/tg/webhook")
	t.Setenv("WEBHOOK_LISTEN_ADDR", ":8080")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BotRunMode != "webhook" {
		t.Fatalf("BotRunMode = %q, want webhook", cfg.BotRunMode)
	}
}

func TestLoad_InvalidRunMode(t *testing.T) {
	t.Setenv("BOT_TOKEN", "test-token")
	t.Setenv("BOT_RUN_MODE", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatalf("Load() error = nil, want error")
	}
}
