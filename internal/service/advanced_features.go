package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *Service) CanAccessFeatureByTGGroupID(tgGroupID, tgUserID int64, feature string) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	isAdmin, err := s.repo.CheckAdmin(group.ID, tgUserID)
	if err != nil || !isAdmin {
		return false, err
	}
	cfg, err := s.getRBACConfig(group.ID)
	if err != nil {
		return false, err
	}
	role := cfg.Roles[strconv.FormatInt(tgUserID, 10)]
	if role == "" {
		role = "super_admin"
	}
	allowed := cfg.FeatureACL[feature]
	if len(allowed) == 0 {
		return true, nil
	}
	for _, r := range allowed {
		if r == role {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) SetRoleByTGGroupID(tgGroupID, targetTGUserID int64, role string) error {
	if role != "super_admin" && role != "admin" {
		return errors.New("invalid role")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getRBACConfig(group.ID)
	if err != nil {
		return err
	}
	cfg.Roles[strconv.FormatInt(targetTGUserID, 10)] = role
	if err := s.saveRBACConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "rbac_set_role_"+role, 0, 0)
}

func (s *Service) SetFeatureACLByTGGroupID(tgGroupID int64, feature string, roles []string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getRBACConfig(group.ID)
	if err != nil {
		return err
	}
	valid := make([]string, 0, len(roles))
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "super_admin" || r == "admin" {
			valid = append(valid, r)
		}
	}
	if len(valid) == 0 {
		return errors.New("empty acl")
	}
	cfg.FeatureACL[feature] = valid
	if err := s.saveRBACConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "rbac_set_acl_"+feature, 0, 0)
}

func (s *Service) RBACSummaryByTGGroupID(tgGroupID int64) (string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	cfg, err := s.getRBACConfig(group.ID)
	if err != nil {
		return "", err
	}
	lines := []string{"权限分级", "角色（TG 用户ID -> 角色）:"}
	if len(cfg.Roles) == 0 {
		lines = append(lines, "默认：所有 Telegram 管理员 = super_admin")
	} else {
		for uid, role := range cfg.Roles {
			lines = append(lines, fmt.Sprintf("- %s -> %s", uid, role))
		}
	}
	lines = append(lines, "", "功能权限（feature -> roles）:")
	if len(cfg.FeatureACL) == 0 {
		lines = append(lines, "默认：全部功能 super_admin/admin 均可操作")
	} else {
		for f, rs := range cfg.FeatureACL {
			lines = append(lines, fmt.Sprintf("- %s -> %s", f, strings.Join(rs, ",")))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func (s *Service) AddGlobalBlacklist(tgUserID int64, reason string) error {
	return s.repo.AddGlobalBlacklist(tgUserID, reason)
}

func (s *Service) RemoveGlobalBlacklist(tgUserID int64) error {
	return s.repo.RemoveGlobalBlacklist(tgUserID)
}

func (s *Service) ListGlobalBlacklist() ([]model.GlobalBlacklist, error) {
	return s.repo.ListGlobalBlacklist()
}

func (s *Service) SetUserLanguage(tgUserID int64, lang string) error {
	if lang != "zh" && lang != "en" {
		return errors.New("invalid language")
	}
	return s.repo.SetUserLanguage(tgUserID, lang)
}

func (s *Service) GetUserLanguage(tgUserID int64) (string, error) {
	return s.repo.GetUserLanguage(tgUserID)
}
func (s *Service) CreateInviteLinkByTGGroupID(bot *tgbotapi.BotAPI, tgGroupID int64, expireHours, memberLimit int) (string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	cfg := tgbotapi.CreateChatInviteLinkConfig{
		ChatConfig: tgbotapi.ChatConfig{ChatID: tgGroupID},
	}
	if expireHours > 0 {
		cfg.ExpireDate = int(time.Now().Add(time.Duration(expireHours) * time.Hour).Unix())
	}
	if memberLimit > 0 {
		cfg.MemberLimit = memberLimit
	}
	resp, err := bot.Request(cfg)
	if err != nil {
		return "", err
	}
	var chatInvite tgbotapi.ChatInviteLink
	if err := json.Unmarshal(resp.Result, &chatInvite); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "create_invite_link", 0, 0)
	return chatInvite.InviteLink, nil
}

func (s *Service) StartChainByTGGroupID(tgGroupID int64, title string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg := chainConfig{Active: true, Title: title, Entries: []string{}}
	if err := s.saveChainConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "chain_start", 0, 0)
}

func (s *Service) AddChainEntryByTGGroupID(tgGroupID int64, text string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getChainConfig(group.ID)
	if err != nil {
		return err
	}
	if !cfg.Active {
		return errors.New("chain not active")
	}
	cfg.Entries = append(cfg.Entries, text)
	if err := s.saveChainConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "chain_add_entry", 0, 0)
}

