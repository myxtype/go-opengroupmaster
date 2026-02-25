package repository

import (
	"errors"
	"time"

	"supervisor/internal/model"

	"gorm.io/gorm"
)

func (r *Repository) CreateChain(groupID uint, intro string, maxParticipants int, deadlineUnix int64) (*model.Chain, error) {
	item := &model.Chain{
		GroupID:               groupID,
		Intro:                 intro,
		MaxParticipants:       maxParticipants,
		DeadlineUnix:          deadlineUnix,
		AnnouncementMessageID: 0,
		Status:                "active",
	}
	if err := r.db.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repository) GetActiveChain(groupID uint) (*model.Chain, error) {
	var item model.Chain
	if err := r.db.Where("group_id = ? and status = ?", groupID, "active").Order("id desc").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetLatestChain(groupID uint) (*model.Chain, error) {
	var item model.Chain
	if err := r.db.Where("group_id = ?", groupID).Order("id desc").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetChainByID(chainID uint) (*model.Chain, error) {
	var item model.Chain
	if err := r.db.Where("id = ?", chainID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) ListActiveChains(groupID uint, limit int) ([]model.Chain, error) {
	if limit <= 0 {
		limit = 10
	}
	out := make([]model.Chain, 0, limit)
	err := r.db.Where("group_id = ? and status = ?", groupID, "active").Order("id desc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) CloseChain(chainID uint) error {
	return r.db.Model(&model.Chain{}).Where("id = ?", chainID).Updates(map[string]any{
		"status":     "closed",
		"updated_at": time.Now(),
	}).Error
}

func (r *Repository) CloseAllActiveChains(groupID uint) error {
	return r.db.Model(&model.Chain{}).Where("group_id = ? and status = ?", groupID, "active").Updates(map[string]any{
		"status":     "closed",
		"updated_at": time.Now(),
	}).Error
}

func (r *Repository) SetChainAnnouncementMessageID(chainID uint, messageID int) error {
	return r.db.Model(&model.Chain{}).Where("id = ?", chainID).Update("announcement_message_id", messageID).Error
}

func (r *Repository) CountChainEntries(chainID uint) (int64, error) {
	var total int64
	err := r.db.Model(&model.ChainEntry{}).Where("chain_id = ?", chainID).Count(&total).Error
	return total, err
}

func (r *Repository) ListChainEntries(chainID uint) ([]model.ChainEntry, error) {
	out := make([]model.ChainEntry, 0)
	err := r.db.Where("chain_id = ?", chainID).Order("id asc").Find(&out).Error
	return out, err
}

func (r *Repository) ListChainEntriesForExport(chainID uint, limit int) ([]model.ChainEntry, error) {
	if limit <= 0 {
		limit = 5000
	}
	out := make([]model.ChainEntry, 0, limit)
	err := r.db.Where("chain_id = ?", chainID).Order("id asc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) GetChainEntry(chainID uint, tgUserID int64) (*model.ChainEntry, error) {
	var item model.ChainEntry
	if err := r.db.Where("chain_id = ? and tg_user_id = ?", chainID, tgUserID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpsertChainEntry(chainID uint, tgUserID int64, displayName, content string) (bool, error) {
	var existed model.ChainEntry
	err := r.db.Where("chain_id = ? and tg_user_id = ?", chainID, tgUserID).First(&existed).Error
	if err == nil {
		existed.DisplayName = displayName
		existed.Content = content
		return false, r.db.Save(&existed).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	created := &model.ChainEntry{
		ChainID:     chainID,
		TGUserID:    tgUserID,
		DisplayName: displayName,
		Content:     content,
	}
	if err := r.db.Create(created).Error; err != nil {
		return false, err
	}
	return true, nil
}
