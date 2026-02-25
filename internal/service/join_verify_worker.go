package service

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	joinVerifyBatchSize = 100
	joinVerifyIdleWait  = time.Minute
)

func (s *Service) StartJoinVerifyWorker(bot *tgbotapi.BotAPI) {
	if bot == nil {
		return
	}

	s.joinVerifyMu.Lock()
	if s.joinVerifyStop != nil {
		s.joinVerifyMu.Unlock()
		return
	}
	wake := make(chan struct{}, 1)
	stop := make(chan struct{})
	done := make(chan struct{})
	s.joinVerifyWake = wake
	s.joinVerifyStop = stop
	s.joinVerifyDone = done
	s.joinVerifyMu.Unlock()

	go s.runJoinVerifyWorker(bot, wake, stop, done)
	s.wakeJoinVerifyWorker()
}

func (s *Service) StopJoinVerifyWorker() {
	s.joinVerifyMu.Lock()
	stop := s.joinVerifyStop
	done := s.joinVerifyDone
	if stop == nil {
		s.joinVerifyMu.Unlock()
		return
	}
	s.joinVerifyWake = nil
	s.joinVerifyStop = nil
	s.joinVerifyDone = nil
	s.joinVerifyMu.Unlock()

	close(stop)
	<-done
}

func (s *Service) runJoinVerifyWorker(bot *tgbotapi.BotAPI, wake <-chan struct{}, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	for {
		s.processDueJoinVerify(bot)
		wait := s.nextJoinVerifyWait()
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

func (s *Service) processDueJoinVerify(bot *tgbotapi.BotAPI) {
	for {
		rows, err := s.repo.ListDueJoinVerifyPendings(time.Now(), joinVerifyBatchSize)
		if err != nil {
			s.logger.Printf("list due join verify pendings failed: %v", err)
			return
		}
		if len(rows) == 0 {
			return
		}

		for _, row := range rows {
			popped, err := s.repo.DeleteDueJoinVerifyPendingByID(row.ID, row.Deadline)
			if err != nil {
				s.logger.Printf("pop join verify pending failed id=%d: %v", row.ID, err)
				continue
			}
			if !popped {
				continue
			}
			s.applyVerifyTimeout(bot, verifyPending{
				ID:            row.ID,
				TGGroupID:     row.TGGroupID,
				TGUserID:      row.TGUserID,
				Deadline:      row.Deadline,
				RestrictUntil: row.RestrictUntil,
				Mode:          row.Mode,
				Answer:        row.Answer,
				MessageID:     row.MessageID,
				TimeoutAction: row.TimeoutAction,
			})
		}

		if len(rows) < joinVerifyBatchSize {
			return
		}
	}
}

func (s *Service) nextJoinVerifyWait() time.Duration {
	nextAt, ok, err := s.repo.NextJoinVerifyPendingDeadline()
	if err != nil {
		s.logger.Printf("query next join verify deadline failed: %v", err)
		return joinVerifyIdleWait
	}
	if !ok {
		return joinVerifyIdleWait
	}
	wait := time.Until(nextAt)
	if wait < 0 {
		return 0
	}
	return wait
}

func (s *Service) wakeJoinVerifyWorker() {
	s.joinVerifyMu.Lock()
	wake := s.joinVerifyWake
	s.joinVerifyMu.Unlock()
	if wake == nil {
		return
	}
	select {
	case wake <- struct{}{}:
	default:
	}
}