func (s *Service) CloseChainByTGGroupID(tgGroupID int64) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getChainConfig(group.ID)
	if err != nil {
		return err
	}
	cfg.Active = false
	if err := s.saveChainConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "chain_close", 0, 0)
}

func (s *Service) ChainViewByTGGroupID(tgGroupID int64) (*ChainView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.getChainConfig(group.ID)
	if err != nil {
		return nil, err
	}
	return &ChainView{Active: cfg.Active, Title: cfg.Title, Entries: cfg.Entries}, nil
}

func (s *Service) CreatePollByTGGroupID(bot *tgbotapi.BotAPI, tgGroupID int64, question string, options []string) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	p := tgbotapi.NewPoll(tgGroupID, question, options...)
	msg, err := bot.Send(p)
	if err != nil {
		return 0, err
	}
	meta := pollMeta{Question: question, MessageID: msg.MessageID, Active: true}
	if err := s.savePollMeta(group.ID, meta); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, "poll_create", 0, 0)
	return msg.MessageID, nil
}

func (s *Service) StopPollByTGGroupID(bot *tgbotapi.BotAPI, tgGroupID int64) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	meta, err := s.getPollMeta(group.ID)
	if err != nil {
		return err
	}
	if !meta.Active || meta.MessageID == 0 {
		return errors.New("no active poll")
	}
	_, err = bot.StopPoll(tgbotapi.NewStopPoll(tgGroupID, meta.MessageID))
	if err != nil {
		return err
	}
	meta.Active = false
	if err := s.savePollMeta(group.ID, meta); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "poll_stop", 0, 0)
}

func (s *Service) AddMonitorKeywordByTGGroupID(tgGroupID int64, keyword string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getKeywordMonitorConfig(group.ID)
	if err != nil {
		return err
	}
	for _, k := range cfg.Keywords {
		if strings.EqualFold(k, keyword) {
			return nil
		}
	}
	cfg.Keywords = append(cfg.Keywords, keyword)
	if err := s.saveKeywordMonitorConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "monitor_add_keyword", 0, 0)
}

func (s *Service) RemoveMonitorKeywordByTGGroupID(tgGroupID int64, keyword string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getKeywordMonitorConfig(group.ID)
	if err != nil {
		return err
	}
	next := make([]string, 0, len(cfg.Keywords))
	for _, k := range cfg.Keywords {
		if !strings.EqualFold(k, keyword) {
			next = append(next, k)
		}
	}
	cfg.Keywords = next
	if err := s.saveKeywordMonitorConfig(group.ID, cfg); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "monitor_remove_keyword", 0, 0)
}

func (s *Service) ListMonitorKeywordsByTGGroupID(tgGroupID int64) ([]string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.getKeywordMonitorConfig(group.ID)
	if err != nil {
		return nil, err
	}
	return cfg.Keywords, nil
}
func (s *Service) notifyKeywordMonitor(bot *tgbotapi.BotAPI, group *model.Group, msg *tgbotapi.Message) error {
	if msg == nil || msg.Text == "" {
		return nil
	}
	cfg, err := s.getKeywordMonitorConfig(group.ID)
	if err != nil {
		return err
	}
	if len(cfg.Keywords) == 0 {
		return nil
	}
	lower := strings.ToLower(msg.Text)
	matched := make([]string, 0, 2)
	for _, k := range cfg.Keywords {
		if k == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(k)) {
			matched = append(matched, k)
		}
	}
	if len(matched) == 0 {
		return nil
	}
	adminIDs, err := s.repo.ListAdminTGUserIDsByGroupID(group.ID)
	if err != nil {
		return err
	}
	notice := fmt.Sprintf("关键词监控命中\\n群：%s(%d)\\n关键词：%s\\n用户：@%s\\n消息：%s",
		group.Title, group.TGGroupID, strings.Join(matched, ","), msg.From.UserName, msg.Text)
	for _, adminID := range adminIDs {
		_, _ = bot.Send(tgbotapi.NewMessage(adminID, notice))
	}
	_ = s.repo.CreateLog(group.ID, "keyword_monitor_hit", 0, 0)
	return nil
}
