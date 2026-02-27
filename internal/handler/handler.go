package handler

import (
	"log"
	"sync"

	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
	Kind        string
	TGGroupID   int64
	TargetTGUID int64
	TargetLabel string
	RuleID      uint
	Page        int
	CronExpr    string
	Keyword     string
	MatchType   string
	Content     string
	RawButtons  string
	MediaType   string
	MediaFileID string
	Pin         bool
	ChainMode   string
	ChainID     uint
	Count       int
	Deadline    int64
}

type Handler struct {
	service *service.Service
	logger  *log.Logger

	mu      sync.Mutex
	pending map[int64]pendingInput
}

func New(svc *service.Service, logger *log.Logger) *Handler {
	return &Handler{service: svc, logger: logger, pending: make(map[int64]pendingInput)}
}

func (h *Handler) HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message != nil {
		h.handleMessage(bot, update.Message)
	}
	if update.ChatMember != nil {
		h.handleChatMemberUpdate(update.ChatMember)
	}
	if update.CallbackQuery != nil {
		h.handleCallback(bot, update.CallbackQuery)
	}
}
