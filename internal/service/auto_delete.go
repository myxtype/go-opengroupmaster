package service

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	autoDeleteBatchSize   = 100
	autoDeleteIdleWait    = time.Minute
	autoDeleteRetryDelay  = time.Minute
	autoDeleteMaxAttempts = 5
)

func (s *Service) StartAutoDeleteWorker(bot *tgbotapi.BotAPI) {
	if bot == nil {
		return
	}

	s.autoDeleteMu.Lock()
	if s.autoDeleteStop != nil {
		s.autoDeleteMu.Unlock()
		return
	}
	wake := make(chan struct{}, 1)
	stop := make(chan struct{})
	done := make(chan struct{})
	s.autoDeleteWake = wake
	s.autoDeleteStop = stop
	s.autoDeleteDone = done
	s.autoDeleteMu.Unlock()

	go s.runAutoDeleteWorker(bot, wake, stop, done)
	s.wakeAutoDeleteWorker()
}

func (s *Service) StopAutoDeleteWorker() {
	s.autoDeleteMu.Lock()
	stop := s.autoDeleteStop
	done := s.autoDeleteDone
	if stop == nil {
		s.autoDeleteMu.Unlock()
		return
	}
	s.autoDeleteWake = nil
	s.autoDeleteStop = nil
	s.autoDeleteDone = nil
	s.autoDeleteMu.Unlock()

	close(stop)
	<-done
}

func (s *Service) ScheduleMessageDelete(chatID int64, messageID int, delay time.Duration) {
	if messageID <= 0 || delay <= 0 {
		return
	}

	if err := s.repo.CreateAutoDeleteTask(chatID, messageID, time.Now().Add(delay)); err != nil {
		s.logger.Printf("create auto delete task failed chat=%d msg=%d: %v", chatID, messageID, err)
		return
	}
	s.wakeAutoDeleteWorker()
}

func (s *Service) runAutoDeleteWorker(bot *tgbotapi.BotAPI, wake <-chan struct{}, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	for {
		s.processDueAutoDeleteTasks(bot)
		wait := s.nextAutoDeleteWait()
		timer := time.NewTimer(wait)
		select {
		case <-stop:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-wake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
		}
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

func (s *Service) nextAutoDeleteWait() time.Duration {
	nextAt, ok, err := s.repo.NextAutoDeleteTaskTime()
	if err != nil {
		s.logger.Printf("query next auto delete task failed: %v", err)
		return autoDeleteIdleWait
	}
	if !ok {
		return autoDeleteIdleWait
	}
	wait := time.Until(nextAt)
	if wait < 0 {
		return 0
	}
	return wait
}

func (s *Service) wakeAutoDeleteWorker() {
	s.autoDeleteMu.Lock()
	wake := s.autoDeleteWake
	s.autoDeleteMu.Unlock()
	if wake == nil {
		return
	}
	select {
	case wake <- struct{}{}:
	default:
	}
}
