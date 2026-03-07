package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

func featureConfigCacheKey(groupID uint, featureKey string) string {
	return fmt.Sprintf("%d:%s", groupID, featureKey)
}

func (s *Service) getFeatureConfigEntryFromCache(groupID uint, featureKey string) (featureConfigCacheEntry, bool) {
	key := featureConfigCacheKey(groupID, featureKey)
	s.configCacheMu.RLock()
	entry, ok := s.configCache[key]
	s.configCacheMu.RUnlock()
	return entry, ok
}

func (s *Service) setFeatureConfigEntryCache(groupID uint, featureKey string, entry featureConfigCacheEntry) {
	key := featureConfigCacheKey(groupID, featureKey)
	s.configCacheMu.Lock()
	s.configCache[key] = entry
	s.configCacheMu.Unlock()
}

func (s *Service) readFeatureConfigEntry(groupID uint, featureKey string) (featureConfigCacheEntry, error) {
	if entry, ok := s.getFeatureConfigEntryFromCache(groupID, featureKey); ok {
		return entry, nil
	}
	setting, err := s.repo.GetGroupSetting(groupID, featureKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			entry := featureConfigCacheEntry{Exists: false, Config: ""}
			s.setFeatureConfigEntryCache(groupID, featureKey, entry)
			return entry, nil
		}
		return featureConfigCacheEntry{}, err
	}
	entry := featureConfigCacheEntry{Exists: true, Config: setting.Config}
	s.setFeatureConfigEntryCache(groupID, featureKey, entry)
	return entry, nil
}

func (s *Service) saveFeatureConfigEntry(groupID uint, featureKey string, config string) error {
	if err := s.repo.UpsertFeatureConfig(groupID, featureKey, config, defaultFeatureEnabled(featureKey)); err != nil {
		return err
	}
	s.setFeatureConfigEntryCache(groupID, featureKey, featureConfigCacheEntry{
		Exists: true,
		Config: config,
	})
	return nil
}

func defaultFeatureEnabled(featureKey string) bool {
	switch featureKey {
	case featureWelcome:
		return true
	default:
		return false
	}
}

func defaultWelcomeConfig() welcomeConfig {
	return welcomeConfig{
		Text:          "欢迎 {user} 加入，先看群规再发言。",
		Mode:          "verify",
		DeleteMinutes: 1,
		MediaFileID:   "",
		ButtonRows:    [][]welcomeButton{},
	}
}

func normalizeWelcomeConfig(cfg welcomeConfig) welcomeConfig {
	if cfg.Text == "" {
		cfg.Text = "欢迎 {user} 加入，先看群规再发言。"
	}
	if cfg.Mode != "join" {
		cfg.Mode = "verify"
	}
	switch cfg.DeleteMinutes {
	case 0, 1, 5, 10, 30:
	default:
		cfg.DeleteMinutes = 1
	}
	cfg.ButtonRows = normalizeWelcomeButtonRows(cfg.ButtonRows)
	return cfg
}

func (s *Service) getWelcomeConfig(groupID uint) (welcomeConfig, error) {
	cfg := defaultWelcomeConfig()
	entry, err := s.readFeatureConfigEntry(groupID, featureWelcome)
	if err != nil {
		return cfg, err
	}
	if entry.Exists && entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &cfg)
		if len(cfg.ButtonRows) == 0 {
			var legacy struct {
				ButtonText string `json:"button_text"`
				ButtonURL  string `json:"button_url"`
			}
			if err := json.Unmarshal([]byte(entry.Config), &legacy); err == nil {
				if strings.TrimSpace(legacy.ButtonText) != "" && strings.TrimSpace(legacy.ButtonURL) != "" {
					if normURL, nErr := normalizeWelcomeButtonURL(legacy.ButtonURL); nErr == nil {
						cfg.ButtonRows = [][]welcomeButton{{{Text: strings.TrimSpace(legacy.ButtonText), URL: normURL}}}
					}
				}
			}
		}
	}
	return normalizeWelcomeConfig(cfg), nil
}

func (s *Service) saveWelcomeConfig(groupID uint, cfg welcomeConfig) error {
	cfg = normalizeWelcomeConfig(cfg)
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featureWelcome, string(b))
}

