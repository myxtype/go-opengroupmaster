package repository

import (
	"time"

	"supervisor/internal/model"

	"gorm.io/gorm"
)

func (r *Repository) CreateAutoDeleteTask(chatID int64, messageID int, executeAt time.Time) error {
	task := &model.AutoDeleteTask{
		ChatID:    chatID,
		MessageID: messageID,
		ExecuteAt: executeAt,
	}
	return r.db.Create(task).Error
}

func (r *Repository) ListDueAutoDeleteTasks(now time.Time, limit int) ([]model.AutoDeleteTask, error) {
	if limit <= 0 {
		limit = 100
	}
	out := make([]model.AutoDeleteTask, 0, limit)
	err := r.db.Where("execute_at <= ?", now).
		Order("execute_at asc, id asc").
		Limit(limit).
		Find(&out).Error
	return out, err
}

func (r *Repository) NextAutoDeleteTaskTime() (time.Time, bool, error) {
	var task model.AutoDeleteTask
	err := r.db.Order("execute_at asc, id asc").Take(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return task.ExecuteAt, true, nil
}

func (r *Repository) DeleteAutoDeleteTask(id uint) error {
	return r.db.Delete(&model.AutoDeleteTask{}, id).Error
}

func (r *Repository) RetryAutoDeleteTask(id uint, nextExecuteAt time.Time) error {
	return r.db.Model(&model.AutoDeleteTask{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"execute_at": nextExecuteAt,
			"attempts":   gorm.Expr("attempts + 1"),
		}).Error
}
