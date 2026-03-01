package service

import (
	"time"
)

// CleanupWordCloudOldData 删除 3 个月前的词云统计数据
// 涉及表：
//   - WordCloudToken: 按群/日期/用户/词语的分词聚合记录
//   - WordCloudDailyUserStat: 按群/日期/用户的每日统计（发言数、贡献词数）
func (s *Service) CleanupWordCloudOldData() {
	cutoffDate := time.Now().AddDate(0, -3, 0) // 3 个月前
	deletedCount, err := s.repo.DeleteWordCloudDataOlderThan(cutoffDate)
	if err != nil && s.logger != nil {
		s.logger.Printf("word cloud cleanup failed: %v", err)
		return
	}
	if s.logger != nil {
		s.logger.Printf("word cloud cleanup completed: deleted data older than %s, affected rows: %d", cutoffDate.Format("2006-01-02"), deletedCount)
	}
}

// CleanupLogOldData 删除 6 个月前的操作审计日志
// 涉及表：
//   - Log: 群组操作审计日志（违禁词、反垃圾、反刷屏、积分等操作记录）
func (s *Service) CleanupLogOldData() {
	cutoffTime := time.Now().AddDate(0, -6, 0) // 6 个月前
	deletedCount, err := s.repo.DeleteLogsWithCreatedAtBefore(cutoffTime)
	if err != nil && s.logger != nil {
		s.logger.Printf("log cleanup failed: %v", err)
		return
	}
	if s.logger != nil {
		s.logger.Printf("log cleanup completed: deleted logs older than %s, affected rows: %d", cutoffTime.Format("2006-01-02 15:04:05"), deletedCount)
	}
}

// CleanupPointEventOldData 删除 1 年前的积分变动流水记录
// 涉及表：
//   - PointEvent: 积分变动流水（签到/发言/邀请/抽奖消耗/手动加减）
func (s *Service) CleanupPointEventOldData() {
	cutoffTime := time.Now().AddDate(-1, 0, 0) // 1 年前
	deletedCount, err := s.repo.DeletePointEventsWithCreatedAtBefore(cutoffTime)
	if err != nil && s.logger != nil {
		s.logger.Printf("point event cleanup failed: %v", err)
		return
	}
	if s.logger != nil {
		s.logger.Printf("point event cleanup completed: deleted events older than %s, affected rows: %d", cutoffTime.Format("2006-01-02 15:04:05"), deletedCount)
	}
}