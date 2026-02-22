package scheduler

import (
	"log"

	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron    *cron.Cron
	service *service.Service
	bot     *tgbotapi.BotAPI
	logger  *log.Logger
}

func New(svc *service.Service, bot *tgbotapi.BotAPI, logger *log.Logger) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		service: svc,
		bot:     bot,
		logger:  logger,
	}
}

func (s *Scheduler) Start() error {
	jobs, err := s.service.Repo().ListEnabledScheduledMessages()
	if err != nil {
		return err
	}
	for _, job := range jobs {
		j := job
		_, err := s.cron.AddFunc(j.CronExpr, func() {
			group, err := s.service.Repo().FindGroupByID(j.GroupID)
			if err != nil || group == nil {
				return
			}
			_, _ = s.bot.Send(tgbotapi.NewMessage(group.TGGroupID, j.Content))
		})
		if err != nil {
			s.logger.Printf("invalid cron expr %s: %v", j.CronExpr, err)
		}
	}
	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}
