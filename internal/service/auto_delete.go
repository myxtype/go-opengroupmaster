package service

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	autoDeleteBatchSize   = 100
	autoDeleteRetryDelay  = time.Minute
	autoDeleteMaxAttempts = 5
)

func (s *Service) ScheduleMessageDelete(chatID int64, messageID int, delay time.Duration) {
	if messageID <= 0 || delay <= 0 {
		return
	}

	if err := s.repo.CreateAutoDeleteTask(chatID, messageID, time.Now().Add(delay)); err != nil {
		s.logger.Printf("create auto delete task failed chat=%d msg=%d: %v", chatID, messageID, err)
		return
	}
}

func (s *Service) processDueAutoDeleteTasks(bot *tgbotapi.BotAPI) {
	for {
		tasks, err := s.repo.ListDueAutoDeleteTasks(time.Now(), autoDeleteBatchSize)
		if err != nil {
			s.logger.Printf("list due auto delete tasks failed: %v", err)
			return
		}
		if len(tasks) == 0 {
			return
		}

		for _, task := range tasks {
			_, err := bot.Request(tgbotapi.NewDeleteMessage(task.ChatID, task.MessageID))
			if err != nil {
				attempt := task.Attempts + 1
				if attempt >= autoDeleteMaxAttempts {
					if delErr := s.repo.DeleteAutoDeleteTask(task.ID); delErr != nil {
						s.logger.Printf("drop auto delete task failed id=%d: %v", task.ID, delErr)
					}
					s.logger.Printf("drop auto delete task id=%d chat=%d msg=%d after %d attempts: %v", task.ID, task.ChatID, task.MessageID, attempt, err)
					continue
				}

				next := time.Now().Add(autoDeleteRetryDelay)
				if retryErr := s.repo.RetryAutoDeleteTask(task.ID, next); retryErr != nil {
					s.logger.Printf("retry auto delete task failed id=%d: %v", task.ID, retryErr)
				}
				s.logger.Printf("auto delete task failed id=%d chat=%d msg=%d, retry at %s: %v", task.ID, task.ChatID, task.MessageID, next.Format(time.RFC3339), err)
				continue
			}

			if delErr := s.repo.DeleteAutoDeleteTask(task.ID); delErr != nil {
				s.logger.Printf("delete auto delete task failed id=%d: %v", task.ID, delErr)
			}
		}

		if len(tasks) < autoDeleteBatchSize {
			return
		}
	}
}

// RunAutoDeleteTick executes one auto-delete maintenance cycle.
func (s *Service) RunAutoDeleteTick(bot *tgbotapi.BotAPI) {
	if bot == nil {
		return
	}
	s.processDueAutoDeleteTasks(bot)
}
