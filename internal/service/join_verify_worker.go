package service

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	joinVerifyBatchSize = 100
)

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
				Mode:          row.Mode,
				Answer:        row.Answer,
				FailCount:     row.FailCount,
				MessageID:     row.MessageID,
				TimeoutAction: row.TimeoutAction,
			})
		}

		if len(rows) < joinVerifyBatchSize {
			return
		}
	}
}

// RunJoinVerifyTick executes one join-verify timeout maintenance cycle.
func (s *Service) RunJoinVerifyTick(bot *tgbotapi.BotAPI) {
	if bot == nil {
		return
	}
	s.processDueJoinVerify(bot)
}
