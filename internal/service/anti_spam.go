package service

import (
	"errors"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *Service) AntiSpamViewByTGGroupID(tgGroupID int64) (*AntiSpamView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return nil, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	aiAvailable := s.antiSpamAIAvailable()
	keywords := append([]string{}, cfg.ExceptionKeywords...)
	return &AntiSpamView{
		Enabled:               state.Enabled,
		BlockPhoto:            cfg.BlockPhoto,
		BlockContactShare:     cfg.BlockContactShare,
		BlockLink:             cfg.BlockLink,
		BlockChannelAlias:     cfg.BlockChannelAlias,
		BlockForwardFromChan:  cfg.BlockForwardFromChannel,
		BlockForwardFromUser:  cfg.BlockForwardFromUser,
		BlockAtGroupID:        cfg.BlockAtGroupID,
		BlockAtUserID:         cfg.BlockAtUserID,
		BlockEthAddress:       cfg.BlockEthAddress,
		BlockLongMessage:      cfg.BlockLongMessage,
		MaxMessageLength:      cfg.MaxMessageLength,
		BlockLongName:         cfg.BlockLongName,
		MaxNameLength:         cfg.MaxNameLength,
		ExceptionKeywordCount: len(keywords),
		ExceptionKeywords:     keywords,
		AIAvailable:           aiAvailable,
		AIEnabled:             cfg.AIEnabled && aiAvailable,
		AISpamScore:           cfg.AISpamScore,
		AIStrictness:          cfg.AIStrictness,
		Penalty:               cfg.Penalty,
		WarnThreshold:         cfg.WarnThreshold,
		WarnAction:            cfg.WarnAction,
		WarnActionMuteMinutes: cfg.WarnActionMuteMinutes,
		WarnActionBanMinutes:  cfg.WarnActionBanMinutes,
		MuteMinutes:           cfg.MuteMinutes,
		BanMinutes:            cfg.BanMinutes,
		WarnDeleteSec:         cfg.WarnDeleteSec,
	}, nil
}

