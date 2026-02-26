package service

import (
	"errors"
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
		Enabled:               state.Enabled,
		WindowSec:             cfg.WindowSec,
		MaxMessages:           cfg.MaxMessages,
		Penalty:               cfg.Penalty,
		WarnThreshold:         cfg.WarnThreshold,
		WarnAction:            cfg.WarnAction,
		WarnActionMuteMinutes: cfg.WarnActionMuteMinutes,
		WarnActionBanMinutes:  cfg.WarnActionBanMinutes,
		MuteMinutes:           cfg.MuteMinutes,
		BanMinutes:            cfg.BanMinutes,
		WarnDeleteSec:         cfg.WarnDeleteSec,
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

func (s *Service) SetAntiFloodMaxMessagesByTGGroupID(tgGroupID int64, n int) (int, error) {
	if !isAllowedAntiFloodMaxMessages(n) {
		return 0, fmt.Errorf("invalid anti flood max messages")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.MaxMessages = n
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

func (s *Service) SetAntiFloodWindowSecByTGGroupID(tgGroupID int64, sec int) (int, error) {
	if !isAllowedAntiFloodWindowSec(sec) {
		return 0, fmt.Errorf("invalid anti flood window seconds")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.WindowSec = sec
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_window_%d", cfg.WindowSec), 0, 0)
	return cfg.WindowSec, nil
}

func (s *Service) SetAntiFloodPenaltyByTGGroupID(tgGroupID int64, penalty string) (string, error) {
	if !isAllowedAntiFloodPenalty(penalty) {
		return "", errors.New("invalid anti flood penalty")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return "", err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.Penalty = penalty
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_anti_flood_penalty_"+cfg.Penalty, 0, 0)
	return cfg.Penalty, nil
}

func (s *Service) SetAntiFloodWarnThresholdByTGGroupID(tgGroupID int64, count int) (int, error) {
	if !isAllowedAntiFloodWarnThreshold(count) {
		return 0, errors.New("invalid anti flood warn threshold")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.WarnThreshold = count
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_warn_threshold_%d", count), 0, 0)
	return count, nil
}

func (s *Service) SetAntiFloodWarnActionByTGGroupID(tgGroupID int64, action string) (string, error) {
	if !isAllowedAntiFloodWarnAction(action) {
		return "", errors.New("invalid anti flood warn action")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return "", err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.WarnAction = action
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_anti_flood_warn_action_"+action, 0, 0)
	return action, nil
}

func (s *Service) SetAntiFloodWarnActionMuteMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedAntiFloodDurationMinutes(minutes) {
		return 0, errors.New("invalid anti flood warn action mute minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.WarnActionMuteMinutes = minutes
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_warn_action_mute_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetAntiFloodWarnActionBanMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedAntiFloodDurationMinutes(minutes) {
		return 0, errors.New("invalid anti flood warn action ban minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.WarnActionBanMinutes = minutes
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_warn_action_ban_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetAntiFloodMuteMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedAntiFloodDurationMinutes(minutes) {
		return 0, errors.New("invalid anti flood mute minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.MuteMinutes = minutes
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_mute_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetAntiFloodBanMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedAntiFloodDurationMinutes(minutes) {
		return 0, errors.New("invalid anti flood ban minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.BanMinutes = minutes
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_ban_minutes_%d", minutes), 0, 0)
	return minutes, nil
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

func (s *Service) SetAntiFloodWarnDeleteSecByTGGroupID(tgGroupID int64, sec int) (int, error) {
	if !isAllowedAntiFloodWarnDeleteSec(sec) {
		return 0, fmt.Errorf("invalid anti flood warn delete seconds")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiFloodConfig(state.Config)
	cfg.WarnDeleteSec = sec
	state.Config = cfg
	if err := s.saveAntiFloodState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_flood_warn_delete_%d", cfg.WarnDeleteSec), 0, 0)
	return cfg.WarnDeleteSec, nil
}

func isAllowedAntiFloodWarnDeleteSec(sec int) bool {
	switch sec {
	case 0, 5, 10, 20, 30, 60:
		return true
	default:
		return false
	}
}

func isAllowedAntiFloodMaxMessages(n int) bool {
	switch n {
	case 3, 5, 8, 10, 15, 20:
		return true
	default:
		return false
	}
}

func isAllowedAntiFloodWindowSec(sec int) bool {
	switch sec {
	case 3, 5, 10, 15, 20, 30:
		return true
	default:
		return false
	}
}

func isAllowedAntiFloodPenalty(v string) bool {
	return isAllowedModerationPenalty(v)
}

func isAllowedAntiFloodWarnAction(v string) bool {
	return isAllowedModerationWarnAction(v)
}

func isAllowedAntiFloodWarnThreshold(v int) bool {
	return isAllowedModerationWarnThreshold(v)
}

func isAllowedAntiFloodDurationMinutes(v int) bool {
	return isAllowedModerationDurationMinutes(v)
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

func antiFloodActionLabel(penalty string, muteMinutes int, banMinutes int) string {
	return moderationPenaltyActionLabel(penalty, muteMinutes, banMinutes)
}
