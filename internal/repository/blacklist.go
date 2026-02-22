package repository

import "supervisor/internal/model"

func (r *Repository) AddGlobalBlacklist(tgUserID int64, reason string) error {
	item := &model.GlobalBlacklist{TGUserID: tgUserID, Reason: reason}
	return r.db.Where("tg_user_id = ?", tgUserID).FirstOrCreate(item).Error
}

func (r *Repository) RemoveGlobalBlacklist(tgUserID int64) error {
	return r.db.Where("tg_user_id = ?", tgUserID).Delete(&model.GlobalBlacklist{}).Error
}

func (r *Repository) IsGlobalBlacklisted(tgUserID int64) (bool, error) {
	var count int64
	err := r.db.Model(&model.GlobalBlacklist{}).Where("tg_user_id = ?", tgUserID).Count(&count).Error
	return count > 0, err
}

func (r *Repository) ListGlobalBlacklist() ([]model.GlobalBlacklist, error) {
	out := make([]model.GlobalBlacklist, 0)
	err := r.db.Order("id desc").Find(&out).Error
	return out, err
}