func defaultAntiSpamConfig() antiSpamConfig {
	return antiSpamConfig{
		BlockPhoto:              false,
		BlockContactShare:       false,
		BlockLink:               false,
		BlockChannelAlias:       false,
		BlockForwardFromChannel: false,
		BlockForwardFromUser:    false,
		BlockExternalReply:      false,
		BlockAtGroupID:          false,
		BlockAtUserID:           false,
		BlockEthAddress:         false,
		BlockLongMessage:        false,
		MaxMessageLength:        100,
		BlockLongName:           false,
		MaxNameLength:           32,
		ExceptionKeywords:       []string{},
		AIEnabled:               false,
		AISpamScore:             70,
		AIStrictness:            antiSpamAIStrictnessMedium,
		Penalty:                 antiFloodPenaltyDeleteOnly,
		WarnThreshold:           3,
		WarnAction:              antiFloodPenaltyMute,
		WarnActionMuteMinutes:   60,
		WarnActionBanMinutes:    60,
		MuteMinutes:             60,
		BanMinutes:              60,
		MuteSec:                 60,
		WarnDeleteSec:           10,
	}
}

func normalizeAntiSpamConfig(cfg antiSpamConfig) antiSpamConfig {
	if cfg.MaxMessageLength <= 0 {
		cfg.MaxMessageLength = 100
	}
	if cfg.MaxNameLength <= 0 {
		cfg.MaxNameLength = 32
	}
	if cfg.AISpamScore <= 0 {
		cfg.AISpamScore = 70
	}
	if cfg.AISpamScore > 100 {
		cfg.AISpamScore = 100
	}
	cfg.AIStrictness = normalizeAntiSpamAIStrictness(cfg.AIStrictness)
	if cfg.WarnDeleteSec < -1 {
		cfg.WarnDeleteSec = -1
	}
	penaltyCfg := normalizeModerationPenaltyConfig(moderationPenaltyConfig{
		Penalty:               cfg.Penalty,
		WarnThreshold:         cfg.WarnThreshold,
		WarnAction:            cfg.WarnAction,
		WarnActionMuteMinutes: cfg.WarnActionMuteMinutes,
		WarnActionBanMinutes:  cfg.WarnActionBanMinutes,
		MuteMinutes:           cfg.MuteMinutes,
		BanMinutes:            cfg.BanMinutes,
	}, antiFloodPenaltyDeleteOnly, cfg.MuteSec)
	cfg.Penalty = penaltyCfg.Penalty
	cfg.WarnThreshold = penaltyCfg.WarnThreshold
	cfg.WarnAction = penaltyCfg.WarnAction
	cfg.WarnActionMuteMinutes = penaltyCfg.WarnActionMuteMinutes
	cfg.WarnActionBanMinutes = penaltyCfg.WarnActionBanMinutes
	cfg.MuteMinutes = penaltyCfg.MuteMinutes
	cfg.BanMinutes = penaltyCfg.BanMinutes
	// 例外的关键词
	cfg.ExceptionKeywords = normalizeKeywordList(cfg.ExceptionKeywords)
	return cfg
}

// 去重并转为小写
func normalizeKeywordList(items []string) []string {
	uniq := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		kw := strings.TrimSpace(item)
		if kw == "" {
			continue
		}
		k := strings.ToLower(kw)
		if _, ok := uniq[k]; ok {
			continue
		}
		uniq[k] = struct{}{}
		out = append(out, kw)
	}
	return out
}

func (s *Service) getAntiSpamState(groupID uint) (antiSpamState, error) {
	s.antiSpamMu.RLock()
	state, ok := s.antiSpamCache[groupID]
	s.antiSpamMu.RUnlock()
	if ok {
		return state, nil
	}

	cfg := defaultAntiSpamConfig()
	setting, err := s.repo.GetGroupSetting(groupID, featureAntiSpam)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			state = antiSpamState{Enabled: false, Config: cfg}
			if saveErr := s.saveAntiSpamState(groupID, state); saveErr != nil {
				return state, saveErr
			}
			return state, nil
		}
		return antiSpamState{}, err
	}

	rawCfg := cfg
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &rawCfg)
	}
	cfg = normalizeAntiSpamConfig(rawCfg)
	state = antiSpamState{
		Enabled: setting.Enabled,
		Config:  cfg,
	}

	s.antiSpamMu.Lock()
	s.antiSpamCache[groupID] = state
	s.antiSpamMu.Unlock()

	if setting.Config == "" || !reflect.DeepEqual(rawCfg, cfg) {
		_ = s.saveAntiSpamState(groupID, state)
	}
	return state, nil
}

