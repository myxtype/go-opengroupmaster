package repository

import "supervisor/internal/model"

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

func (r *Repository) UpsertFeatureConfig(groupID uint, featureKey string, config string) error {
	setting := &model.GroupSetting{GroupID: groupID, FeatureKey: featureKey}
	if err := r.db.Where("group_id = ? and feature_key = ?", groupID, featureKey).FirstOrCreate(setting).Error; err != nil {
		return err
	}
	setting.Config = config
	return r.db.Save(setting).Error
}
