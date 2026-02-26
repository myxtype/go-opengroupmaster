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
	ID          uint   `gorm:"primaryKey"`
	GroupID     uint   `gorm:"index;not null"`
	Content     string `gorm:"type:text;not null"`
	CronExpr    string `gorm:"not null"`
	Enabled     bool   `gorm:"default:true"`
	ButtonRows  string `gorm:"type:text"`
	MediaType   string `gorm:"default:''"` // photo/video/document/animation，空表示纯文本
	MediaFileID string `gorm:"type:text"`
	PinMessage  bool   `gorm:"default:false"`
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

// JoinVerifyPending 表示待完成的进群验证任务（持久化，支持重启恢复）。
type JoinVerifyPending struct {
	ID            uint      `gorm:"primaryKey"`
	TGGroupID     int64     `gorm:"uniqueIndex:idx_join_verify_pending_user,priority:1;index:idx_join_verify_pending_deadline,priority:2;not null"`
	TGUserID      int64     `gorm:"uniqueIndex:idx_join_verify_pending_user,priority:2;index:idx_join_verify_pending_deadline,priority:3;not null"`
	Mode          string    `gorm:"size:32;not null"`
	Answer        string    `gorm:"type:text"`
	MessageID     int       `gorm:"not null;default:0"`
	TimeoutAction string    `gorm:"size:16;not null;default:mute"`
	Deadline      time.Time `gorm:"index:idx_join_verify_pending_deadline,priority:1;not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
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

// Chain 表示群组中的一次接龙活动。
type Chain struct {
	ID                    uint      `gorm:"primaryKey"`
	GroupID               uint      `gorm:"index:idx_chain_group_created,priority:1;index:idx_chain_group_status,priority:1;not null"`
	Intro                 string    `gorm:"type:text;not null"`
	MaxParticipants       int       `gorm:"default:0"`
	DeadlineUnix          int64     `gorm:"default:0"`
	AnnouncementMessageID int       `gorm:"default:0"`
	Status                string    `gorm:"index:idx_chain_group_status,priority:2;not null;default:active"`
	CreatedAt             time.Time `gorm:"index:idx_chain_group_created,priority:2;autoCreateTime"`
	UpdatedAt             time.Time
}

// ChainEntry 表示接龙中的用户参与记录（同一接龙每个 TG 用户仅保留一条，可覆盖更新）。
type ChainEntry struct {
	ID          uint      `gorm:"primaryKey"`
	ChainID     uint      `gorm:"uniqueIndex:idx_chain_entry_user,priority:1;index:idx_chain_entry_chain,priority:1;not null"`
	TGUserID    int64     `gorm:"uniqueIndex:idx_chain_entry_user,priority:2;index:idx_chain_entry_chain,priority:2;not null"`
	DisplayName string    `gorm:"not null"`
	Content     string    `gorm:"type:text;not null"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
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

// InviteLink 表示群组中成员生成的邀请链接记录。
type InviteLink struct {
	ID              uint   `gorm:"primaryKey"`
	GroupID         uint   `gorm:"uniqueIndex:idx_invite_link_group_link,priority:1;index:idx_invite_link_group_created,priority:1;index:idx_invite_link_creator;not null"`
	CreatorTGUserID int64  `gorm:"index:idx_invite_link_creator;not null"`
	Link            string `gorm:"uniqueIndex:idx_invite_link_group_link,priority:2;type:text;not null"`
	ExpireDate      int64
	MemberLimit     int
	CreatedAt       time.Time `gorm:"index:idx_invite_link_group_created,priority:2;autoCreateTime"`
}

// InviteEvent 表示一次有效邀请（仅首次进群计数）。
type InviteEvent struct {
	ID              uint      `gorm:"primaryKey"`
	GroupID         uint      `gorm:"uniqueIndex:idx_invite_event_group_invitee,priority:1;index:idx_invite_event_group_inviter,priority:1;index:idx_invite_event_group_joined,priority:1;not null"`
	InviterTGUserID int64     `gorm:"index:idx_invite_event_group_inviter,priority:2;not null"`
	InviteeTGUserID int64     `gorm:"uniqueIndex:idx_invite_event_group_invitee,priority:2;not null"`
	Link            string    `gorm:"type:text;not null"`
	JoinedAt        time.Time `gorm:"index:idx_invite_event_group_joined,priority:2;not null"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
}

// GroupMemberJoin 表示成员首次进群记录（用于邀请防作弊）。
type GroupMemberJoin struct {
	ID          uint      `gorm:"primaryKey"`
	GroupID     uint      `gorm:"uniqueIndex:idx_group_member_first_join,priority:1;index;not null"`
	TGUserID    int64     `gorm:"uniqueIndex:idx_group_member_first_join,priority:2;index;not null"`
	FirstJoinAt time.Time `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

// AISpamCache 表示 AI 反垃圾判定缓存（避免同内容重复请求模型）。
type AISpamCache struct {
	ID          uint      `gorm:"primaryKey"`
	ChatID      int64     `gorm:"uniqueIndex:idx_ai_spam_cache_chat_hash,priority:1;index:idx_ai_spam_cache_created,priority:2;not null"`
	ContentHash string    `gorm:"size:64;uniqueIndex:idx_ai_spam_cache_chat_hash,priority:2;not null"`
	ResultJSON  string    `gorm:"type:text;not null"`
	CreatedAt   time.Time `gorm:"index:idx_ai_spam_cache_created,priority:1;autoCreateTime"`
}