func (s *Service) saveAntiSpamState(groupID uint, state antiSpamState) error {
	state.Config = normalizeAntiSpamConfig(state.Config)
	if err := s.repo.UpsertFeatureEnabled(groupID, featureAntiSpam, state.Enabled); err != nil {
		return err
	}
	b, err := json.Marshal(state.Config)
	if err != nil {
		return err
	}
	if err := s.repo.UpsertFeatureConfig(groupID, featureAntiSpam, string(b), defaultFeatureEnabled(featureAntiSpam)); err != nil {
		return err
	}
	s.antiSpamMu.Lock()
	s.antiSpamCache[groupID] = state
	s.antiSpamMu.Unlock()
	return nil
}

func defaultAntiFloodConfig() antiFloodConfig {
	return antiFloodConfig{
		WindowSec:             5,
		MaxMessages:           5,
		Penalty:               antiFloodPenaltyDeleteOnly,
		WarnThreshold:         3,
		WarnAction:            antiFloodPenaltyMute,
		WarnActionMuteMinutes: 60,
		WarnActionBanMinutes:  60,
		MuteMinutes:           60,
		BanMinutes:            60,
		MuteSec:               60,
		WarnDeleteSec:         10,
		RepeatWindow:          20,
		RepeatThreshold:       3,
	}
}

func normalizeAntiFloodConfig(cfg antiFloodConfig) antiFloodConfig {
	if cfg.WindowSec <= 0 {
		cfg.WindowSec = 5
	}
	if cfg.MaxMessages <= 0 {
		cfg.MaxMessages = 5
	}
	if cfg.WarnDeleteSec < 0 {
		cfg.WarnDeleteSec = 0
	}
	penaltyCfg := normalizeModerationPenaltyConfig(moderationPenaltyConfig{
		Penalty:               cfg.Penalty,
		WarnThreshold:         cfg.WarnThreshold,
		WarnAction:            cfg.WarnAction,
		WarnActionMuteMinutes: cfg.WarnActionMuteMinutes,
		WarnActionBanMinutes:  cfg.WarnActionBanMinutes,
		MuteMinutes:           cfg.MuteMinutes,
		BanMinutes:            cfg.BanMinutes,
	}, antiFloodPenaltyDeleteOnly, cfg.MuteSec)
	cfg.Penalty = penaltyCfg.Penalty
	cfg.WarnThreshold = penaltyCfg.WarnThreshold
	cfg.WarnAction = penaltyCfg.WarnAction
	cfg.WarnActionMuteMinutes = penaltyCfg.WarnActionMuteMinutes
	cfg.WarnActionBanMinutes = penaltyCfg.WarnActionBanMinutes
	cfg.MuteMinutes = penaltyCfg.MuteMinutes
	cfg.BanMinutes = penaltyCfg.BanMinutes
	return cfg
}

func (s *Service) getAntiFloodState(groupID uint) (antiFloodState, error) {
	s.antiFloodMu.RLock()
	state, ok := s.antiFloodCache[groupID]
	s.antiFloodMu.RUnlock()
	if ok {
		return state, nil
	}

	cfg := defaultAntiFloodConfig()
	enabled := false
	setting, err := s.repo.GetGroupSetting(groupID, featureAntiFlood)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			state = antiFloodState{Enabled: false, Config: cfg}
			if saveErr := s.saveAntiFloodState(groupID, state); saveErr != nil {
				return state, saveErr
			}
			return state, nil
		}
		return antiFloodState{}, err
	}

	enabled = setting.Enabled
	rawCfg := cfg
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &rawCfg)
	}
	cfg = normalizeAntiFloodConfig(rawCfg)

	state = antiFloodState{Enabled: enabled, Config: cfg}
	s.antiFloodMu.Lock()
	s.antiFloodCache[groupID] = state
	s.antiFloodMu.Unlock()

	// Ensure db has normalized config and key exists for persistence across restarts.
	if setting.Config == "" || rawCfg != cfg {
		_ = s.saveAntiFloodState(groupID, state)
	}
	return state, nil
}

