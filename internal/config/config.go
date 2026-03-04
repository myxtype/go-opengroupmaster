package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken              string
	DBPath                string
	BotRunMode            string
	BotDebug              bool
	GormLogSilent         bool
	AdminSyncIntervalSecs int
	WebhookURL            string
	WebhookListenAddr     string
	WebhookSecretToken    string
	WebhookDropPending    bool
	WordCloudFontPath     string
	WordCloudJiebaDictDir string
	AntiSpamAIModel       string
	AntiSpamAIServerURL   string
	AntiSpamAITimeoutSecs int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		BotToken:              os.Getenv("BOT_TOKEN"),
		DBPath:                envOrDefault("DB_PATH", "sqlite://./data/bot.db"),
		BotRunMode:            strings.ToLower(strings.TrimSpace(envOrDefault("BOT_RUN_MODE", "polling"))),
		BotDebug:              parseBool(os.Getenv("BOT_DEBUG")),
		GormLogSilent:         parseBool(os.Getenv("GORM_LOG_SILENT")),
		AdminSyncIntervalSecs: parseIntDefault(os.Getenv("ADMIN_SYNC_INTERVAL_SECS"), 300),
		WebhookURL:            strings.TrimSpace(os.Getenv("WEBHOOK_URL")),
		WebhookListenAddr:     strings.TrimSpace(envOrDefault("WEBHOOK_LISTEN_ADDR", ":8080")),
		WebhookSecretToken:    strings.TrimSpace(os.Getenv("WEBHOOK_SECRET_TOKEN")),
		WebhookDropPending:    parseBool(os.Getenv("WEBHOOK_DROP_PENDING_UPDATES")),
		WordCloudFontPath:     strings.TrimSpace(os.Getenv("WORDCLOUD_FONT_PATH")),
		WordCloudJiebaDictDir: strings.TrimSpace(os.Getenv("WORDCLOUD_JIEBA_DICT_DIR")),
		AntiSpamAIModel:       strings.TrimSpace(os.Getenv("ANTI_SPAM_AI_MODEL")),
		AntiSpamAIServerURL:   envOrDefault("ANTI_SPAM_AI_SERVER_URL", "http://127.0.0.1:11434"),
		AntiSpamAITimeoutSecs: parseIntDefault(os.Getenv("ANTI_SPAM_AI_TIMEOUT_SECS"), 8),
	}
	if cfg.AdminSyncIntervalSecs < 1 {
		cfg.AdminSyncIntervalSecs = 300
	}
	if strings.TrimSpace(cfg.AntiSpamAIServerURL) == "" {
		cfg.AntiSpamAIServerURL = "http://127.0.0.1:11434"
	}
	if cfg.AntiSpamAITimeoutSecs < 1 {
		cfg.AntiSpamAITimeoutSecs = 8
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}
	if cfg.BotRunMode != "polling" && cfg.BotRunMode != "webhook" {
		return nil, fmt.Errorf("BOT_RUN_MODE must be polling or webhook")
	}
	if cfg.BotRunMode == "webhook" {
		if cfg.WebhookURL == "" {
			return nil, fmt.Errorf("WEBHOOK_URL is required when BOT_RUN_MODE=webhook")
		}
		u, err := url.Parse(cfg.WebhookURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("WEBHOOK_URL must be a valid absolute URL")
		}
		if cfg.WebhookListenAddr == "" {
			return nil, fmt.Errorf("WEBHOOK_LISTEN_ADDR is required when BOT_RUN_MODE=webhook")
		}
	}
	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseBool(v string) bool {
	b, _ := strconv.ParseBool(v)
	return b
}

func parseIntDefault(v string, fallback int) int {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
