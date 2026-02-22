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
	s.cron.Start()
	return nil
}

func (s *Scheduler) AddJob(job model.ScheduledMessage) error {
	if !job.Enabled {
		return nil
	}
	s.mu.Lock()
	if old, ok := s.entryID[job.ID]; ok {
		s.cron.Remove(old)
		delete(s.entryID, job.ID)
	}
	s.mu.Unlock()

	j := job
	entry, err := s.cron.AddFunc(j.CronExpr, func() {
		group, err := s.service.Repo().FindGroupByID(j.GroupID)
		if err != nil || group == nil {
			return
		}
		_, _ = s.bot.Send(tgbotapi.NewMessage(group.TGGroupID, j.Content))
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