func (s *Service) saveAntiFloodState(groupID uint, state antiFloodState) error {
	state.Config = normalizeAntiFloodConfig(state.Config)
	if err := s.repo.UpsertFeatureEnabled(groupID, featureAntiFlood, state.Enabled); err != nil {
		return err
	}
	b, err := json.Marshal(state.Config)
	if err != nil {
		return err
	}
	if err := s.repo.UpsertFeatureConfig(groupID, featureAntiFlood, string(b), defaultFeatureEnabled(featureAntiFlood)); err != nil {
		return err
	}
	s.antiFloodMu.Lock()
	s.antiFloodCache[groupID] = state
	s.antiFloodMu.Unlock()
	return nil
}

func defaultNightModeConfig() nightModeConfig {
	return nightModeConfig{
		Mode:      "delete_media",
		StartHour: nightDefaultStartHour,
		EndHour:   nightDefaultEndHour,
	}
}

func normalizeNightModeConfig(cfg nightModeConfig) nightModeConfig {
	switch cfg.Mode {
	case "delete_media", "global_mute":
	default:
		cfg.Mode = "delete_media"
	}
	if cfg.StartHour < 0 || cfg.StartHour > 23 {
		cfg.StartHour = nightDefaultStartHour
	}
	if cfg.EndHour < 0 || cfg.EndHour > 23 {
		cfg.EndHour = nightDefaultEndHour
	}
	return cfg
}

func (s *Service) getNightModeState(groupID uint) (nightModeState, error) {
	s.nightModeMu.RLock()
	state, ok := s.nightModeCache[groupID]
	s.nightModeMu.RUnlock()
	if ok {
		return state, nil
	}

	cfg := defaultNightModeConfig()
	setting, err := s.repo.GetGroupSetting(groupID, featureNightMode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			state = nightModeState{Enabled: false, Config: cfg}
			if saveErr := s.saveNightModeState(groupID, state); saveErr != nil {
				return state, saveErr
			}
			return state, nil
		}
		return nightModeState{}, err
	}

	rawCfg := cfg
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &rawCfg)
		fields := map[string]json.RawMessage{}
		if err := json.Unmarshal([]byte(setting.Config), &fields); err == nil {
			if _, ok := fields["start_hour"]; !ok {
				rawCfg.StartHour = cfg.StartHour
			}
			if _, ok := fields["end_hour"]; !ok {
				rawCfg.EndHour = cfg.EndHour
			}
		}
	}
	cfg = normalizeNightModeConfig(rawCfg)
	state = nightModeState{
		Enabled: setting.Enabled,
		Config:  cfg,
	}

	s.nightModeMu.Lock()
	s.nightModeCache[groupID] = state
	s.nightModeMu.Unlock()

	if setting.Config == "" || !reflect.DeepEqual(rawCfg, cfg) {
		_ = s.saveNightModeState(groupID, state)
	}
	return state, nil
}

func (s *Service) saveNightModeState(groupID uint, state nightModeState) error {
	state.Config = normalizeNightModeConfig(state.Config)
	if err := s.repo.UpsertFeatureEnabled(groupID, featureNightMode, state.Enabled); err != nil {
		return err
	}
	b, err := json.Marshal(state.Config)
	if err != nil {
		return err
	}
	if err := s.repo.UpsertFeatureConfig(groupID, featureNightMode, string(b), defaultFeatureEnabled(featureNightMode)); err != nil {
		return err
	}
	s.nightModeMu.Lock()
	s.nightModeCache[groupID] = state
	s.nightModeMu.Unlock()
	return nil
}

func (s *Service) getJoinVerifyConfig(groupID uint) (joinVerifyConfig, error) {
	cfg := joinVerifyConfig{Type: "button", TimeoutMinutes: 5, TimeoutAction: "mute"}
	entry, err := s.readFeatureConfigEntry(groupID, featureJoinVerify)
	if err != nil {
		return cfg, err
	}
	if !entry.Exists {
		if saveErr := s.saveJoinVerifyConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
		return cfg, nil
	}
	rawCfg := cfg
	if entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &rawCfg)
	}
	cfg = rawCfg
	switch cfg.Type {
	case "button", "math", "captcha", "zhchar", "zhvoice":
	default:
		cfg.Type = "button"
	}
	if cfg.TimeoutMinutes <= 0 && cfg.TimeoutSec > 0 {
		switch {
		case cfg.TimeoutSec <= 60:
			cfg.TimeoutMinutes = 1
		case cfg.TimeoutSec <= 300:
			cfg.TimeoutMinutes = 5
		default:
			cfg.TimeoutMinutes = 10
		}
	}
	switch cfg.TimeoutMinutes {
	case 1, 5, 10:
	default:
		cfg.TimeoutMinutes = 5
	}
	switch cfg.TimeoutAction {
	case "mute", "kick":
	default:
		cfg.TimeoutAction = "mute"
	}
	if entry.Config == "" || rawCfg != cfg {
		if saveErr := s.saveJoinVerifyConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
	}
	return cfg, nil
}

