package repository

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"supervisor/internal/model"
)

func (r *Repository) CreateLog(groupID uint, action string, operatorID, targetID uint) error {
	if action == "" {
		return fmt.Errorf("action is required")
	}
	l := &model.Log{GroupID: groupID, Action: action, OperatorID: operatorID, TargetID: targetID}
	return r.db.Create(l).Error
}

func (r *Repository) ListLogsPage(groupID uint, page, pageSize int, action string) ([]model.Log, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	q := r.db.Model(&model.Log{}).Where("group_id = ?", groupID)
	q = applyLogActionFilter(q, action)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	out := make([]model.Log, 0, pageSize)
	err := q.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&out).Error
	return out, total, err
}

func (r *Repository) ListLogsForExport(groupID uint, action string, limit int) ([]model.Log, error) {
	if limit <= 0 {
		limit = 1000
	}
	q := r.db.Where("group_id = ?", groupID)
	q = applyLogActionFilter(q, action)
	out := make([]model.Log, 0, limit)
	err := q.Order("id desc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) CountBannedWordWarnsSinceLastAction(groupID, targetID uint) (int64, error) {
	var total int64
	sub := r.db.Model(&model.Log{}).
		Select("COALESCE(MAX(id), 0)").
		Where("group_id = ? and target_id = ? and action = ?", groupID, targetID, "banned_word_warn_action_applied")
	err := r.db.Model(&model.Log{}).
		Where("group_id = ? and target_id = ? and action = ?", groupID, targetID, "banned_word_warn").
		Where("id > (?)", sub).
		Count(&total).Error
	return total, err
}

func (r *Repository) CountAntiSpamWarnsSinceLastAction(groupID, targetID uint) (int64, error) {
	var total int64
	sub := r.db.Model(&model.Log{}).
		Select("COALESCE(MAX(id), 0)").
		Where("group_id = ? and target_id = ? and action = ?", groupID, targetID, "anti_spam_warn_action_applied")
	err := r.db.Model(&model.Log{}).
		Where("group_id = ? and target_id = ? and action = ?", groupID, targetID, "anti_spam_warn").
		Where("id > (?)", sub).
		Count(&total).Error
	return total, err
}

func (r *Repository) CountAntiFloodWarnsSinceLastAction(groupID, targetID uint) (int64, error) {
	var total int64
	sub := r.db.Model(&model.Log{}).
		Select("COALESCE(MAX(id), 0)").
		Where("group_id = ? and target_id = ? and action = ?", groupID, targetID, "anti_flood_warn_action_applied")
	err := r.db.Model(&model.Log{}).
		Where("group_id = ? and target_id = ? and action = ?", groupID, targetID, "anti_flood_warn").
		Where("id > (?)", sub).
		Count(&total).Error
	return total, err
}

func applyLogActionFilter(q *gorm.DB, action string) *gorm.DB {
	if action == "" || action == "all" {
		return q
	}
	if strings.HasSuffix(action, "*") {
		prefix := strings.TrimSuffix(action, "*")
		return q.Where("action LIKE ?", prefix+"%")
	}
	return q.Where("action = ?", action)
}

// DeleteLogsWithCreatedAtBefore 删除指定时间前的操作审计日志
// 涉及表：Log - 群组操作审计日志
// 返回删除的记录数
func (r *Repository) DeleteLogsWithCreatedAtBefore(cutoffTime time.Time) (int64, error) {
	result := r.db.Where("created_at < ?", cutoffTime).Delete(&model.Log{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
