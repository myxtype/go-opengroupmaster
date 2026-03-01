package repository

import (
	"time"

	"supervisor/internal/model"

	"gorm.io/gorm"
)

func (r *Repository) AddPoints(groupID, userID uint, delta int) error {
	_, _, err := r.AdjustPoints(groupID, userID, delta)
	return err
}

func (r *Repository) AdjustPoints(groupID, userID uint, delta int) (int, int, error) {
	applied := 0
	current := 0
	err := r.db.Transaction(func(txTx *gorm.DB) error {
		up := &model.UserPoint{GroupID: groupID, UserID: userID}
		if err := txTx.Where("group_id = ? and user_id = ?", groupID, userID).FirstOrCreate(up).Error; err != nil {
			return err
		}
		next := up.Points + delta
		if next < 0 {
			next = 0
		}
		applied = next - up.Points
		up.Points = next
		current = next
		return txTx.Save(up).Error
	})
	return applied, current, err
}

func (r *Repository) UserPoints(groupID, userID uint) (int, error) {
	up := &model.UserPoint{GroupID: groupID, UserID: userID}
	if err := r.db.Where("group_id = ? and user_id = ?", groupID, userID).FirstOrCreate(up).Error; err != nil {
		return 0, err
	}
	return up.Points, nil
}

func (r *Repository) TopUsersByPoints(groupID uint, limit int) ([]model.UserPoint, error) {
	if limit <= 0 {
		limit = 10
	}
	out := make([]model.UserPoint, 0, limit)
	err := r.db.Where("group_id = ?", groupID).Order("points desc, user_id asc").Limit(limit).Find(&out).Error
	return out, err
}

type pointEventSum struct {
	Total int64
}

func (r *Repository) CreatePointEvent(event *model.PointEvent) error {
	if event == nil {
		return nil
	}
	return r.db.Create(event).Error
}

func (r *Repository) SumPointEventDeltaByDayAndType(groupID, userID uint, dayKey, eventType string) (int, error) {
	if dayKey == "" || eventType == "" {
		return 0, nil
	}
	var out pointEventSum
	err := r.db.Model(&model.PointEvent{}).
		Select("coalesce(sum(delta), 0) as total").
		Where("group_id = ? and user_id = ? and day_key = ? and type = ?", groupID, userID, dayKey, eventType).
		Scan(&out).Error
	return int(out.Total), err
}

func (r *Repository) ExistsPointEventByDayAndType(groupID, userID uint, dayKey, eventType string) (bool, error) {
	if dayKey == "" || eventType == "" {
		return false, nil
	}
	var total int64
	err := r.db.Model(&model.PointEvent{}).
		Where("group_id = ? and user_id = ? and day_key = ? and type = ?", groupID, userID, dayKey, eventType).
		Count(&total).Error
	return total > 0, err
}

// DeletePointEventsWithCreatedAtBefore 删除指定时间前的积分变动流水记录
// 涉及表：PointEvent - 积分变动流水（签到/发言/邀请/抽奖消耗/手动加减）
// 返回删除的记录数
func (r *Repository) DeletePointEventsWithCreatedAtBefore(cutoffTime time.Time) (int64, error) {
	result := r.db.Where("created_at < ?", cutoffTime).Delete(&model.PointEvent{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
