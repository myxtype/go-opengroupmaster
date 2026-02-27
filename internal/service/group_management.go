package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"supervisor/internal/model"
	"supervisor/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func (s *Service) RegisterGroupAndUser(msg *tgbotapi.Message) (*model.Group, *model.User, error) {
	user, err := s.repo.UpsertUserFromTG(msg.From)
	if err != nil {
		return nil, nil, err
	}
	group, err := s.repo.UpsertGroup(msg.Chat)
	if err != nil {
		return nil, nil, err
	}
	return group, user, nil
}

func (s *Service) SyncGroupAdmins(bot *tgbotapi.BotAPI, group *model.Group) error {
	if !s.tryBeginAdminSync(group.TGGroupID) {
		return nil
	}
	admins, err := bot.GetChatAdministrators(tgbotapi.ChatAdministratorsConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: group.TGGroupID}})
	if err != nil {
		s.finishAdminSync(group.TGGroupID, false)
		return err
	}
	rows := make([]model.GroupAdmin, 0, len(admins))
	for _, a := range admins {
		u, err := s.repo.UpsertUserFromTG(a.User)
		if err != nil {
			s.finishAdminSync(group.TGGroupID, false)
			return err
		}
		role := "admin"
		if a.Status == "creator" {
			role = "owner"
		}
		rows = append(rows, model.GroupAdmin{GroupID: group.ID, UserID: u.ID, Role: role})
	}
	if err := s.repo.ReplaceGroupAdmins(group.ID, rows); err != nil {
		s.finishAdminSync(group.TGGroupID, false)
		return err
	}
	if err := s.repo.CreateDefaultDataIfEmpty(group.ID); err != nil {
		s.finishAdminSync(group.TGGroupID, false)
		return err
	}
	s.finishAdminSync(group.TGGroupID, true)
	return nil
}

func (s *Service) ListManageableGroups(tgUserID int64) ([]model.Group, error) {
	return s.repo.ListGroupsByAdminTGUserID(tgUserID)
}
func (s *Service) Repo() *repository.Repository {
	return s.repo
}

func (s *Service) IsAdminByTGGroupID(tgGroupID, tgUserID int64) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	return s.repo.CheckAdmin(group.ID, tgUserID)
}

func (s *Service) GroupPanelSummary(tgGroupID int64) (string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	autoCount, err := s.repo.CountAutoReplies(group.ID)
	if err != nil {
		return "", err
	}
	bwCount, err := s.repo.CountBannedWords(group.ID)
	if err != nil {
		return "", err
	}
	welcomeEnabled, err := s.IsFeatureEnabled(group.ID, featureWelcome, true)
	if err != nil {
		return "", err
	}
	antiSpamEnabled, err := s.IsFeatureEnabled(group.ID, featureAntiSpam, false)
	if err != nil {
		return "", err
	}
	antiFloodEnabled, err := s.IsFeatureEnabled(group.ID, featureAntiFlood, false)
	if err != nil {
		return "", err
	}
	verifyEnabled, err := s.IsFeatureEnabled(group.ID, featureJoinVerify, false)
	if err != nil {
		return "", err
	}
	newbieEnabled, err := s.IsFeatureEnabled(group.ID, featureNewbieLimit, false)
	if err != nil {
		return "", err
	}
	pointsEnabled, err := s.IsFeatureEnabled(group.ID, featurePoints, false)
	if err != nil {
		return "", err
	}
	nightState, err := s.getNightModeState(group.ID)
	if err != nil {
		return "", err
	}
	verifyCfg, _ := s.getJoinVerifyConfig(group.ID)
	newbieMinutes, _ := s.getNewbieLimitMinutes(group.ID)

	welcomeText := onOff(welcomeEnabled)
	antiSpamText := onOff(antiSpamEnabled)
	antiFloodText := onOff(antiFloodEnabled)
	verifyText := onOff(verifyEnabled)
	newbieText := onOff(newbieEnabled)
	pointsText := onOff(pointsEnabled)
	nightCfg := normalizeNightModeConfig(nightState.Config)
	nightText := onOff(nightState.Enabled)
	nightDesc := fmt.Sprintf("%s，%s，%s", formatUTCOffset(nightCfg.TimezoneOffsetMinutes), formatNightWindow(nightCfg.StartHour, nightCfg.EndHour), nightModeLabelForSummary(nightCfg.Mode))
	lines := []string{
		fmt.Sprintf("🏠 %s", group.Title),
		fmt.Sprintf("🆔 群ID: %d", group.TGGroupID),
		"",
		"【内容】",
		fmt.Sprintf("自动回复: %d 条   违禁词: %d 条", autoCount, bwCount),
		fmt.Sprintf("欢迎消息: %s", welcomeText),
		fmt.Sprintf("积分系统: %s", pointsText),
		"",
		"【风控】",
		fmt.Sprintf("反垃圾: %s   反刷屏: %s", antiSpamText, antiFloodText),
		fmt.Sprintf("夜间模式: %s（%s）", nightText, nightDesc),
		fmt.Sprintf("进群验证: %s（%s）", verifyText, verifyTypeLabelForSummary(verifyCfg.Type)),
		fmt.Sprintf("新成员限制: %s（%d 分钟）", newbieText, newbieMinutes),
		"",
		"点击下方按钮进入对应二级面板",
	}
	return strings.Join(lines, "\n"), nil
}