func (s *Service) saveJoinVerifyConfig(groupID uint, cfg joinVerifyConfig) error {
	cfg.TimeoutSec = cfg.TimeoutMinutes * 60
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featureJoinVerify, string(b))
}

func (s *Service) getSystemCleanConfig(groupID uint) (systemCleanConfig, error) {
	cfg := systemCleanConfig{
		Join:  true,
		Leave: true,
		Pin:   false,
		Photo: false,
		Title: false,
	}
	entry, err := s.readFeatureConfigEntry(groupID, featureSystemClean)
	if err != nil {
		return cfg, err
	}
	if !entry.Exists {
		if saveErr := s.saveSystemCleanConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
		return cfg, nil
	}
	if entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) saveSystemCleanConfig(groupID uint, cfg systemCleanConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featureSystemClean, string(b))
}

func defaultLotteryConfig() lotteryConfig {
	return lotteryConfig{
		PublishPin:           true,
		ResultPin:            true,
		DeleteKeywordMinutes: 5,
	}
}

func normalizeLotteryConfig(cfg lotteryConfig) lotteryConfig {
	switch cfg.DeleteKeywordMinutes {
	case 0, 1, 3, 5, 10, 30:
	default:
		cfg.DeleteKeywordMinutes = 5
	}
	return cfg
}

func (s *Service) getLotteryConfig(groupID uint) (lotteryConfig, error) {
	cfg := defaultLotteryConfig()
	entry, err := s.readFeatureConfigEntry(groupID, featureLottery)
	if err != nil {
		return cfg, err
	}
	if !entry.Exists {
		if saveErr := s.saveLotteryConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
		return cfg, nil
	}
	rawCfg := cfg
	if entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &rawCfg)
	}
	cfg = normalizeLotteryConfig(rawCfg)
	if entry.Config == "" || rawCfg != cfg {
		if saveErr := s.saveLotteryConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
	}
	return cfg, nil
}

func (s *Service) saveLotteryConfig(groupID uint, cfg lotteryConfig) error {
	cfg = normalizeLotteryConfig(cfg)
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featureLottery, string(b))
}

func defaultBannedWordConfig() bannedWordConfig {
	return bannedWordConfig{
		Penalty:               antiFloodPenaltyWarn,
		WarnThreshold:         3,
		WarnAction:            antiFloodPenaltyMute,
		WarnActionMuteMinutes: 60,
		WarnActionBanMinutes:  60,
		MuteMinutes:           60,
		BanMinutes:            60,
		WarnDeleteMinutes:     10,
	}
}

func normalizeBannedWordConfig(cfg bannedWordConfig) bannedWordConfig {
	penaltyCfg := normalizeModerationPenaltyConfig(moderationPenaltyConfig{
		Penalty:               cfg.Penalty,
		WarnThreshold:         cfg.WarnThreshold,
		WarnAction:            cfg.WarnAction,
		WarnActionMuteMinutes: cfg.WarnActionMuteMinutes,
		WarnActionBanMinutes:  cfg.WarnActionBanMinutes,
		MuteMinutes:           cfg.MuteMinutes,
		BanMinutes:            cfg.BanMinutes,
	}, antiFloodPenaltyWarn, 0)
	cfg.Penalty = penaltyCfg.Penalty
	cfg.WarnThreshold = penaltyCfg.WarnThreshold
	cfg.WarnAction = penaltyCfg.WarnAction
	cfg.WarnActionMuteMinutes = penaltyCfg.WarnActionMuteMinutes
	cfg.WarnActionBanMinutes = penaltyCfg.WarnActionBanMinutes
	cfg.MuteMinutes = penaltyCfg.MuteMinutes
	cfg.BanMinutes = penaltyCfg.BanMinutes
	if cfg.WarnDeleteMinutes < 0 {
		cfg.WarnDeleteMinutes = 10
	}
	if cfg.WarnDeleteMinutes > 1440 {
		cfg.WarnDeleteMinutes = 10
	}
	return cfg
}

