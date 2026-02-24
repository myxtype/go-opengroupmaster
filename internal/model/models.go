package model

import "time"

// User 表示系统中的 Telegram 用户。
type User struct {
	ID        uint  `gorm:"primaryKey"`
	TGUserID  int64 `gorm:"uniqueIndex;not null"`
	Username  string
	FirstName string
	LastName  string
	Language  string `gorm:"default:zh"`
	CreatedAt time.Time
}

// Group 表示一个被机器人管理的 Telegram 群组。
type Group struct {
	ID        uint  `gorm:"primaryKey"`
	TGGroupID int64 `gorm:"uniqueIndex;not null"`
	Title     string
	OwnerID   uint
	BotAdded  bool `gorm:"default:false"`
	CreatedAt time.Time
}

// GroupAdmin 表示群组管理员与权限角色的关联关系。
type GroupAdmin struct {
	ID      uint   `gorm:"primaryKey"`
	GroupID uint   `gorm:"index;not null"`
	UserID  uint   `gorm:"index;not null"`
	Role    string `gorm:"default:admin"`
}

// GroupSetting 表示群组功能开关和配置项。
type GroupSetting struct {
	ID         uint   `gorm:"primaryKey"`
	GroupID    uint   `gorm:"index;not null"`
	FeatureKey string `gorm:"index;not null"`
	Enabled    bool   `gorm:"default:true"`
	Config     string `gorm:"type:text"`
}

// AutoReply 表示群组自动回复规则。
type AutoReply struct {
	ID         uint   `gorm:"primaryKey"`
	GroupID    uint   `gorm:"index;not null"`
	Keyword    string `gorm:"index;not null"`
	Reply      string `gorm:"type:text;not null"`
	MatchType  string `gorm:"default:exact"`
	ButtonRows string `gorm:"type:text"`
}

// ScheduledMessage 表示按计划在群组发送的定时消息。
type ScheduledMessage struct {
	ID         uint   `gorm:"primaryKey"`
	GroupID    uint   `gorm:"index;not null"`
	Content    string `gorm:"type:text;not null"`
	CronExpr   string `gorm:"not null"`
	Enabled    bool   `gorm:"default:true"`
	ButtonRows string `gorm:"type:text"`
}

// AutoDeleteTask 表示待执行的消息自动删除任务（持久化队列）。
type AutoDeleteTask struct {
	ID        uint      `gorm:"primaryKey"`
	ChatID    int64     `gorm:"index:idx_auto_delete_due,priority:2;not null"`
	MessageID int       `gorm:"not null"`
	ExecuteAt time.Time `gorm:"index:idx_auto_delete_due,priority:1;not null"`
	Attempts  int       `gorm:"default:0"`
	CreatedAt time.Time
}

// BannedWord 表示群组的违禁词条目。
type BannedWord struct {
	ID      uint   `gorm:"primaryKey"`
	GroupID uint   `gorm:"index;not null"`
	Word    string `gorm:"index;not null"`
}

// UserPoint 表示用户在群组内的积分记录。
type UserPoint struct {
	ID      uint `gorm:"primaryKey"`
	GroupID uint `gorm:"index;not null"`
	UserID  uint `gorm:"index;not null"`
	Points  int  `gorm:"default:0"`
}

// Lottery 表示群组内的一次抽奖活动。
type Lottery struct {
	ID           uint   `gorm:"primaryKey"`
	GroupID      uint   `gorm:"index;not null"`
	Title        string `gorm:"not null"`
	JoinKeyword  string `gorm:"not null;default:参加"`
	WinnersCount int    `gorm:"default:1"`
	Status       string `gorm:"default:active"`
}

// LotteryParticipant 表示抽奖活动参与者记录。
type LotteryParticipant struct {
	ID        uint `gorm:"primaryKey"`
	LotteryID uint `gorm:"index;not null"`
	UserID    uint `gorm:"index;not null"`
}

// Log 表示群组操作审计日志。
type Log struct {
	ID         uint      `gorm:"primaryKey"`
	GroupID    uint      `gorm:"index;not null"`
	Action     string    `gorm:"not null"`
	OperatorID uint      `gorm:"index"`
	TargetID   uint      `gorm:"index"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

// GroupBlacklist 表示群组级别的用户黑名单记录。
type GroupBlacklist struct {
	ID        uint  `gorm:"primaryKey"`
	GroupID   uint  `gorm:"uniqueIndex:idx_group_blacklist_user;index;not null"`
	TGUserID  int64 `gorm:"uniqueIndex:idx_group_blacklist_user;index;not null"`
	Reason    string
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
