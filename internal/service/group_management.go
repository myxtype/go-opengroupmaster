package service

import (
	"errors"
	"fmt"

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
	admins, err := bot.GetChatAdministrators(tgbotapi.ChatAdministratorsConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: group.TGGroupID}})
	if err != nil {
		return err
	}
	rows := make([]model.GroupAdmin, 0, len(admins))
	for _, a := range admins {
		u, err := s.repo.UpsertUserFromTG(a.User)
		if err != nil {
			return err
		}
		role := "admin"
		if a.Status == "creator" {
			role = "owner"
		}
		rows = append(rows, model.GroupAdmin{GroupID: group.ID, UserID: u.ID, Role: role})
	}
	if err := s.repo.ReplaceGroupAdmins(group.ID, rows); err != nil {
		return err
	}
	return s.repo.CreateDefaultDataIfEmpty(group.ID)
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
	verifyCfg, _ := s.getJoinVerifyConfig(group.ID)
	newbieMinutes, _ := s.getNewbieLimitMinutes(group.ID)

	welcomeText := onOff(welcomeEnabled)
	antiSpamText := onOff(antiSpamEnabled)
	antiFloodText := onOff(antiFloodEnabled)
	verifyText := onOff(verifyEnabled)
	newbieText := onOff(newbieEnabled)
	return fmt.Sprintf(
		"群组：%s\n群ID：%d\n自动回复：%d 条\n违禁词：%d 条\n欢迎消息：%s\n反垃圾：%s\n反刷屏：%s\n进群验证：%s（%s）\n新成员限制：%s（%d 分钟）",
		group.Title, group.TGGroupID, autoCount, bwCount, welcomeText, antiSpamText, antiFloodText, verifyText, verifyCfg.Type, newbieText, newbieMinutes,
	), nil
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

func (s *Service) SetWelcomeButtonByTGGroupID(tgGroupID int64, text, url string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return err
	}
	cfg.ButtonText = text
	cfg.ButtonURL = url
	if err := s.saveWelcomeConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "set_welcome_button", 0, 0)
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
