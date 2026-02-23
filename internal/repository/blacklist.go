package repository

import "supervisor/internal/model"

func (r *Repository) AddGroupBlacklist(groupID uint, tgUserID int64, reason string) error {
	item := &model.GroupBlacklist{GroupID: groupID, TGUserID: tgUserID}
	if err := r.db.Where("group_id = ? AND tg_user_id = ?", groupID, tgUserID).FirstOrCreate(item).Error; err != nil {
		return err
	}
	if reason != "" && item.Reason != reason {
		item.Reason = reason
		return r.db.Save(item).Error
	}
	return nil
}

func (r *Repository) RemoveGroupBlacklist(groupID uint, tgUserID int64) error {
	return r.db.Where("group_id = ? AND tg_user_id = ?", groupID, tgUserID).Delete(&model.GroupBlacklist{}).Error
}

func (r *Repository) IsGroupBlacklisted(groupID uint, tgUserID int64) (bool, error) {
	var count int64
	err := r.db.Model(&model.GroupBlacklist{}).Where("group_id = ? AND tg_user_id = ?", groupID, tgUserID).Count(&count).Error
	return count > 0, err
}

func (r *Repository) ListGroupBlacklist(groupID uint) ([]model.GroupBlacklist, error) {
	out := make([]model.GroupBlacklist, 0)
	err := r.db.Where("group_id = ?", groupID).Order("id desc").Find(&out).Error
	return out, err
}
