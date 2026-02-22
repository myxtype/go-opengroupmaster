package service

import (
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

func (s *Service) getWelcomeText(groupID uint) (string, error) {
	cfg := welcomeConfig{Text: "欢迎新成员加入，先看群规再发言。"}
	setting, err := s.repo.GetGroupSetting(groupID, featureWelcome)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg.Text, nil
		}
		return "", err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	if cfg.Text == "" {
		cfg.Text = "欢迎新成员加入，先看群规再发言。"
	}
	return cfg.Text, nil
}

func (s *Service) saveWelcomeText(groupID uint, text string) error {
	if text == "" {
		text = "欢迎新成员加入，先看群规再发言。"
	}
	b, err := json.Marshal(welcomeConfig{Text: text})
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureWelcome, string(b))
}

func (s *Service) getJoinVerifyConfig(groupID uint) (joinVerifyConfig, error) {
	cfg := joinVerifyConfig{Type: "button", TimeoutSec: 120}
	setting, err := s.repo.GetGroupSetting(groupID, featureJoinVerify)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	if cfg.Type != "math" {
		cfg.Type = "button"
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 120
	}
	return cfg, nil
}

func (s *Service) saveJoinVerifyConfig(groupID uint, cfg joinVerifyConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureJoinVerify, string(b))
}

func (s *Service) getSystemCleanConfig(groupID uint) (systemCleanConfig, error) {
	cfg := systemCleanConfig{
		Join:  true,
		Leave: true,
		Pin:   false,
		Photo: false,
		Title: false,
	}
	setting, err := s.repo.GetGroupSetting(groupID, featureSystemClean)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if saveErr := s.saveSystemCleanConfig(groupID, cfg); saveErr != nil {
				return cfg, saveErr
			}
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) saveSystemCleanConfig(groupID uint, cfg systemCleanConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureSystemClean, string(b))
}

func (s *Service) getKeywordMonitorConfig(groupID uint) (keywordMonitorConfig, error) {
	cfg := keywordMonitorConfig{Keywords: []string{}}
	setting, err := s.repo.GetGroupSetting(groupID, featureKeywordMonitor)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) saveKeywordMonitorConfig(groupID uint, cfg keywordMonitorConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureKeywordMonitor, string(b))
}

func (s *Service) getChainConfig(groupID uint) (chainConfig, error) {
	cfg := chainConfig{Active: false, Title: "", Entries: []string{}}
	setting, err := s.repo.GetGroupSetting(groupID, featureChain)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) saveChainConfig(groupID uint, cfg chainConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureChain, string(b))
}

func (s *Service) getPollMeta(groupID uint) (pollMeta, error) {
	cfg := pollMeta{}
	setting, err := s.repo.GetGroupSetting(groupID, featurePollMeta)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) savePollMeta(groupID uint, cfg pollMeta) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featurePollMeta, string(b))
}

func (s *Service) getRBACConfig(groupID uint) (rbacConfig, error) {
	cfg := rbacConfig{Roles: map[string]string{}, FeatureACL: map[string][]string{}}
	setting, err := s.repo.GetGroupSetting(groupID, featureRBAC)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	if cfg.Roles == nil {
		cfg.Roles = map[string]string{}
	}
	if cfg.FeatureACL == nil {
		cfg.FeatureACL = map[string][]string{}
	}
	return cfg, nil
}

func (s *Service) saveRBACConfig(groupID uint, cfg rbacConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureRBAC, string(b))
}

func (s *Service) getNewbieLimitMinutes(groupID uint) (int, error) {
	cfg := newbieLimitConfig{Minutes: 10}
	setting, err := s.repo.GetGroupSetting(groupID, featureNewbieLimit)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg.Minutes, nil
		}
		return 10, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	if cfg.Minutes <= 0 {
		cfg.Minutes = 10
	}
	return cfg.Minutes, nil
}

func (s *Service) saveNewbieLimitMinutes(groupID uint, minutes int) error {
	if minutes <= 0 {
		minutes = 10
	}
	b, err := json.Marshal(newbieLimitConfig{Minutes: minutes})
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureNewbieLimit, string(b))
}
func onOff(v bool) string {
	if v {
		return "开启"
	}
	return "关闭"
}
