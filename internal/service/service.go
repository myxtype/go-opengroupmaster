package service

import (
	"log"
	"sync"
	"time"

	"supervisor/internal/model"
	"supervisor/internal/repository"
)

const featureWelcome = "welcome"
const featureAntiSpam = "anti_spam"
const featureAntiFlood = "anti_flood"
const featureJoinVerify = "join_verify"
const featureNewbieLimit = "newbie_limit"
const featureSystemClean = "system_clean"
const featureKeywordMonitor = "keyword_monitor"
const featureChain = "chain"
const featurePollMeta = "poll_meta"
const featureRBAC = "rbac"

type verifyPending struct {
	Deadline time.Time
	Mode     string
	Answer   int
}

type floodEvent struct {
	Timestamp int64
	Text      string
}

type joinVerifyConfig struct {
	Type       string `json:"type"`
	TimeoutSec int    `json:"timeout_sec"`
}

type newbieLimitConfig struct {
	Minutes int `json:"minutes"`
}

type welcomeConfig struct {
	Text string `json:"text"`
}

type antiSpamConfig struct {
	WhitelistDomains []string `json:"whitelist_domains"`
}

type antiFloodConfig struct {
	WindowSec       int `json:"window_sec"`
	MaxMessages     int `json:"max_messages"`
	MuteSec         int `json:"mute_sec"`
	RepeatWindow    int `json:"repeat_window_sec"`
	RepeatThreshold int `json:"repeat_threshold"`
}

type systemCleanConfig struct {
	Join  bool `json:"join"`
	Leave bool `json:"leave"`
	Pin   bool `json:"pin"`
	Photo bool `json:"photo"`
	Title bool `json:"title"`
}

type keywordMonitorConfig struct {
	Keywords []string `json:"keywords"`
}

type chainConfig struct {
	Active  bool     `json:"active"`
	Title   string   `json:"title"`
	Entries []string `json:"entries"`
}

type pollMeta struct {
	Question  string `json:"question"`
	MessageID int    `json:"message_id"`
	Active    bool   `json:"active"`
}

type rbacConfig struct {
	Roles      map[string]string   `json:"roles"`
	FeatureACL map[string][]string `json:"feature_acl"`
}

type Service struct {
	repo            *repository.Repository
	logger          *log.Logger
	scheduleRuntime ScheduleRuntime
	mu              sync.Mutex
	flood           map[string][]floodEvent
	joinAt          map[string]time.Time
	verify          map[string]verifyPending
}

type AutoReplyPage struct {
	Items    []model.AutoReply
	Page     int
	PageSize int
	Total    int64
}

type BannedWordPage struct {
	Items    []model.BannedWord
	Page     int
	PageSize int
	Total    int64
}

type ScheduledMessagePage struct {
	Items    []model.ScheduledMessage
	Page     int
	PageSize int
	Total    int64
}

type GroupStats struct {
	GroupTitle string
	GroupID    int64
	TopUsers   []UserScore
}

type UserScore struct {
	DisplayName string
	Points      int
}

type LogPage struct {
	Items    []model.Log
	Page     int
	PageSize int
	Total    int64
}

type SystemCleanView struct {
	Join  bool
	Leave bool
	Pin   bool
	Photo bool
	Title bool
}

type ChainView struct {
	Active  bool
	Title   string
	Entries []string
}

type LotteryPanelView struct {
	ActiveID           uint
	ActiveTitle        string
	ActiveJoinKeyword  string
	ActiveWinnersCount int
	ActiveParticipants int64
	LatestID           uint
	LatestTitle        string
	LatestJoinKeyword  string
	LatestStatus       string
}

func New(repo *repository.Repository, logger *log.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
		flood:  make(map[string][]floodEvent),
		joinAt: make(map[string]time.Time),
		verify: make(map[string]verifyPending),
	}
}

type ScheduleRuntime interface {
	AddJob(job model.ScheduledMessage) error
}

func (s *Service) SetScheduleRuntime(runtime ScheduleRuntime) {
	s.scheduleRuntime = runtime
}
