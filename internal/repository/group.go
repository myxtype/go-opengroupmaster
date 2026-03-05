package repository

import (
	"errors"

	"supervisor/internal/model"

	"github.com/go-telegram/bot/models"
	"gorm.io/gorm"
)

func (r *Repository) UpsertGroup(chat *models.Chat) (*model.Group, error) {
	if chat == nil {
		return nil, errors.New("nil chat")
	}
	g := &model.Group{TGGroupID: chat.ID}
	if err := r.db.Where(&model.Group{TGGroupID: chat.ID}).FirstOrCreate(g).Error; err != nil {
		return nil, err
	}
	g.Title = chat.Title
	g.BotAdded = true
	if err := r.db.Save(g).Error; err != nil {
		return nil, err
	}
	return g, nil
}

func (r *Repository) ReplaceGroupAdmins(groupID uint, admins []model.GroupAdmin) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&model.GroupAdmin{}).Error; err != nil {
			return err
		}
		if len(admins) == 0 {
			return nil
		}
		return tx.Create(&admins).Error
	})
}

func (r *Repository) ListGroupsByAdminTGUserID(tgUserID int64) ([]model.Group, error) {
	var groups []model.Group
	err := r.db.
		Table("groups").
		Select("groups.*").
		Joins("join group_admins on group_admins.group_id = groups.id").
		Joins("join users on users.id = group_admins.user_id").
		Where("users.tg_user_id = ?", tgUserID).
		Scan(&groups).Error
	return groups, err
}

func (r *Repository) FindGroupByTGID(tgGroupID int64) (*model.Group, error) {
	var g model.Group
	if err := r.db.Where("tg_group_id = ?", tgGroupID).First(&g).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repository) FindGroupByID(groupID uint) (*model.Group, error) {
	var g model.Group
	if err := r.db.First(&g, groupID).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repository) UpdateGroupTimezoneOffsetMinutes(groupID uint, offsetMinutes int) error {
	return r.db.Model(&model.Group{}).
		Where("id = ?", groupID).
		Update("timezone_offset_minutes", offsetMinutes).Error
}

func (r *Repository) CheckAdmin(groupID uint, tgUserID int64) (bool, error) {
	var count int64
	err := r.db.
		Table("group_admins").
		Joins("join users on users.id = group_admins.user_id").
		Where("group_admins.group_id = ? and users.tg_user_id = ?", groupID, tgUserID).
		Count(&count).Error
	return count > 0, err
}

func (r *Repository) ListAdminTGUserIDsByGroupID(groupID uint) ([]int64, error) {
	rows := make([]struct {
		TGUserID int64
	}, 0)
	err := r.db.
		Table("group_admins").
		Select("users.tg_user_id as tg_user_id").
		Joins("join users on users.id = group_admins.user_id").
		Where("group_admins.group_id = ?", groupID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.TGUserID)
	}
	return out, nil
}