func (s *Service) SetAntiSpamEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return false, err
	}
	state.Enabled = enabled
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) SetAntiSpamPenaltyByTGGroupID(tgGroupID int64, penalty string) (string, error) {
	if !isAllowedAntiSpamPenalty(penalty) {
		return "", errors.New("invalid anti spam penalty")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return "", err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.Penalty = penalty
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_anti_spam_penalty_"+cfg.Penalty, 0, 0)
	return cfg.Penalty, nil
}

func (s *Service) SetAntiSpamWarnThresholdByTGGroupID(tgGroupID int64, count int) (int, error) {
	if !isAllowedAntiSpamWarnThreshold(count) {
		return 0, errors.New("invalid anti spam warn threshold")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.WarnThreshold = count
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_warn_threshold_%d", count), 0, 0)
	return count, nil
}

func (s *Service) SetAntiSpamWarnActionByTGGroupID(tgGroupID int64, action string) (string, error) {
	if !isAllowedAntiSpamWarnAction(action) {
		return "", errors.New("invalid anti spam warn action")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return "", err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.WarnAction = action
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_anti_spam_warn_action_"+action, 0, 0)
	return action, nil
}

func (s *Service) SetAntiSpamWarnActionMuteMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedAntiSpamDurationMinutes(minutes) {
		return 0, errors.New("invalid anti spam warn action mute minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.WarnActionMuteMinutes = minutes
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_warn_action_mute_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetAntiSpamWarnActionBanMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedAntiSpamDurationMinutes(minutes) {
		return 0, errors.New("invalid anti spam warn action ban minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.WarnActionBanMinutes = minutes
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_warn_action_ban_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetAntiSpamMuteMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedAntiSpamDurationMinutes(minutes) {
		return 0, errors.New("invalid anti spam mute minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.MuteMinutes = minutes
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_mute_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetAntiSpamBanMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedAntiSpamDurationMinutes(minutes) {
		return 0, errors.New("invalid anti spam ban minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.BanMinutes = minutes
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_ban_minutes_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) ToggleAntiSpamOptionByTGGroupID(tgGroupID int64, option string) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return false, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)

	var (
		next bool
		log  string
	)
	switch option {
	case "photo":
		cfg.BlockPhoto = !cfg.BlockPhoto
		next = cfg.BlockPhoto
		log = "set_anti_spam_photo"
	case "contact":
		cfg.BlockContactShare = !cfg.BlockContactShare
		next = cfg.BlockContactShare
		log = "set_anti_spam_contact"
	case "link":
		cfg.BlockLink = !cfg.BlockLink
		next = cfg.BlockLink
		log = "set_anti_spam_link"
	case "senderchat":
		cfg.BlockChannelAlias = !cfg.BlockChannelAlias
		next = cfg.BlockChannelAlias
		log = "set_anti_spam_senderchat"
	case "fwdchan":
		cfg.BlockForwardFromChannel = !cfg.BlockForwardFromChannel
		next = cfg.BlockForwardFromChannel
		log = "set_anti_spam_fwdchan"
	case "fwduser":
		cfg.BlockForwardFromUser = !cfg.BlockForwardFromUser
		next = cfg.BlockForwardFromUser
		log = "set_anti_spam_fwduser"
	case "atgroup":
		cfg.BlockAtGroupID = !cfg.BlockAtGroupID
		next = cfg.BlockAtGroupID
		log = "set_anti_spam_atgroup"
	case "atuser":
		cfg.BlockAtUserID = !cfg.BlockAtUserID
		next = cfg.BlockAtUserID
		log = "set_anti_spam_atuser"
	case "eth":
		cfg.BlockEthAddress = !cfg.BlockEthAddress
		next = cfg.BlockEthAddress
		log = "set_anti_spam_eth"
	case "longmsg":
		cfg.BlockLongMessage = !cfg.BlockLongMessage
		next = cfg.BlockLongMessage
		log = "set_anti_spam_longmsg"
	case "longname":
		cfg.BlockLongName = !cfg.BlockLongName
		next = cfg.BlockLongName
		log = "set_anti_spam_longname"
	default:
		return false, errors.New("invalid anti spam option")
	}

	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("%s_%t", log, next), 0, 0)
	return next, nil
}

func (s *Service) SetAntiSpamMaxMessageLengthByTGGroupID(tgGroupID int64, maxLen int) (int, error) {
	if maxLen <= 0 {
		return 0, errors.New("invalid max message length")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.MaxMessageLength = maxLen
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_max_msg_len_%d", maxLen), 0, 0)
	return maxLen, nil
}

func (s *Service) SetAntiSpamMaxNameLengthByTGGroupID(tgGroupID int64, maxLen int) (int, error) {
	if maxLen <= 0 {
		return 0, errors.New("invalid max name length")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.MaxNameLength = maxLen
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_max_name_len_%d", maxLen), 0, 0)
	return maxLen, nil
}

func (s *Service) AddAntiSpamExceptionByTGGroupID(tgGroupID int64, keyword string) (int, error) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return 0, errors.New("empty keyword")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.ExceptionKeywords = append(cfg.ExceptionKeywords, kw)
	cfg = normalizeAntiSpamConfig(cfg)
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, "add_anti_spam_exception", 0, 0)
	return len(cfg.ExceptionKeywords), nil
}

func (s *Service) RemoveAntiSpamExceptionByTGGroupID(tgGroupID int64, keyword string) (int, error) {
	kw := strings.TrimSpace(strings.ToLower(keyword))
	if kw == "" {
		return 0, errors.New("empty keyword")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	filtered := make([]string, 0, len(cfg.ExceptionKeywords))
	for _, item := range cfg.ExceptionKeywords {
		if strings.ToLower(strings.TrimSpace(item)) == kw {
			continue
		}
		filtered = append(filtered, item)
	}
	cfg.ExceptionKeywords = filtered
	cfg = normalizeAntiSpamConfig(cfg)
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, "remove_anti_spam_exception", 0, 0)
	return len(cfg.ExceptionKeywords), nil
}

func (s *Service) SetAntiSpamWarnDeleteSecByTGGroupID(tgGroupID int64, sec int) (int, error) {
	if !isAllowedAntiSpamWarnDeleteSec(sec) {
		return 0, errors.New("invalid anti spam warn delete seconds")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.WarnDeleteSec = sec
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_warn_delete_%d", cfg.WarnDeleteSec), 0, 0)
	return cfg.WarnDeleteSec, nil
}

func (s *Service) SetAntiSpamAIEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	if enabled && !s.antiSpamAIAvailable() {
		return false, errors.New("anti spam ai model is not configured")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return false, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.AIEnabled = enabled
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_ai_enabled_%t", enabled), 0, 0)
	return cfg.AIEnabled, nil
}

func (s *Service) antiSpamAIAvailable() bool {
	return s != nil && s.spamAI != nil
}

func (s *Service) SetAntiSpamAISpamScoreByTGGroupID(tgGroupID int64, spamScore int) (int, error) {
	if spamScore < 1 || spamScore > 100 {
		return 0, errors.New("invalid ai spam score")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return 0, err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.AISpamScore = spamScore
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_ai_spam_score_%d", spamScore), 0, 0)
	return cfg.AISpamScore, nil
}

func (s *Service) SetAntiSpamAIStrictnessByTGGroupID(tgGroupID int64, strictness string) (string, error) {
	if !isAllowedAntiSpamAIStrictness(strictness) {
		return "", errors.New("invalid ai strictness")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return "", err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	cfg.AIStrictness = normalizeAntiSpamAIStrictness(strictness)
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_anti_spam_ai_strictness_%s", cfg.AIStrictness), 0, 0)
	return cfg.AIStrictness, nil
}

func (s *Service) ReleaseAntiSpamPenaltyByTGGroupID(bot *tgbotapi.BotAPI, tgGroupID, tgUserID int64) error {
	if bot == nil {
		return errors.New("nil bot")
	}
	_, _ = bot.Request(tgbotapi.UnbanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
		OnlyIfBanned:     true,
	})
	return s.restoreMemberSpeak(bot, tgGroupID, tgUserID)
}

func (s *Service) RevokeAntiSpamWarnByTGGroupID(tgGroupID, operatorTGUserID, targetTGUserID int64) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	target, err := s.repo.EnsureUserByTGUserID(targetTGUserID)
	if err != nil {
		return err
	}
	deleted, err := s.repo.DeleteLatestWarnLog(group.ID, target.ID, "anti_spam_warn", "anti_spam_warn_action_applied")
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
	_ = s.repo.CreateLog(group.ID, "anti_spam_warn_revoked", operatorID, target.ID)
	return nil
}

func isAllowedAntiSpamWarnDeleteSec(sec int) bool {
	switch sec {
	case -1, 0, 10, 30, 60, 300, 600, 1800, 3600, 21600, 43200:
		return true
	default:
		return false
	}
}

func isAllowedAntiSpamPenalty(v string) bool {
	return isAllowedModerationPenalty(v)
}

func isAllowedAntiSpamWarnAction(v string) bool {
	return isAllowedModerationWarnAction(v)
}

func isAllowedAntiSpamWarnThreshold(v int) bool {
	return isAllowedModerationWarnThreshold(v)
}

func isAllowedAntiSpamDurationMinutes(v int) bool {
	return isAllowedModerationDurationMinutes(v)
}

func isAllowedAntiSpamAIStrictness(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "low", "medium", "high", "低", "中", "高":
		return true
	default:
		return false
	}
}
