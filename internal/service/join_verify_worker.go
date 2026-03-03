package service

import (
	"time"

	tgbot "github.com/go-telegram/bot"
)

const (
	joinVerifyBatchSize = 100 // 每次批量处理的待验证用户数量
)

// processDueJoinVerify 处理超时的进群验证任务
// 进群验证是持久化的（SQLite 表），支持重启恢复
// 逻辑：查询超时待处理任务 -> 标记为完成 -> 应用超时惩罚
func (s *Service) processDueJoinVerify(bot *tgbot.Bot) {
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
			// 原子性地删除并获取待处理记录
			popped, err := s.repo.DeleteDueJoinVerifyPendingByID(row.ID, row.Deadline)
			if err != nil {
				s.logger.Printf("pop join verify pending failed id=%d: %v", row.ID, err)
				continue
			}
			if !popped {
				continue
			}
			// 应用超时惩罚（禁言/踢出）
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

		// 批量处理完后继续循环，直到没有更多超时任务
		if len(rows) < joinVerifyBatchSize {
			return
		}
	}
}

// RunJoinVerifyTick 执行一次进群验证超时检查
// 由 scheduler 每 30 秒调用一次
func (s *Service) RunJoinVerifyTick(bot *tgbot.Bot) {
	if bot == nil {
		return
	}
	s.processDueJoinVerify(bot)
}
