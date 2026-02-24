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

func (r *Repository) CreateDefaultDataIfEmpty(groupID uint) error {
	var count int64
	if err := r.db.Model(&model.AutoReply{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := r.db.Create(&model.AutoReply{GroupID: groupID, Keyword: "你好", Reply: "你好，我是 GroupMaster Bot", MatchType: "exact", ButtonRows: ""}).Error; err != nil {
			return err
		}
	}
	if err := r.db.Model(&model.BannedWord{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := r.db.Create(&model.BannedWord{GroupID: groupID, Word: "spam"}).Error; err != nil {
			return err
		}
	}
	return nil
}