func nightModeLabelForSummary(mode string) string {
	if mode == nightModeGlobalMute {
		return "全局禁言"
	}
	return "删除媒体"
}

func verifyTypeLabelForSummary(v string) string {
	switch v {
	case "math":
		return "数学题"
	case "captcha":
		return "验证码"
	case "zhchar":
		return "中文字符验证码"
	case "zhvoice":
		return "中文语音验证码"
	default:
		return "按钮"
	}
}

func (s *Service) IsFeatureEnabled(groupID uint, featureKey string, defaultValue bool) (bool, error) {
	setting, err := s.repo.GetGroupSetting(groupID, featureKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultValue, nil
		}
		return false, err
	}
	return setting.Enabled, nil
}

func (s *Service) tryBeginAdminSync(tgGroupID int64) bool {
	now := time.Now()
	s.adminSyncMu.Lock()
	defer s.adminSyncMu.Unlock()
	last, ok := s.adminSyncAt[tgGroupID]
	if ok && now.Sub(last) < s.adminSyncEvery {
		return false
	}
	s.adminSyncAt[tgGroupID] = now
	return true
}

func (s *Service) finishAdminSync(tgGroupID int64, ok bool) {
	if ok {
		return
	}
	s.adminSyncMu.Lock()
	defer s.adminSyncMu.Unlock()
	delete(s.adminSyncAt, tgGroupID)
}

func (s *Service) ToggleWelcomeByTGGroupID(tgGroupID int64) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureWelcome, true)
	if err != nil {
		return false, err
	}
	next := !enabled
	if err := s.repo.UpsertFeatureEnabled(group.ID, featureWelcome, next); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, "toggle_welcome", 0, 0)
	return next, nil
}

func (s *Service) SetWelcomeEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	if err := s.repo.UpsertFeatureEnabled(group.ID, featureWelcome, enabled); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_welcome_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) SetWelcomeTextByTGGroupID(tgGroupID int64, text string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return err
	}
	cfg.Text = text
	if err := s.saveWelcomeConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "set_welcome_text", 0, 0)
}

func (s *Service) WelcomeViewByTGGroupID(tgGroupID int64) (*welcomeConfig, bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, false, err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return nil, false, err
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureWelcome, true)
	if err != nil {
		return nil, false, err
	}
	return &cfg, enabled, nil
}

func (s *Service) ToggleWelcomeModeByTGGroupID(tgGroupID int64) (string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return "", err
	}
	if cfg.Mode == "verify" {
		cfg.Mode = "join"
	} else {
		cfg.Mode = "verify"
	}
	if err := s.saveWelcomeConfig(group.ID, cfg); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "toggle_welcome_mode_"+cfg.Mode, 0, 0)
	return cfg.Mode, nil
}

