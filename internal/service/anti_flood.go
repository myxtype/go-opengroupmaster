package service

import (
	"fmt"
)

func (s *Service) AntiFloodViewByTGGroupID(tgGroupID int64) (*AntiFloodView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return nil, err
	}
	return antiFloodStateToView(state), nil
}

func antiFloodStateToView(state antiFloodState) *AntiFloodView {
	cfg := normalizeAntiFloodConfig(state.Config)
	return &AntiFloodView{
		Enabled:       state.Enabled,
		WindowSec:     cfg.WindowSec,
		MaxMessages:   cfg.MaxMessages,
		Penalty:       cfg.Penalty,
		MuteSec:       cfg.MuteSec,
		WarnDeleteSec: cfg.WarnDeleteSec,
	}
}

func (s *Service) SetAntiFloodEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return false, err
	}
	state.Enabled = enabled
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) CycleAntiFloodMaxMessagesByTGGroupID(tgGroupID int64) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.MaxMessages = nextCycleInt(cfg.MaxMessages, []int{3, 5, 8, 10, 15, 20})
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_max_messages_%d", cfg.MaxMessages), 0, 0)
	return cfg.MaxMessages, nil
}

func (s *Service) CycleAntiFloodWindowSecByTGGroupID(tgGroupID int64) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.WindowSec = nextCycleInt(cfg.WindowSec, []int{3, 5, 10, 15, 20, 30})
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_window_%d", cfg.WindowSec), 0, 0)
	return cfg.WindowSec, nil
}

func (s *Service) SetAntiFloodPenaltyByTGGroupID(tgGroupID int64, penalty string) (string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return "", err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	switch penalty {
	case antiFloodPenaltyWarn, antiFloodPenaltyMute, antiFloodPenaltyKick, antiFloodPenaltyKickBan, antiFloodPenaltyDeleteOnly:
		cfg.Penalty = penalty
	default:
		cfg.Penalty = antiFloodPenaltyDeleteOnly
	}
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_anti_flood_penalty_"+cfg.Penalty, 0, 0)
	return cfg.Penalty, nil
}

func (s *Service) CycleAntiFloodWarnDeleteSecByTGGroupID(tgGroupID int64) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.WarnDeleteSec = nextCycleInt(cfg.WarnDeleteSec, []int{0, 5, 10, 20, 30, 60})
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_warn_delete_%d", cfg.WarnDeleteSec), 0, 0)
	return cfg.WarnDeleteSec, nil
}

func nextCycleInt(current int, options []int) int {
	if len(options) == 0 {
		return current
	}
	for i, v := range options {
		if v == current {
			return options[(i+1)%len(options)]
		}
	}
	return options[0]
}

func antiFloodActionLabel(penalty string, muteSec int) string {
	switch penalty {
	case antiFloodPenaltyWarn:
		return "警告"
	case antiFloodPenaltyMute:
		return fmt.Sprintf("禁言 %d 秒", muteSec)
	case antiFloodPenaltyKick:
		return "踢出"
	case antiFloodPenaltyKickBan:
		return "踢出+封禁"
	default:
		return "撤回消息+不处罚"
	}
}
