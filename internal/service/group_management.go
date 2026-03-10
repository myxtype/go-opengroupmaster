package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"supervisor/internal/model"
	"supervisor/internal/repository"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gorm.io/gorm"
)

var ErrSyncAdminsForbidden = errors.New("sync admins forbidden")

func (s *Service) RegisterGroupAndUser(msg *models.Message) (*model.Group, *model.User, error) {
	user, err := s.repo.UpsertUserFromTG(msg.From)
	if err != nil {
		return nil, nil, err
	}
	group, err := s.repo.UpsertGroup(&msg.Chat)
	if err != nil {
		return nil, nil, err
	}
	return group, user, nil
}

func (s *Service) RegisterGroup(chat *models.Chat) (*model.Group, error) {
	return s.repo.UpsertGroup(chat)
}

func (s *Service) SyncGroupAdmins(bot *tgbot.Bot, group *model.Group) error {
	return s.syncGroupAdmins(bot, group, false)
}

func (s *Service) ForceSyncGroupAdminsByTGGroupID(bot *tgbot.Bot, tgGroupID, requesterTGUserID int64) error {
	if bot == nil || tgGroupID == 0 || requesterTGUserID == 0 {
		return errors.New("invalid force sync params")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	member, err := bot.GetChatMember(context.Background(), &tgbot.GetChatMemberParams{
		ChatID: tgGroupID,
		UserID: requesterTGUserID,
	})
	if err != nil {
		return err
	}
	if member == nil || !isGroupAdminChatMember(*member) {
		return ErrSyncAdminsForbidden
	}
	return s.syncGroupAdmins(bot, group, true)
}

func (s *Service) syncGroupAdmins(bot *tgbot.Bot, group *model.Group, force bool) error {
	if bot == nil || group == nil {
		return errors.New("invalid sync params")
	}
	if !s.beginAdminSync(group.TGGroupID, force) {
		return nil
	}
	admins, err := bot.GetChatAdministrators(context.Background(), &tgbot.GetChatAdministratorsParams{ChatID: group.TGGroupID})
	if err != nil {
		s.finishAdminSync(group.TGGroupID, false)
		return err
	}
	rows := make([]model.GroupAdmin, 0, len(admins))
	for _, a := range admins {
		au := chatMemberUser(a)
		if au == nil {
			continue
		}
		u, err := s.repo.UpsertUserFromTG(au)
		if err != nil {
			s.finishAdminSync(group.TGGroupID, false)
			return err
		}
		role := "admin"
		if a.Type == models.ChatMemberTypeOwner {
			role = "owner"
		}
		rows = append(rows, model.GroupAdmin{GroupID: group.ID, UserID: u.ID, Role: role})
	}
	if err := s.repo.ReplaceGroupAdmins(group.ID, rows); err != nil {
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
	groupTZ := formatUTCOffset(normalizeGroupTimezoneOffsetMinutes(group.TimezoneOffsetMinutes))
	nightDesc := fmt.Sprintf("%s，%s，%s", groupTZ, formatNightWindow(nightCfg.StartHour, nightCfg.EndHour), nightModeLabelForSummary(nightCfg.Mode))
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

func (s *Service) beginAdminSync(tgGroupID int64, force bool) bool {
	now := time.Now()
	s.adminSyncMu.Lock()
	defer s.adminSyncMu.Unlock()
	if !force {
		last, ok := s.adminSyncAt[tgGroupID]
		if ok && now.Sub(last) < s.adminSyncEvery {
			return false
		}
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

func (s *Service) MuteMemberByTGGroupID(bot *tgbot.Bot, tgGroupID, targetTGUserID int64, minutes int) error {
	if bot == nil || targetTGUserID == 0 || minutes <= 0 {
		return errors.New("invalid mute params")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	permissions := models.ChatPermissions{}
	_, err = bot.RestrictChatMember(context.Background(), &tgbot.RestrictChatMemberParams{
		ChatID:      tgGroupID,
		UserID:      targetTGUserID,
		UntilDate:   int(time.Now().Add(time.Duration(minutes) * time.Minute).Unix()),
		Permissions: &permissions,
	})
	if err != nil {
		return err
	}
	targetID := uint(0)
	if u, uErr := s.repo.EnsureUserByTGUserID(targetTGUserID); uErr == nil {
		targetID = u.ID
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("cmd_mute_%d", minutes), 0, targetID)
	return nil
}

func (s *Service) UnmuteMemberByTGGroupID(bot *tgbot.Bot, tgGroupID, targetTGUserID int64) error {
	if bot == nil || targetTGUserID == 0 {
		return errors.New("invalid unmute params")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	permissions := models.ChatPermissions{
		CanSendMessages:       true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
	}
	_, err = bot.RestrictChatMember(context.Background(), &tgbot.RestrictChatMemberParams{
		ChatID:      tgGroupID,
		UserID:      targetTGUserID,
		Permissions: &permissions,
	})
	if err != nil {
		return err
	}
	targetID := uint(0)
	if u, uErr := s.repo.EnsureUserByTGUserID(targetTGUserID); uErr == nil {
		targetID = u.ID
	}
	_ = s.repo.CreateLog(group.ID, "cmd_unmute", 0, targetID)
	return nil
}

func (s *Service) BanMemberByTGGroupID(bot *tgbot.Bot, tgGroupID, targetTGUserID int64, minutes int) error {
	if bot == nil || targetTGUserID == 0 {
		return errors.New("invalid ban params")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	req := &tgbot.BanChatMemberParams{ChatID: tgGroupID, UserID: targetTGUserID}
	if minutes > 0 {
		req.UntilDate = int(time.Now().Add(time.Duration(minutes) * time.Minute).Unix())
	}
	_, err = bot.BanChatMember(context.Background(), req)
	if err != nil {
		return err
	}
	targetID := uint(0)
	if u, uErr := s.repo.EnsureUserByTGUserID(targetTGUserID); uErr == nil {
		targetID = u.ID
	}
	action := "cmd_ban_perm"
	if minutes > 0 {
		action = fmt.Sprintf("cmd_ban_%d", minutes)
	}
	_ = s.repo.CreateLog(group.ID, action, 0, targetID)
	return nil
}

func (s *Service) UnbanMemberByTGGroupID(bot *tgbot.Bot, tgGroupID, targetTGUserID int64) error {
	if bot == nil || targetTGUserID == 0 {
		return errors.New("invalid unban params")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	_, err = bot.UnbanChatMember(context.Background(), &tgbot.UnbanChatMemberParams{
		ChatID: tgGroupID,
		UserID: targetTGUserID,
	})
	if err != nil {
		return err
	}
	targetID := uint(0)
	if u, uErr := s.repo.EnsureUserByTGUserID(targetTGUserID); uErr == nil {
		targetID = u.ID
	}
	_ = s.repo.CreateLog(group.ID, "cmd_unban", 0, targetID)
	return nil
}

func (s *Service) KickMemberByTGGroupID(bot *tgbot.Bot, tgGroupID, targetTGUserID int64) error {
	if bot == nil || targetTGUserID == 0 {
		return errors.New("invalid kick params")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	_, err = bot.BanChatMember(context.Background(), &tgbot.BanChatMemberParams{
		ChatID: tgGroupID,
		UserID: targetTGUserID,
	})
	if err != nil {
		return err
	}
	_, err = bot.UnbanChatMember(context.Background(), &tgbot.UnbanChatMemberParams{
		ChatID: tgGroupID,
		UserID: targetTGUserID,
	})
	if err != nil {
		return err
	}
	targetID := uint(0)
	if u, uErr := s.repo.EnsureUserByTGUserID(targetTGUserID); uErr == nil {
		targetID = u.ID
	}
	_ = s.repo.CreateLog(group.ID, "cmd_kick", 0, targetID)
	return nil
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

func (s *Service) SendWelcomePreviewByTGGroupID(bot *tgbot.Bot, tgGroupID, previewChatID, tgUserID int64) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return err
	}
	previewUser := models.User{
		ID:        tgUserID,
		FirstName: "预览用户",
	}
	if member, mErr := bot.GetChatMember(context.Background(), &tgbot.GetChatMemberParams{
		ChatID: tgGroupID,
		UserID: tgUserID,
	}); mErr == nil {
		if u := chatMemberUser(*member); u != nil {
			previewUser = *u
		}
	}
	return s.sendWelcomePreview(bot, previewChatID, []models.User{previewUser}, cfg)
}

func chatMemberStatus(cm models.ChatMember) string {
	return string(cm.Type)
}

func isGroupAdminChatMember(cm models.ChatMember) bool {
	return cm.Type == models.ChatMemberTypeOwner || cm.Type == models.ChatMemberTypeAdministrator
}

func chatMemberUser(cm models.ChatMember) *models.User {
	switch cm.Type {
	case models.ChatMemberTypeOwner:
		if cm.Owner != nil {
			return cm.Owner.User
		}
	case models.ChatMemberTypeAdministrator:
		if cm.Administrator != nil {
			u := cm.Administrator.User
			return &u
		}
	case models.ChatMemberTypeMember:
		if cm.Member != nil {
			return cm.Member.User
		}
	case models.ChatMemberTypeRestricted:
		if cm.Restricted != nil {
			return cm.Restricted.User
		}
	case models.ChatMemberTypeLeft:
		if cm.Left != nil {
			return cm.Left.User
		}
	case models.ChatMemberTypeBanned:
		if cm.Banned != nil {
			return cm.Banned.User
		}
	}
	return nil
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
