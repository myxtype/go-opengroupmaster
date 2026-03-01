package service

import (
	"time"
)

// CleanupWordCloudOldData deletes wordcloud data older than 3 months.
func (s *Service) CleanupWordCloudOldData() {
	cutoffDate := time.Now().AddDate(0, -3, 0) // 3 months ago
	deletedCount, err := s.repo.DeleteWordCloudDataOlderThan(cutoffDate)
	if err != nil && s.logger != nil {
		s.logger.Printf("word cloud cleanup failed: %v", err)
		return
	}
	if s.logger != nil {
		s.logger.Printf("word cloud cleanup completed: deleted data older than %s, affected rows: %d", cutoffDate.Format("2006-01-02"), deletedCount)
	}
}

// CleanupLogOldData deletes log records older than 6 months.
func (s *Service) CleanupLogOldData() {
	cutoffTime := time.Now().AddDate(0, -6, 0) // 6 months ago
	deletedCount, err := s.repo.DeleteLogsWithCreatedAtBefore(cutoffTime)
	if err != nil && s.logger != nil {
		s.logger.Printf("log cleanup failed: %v", err)
		return
	}
	if s.logger != nil {
		s.logger.Printf("log cleanup completed: deleted logs older than %s, affected rows: %d", cutoffTime.Format("2006-01-02 15:04:05"), deletedCount)
	}
}

// CleanupPointEventOldData deletes point event records older than 1 year.
func (s *Service) CleanupPointEventOldData() {
	cutoffTime := time.Now().AddDate(-1, 0, 0) // 1 year ago
	deletedCount, err := s.repo.DeletePointEventsWithCreatedAtBefore(cutoffTime)
	if err != nil && s.logger != nil {
		s.logger.Printf("point event cleanup failed: %v", err)
		return
	}
	if s.logger != nil {
		s.logger.Printf("point event cleanup completed: deleted events older than %s, affected rows: %d", cutoffTime.Format("2006-01-02 15:04:05"), deletedCount)
	}
}