func (s *Service) CycleWelcomeDeleteMinutesByTGGroupID(tgGroupID int64) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return 0, err
	}
	next := 1
	switch cfg.DeleteMinutes {
	case 0:
		next = 1
	case 1:
		next = 5
	case 5:
		next = 10
	case 10:
		next = 30
	default:
		next = 0
	}
	cfg.DeleteMinutes = next
	if err := s.saveWelcomeConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_welcome_delete_%d", next), 0, 0)
	return next, nil
}

func (s *Service) SetWelcomeDeleteMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedWelcomeDeleteMinutes(minutes) {
		return 0, errors.New("invalid welcome delete minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.DeleteMinutes = minutes
	if err := s.saveWelcomeConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_welcome_delete_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) SetWelcomeMediaByTGGroupID(tgGroupID int64, fileID string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return err
	}
	cfg.MediaFileID = fileID
	if err := s.saveWelcomeConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "set_welcome_media", 0, 0)
}

func (s *Service) SetWelcomeButtonsByTGGroupID(tgGroupID int64, raw string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return err
	}
	rows, err := parseWelcomeButtonsInput(raw)
	if err != nil {
		return err
	}
	cfg.ButtonRows = rows
	if err := s.saveWelcomeConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "set_welcome_buttons_multi", 0, 0)
}

func (s *Service) ClearWelcomeButtonsByTGGroupID(tgGroupID int64) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return err
	}
	cfg.ButtonRows = [][]welcomeButton{}
	if err := s.saveWelcomeConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "clear_welcome_buttons", 0, 0)
}

func (s *Service) SendWelcomePreviewByTGGroupID(bot *tgbotapi.BotAPI, tgGroupID, previewChatID, tgUserID int64) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return err
	}
	previewUser := tgbotapi.User{
		ID:        tgUserID,
		FirstName: "预览用户",
	}
	if member, mErr := bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: tgGroupID, UserID: tgUserID},
	}); mErr == nil && member.User != nil {
		previewUser = *member.User
	}
	return s.sendWelcomePreview(bot, previewChatID, []tgbotapi.User{previewUser}, cfg)
}

func (s *Service) ToggleFeatureByTGGroupID(tgGroupID int64, featureKey string) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureKey, false)
	if err != nil {
		return false, err
	}
	next := !enabled
	if err := s.repo.UpsertFeatureEnabled(group.ID, featureKey, next); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, "toggle_"+featureKey, 0, 0)
	return next, nil
}
func (s *Service) ToggleJoinVerifyTypeByTGGroupID(tgGroupID int64) (string, error) {
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
	case "zhchar":
		cfg.Type = "zhvoice"
	default:
		cfg.Type = "button"
	}
	if err := s.saveJoinVerifyConfig(group.ID, cfg); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "switch_verify_type_"+cfg.Type, 0, 0)
	return cfg.Type, nil
}

func (s *Service) CycleNewbieLimitMinutesByTGGroupID(tgGroupID int64) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	minutes, err := s.getNewbieLimitMinutes(group.ID)
	if err != nil {
		return 0, err
	}
	next := 10
	switch minutes {
	case 10:
		next = 30
	case 30:
		next = 60
	default:
		next = 10
	}
	if err := s.saveNewbieLimitMinutes(group.ID, next); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_newbie_limit_%d", next), 0, 0)
	return next, nil
}

func (s *Service) SetNewbieLimitMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedNewbieLimitMinutes(minutes) {
		return 0, errors.New("invalid newbie limit minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	if err := s.saveNewbieLimitMinutes(group.ID, minutes); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_newbie_limit_%d", minutes), 0, 0)
	return minutes, nil
}

func isAllowedWelcomeDeleteMinutes(minutes int) bool {
	switch minutes {
	case 0, 1, 5, 10, 30:
		return true
	default:
		return false
	}
}

func isAllowedNewbieLimitMinutes(minutes int) bool {
	switch minutes {
	case 10, 30, 60:
		return true
	default:
		return false
	}
}
