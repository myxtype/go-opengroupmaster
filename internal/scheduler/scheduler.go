package scheduler

import (
	"log"
	"sync"

	"supervisor/internal/model"
	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron    *cron.Cron
	service *service.Service
	bot     *tgbotapi.BotAPI
	logger  *log.Logger
	mu      sync.Mutex
	entryID map[uint]cron.EntryID
}

func New(svc *service.Service, bot *tgbotapi.BotAPI, logger *log.Logger) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		service: svc,
		bot:     bot,
		logger:  logger,
		entryID: make(map[uint]cron.EntryID),
	}
}

func (s *Scheduler) Start() error {
	jobs, err := s.service.Repo().ListEnabledScheduledMessages()
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := s.AddJob(job); err != nil {
			s.logger.Printf("invalid cron expr %s: %v", job.CronExpr, err)
		}
	}
	// Run once on startup, then every 30 seconds.
	s.runMaintenanceTick()
	if _, err := s.cron.AddFunc("@every 30s", s.runMaintenanceTick); err != nil {
		return err
	}
	// Run daily maintenance tasks once on startup, then daily at 3:00 AM.
	s.runDailyMaintenanceTasks()
	if _, err := s.cron.AddFunc("0 3 * * *", s.runDailyMaintenanceTasks); err != nil {
		return err
	}
	s.cron.Start()
	return nil
}

func (s *Scheduler) AddJob(job model.ScheduledMessage) error {
	s.mu.Lock()
	if old, ok := s.entryID[job.ID]; ok {
		s.cron.Remove(old)
		delete(s.entryID, job.ID)
	}
	s.mu.Unlock()
	if !job.Enabled {
		return nil
	}

	j := job
	entry, err := s.cron.AddFunc(j.CronExpr, func() {
		group, err := s.service.Repo().FindGroupByID(j.GroupID)
		if err != nil || group == nil {
			return
		}
		var sent tgbotapi.Message
		switch j.MediaType {
		case "photo":
			out := tgbotapi.NewPhoto(group.TGGroupID, tgbotapi.FileID(j.MediaFileID))
			out.Caption = j.Content
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.Send(out)
		case "video":
			out := tgbotapi.NewVideo(group.TGGroupID, tgbotapi.FileID(j.MediaFileID))
			out.Caption = j.Content
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.Send(out)
		case "document":
			out := tgbotapi.NewDocument(group.TGGroupID, tgbotapi.FileID(j.MediaFileID))
			out.Caption = j.Content
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.Send(out)
		case "animation":
			out := tgbotapi.NewAnimation(group.TGGroupID, tgbotapi.FileID(j.MediaFileID))
			out.Caption = j.Content
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.Send(out)
		default:
			out := tgbotapi.NewMessage(group.TGGroupID, j.Content)
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.Send(out)
		}
		if j.PinMessage && sent.MessageID > 0 {
			_, _ = s.bot.Request(tgbotapi.PinChatMessageConfig{
				ChatID:              group.TGGroupID,
				MessageID:           sent.MessageID,
				DisableNotification: true,
			})
		}
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.entryID[job.ID] = entry
	s.mu.Unlock()
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// 内部维护任务，包括自动删除、群组验证、词云生成等。
func (s *Scheduler) runMaintenanceTick() {
	s.service.RunAutoDeleteTick(s.bot)
	s.service.RunJoinVerifyTick(s.bot)
	s.service.RunWordCloudTick(s.bot)
}

// runDailyMaintenanceTasks executes daily maintenance tasks.
func (s *Scheduler) runDailyMaintenanceTasks() {
	s.service.CleanupWordCloudOldData()
	s.service.CleanupLogOldData()
	s.service.CleanupPointEventOldData()
}
