package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken              string
	DBPath                string
	BotDebug              bool
	UpdateWorkers         int
	AdminSyncIntervalSecs int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		BotToken:              os.Getenv("BOT_TOKEN"),
		DBPath:                envOrDefault("DB_PATH", "./data/bot.db"),
		BotDebug:              parseBool(os.Getenv("BOT_DEBUG")),
		UpdateWorkers:         parseIntDefault(os.Getenv("UPDATE_WORKERS"), 8),
		AdminSyncIntervalSecs: parseIntDefault(os.Getenv("ADMIN_SYNC_INTERVAL_SECS"), 300),
	}
	if cfg.UpdateWorkers < 1 {
		cfg.UpdateWorkers = 1
	}
	if cfg.AdminSyncIntervalSecs < 1 {
		cfg.AdminSyncIntervalSecs = 300
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
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
