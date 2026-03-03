package scheduler

import (
	"context"
	"log"
	"sync"

	"supervisor/internal/model"
	"supervisor/internal/service"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron    *cron.Cron
	service *service.Service
	bot     *tgbot.Bot
	logger  *log.Logger
	mu      sync.Mutex
	entryID map[uint]cron.EntryID
}

func New(svc *service.Service, bot *tgbot.Bot, logger *log.Logger) *Scheduler {
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
	// 高频内部维护任务
	if _, err := s.cron.AddFunc("@every 30s", s.runMaintenanceTick); err != nil {
		return err
	}
	// 每日维护任务
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
		ctx := context.Background()
		group, err := s.service.Repo().FindGroupByID(j.GroupID)
		if err != nil || group == nil {
			return
		}
		var sent *models.Message
		switch j.MediaType {
		case "photo":
			out := &tgbot.SendPhotoParams{
				ChatID:  group.TGGroupID,
				Photo:   &models.InputFileString{Data: j.MediaFileID},
				Caption: j.Content,
			}
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.SendPhoto(ctx, out)
		case "video":
			out := &tgbot.SendVideoParams{
				ChatID:  group.TGGroupID,
				Video:   &models.InputFileString{Data: j.MediaFileID},
				Caption: j.Content,
			}
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.SendVideo(ctx, out)
		case "document":
			out := &tgbot.SendDocumentParams{
				ChatID:   group.TGGroupID,
				Document: &models.InputFileString{Data: j.MediaFileID},
				Caption:  j.Content,
			}
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.SendDocument(ctx, out)
		case "animation":
			out := &tgbot.SendAnimationParams{
				ChatID:    group.TGGroupID,
				Animation: &models.InputFileString{Data: j.MediaFileID},
				Caption:   j.Content,
			}
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.SendAnimation(ctx, out)
		default:
			out := &tgbot.SendMessageParams{
				ChatID: group.TGGroupID,
				Text:   j.Content,
			}
			if markup, ok := service.InlineKeyboardFromButtonRowsJSON(j.ButtonRows); ok {
				out.ReplyMarkup = markup
			}
			sent, _ = s.bot.SendMessage(ctx, out)
		}
		if j.PinMessage && sent != nil && sent.ID > 0 {
			_, _ = s.bot.PinChatMessage(ctx, &tgbot.PinChatMessageParams{
				ChatID:              group.TGGroupID,
				MessageID:           sent.ID,
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
