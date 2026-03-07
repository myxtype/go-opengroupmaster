package repository

import (
	"errors"

	"supervisor/internal/model"

	"gorm.io/gorm"
)

func (r *Repository) GetGroupSetting(groupID uint, featureKey string) (*model.GroupSetting, error) {
	var setting model.GroupSetting
	if err := r.db.Where("group_id = ? and feature_key = ?", groupID, featureKey).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *Repository) UpsertFeatureEnabled(groupID uint, featureKey string, enabled bool) error {
	setting := &model.GroupSetting{GroupID: groupID, FeatureKey: featureKey}
	if err := r.db.Where("group_id = ? and feature_key = ?", groupID, featureKey).FirstOrCreate(setting).Error; err != nil {
		return err
	}
	setting.Enabled = enabled
	return r.db.Save(setting).Error
}

func (r *Repository) UpsertFeatureConfig(groupID uint, featureKey string, config string, enabledOnCreate bool) error {
	setting := &model.GroupSetting{}
	err := r.db.Where("group_id = ? and feature_key = ?", groupID, featureKey).First(setting).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		setting = &model.GroupSetting{
			GroupID:    groupID,
			FeatureKey: featureKey,
			Enabled:    enabledOnCreate,
			Config:     config,
		}
		return r.db.Create(setting).Error
	}
	setting.Config = config
	return r.db.Save(setting).Error
}
