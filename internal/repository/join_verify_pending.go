package repository

import (
	"errors"
	"time"

	"supervisor/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) UpsertJoinVerifyPending(pending *model.JoinVerifyPending) error {
	if pending == nil {
		return errors.New("pending is nil")
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tg_group_id"},
			{Name: "tg_user_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"mode",
			"answer",
			"message_id",
			"timeout_action",
			"deadline",
			"restrict_until",
			"updated_at",
		}),
	}).Create(pending).Error
}

func (r *Repository) GetJoinVerifyPending(tgGroupID, tgUserID int64) (*model.JoinVerifyPending, error) {
	var pending model.JoinVerifyPending
	if err := r.db.Where("tg_group_id = ? AND tg_user_id = ?", tgGroupID, tgUserID).First(&pending).Error; err != nil {
		return nil, err
	}
	return &pending, nil
}

func (r *Repository) DeleteJoinVerifyPendingByID(id uint) (bool, error) {
	if id == 0 {
		return false, nil
	}
	tx := r.db.Delete(&model.JoinVerifyPending{}, id)
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (r *Repository) DeleteDueJoinVerifyPendingByID(id uint, deadline time.Time) (bool, error) {
	if id == 0 {
		return false, nil
	}
	tx := r.db.Where("id = ? AND deadline <= ?", id, deadline).Delete(&model.JoinVerifyPending{})
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (r *Repository) ListDueJoinVerifyPendings(now time.Time, limit int) ([]model.JoinVerifyPending, error) {
	if limit <= 0 {
		limit = 100
	}
	out := make([]model.JoinVerifyPending, 0, limit)
	err := r.db.Where("deadline <= ?", now).
		Order("deadline ASC, id ASC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

func (r *Repository) NextJoinVerifyPendingDeadline() (time.Time, bool, error) {
	var pending model.JoinVerifyPending
	err := r.db.Order("deadline ASC, id ASC").Take(&pending).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return pending.Deadline, true, nil
}