func (s *Service) getBannedWordConfig(groupID uint) (bannedWordConfig, error) {
	cfg := defaultBannedWordConfig()
	entry, err := s.readFeatureConfigEntry(groupID, featureBannedWords)
	if err != nil {
		return cfg, err
	}
	if !entry.Exists {
		if saveErr := s.saveBannedWordConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
		return cfg, nil
	}
	rawCfg := cfg
	if entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &rawCfg)
	}
	cfg = normalizeBannedWordConfig(rawCfg)
	if entry.Config == "" || rawCfg != cfg {
		if saveErr := s.saveBannedWordConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
	}
	return cfg, nil
}

func (s *Service) saveBannedWordConfig(groupID uint, cfg bannedWordConfig) error {
	cfg = normalizeBannedWordConfig(cfg)
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featureBannedWords, string(b))
}

func defaultInviteConfig() inviteConfig {
	return inviteConfig{
		ExpireDate:    0,
		MemberLimit:   0,
		GenerateLimit: 0,
	}
}

func normalizeInviteConfig(cfg inviteConfig) inviteConfig {
	if cfg.ExpireDate < 0 {
		cfg.ExpireDate = 0
	}
	if cfg.MemberLimit < 0 {
		cfg.MemberLimit = 0
	}
	if cfg.MemberLimit > 99999 {
		cfg.MemberLimit = 99999
	}
	if cfg.GenerateLimit < 0 {
		cfg.GenerateLimit = 0
	}
	return cfg
}

func (s *Service) getInviteConfig(groupID uint) (inviteConfig, error) {
	cfg := defaultInviteConfig()
	entry, err := s.readFeatureConfigEntry(groupID, featureInvite)
	if err != nil {
		return cfg, err
	}
	if !entry.Exists {
		if saveErr := s.saveInviteConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
		return cfg, nil
	}
	rawCfg := cfg
	if entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &rawCfg)
	}
	cfg = normalizeInviteConfig(rawCfg)
	if entry.Config == "" || rawCfg != cfg {
		if saveErr := s.saveInviteConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
	}
	return cfg, nil
}

func (s *Service) saveInviteConfig(groupID uint, cfg inviteConfig) error {
	cfg = normalizeInviteConfig(cfg)
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featureInvite, string(b))
}

func defaultPointsConfig() pointsConfig {
	return pointsConfig{
		CheckinKeyword: "签到",
		CheckinReward:  1,
		MessageReward:  1,
		MessageDaily:   0,
		MessageMinLen:  0,
		InviteReward:   1,
		InviteDaily:    0,
		BalanceAlias:   "积分",
		RankAlias:      "积分排行",
		LotteryCost:    1,
	}
}

func normalizePointsConfig(cfg pointsConfig) pointsConfig {
	if strings.TrimSpace(cfg.CheckinKeyword) == "" {
		cfg.CheckinKeyword = "签到"
	}
	cfg.CheckinKeyword = strings.TrimSpace(cfg.CheckinKeyword)
	if cfg.CheckinReward <= 0 {
		cfg.CheckinReward = 1
	}
	if cfg.CheckinReward > 100000 {
		cfg.CheckinReward = 100000
	}
	if cfg.MessageReward <= 0 {
		cfg.MessageReward = 1
	}
	if cfg.MessageReward > 100000 {
		cfg.MessageReward = 100000
	}
	if cfg.MessageDaily < 0 {
		cfg.MessageDaily = 0
	}
	if cfg.MessageMinLen < 0 {
		cfg.MessageMinLen = 0
	}
	if cfg.InviteReward < 0 {
		cfg.InviteReward = 0
	}
	if cfg.InviteReward > 100000 {
		cfg.InviteReward = 100000
	}
	if cfg.InviteDaily < 0 {
		cfg.InviteDaily = 0
	}
	if strings.TrimSpace(cfg.BalanceAlias) == "" {
		cfg.BalanceAlias = "积分"
	}
	cfg.BalanceAlias = strings.TrimSpace(cfg.BalanceAlias)
	if strings.TrimSpace(cfg.RankAlias) == "" {
		cfg.RankAlias = "积分排行"
	}
	cfg.RankAlias = strings.TrimSpace(cfg.RankAlias)
	if cfg.LotteryCost < 0 {
		cfg.LotteryCost = 0
	}
	if cfg.LotteryCost > 100000 {
		cfg.LotteryCost = 100000
	}
	return cfg
}

