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
const featureNightMode = "night_mode"
const featureJoinVerify = "join_verify"
const featureNewbieLimit = "newbie_limit"
const featureSystemClean = "system_clean"
const featureKeywordMonitor = "keyword_monitor"
const featureChain = "chain"
const featurePollMeta = "poll_meta"
const featureRBAC = "rbac"
const featureLottery = "lottery"

const (
	antiFloodPenaltyWarn       = "warn"
	antiFloodPenaltyMute       = "mute"
	antiFloodPenaltyKick       = "kick"
	antiFloodPenaltyKickBan    = "kick_ban"
	antiFloodPenaltyDeleteOnly = "delete_only"
)

type verifyPending struct {
	Deadline      time.Time
	Mode          string
	Answer        string
	MessageID     int
	TimeoutAction string
}

type floodEvent struct {
	Timestamp int64
	Text      string
}

type joinVerifyConfig struct {
	Type           string `json:"type"`
	TimeoutSec     int    `json:"timeout_sec,omitempty"`
	TimeoutMinutes int    `json:"timeout_minutes"`
	TimeoutAction  string `json:"timeout_action"`
}

type newbieLimitConfig struct {
	Minutes int `json:"minutes"`
}

type welcomeConfig struct {
	Text          string            `json:"text"`
	Mode          string            `json:"mode"`
	DeleteMinutes int               `json:"delete_minutes"`
	MediaFileID   string            `json:"media_file_id"`
	ButtonRows    [][]welcomeButton `json:"button_rows,omitempty"`
}

type welcomeButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type antiSpamConfig struct {
	BlockPhoto              bool     `json:"block_photo"`
	BlockLink               bool     `json:"block_link"`
	SmartDetectEnabled      bool     `json:"smart_detect_enabled"`
	SmartDeleteScore        int      `json:"smart_delete_score"`
	SmartPunishScore        int      `json:"smart_punish_score"`
	BlockChannelAlias       bool     `json:"block_channel_alias"`
	BlockForwardFromChannel bool     `json:"block_forward_channel"`
	BlockForwardFromUser    bool     `json:"block_forward_user"`
	BlockAtGroupID          bool     `json:"block_at_group_id"`
	BlockAtUserID           bool     `json:"block_at_user_id"`
	BlockEthAddress         bool     `json:"block_eth_address"`
	BlockLongMessage        bool     `json:"block_long_message"`
	MaxMessageLength        int      `json:"max_message_length"`
	BlockLongName           bool     `json:"block_long_name"`
	MaxNameLength           int      `json:"max_name_length"`
	ExceptionKeywords       []string `json:"exception_keywords"`
	Penalty                 string   `json:"penalty"`
	MuteSec                 int      `json:"mute_sec"`
	WarnDeleteSec           int      `json:"warn_delete_sec"`
}

type antiFloodConfig struct {
	WindowSec       int    `json:"window_sec"`
	MaxMessages     int    `json:"max_messages"`
	Penalty         string `json:"penalty"`
	MuteSec         int    `json:"mute_sec"`
	WarnDeleteSec   int    `json:"warn_delete_sec"`
	RepeatWindow    int    `json:"repeat_window_sec"`
	RepeatThreshold int    `json:"repeat_threshold"`
}

type antiFloodState struct {
	Enabled bool
	Config  antiFloodConfig
}

type antiSpamState struct {
	Enabled bool
	Config  antiSpamConfig
}

type nightModeConfig struct {
	TimezoneOffsetMinutes int    `json:"timezone_offset_minutes"`
	Mode                  string `json:"mode"`
}

type nightModeState struct {
	Enabled bool
	Config  nightModeConfig
}

type featureConfigCacheEntry struct {
	Exists bool
	Config string
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

type lotteryConfig struct {
	PublishPin           bool `json:"publish_pin"`
	ResultPin            bool `json:"result_pin"`
	DeleteKeywordMinutes int  `json:"delete_keyword_minutes"`
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
	adminSyncMu     sync.Mutex
	adminSyncAt     map[int64]time.Time
	adminSyncEvery  time.Duration
	configCacheMu   sync.RWMutex
	configCache     map[string]featureConfigCacheEntry
	antiSpamMu      sync.RWMutex
	antiSpamCache   map[uint]antiSpamState
	antiFloodMu     sync.RWMutex
	antiFloodCache  map[uint]antiFloodState
	nightModeMu     sync.RWMutex
	nightModeCache  map[uint]nightModeState
	flood           map[string][]floodEvent
	spamRecent      map[string][]floodEvent
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

type JoinVerifyView struct {
	Enabled        bool
	Type           string
	TimeoutMinutes int
	TimeoutAction  string
}

type NewbieLimitView struct {
	Enabled bool
	Minutes int
}

type AntiFloodView struct {
	Enabled       bool
	WindowSec     int
	MaxMessages   int
	Penalty       string
	MuteSec       int
	WarnDeleteSec int
}

type AntiSpamView struct {
	Enabled               bool
	BlockPhoto            bool
	BlockLink             bool
	SmartDetectEnabled    bool
	SmartDeleteScore      int
	SmartPunishScore      int
	BlockChannelAlias     bool
	BlockForwardFromChan  bool
	BlockForwardFromUser  bool
	BlockAtGroupID        bool
	BlockAtUserID         bool
	BlockEthAddress       bool
	BlockLongMessage      bool
	MaxMessageLength      int
	BlockLongName         bool
	MaxNameLength         int
	ExceptionKeywordCount int
	ExceptionKeywords     []string
	Penalty               string
	MuteSec               int
	WarnDeleteSec         int
}

type NightModeView struct {
	Enabled      bool
	TimezoneText string
	Mode         string
	NightWindow  string
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
	PublishPin         bool
	ResultPin          bool
	DeleteKeywordMins  int
}

func New(repo *repository.Repository, logger *log.Logger) *Service {
	return &Service{
		repo:           repo,
		logger:         logger,
		configCache:    make(map[string]featureConfigCacheEntry),
		antiSpamCache:  make(map[uint]antiSpamState),
		antiFloodCache: make(map[uint]antiFloodState),
		nightModeCache: make(map[uint]nightModeState),
		flood:          make(map[string][]floodEvent),
		spamRecent:     make(map[string][]floodEvent),
		joinAt:         make(map[string]time.Time),
		verify:         make(map[string]verifyPending),
		adminSyncAt:    make(map[int64]time.Time),
		adminSyncEvery: 3 * time.Minute,
	}
}

type ScheduleRuntime interface {
	AddJob(job model.ScheduledMessage) error
}

func (s *Service) SetScheduleRuntime(runtime ScheduleRuntime) {
	s.scheduleRuntime = runtime
}

func (s *Service) SetAdminSyncInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	s.adminSyncMu.Lock()
	defer s.adminSyncMu.Unlock()
	s.adminSyncEvery = d
}
