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

func TestGetPointsConfigDoesNotEnableFeature(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "points.db")
	repo, err := repository.New("sqlite://"+dbPath, true)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	svc := New(repo, log.New(io.Discard, "", 0), &config.Config{})

	group, err := repo.UpsertGroup(&models.Chat{ID: -10001, Title: "积分测试群"})
	if err != nil {
		t.Fatalf("upsert group failed: %v", err)
	}

	cfg, err := svc.getPointsConfig(group.ID)
	if err != nil {
		t.Fatalf("get points config failed: %v", err)
	}
	if cfg.BalanceAlias != "积分" {
		t.Fatalf("unexpected points config: %+v", cfg)
	}

	enabled, err := svc.pointsEnabled(group.ID)
	if err != nil {
		t.Fatalf("points enabled check failed: %v", err)
	}
	if enabled {
		t.Fatalf("expected points feature to remain disabled after reading config")
	}
}

func TestGetWelcomeConfigKeepsDefaultFeatureEnabled(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "welcome.db")
	repo, err := repository.New("sqlite://"+dbPath, true)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	svc := New(repo, log.New(io.Discard, "", 0), &config.Config{})

	group, err := repo.UpsertGroup(&models.Chat{ID: -10002, Title: "欢迎测试群"})
	if err != nil {
		t.Fatalf("upsert group failed: %v", err)
	}

	cfg, err := svc.getWelcomeConfig(group.ID)
	if err != nil {
		t.Fatalf("get welcome config failed: %v", err)
	}
	if cfg.Mode != "verify" {
		t.Fatalf("unexpected welcome config: %+v", cfg)
	}

	enabled, err := svc.IsFeatureEnabled(group.ID, featureWelcome, true)
	if err != nil {
		t.Fatalf("welcome enabled check failed: %v", err)
	}
	if !enabled {
		t.Fatalf("expected welcome feature to stay enabled after reading config")
	}
}
