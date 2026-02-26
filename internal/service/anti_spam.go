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
	keywords := append([]string{}, cfg.ExceptionKeywords...)
	return &AntiSpamView{
		Enabled:               state.Enabled,
		BlockPhoto:            cfg.BlockPhoto,
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
		AIEnabled:             cfg.AIEnabled,
		AISpamScore:           cfg.AISpamScore,
		Penalty:               cfg.Penalty,
		MuteSec:               cfg.MuteSec,
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
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	state, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return "", err
	}
	cfg := normalizeAntiSpamConfig(state.Config)
	switch penalty {
	case antiFloodPenaltyWarn, antiFloodPenaltyMute, antiFloodPenaltyKick, antiFloodPenaltyKickBan, antiFloodPenaltyDeleteOnly:
		cfg.Penalty = penalty
	default:
		cfg.Penalty = antiFloodPenaltyDeleteOnly
	}
	state.Config = cfg
	if err := s.saveAntiSpamState(group.ID, state); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_anti_spam_penalty_"+cfg.Penalty, 0, 0)
	return cfg.Penalty, nil
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

func isAllowedAntiSpamWarnDeleteSec(sec int) bool {
	switch sec {
	case -1, 0, 10, 30, 60, 300, 600, 1800, 3600, 21600, 43200:
		return true
	default:
		return false
	}
}
