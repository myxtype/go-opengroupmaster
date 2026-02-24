package repository

import (
	"errors"
	"fmt"
	"time"

	"supervisor/internal/model"

	"gorm.io/gorm"
)

func (r *Repository) CreateInviteLink(item *model.InviteLink) error {
	if item == nil {
		return fmt.Errorf("nil invite link")
	}
	return r.db.Create(item).Error
}

func (r *Repository) CountInviteLinks(groupID uint) (int64, error) {
	var total int64
	err := r.db.Model(&model.InviteLink{}).Where("group_id = ?", groupID).Count(&total).Error
	return total, err
}

func (r *Repository) CountInviteLinksByCreator(groupID uint, creatorTGUserID int64) (int64, error) {
	var total int64
	err := r.db.Model(&model.InviteLink{}).
		Where("group_id = ? AND creator_tg_user_id = ?", groupID, creatorTGUserID).
		Count(&total).Error
	return total, err
}

func (r *Repository) FindInviteLinkByLink(groupID uint, link string) (*model.InviteLink, error) {
	var out model.InviteLink
	if err := r.db.Where("group_id = ? AND link = ?", groupID, link).First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *Repository) CreateInviteEvent(item *model.InviteEvent) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("nil invite event")
	}
	if item.JoinedAt.IsZero() {
		item.JoinedAt = time.Now()
	}
	tx := r.db.Where("group_id = ? AND invitee_tg_user_id = ?", item.GroupID, item.InviteeTGUserID).FirstOrCreate(item)
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (r *Repository) CountInviteEvents(groupID uint) (int64, error) {
	var total int64
	err := r.db.Model(&model.InviteEvent{}).Where("group_id = ?", groupID).Count(&total).Error
	return total, err
}

func (r *Repository) CountInviteEventsByInviter(groupID uint, inviterTGUserID int64) (int64, error) {
	var total int64
	err := r.db.Model(&model.InviteEvent{}).
		Where("group_id = ? AND inviter_tg_user_id = ?", groupID, inviterTGUserID).
		Count(&total).Error
	return total, err
}

func (r *Repository) ListInviteEventsForExport(groupID uint, limit int) ([]model.InviteEvent, error) {
	if limit <= 0 {
		limit = 5000
	}
	out := make([]model.InviteEvent, 0, limit)
	err := r.db.Where("group_id = ?", groupID).
		Order("joined_at desc").
		Limit(limit).
		Find(&out).Error
	return out, err
}

func (r *Repository) MarkGroupMemberFirstJoin(groupID uint, tgUserID int64, joinedAt time.Time) (bool, error) {
	if joinedAt.IsZero() {
		joinedAt = time.Now()
	}
	rec := model.GroupMemberJoin{
		GroupID:     groupID,
		TGUserID:    tgUserID,
		FirstJoinAt: joinedAt,
	}
	tx := r.db.Where("group_id = ? AND tg_user_id = ?", groupID, tgUserID).FirstOrCreate(&rec)
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (r *Repository) ClearInviteData(groupID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&model.InviteEvent{}).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Where("group_id = ?", groupID).Delete(&model.InviteLink{}).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return nil
	})
}
