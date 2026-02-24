package model

import "time"

type User struct {
	ID        uint  `gorm:"primaryKey"`
	TGUserID  int64 `gorm:"uniqueIndex;not null"`
	Username  string
	FirstName string
	LastName  string
	Language  string `gorm:"default:zh"`
	CreatedAt time.Time
}

type Group struct {
	ID        uint  `gorm:"primaryKey"`
	TGGroupID int64 `gorm:"uniqueIndex;not null"`
	Title     string
	OwnerID   uint
	BotAdded  bool `gorm:"default:false"`
	CreatedAt time.Time
}

type GroupAdmin struct {
	ID      uint   `gorm:"primaryKey"`
	GroupID uint   `gorm:"index;not null"`
	UserID  uint   `gorm:"index;not null"`
	Role    string `gorm:"default:admin"`
}

type GroupSetting struct {
	ID         uint   `gorm:"primaryKey"`
	GroupID    uint   `gorm:"index;not null"`
	FeatureKey string `gorm:"index;not null"`
	Enabled    bool   `gorm:"default:true"`
	Config     string `gorm:"type:text"`
}

type AutoReply struct {
	ID         uint   `gorm:"primaryKey"`
	GroupID    uint   `gorm:"index;not null"`
	Keyword    string `gorm:"index;not null"`
	Reply      string `gorm:"type:text;not null"`
	MatchType  string `gorm:"default:exact"`
	ButtonRows string `gorm:"type:text"`
}

type ScheduledMessage struct {
	ID         uint   `gorm:"primaryKey"`
	GroupID    uint   `gorm:"index;not null"`
	Content    string `gorm:"type:text;not null"`
	CronExpr   string `gorm:"not null"`
	Enabled    bool   `gorm:"default:true"`
	ButtonRows string `gorm:"type:text"`
}

type BannedWord struct {
	ID      uint   `gorm:"primaryKey"`
	GroupID uint   `gorm:"index;not null"`
	Word    string `gorm:"index;not null"`
}

type UserPoint struct {
	ID      uint `gorm:"primaryKey"`
	GroupID uint `gorm:"index;not null"`
	UserID  uint `gorm:"index;not null"`
	Points  int  `gorm:"default:0"`
}

type Lottery struct {
	ID           uint   `gorm:"primaryKey"`
	GroupID      uint   `gorm:"index;not null"`
	Title        string `gorm:"not null"`
	JoinKeyword  string `gorm:"not null;default:参加"`
	WinnersCount int    `gorm:"default:1"`
	Status       string `gorm:"default:active"`
}

type LotteryParticipant struct {
	ID        uint `gorm:"primaryKey"`
	LotteryID uint `gorm:"index;not null"`
	UserID    uint `gorm:"index;not null"`
}

type Log struct {
	ID         uint      `gorm:"primaryKey"`
	GroupID    uint      `gorm:"index;not null"`
	Action     string    `gorm:"not null"`
	OperatorID uint      `gorm:"index"`
	TargetID   uint      `gorm:"index"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

type GroupBlacklist struct {
	ID        uint  `gorm:"primaryKey"`
	GroupID   uint  `gorm:"uniqueIndex:idx_group_blacklist_user;index;not null"`
	TGUserID  int64 `gorm:"uniqueIndex:idx_group_blacklist_user;index;not null"`
	Reason    string
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
