package service

import "fmt"

func (s *Service) JoinVerifyViewByTGGroupID(tgGroupID int64) (*JoinVerifyView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureJoinVerify, false)
	if err != nil {
		return nil, err
	}
	cfg, err := s.getJoinVerifyConfig(group.ID)
	if err != nil {
		return nil, err
	}
	return &JoinVerifyView{
		Enabled:        enabled,
		Type:           cfg.Type,
		TimeoutMinutes: cfg.TimeoutMinutes,
		TimeoutAction:  cfg.TimeoutAction,
	}, nil
}

func (s *Service) SetJoinVerifyEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	if err := s.repo.UpsertFeatureEnabled(group.ID, featureJoinVerify, enabled); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_join_verify_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) CycleJoinVerifyTimeoutMinutesByTGGroupID(tgGroupID int64) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getJoinVerifyConfig(group.ID)
	if err != nil {
		return 0, err
	}
	switch cfg.TimeoutMinutes {
	case 1:
		cfg.TimeoutMinutes = 5
	case 5:
		cfg.TimeoutMinutes = 10
	default:
		cfg.TimeoutMinutes = 1
	}
	if err := s.saveJoinVerifyConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_join_verify_timeout_minutes_%d", cfg.TimeoutMinutes), 0, 0)
	return cfg.TimeoutMinutes, nil
}

func (s *Service) ToggleJoinVerifyTimeoutActionByTGGroupID(tgGroupID int64) (string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	cfg, err := s.getJoinVerifyConfig(group.ID)
	if err != nil {
		return "", err
	}
	if cfg.TimeoutAction == "kick" {
		cfg.TimeoutAction = "mute"
	} else {
		cfg.TimeoutAction = "kick"
	}
	if err := s.saveJoinVerifyConfig(group.ID, cfg); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_join_verify_timeout_action_"+cfg.TimeoutAction, 0, 0)
	return cfg.TimeoutAction, nil
}

func (s *Service) SetJoinVerifyTypeByTGGroupID(tgGroupID int64, verifyType string) (string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	cfg, err := s.getJoinVerifyConfig(group.ID)
	if err != nil {
		return "", err
	}
	switch verifyType {
	case "button", "math", "captcha", "zhchar":
		cfg.Type = verifyType
	default:
		cfg.Type = "button"
	}
	if err := s.saveJoinVerifyConfig(group.ID, cfg); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_join_verify_type_"+cfg.Type, 0, 0)
	return cfg.Type, nil
}

func (s *Service) CycleJoinVerifyTypeByTGGroupID(tgGroupID int64) (string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	cfg, err := s.getJoinVerifyConfig(group.ID)
	if err != nil {
		return "", err
	}
	switch cfg.Type {
	case "button":
		cfg.Type = "math"
	case "math":
		cfg.Type = "captcha"
	case "captcha":
		cfg.Type = "zhchar"
	default:
		cfg.Type = "button"
	}
	if err := s.saveJoinVerifyConfig(group.ID, cfg); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "switch_verify_type_"+cfg.Type, 0, 0)
	return cfg.Type, nil
}
