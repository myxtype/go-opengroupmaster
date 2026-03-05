package handler

import (
	"log"
	"sync"

	"supervisor/internal/service"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	groupPageSize  = 6
	rulesPageSize  = 5
	cbMenuGroups   = "menu:groups"
	cbMenuSettings = "menu:settings"
	cbGroupPrefix  = "group:"
	cbFeaturePref  = "feat:"
	cbGroupsPagePF = "menu:groups:page:"
	cbVerifyPF     = "verify:"
)

type renderTarget struct {
	ChatID    int64
	MessageID int
	Edit      bool
}

type pendingInput struct {
	Kind           string
	TGGroupID      int64
	TargetTGUID    int64
	TargetLabel    string
	RuleID         uint
	Page           int
	CronExpr       string
	Keyword        string
	MatchType      string
	Content        string
	RawButtons     string
	MediaType      string
	MediaFileID    string
	Pin            bool
	ChainMode      string
	ChainID        uint
	Count          int
	Deadline       int64
	PollQuestion   string
	PollOptions    []string
	LotteryTitle   string
	LotteryWinners int
}

type Handler struct {
	service     *service.Service
	logger      *log.Logger
	botUsername string
	botName     string

	mu      sync.Mutex
	pending map[int64]pendingInput
}

func New(svc *service.Service, logger *log.Logger) *Handler {
	return &Handler{service: svc, logger: logger, pending: make(map[int64]pendingInput)}
}

func (h *Handler) SetBotUsername(username string) {
	h.botUsername = username
}

func (h *Handler) SetBotName(name string) {
	h.botName = name
}

func (h *Handler) HandleUpdate(bot *tgbot.Bot, update *models.Update) {
	if update.Message != nil {
		h.handleMessage(bot, update.Message)
	}
	if update.EditedMessage != nil {
		h.handleEditedMessage(bot, update.EditedMessage)
	}
	if update.ChatMember != nil {
		h.handleChatMemberUpdate(update.ChatMember)
	}
	if update.CallbackQuery != nil {
		h.handleCallback(bot, update.CallbackQuery)
	}
}
