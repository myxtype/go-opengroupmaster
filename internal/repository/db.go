package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"supervisor/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func New(dbPath string) (*Repository, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		return nil, fmt.Errorf("enable wal: %w", err)
	}

	if err := db.AutoMigrate(
		&model.User{}, &model.Group{}, &model.GroupAdmin{}, &model.GroupSetting{},
		&model.AutoReply{}, &model.ScheduledMessage{}, &model.BannedWord{},
		&model.UserPoint{}, &model.Lottery{}, &model.LotteryParticipant{}, &model.Log{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}
