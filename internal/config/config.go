package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken string
	DBPath   string
	BotDebug bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		BotToken: os.Getenv("BOT_TOKEN"),
		DBPath:   envOrDefault("DB_PATH", "./data/bot.db"),
		BotDebug: parseBool(os.Getenv("BOT_DEBUG")),
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
