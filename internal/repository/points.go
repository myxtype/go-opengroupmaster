package repository

import (
	"errors"
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

type PointEventSummary struct {
	EventsTotal int64
	UsersTotal  int64
	DeltaTotal  int64
}

type UserPointsSummary struct {
	UsersTotal  int64
	PointsTotal int64
}

type PointEventUserTotal struct {
	UserID    uint
	TGUserID  int64
	Username  string
	FirstName string
	LastName  string
	Points    int64
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

func (r *Repository) SummarizeUserPoints(groupID uint) (UserPointsSummary, error) {
	var out UserPointsSummary
	err := r.db.Model(&model.UserPoint{}).
		Select("count(*) as users_total, coalesce(sum(points), 0) as points_total").
		Where("group_id = ? and points > 0", groupID).
		Scan(&out).Error
	return out, err
}

func (r *Repository) SummarizePointEvents(groupID uint, eventType, dayKey string) (PointEventSummary, error) {
	if eventType == "" {
		return PointEventSummary{}, errors.New("empty event type")
	}
	var out PointEventSummary
	query := r.db.Model(&model.PointEvent{}).
		Select("count(*) as events_total, count(distinct user_id) as users_total, coalesce(sum(delta), 0) as delta_total").
		Where("group_id = ? and type = ?", groupID, eventType)
	if dayKey != "" {
		query = query.Where("day_key = ?", dayKey)
	}
	if err := query.Scan(&out).Error; err != nil {
		return PointEventSummary{}, err
	}
	return out, nil
}

func (r *Repository) SummarizePointEventsSinceDay(groupID uint, eventType, sinceDayKey string) (PointEventSummary, error) {
	if eventType == "" {
		return PointEventSummary{}, errors.New("empty event type")
	}
	if sinceDayKey == "" {
		return PointEventSummary{}, errors.New("empty since day key")
	}
	var out PointEventSummary
	err := r.db.Model(&model.PointEvent{}).
		Select("count(*) as events_total, count(distinct user_id) as users_total, coalesce(sum(delta), 0) as delta_total").
		Where("group_id = ? and type = ? and day_key >= ?", groupID, eventType, sinceDayKey).
		Scan(&out).Error
	if err != nil {
		return PointEventSummary{}, err
	}
	return out, nil
}

func (r *Repository) SummarizePointEventsSinceTime(groupID uint, eventType string, since time.Time) (PointEventSummary, error) {
	if eventType == "" {
		return PointEventSummary{}, errors.New("empty event type")
	}
	if since.IsZero() {
		return PointEventSummary{}, errors.New("empty since time")
	}
	var out PointEventSummary
	err := r.db.Model(&model.PointEvent{}).
		Select("count(*) as events_total, count(distinct user_id) as users_total, coalesce(sum(delta), 0) as delta_total").
		Where("group_id = ? and type = ? and created_at >= ?", groupID, eventType, since).
		Scan(&out).Error
	if err != nil {
		return PointEventSummary{}, err
	}
	return out, nil
}

func (r *Repository) TopUsersByPointEventType(groupID uint, eventType string, limit int) ([]PointEventUserTotal, error) {
	if eventType == "" {
		return nil, errors.New("empty event type")
	}
	if limit <= 0 {
		limit = 10
	}
	out := make([]PointEventUserTotal, 0, limit)
	err := r.db.Table("point_events pe").
		Select(`
			pe.user_id as user_id,
			coalesce(u.tg_user_id, 0) as tg_user_id,
			coalesce(u.username, '') as username,
			coalesce(u.first_name, '') as first_name,
			coalesce(u.last_name, '') as last_name,
			coalesce(sum(pe.delta), 0) as points
		`).
		Joins("left join users u on u.id = pe.user_id").
		Where("pe.group_id = ? and pe.type = ?", groupID, eventType).
		Group("pe.user_id, u.tg_user_id, u.username, u.first_name, u.last_name").
		Order("points desc, pe.user_id asc").
		Limit(limit).
		Scan(&out).Error
	return out, err
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
