package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"supervisor/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Repository struct {
	db *gorm.DB
}

func New(dbPath string, gormLogSilent bool) (*Repository, error) {
	dsn := strings.TrimSpace(dbPath)
	if dsn == "" {
		return nil, fmt.Errorf("db dsn is empty")
	}

	dialector, isPostgres, sqliteDSN, err := openDialector(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse db dsn: %w", err)
	}
	if !isPostgres {
		if err := ensureSQLiteDir(sqliteDSN); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	gormCfg := &gorm.Config{}
	if gormLogSilent {
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	}

	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		if isPostgres {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if !isPostgres {
		if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
			return nil, fmt.Errorf("enable wal: %w", err)
		}
	}

	if err := db.AutoMigrate(
		&model.User{}, &model.Group{}, &model.GroupAdmin{}, &model.GroupSetting{},
		&model.AutoReply{}, &model.ScheduledMessage{}, &model.BannedWord{},
		&model.UserPoint{}, &model.PointEvent{}, &model.Lottery{}, &model.LotteryParticipant{}, &model.Chain{}, &model.ChainEntry{}, &model.Log{},
		&model.GroupBlacklist{}, &model.AutoDeleteTask{}, &model.JoinVerifyPending{},
		&model.InviteLink{}, &model.InviteEvent{}, &model.GroupMemberJoin{},
		&model.AISpamCache{},
		&model.WordCloudToken{}, &model.WordCloudDailyUserStat{}, &model.WordCloudBlacklistWord{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

func openDialector(dsn string) (gorm.Dialector, bool, string, error) {
	lower := strings.ToLower(dsn)
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return postgres.Open(dsn), true, "", nil
	}
	if strings.HasPrefix(lower, "pgsql://") {
		return postgres.Open("postgres://" + dsn[len("pgsql://"):]), true, "", nil
	}
	if strings.HasPrefix(lower, "sqlite://") {
		sqliteDSN := dsn[len("sqlite://"):]
		return sqlite.Open(sqliteDSN), false, sqliteDSN, nil
	}
	if strings.HasPrefix(lower, "sqlite3://") {
		sqliteDSN := dsn[len("sqlite3://"):]
		return sqlite.Open(sqliteDSN), false, sqliteDSN, nil
	}
	return nil, false, "", fmt.Errorf("unsupported db dsn format, must start with sqlite://, sqlite3://, postgres://, postgresql:// or pgsql://")
}

func ensureSQLiteDir(dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return fmt.Errorf("sqlite dsn is empty")
	}

	lower := strings.ToLower(dsn)
	if dsn == ":memory:" || strings.HasPrefix(lower, "file:") {
		return nil
	}
	return os.MkdirAll(filepath.Dir(dsn), 0o755)
}
