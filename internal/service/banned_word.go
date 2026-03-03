package service

import (
	"errors"
	"fmt"
)

func (s *Service) BannedWordViewByTGGroupID(tgGroupID int64) (*BannedWordView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	enabled, cfg, err := s.bannedWordStateByGroupID(group.ID)
	if err != nil {
		return nil, err
	}
	return &BannedWordView{
		Enabled:               enabled,
		Penalty:               cfg.Penalty,
		WarnThreshold:         cfg.WarnThreshold,
		WarnAction:            cfg.WarnAction,
		WarnActionMuteMinutes: cfg.WarnActionMuteMinutes,
		WarnActionBanMinutes:  cfg.WarnActionBanMinutes,
		MuteMinutes:           cfg.MuteMinutes,
		BanMinutes:            cfg.BanMinutes,
		WarnDeleteMinutes:     cfg.WarnDeleteMinutes,
	}, nil
}

func (s *Service) SetBannedWordEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	if err := s.repo.UpsertFeatureEnabled(group.ID, featureBannedWords, enabled); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_banned_words_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) SetBannedWordPenaltyByTGGroupID(tgGroupID int64, penalty string) (string, error) {
	if !isAllowedBannedWordPenalty(penalty) {
		return "", errors.New("invalid banned word penalty")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	cfg, err := s.getBannedWordConfig(group.ID)
	if err != nil {
		return "", err
	}
	cfg.Penalty = penalty
	if err := s.saveBannedWordConfig(group.ID, cfg); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_banned_words_penalty_"+penalty, 0, 0)
	return penalty, nil
}

func (s *Service) SetBannedWordWarnThresholdByTGGroupID(tgGroupID int64, count int) (int, error) {
	if !isAllowedBannedWordWarnThreshold(count) {
		return 0, errors.New("invalid banned word warn threshold")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getBannedWordConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.WarnThreshold = count
	if err := s.saveBannedWordConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_banned_words_warn_threshold_%d", count), 0, 0)
	return count, nil
}

func (s *Service) SetBannedWordWarnActionByTGGroupID(tgGroupID int64, action string) (string, error) {
	if !isAllowedBannedWordWarnAction(action) {
		return "", errors.New("invalid banned word warn action")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	cfg, err := s.getBannedWordConfig(group.ID)
	if err != nil {
		return "", err
	}
	cfg.WarnAction = action
	if err := s.saveBannedWordConfig(group.ID, cfg); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_banned_words_warn_action_"+action, 0, 0)
	return action, nil
}

func (s *Service) SetBannedWordWarnActionMuteMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedBannedWordDurationMinutes(minutes) {
		return 0, errors.New("invalid banned word warn action mute minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getBannedWordConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.WarnActionMuteMinutes = minutes
	if err := s.saveBannedWordConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_banned_words_warn_action_mute_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetBannedWordWarnActionBanMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedBannedWordDurationMinutes(minutes) {
		return 0, errors.New("invalid banned word warn action ban minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getBannedWordConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.WarnActionBanMinutes = minutes
	if err := s.saveBannedWordConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_banned_words_warn_action_ban_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetBannedWordMuteMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedBannedWordDurationMinutes(minutes) {
		return 0, errors.New("invalid banned word mute minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getBannedWordConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.MuteMinutes = minutes
	if err := s.saveBannedWordConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_banned_words_mute_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetBannedWordBanMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedBannedWordDurationMinutes(minutes) {
		return 0, errors.New("invalid banned word ban minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getBannedWordConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.BanMinutes = minutes
	if err := s.saveBannedWordConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_banned_words_ban_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetBannedWordWarnDeleteMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedBannedWordWarnDeleteMinutes(minutes) {
		return 0, errors.New("invalid banned word warn delete minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getBannedWordConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.WarnDeleteMinutes = minutes
	if err := s.saveBannedWordConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_banned_words_warn_delete_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) bannedWordStateByGroupID(groupID uint) (bool, bannedWordConfig, error) {
	enabled, err := s.IsFeatureEnabled(groupID, featureBannedWords, true)
	if err != nil {
		return false, bannedWordConfig{}, err
	}
	cfg, err := s.getBannedWordConfig(groupID)
	if err != nil {
		return false, bannedWordConfig{}, err
	}
	return enabled, cfg, nil
}

func (s *Service) RevokeBannedWordWarnByTGGroupID(tgGroupID, operatorTGUserID, targetTGUserID int64) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	target, err := s.repo.EnsureUserByTGUserID(targetTGUserID)
	if err != nil {
		return err
	}
	deleted, err := s.repo.DeleteLatestWarnLog(group.ID, target.ID, "banned_word_warn", "banned_word_warn_action_applied")
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNoModerationWarnToRevoke
	}
	operatorID := uint(0)
	if operatorTGUserID > 0 {
		if operator, opErr := s.repo.EnsureUserByTGUserID(operatorTGUserID); opErr == nil {
			operatorID = operator.ID
		}
	}
	_ = s.repo.CreateLog(group.ID, "banned_word_warn_revoked", operatorID, target.ID)
	return nil
}

func isAllowedBannedWordPenalty(v string) bool {
	return isAllowedModerationPenalty(v)
}

func isAllowedBannedWordWarnAction(v string) bool {
	return isAllowedModerationWarnAction(v)
}

func isAllowedBannedWordWarnThreshold(v int) bool {
	return isAllowedModerationWarnThreshold(v)
}

func isAllowedBannedWordDurationMinutes(v int) bool {
	return isAllowedModerationDurationMinutes(v)
}

func isAllowedBannedWordWarnDeleteMinutes(v int) bool {
	return v >= 0 && v <= 1440
}