func (s *Service) getPointsConfig(groupID uint) (pointsConfig, error) {
	cfg := defaultPointsConfig()
	entry, err := s.readFeatureConfigEntry(groupID, featurePoints)
	if err != nil {
		return cfg, err
	}
	if !entry.Exists {
		if saveErr := s.savePointsConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
		return cfg, nil
	}
	rawCfg := cfg
	if entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &rawCfg)
	}
	cfg = normalizePointsConfig(rawCfg)
	if entry.Config == "" || !reflect.DeepEqual(rawCfg, cfg) {
		if saveErr := s.savePointsConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
	}
	return cfg, nil
}

func (s *Service) savePointsConfig(groupID uint, cfg pointsConfig) error {
	cfg = normalizePointsConfig(cfg)
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featurePoints, string(b))
}

func (s *Service) getKeywordMonitorConfig(groupID uint) (keywordMonitorConfig, error) {
	cfg := keywordMonitorConfig{Keywords: []string{}}
	entry, err := s.readFeatureConfigEntry(groupID, featureKeywordMonitor)
	if err != nil {
		return cfg, err
	}
	if entry.Exists && entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) saveKeywordMonitorConfig(groupID uint, cfg keywordMonitorConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featureKeywordMonitor, string(b))
}

func (s *Service) getPollMeta(groupID uint) (pollMeta, error) {
	cfg := pollMeta{}
	entry, err := s.readFeatureConfigEntry(groupID, featurePollMeta)
	if err != nil {
		return cfg, err
	}
	if entry.Exists && entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) savePollMeta(groupID uint, cfg pollMeta) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featurePollMeta, string(b))
}

func (s *Service) getRBACConfig(groupID uint) (rbacConfig, error) {
	cfg := rbacConfig{Roles: map[string]string{}, FeatureACL: map[string][]string{}}
	entry, err := s.readFeatureConfigEntry(groupID, featureRBAC)
	if err != nil {
		return cfg, err
	}
	if entry.Exists && entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &cfg)
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
	return s.saveFeatureConfigEntry(groupID, featureRBAC, string(b))
}

func (s *Service) getNewbieLimitMinutes(groupID uint) (int, error) {
	cfg := newbieLimitConfig{Minutes: 10}
	entry, err := s.readFeatureConfigEntry(groupID, featureNewbieLimit)
	if err != nil {
		return 10, err
	}
	if entry.Exists && entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &cfg)
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
	return s.saveFeatureConfigEntry(groupID, featureNewbieLimit, string(b))
}

func defaultWordCloudConfig() wordCloudConfig {
	return wordCloudConfig{
		PushHour:    18,
		PushMinute:  0,
		LastPushDay: "",
	}
}

func normalizeWordCloudConfig(cfg wordCloudConfig) wordCloudConfig {
	if cfg.PushHour == -1 {
		cfg.PushMinute = 0
	} else {
		if cfg.PushHour < -1 || cfg.PushHour > 23 {
			cfg.PushHour = 18
		}
		if cfg.PushMinute < 0 || cfg.PushMinute > 59 {
			cfg.PushMinute = 0
		}
	}
	cfg.LastPushDay = strings.TrimSpace(cfg.LastPushDay)
	return cfg
}

func (s *Service) getWordCloudConfig(groupID uint) (wordCloudConfig, error) {
	cfg := defaultWordCloudConfig()
	entry, err := s.readFeatureConfigEntry(groupID, featureWordCloud)
	if err != nil {
		return cfg, err
	}
	if !entry.Exists {
		if saveErr := s.saveWordCloudConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
		return cfg, nil
	}
	rawCfg := cfg
	if entry.Config != "" {
		_ = json.Unmarshal([]byte(entry.Config), &rawCfg)
	}
	cfg = normalizeWordCloudConfig(rawCfg)
	if entry.Config == "" || !reflect.DeepEqual(rawCfg, cfg) {
		if saveErr := s.saveWordCloudConfig(groupID, cfg); saveErr != nil {
			return cfg, saveErr
		}
	}
	return cfg, nil
}

func (s *Service) saveWordCloudConfig(groupID uint, cfg wordCloudConfig) error {
	cfg = normalizeWordCloudConfig(cfg)
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.saveFeatureConfigEntry(groupID, featureWordCloud, string(b))
}

func onOff(v bool) string {
	if v {
		return "开启"
	}
	return "关闭"
}
