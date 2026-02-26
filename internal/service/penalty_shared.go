package service

import "fmt"

type moderationPenaltyConfig struct {
	Penalty               string
	WarnThreshold         int
	WarnAction            string
	WarnActionMuteMinutes int
	WarnActionBanMinutes  int
	MuteMinutes           int
	BanMinutes            int
}

func normalizeModerationPenaltyConfig(cfg moderationPenaltyConfig, defaultPenalty string, legacyMuteSec int) moderationPenaltyConfig {
	if defaultPenalty == "" {
		defaultPenalty = antiFloodPenaltyDeleteOnly
	}
	if !isAllowedModerationPenalty(cfg.Penalty) {
		cfg.Penalty = defaultPenalty
	}
	if !isAllowedModerationWarnAction(cfg.WarnAction) {
		cfg.WarnAction = antiFloodPenaltyMute
	}
	if !isAllowedModerationWarnThreshold(cfg.WarnThreshold) {
		cfg.WarnThreshold = 3
	}

	legacyMuteMinutes := legacyMuteSec / 60
	if legacyMuteMinutes <= 0 {
		legacyMuteMinutes = 60
	}
	if !isAllowedModerationDurationMinutes(cfg.WarnActionMuteMinutes) {
		cfg.WarnActionMuteMinutes = legacyMuteMinutes
	}
	if !isAllowedModerationDurationMinutes(cfg.WarnActionBanMinutes) {
		cfg.WarnActionBanMinutes = 60
	}
	if !isAllowedModerationDurationMinutes(cfg.MuteMinutes) {
		cfg.MuteMinutes = legacyMuteMinutes
	}
	if !isAllowedModerationDurationMinutes(cfg.BanMinutes) {
		cfg.BanMinutes = 60
	}
	return cfg
}

func isAllowedModerationPenalty(v string) bool {
	switch v {
	case antiFloodPenaltyWarn, antiFloodPenaltyMute, antiFloodPenaltyKick, antiFloodPenaltyKickBan, antiFloodPenaltyDeleteOnly:
		return true
	default:
		return false
	}
}

func isAllowedModerationWarnAction(v string) bool {
	switch v {
	case antiFloodPenaltyMute, antiFloodPenaltyKick, antiFloodPenaltyKickBan:
		return true
	default:
		return false
	}
}

func isAllowedModerationWarnThreshold(v int) bool {
	return v > 0 && v <= 99
}

func isAllowedModerationDurationMinutes(v int) bool {
	return v > 0 && v <= 10080
}

func moderationPenaltyActionLabel(penalty string, muteMinutes int, banMinutes int) string {
	switch penalty {
	case antiFloodPenaltyWarn:
		return "警告"
	case antiFloodPenaltyMute:
		return fmt.Sprintf("禁言 %d 分钟", muteMinutes)
	case antiFloodPenaltyKick:
		return "踢出"
	case antiFloodPenaltyKickBan:
		return fmt.Sprintf("踢出+封禁 %d 分钟", banMinutes)
	default:
		return "撤回消息+不处罚"
	}
}

func (s *Service) resolveWarnablePenalty(
	groupID uint,
	targetID uint,
	cfg moderationPenaltyConfig,
	countWarns func(groupID, targetID uint) (int64, error),
	warnLogAction string,
	warnAppliedAction string,
) (string, string, int, int) {
	if cfg.Penalty != antiFloodPenaltyWarn {
		return cfg.Penalty, moderationPenaltyActionLabel(cfg.Penalty, cfg.MuteMinutes, cfg.BanMinutes), cfg.MuteMinutes, cfg.BanMinutes
	}
	warns, err := countWarns(groupID, targetID)
	if err != nil {
		return cfg.Penalty, moderationPenaltyActionLabel(cfg.Penalty, cfg.MuteMinutes, cfg.BanMinutes), cfg.MuteMinutes, cfg.BanMinutes
	}
	nextWarn := int(warns) + 1
	_ = s.repo.CreateLog(groupID, warnLogAction, 0, targetID)
	if nextWarn >= cfg.WarnThreshold {
		_ = s.repo.CreateLog(groupID, warnAppliedAction, 0, targetID)
		return cfg.WarnAction, moderationPenaltyActionLabel(cfg.WarnAction, cfg.WarnActionMuteMinutes, cfg.WarnActionBanMinutes), cfg.WarnActionMuteMinutes, cfg.WarnActionBanMinutes
	}
	return cfg.Penalty, fmt.Sprintf("警告（%d/%d）", nextWarn, cfg.WarnThreshold), cfg.WarnActionMuteMinutes, cfg.WarnActionBanMinutes
}
