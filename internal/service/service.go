package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"supervisor/internal/model"
	"supervisor/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

const featureWelcome = "welcome"
const featureAntiSpam = "anti_spam"
const featureAntiFlood = "anti_flood"
const featureJoinVerify = "join_verify"
const featureNewbieLimit = "newbie_limit"
const featureSystemClean = "system_clean"
const featureKeywordMonitor = "keyword_monitor"
const featureChain = "chain"
const featurePollMeta = "poll_meta"
const featureRBAC = "rbac"

type verifyPending struct {
	Deadline time.Time
	Mode     string
	Answer   int
}

type joinVerifyConfig struct {
	Type       string `json:"type"`
	TimeoutSec int    `json:"timeout_sec"`
}

type newbieLimitConfig struct {
	Minutes int `json:"minutes"`
}

type systemCleanConfig struct {
	Join  bool `json:"join"`
	Leave bool `json:"leave"`
	Pin   bool `json:"pin"`
	Photo bool `json:"photo"`
	Title bool `json:"title"`
}

type keywordMonitorConfig struct {
	Keywords []string `json:"keywords"`
}

type chainConfig struct {
	Active  bool     `json:"active"`
	Title   string   `json:"title"`
	Entries []string `json:"entries"`
}

type pollMeta struct {
	Question  string `json:"question"`
	MessageID int    `json:"message_id"`
	Active    bool   `json:"active"`
}

type rbacConfig struct {
	Roles      map[string]string   `json:"roles"`
	FeatureACL map[string][]string `json:"feature_acl"`
}

type Service struct {
	repo   *repository.Repository
	logger *log.Logger
	mu     sync.Mutex
	flood  map[string][]int64
	joinAt map[string]time.Time
	verify map[string]verifyPending
}

type AutoReplyPage struct {
	Items    []model.AutoReply
	Page     int
	PageSize int
	Total    int64
}

type BannedWordPage struct {
	Items    []model.BannedWord
	Page     int
	PageSize int
	Total    int64
}

type ScheduledMessagePage struct {
	Items    []model.ScheduledMessage
	Page     int
	PageSize int
	Total    int64
}

type GroupStats struct {
	GroupTitle string
	GroupID    int64
	TopUsers   []UserScore
}

type UserScore struct {
	DisplayName string
	Points      int
}

type LogPage struct {
	Items    []model.Log
	Page     int
	PageSize int
	Total    int64
}

type SystemCleanView struct {
	Join  bool
	Leave bool
	Pin   bool
	Photo bool
	Title bool
}

type ChainView struct {
	Active  bool
	Title   string
	Entries []string
}

func New(repo *repository.Repository, logger *log.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
		flood:  make(map[string][]int64),
		joinAt: make(map[string]time.Time),
		verify: make(map[string]verifyPending),
	}
}

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

func (s *Service) CheckMessageAndRespond(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) error {
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if msg.From != nil {
		blacklisted, err := s.repo.IsGlobalBlacklisted(msg.From.ID)
		if err == nil && blacklisted {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
				UntilDate:        time.Now().Add(24 * time.Hour).Unix(),
			})
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 命中全局黑名单，已移出群组", msg.From.UserName)))
			_ = s.repo.CreateLog(group.ID, "global_blacklist_kick", 0, 0)
			return nil
		}
	}

	if msg.Text != "" {
		handled, err := s.applyModeration(bot, msg, group)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}

		limited, err := s.applyNewbieLimit(bot, msg, group)
		if err != nil {
			return err
		}
		if limited {
			return nil
		}

		_ = s.notifyKeywordMonitor(bot, group, msg)

		banned, err := s.repo.ContainsBannedWord(group.ID, msg.Text)
		if err != nil {
			return err
		}
		if banned {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			warn := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 消息触发违禁词，已删除", msg.From.UserName))
			_, _ = bot.Send(warn)
			_ = s.repo.CreateLog(group.ID, "banned_word_delete", 0, 0)
			return nil
		}

		rule, err := s.repo.MatchAutoReply(group.ID, msg.Text)
		if err != nil {
			return err
		}
		if rule != nil {
			reply := tgbotapi.NewMessage(msg.Chat.ID, rule.Reply)
			reply.ReplyToMessageID = msg.MessageID
			_, _ = bot.Send(reply)
		}
	}

	if msg.From != nil {
		u, err := s.repo.UpsertUserFromTG(msg.From)
		if err == nil {
			_ = s.repo.AddPoints(group.ID, u.ID, 1)
		}
	}

	return nil
}

func (s *Service) applyModeration(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, group *model.Group) (bool, error) {
	if msg.From == nil {
		return false, nil
	}
	isAdmin, err := s.repo.CheckAdmin(group.ID, msg.From.ID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return false, nil
	}

	antiSpam, err := s.IsFeatureEnabled(group.ID, featureAntiSpam, false)
	if err != nil {
		return false, err
	}
	if antiSpam && containsLink(msg.Text) {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 检测到链接，消息已删除", msg.From.UserName)))
		_ = s.repo.CreateLog(group.ID, "anti_spam_delete", 0, 0)
		return true, nil
	}

	antiFlood, err := s.IsFeatureEnabled(group.ID, featureAntiFlood, false)
	if err != nil {
		return false, err
	}
	if antiFlood && s.isFlooding(group.TGGroupID, msg.From.ID) {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		restrict := tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
			UntilDate:        time.Now().Add(60 * time.Second).Unix(),
			Permissions:      &tgbotapi.ChatPermissions{},
		}
		_, _ = bot.Request(restrict)
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 触发反刷屏，已禁言 60 秒", msg.From.UserName)))
		_ = s.repo.CreateLog(group.ID, "anti_flood_restrict", 0, 0)
		return true, nil
	}
	return false, nil
}

func (s *Service) OnNewMembers(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) error {
	if len(msg.NewChatMembers) == 0 {
		return nil
	}
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		return nil
	}
	for _, m := range msg.NewChatMembers {
		s.markJoin(group.TGGroupID, m.ID)
	}

	verifyEnabled, err := s.IsFeatureEnabled(group.ID, featureJoinVerify, false)
	if err == nil && verifyEnabled {
		cfg, _ := s.getJoinVerifyConfig(group.ID)
		timeout := cfg.TimeoutSec
		if timeout <= 0 {
			timeout = 120
		}
		for _, m := range msg.NewChatMembers {
			if m.IsBot {
				continue
			}
			restrict := tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: m.ID},
				UntilDate:        time.Now().Add(time.Duration(timeout) * time.Second).Unix(),
				Permissions:      &tgbotapi.ChatPermissions{},
			}
			_, _ = bot.Request(restrict)

			pending := verifyPending{Deadline: time.Now().Add(time.Duration(timeout) * time.Second), Mode: cfg.Type}
			verifyText := fmt.Sprintf("新成员 @%s 请在 %d 秒内完成验证，否则将被移出。", m.UserName, timeout)
			keyboard := tgbotapi.NewInlineKeyboardMarkup()
			if cfg.Type == "math" {
				a := rand.Intn(9) + 1
				b := rand.Intn(9) + 1
				answer := a + b
				pending.Answer = answer
				verifyText = fmt.Sprintf("新成员 @%s 请完成算术验证：%d + %d = ?（%d 秒内）", m.UserName, a, b, timeout)
				options := buildMathOptions(answer)
				row := make([]tgbotapi.InlineKeyboardButton, 0, len(options))
				for _, opt := range options {
					row = append(row, tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(opt), fmt.Sprintf("verify:math:%d:%d:%d", group.TGGroupID, m.ID, opt)))
				}
				keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
			} else {
				keyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("我已验证", fmt.Sprintf("verify:pass:%d:%d", group.TGGroupID, m.ID)),
					),
				)
			}
			s.addVerifyPending(group.TGGroupID, m.ID, pending)
			verifyMsg := tgbotapi.NewMessage(msg.Chat.ID, verifyText)
			verifyMsg.ReplyMarkup = keyboard
			_, _ = bot.Send(verifyMsg)
			_ = s.repo.CreateLog(group.ID, "join_verify_pending", 0, 0)
			go s.verifyTimeoutKick(bot, group.TGGroupID, m.ID, time.Duration(timeout)*time.Second)
		}
	}

	welcomeEnabled, err := s.IsFeatureEnabled(group.ID, featureWelcome, true)
	if err != nil || !welcomeEnabled {
		return err
	}
	text := "欢迎新成员加入，先看群规再发言。"
	_, err = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
	return err
}

func (s *Service) HandleSystemMessageCleanup(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) error {
	if msg == nil || msg.Chat == nil {
		return nil
	}
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		return nil
	}
	cfg, err := s.getSystemCleanConfig(group.ID)
	if err != nil {
		return err
	}

	shouldDelete := false
	action := ""
	switch {
	case len(msg.NewChatMembers) > 0:
		shouldDelete = cfg.Join
		action = "clean_join_message"
	case msg.LeftChatMember != nil:
		shouldDelete = cfg.Leave
		action = "clean_leave_message"
	case msg.PinnedMessage != nil:
		shouldDelete = cfg.Pin
		action = "clean_pin_message"
	case len(msg.NewChatPhoto) > 0 || msg.DeleteChatPhoto:
		shouldDelete = cfg.Photo
		action = "clean_photo_change_message"
	case msg.NewChatTitle != "":
		shouldDelete = cfg.Title
		action = "clean_title_change_message"
	}
	if !shouldDelete || action == "" {
		return nil
	}
	_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	_ = s.repo.CreateLog(group.ID, action, 0, 0)
	return nil
}

func (s *Service) PassVerification(bot *tgbotapi.BotAPI, tgGroupID, tgUserID, actorID int64, answer *int) error {
	if actorID != tgUserID {
		return errors.New("only target user can verify")
	}
	pending, ok := s.getVerifyPending(tgGroupID, tgUserID)
	if !ok || time.Now().After(pending.Deadline) {
		s.popVerifyPending(tgGroupID, tgUserID)
		return errors.New("verification expired")
	}
	if pending.Mode == "math" {
		if answer == nil || *answer != pending.Answer {
			return errors.New("wrong answer")
		}
	}
	if !s.popVerifyPending(tgGroupID, tgUserID) {
		return errors.New("verification expired")
	}

	perms := &tgbotapi.ChatPermissions{
		CanSendMessages:       true,
		CanSendMediaMessages:  true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
	}
	_, err := bot.Request(tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
		Permissions:      perms,
	})
	if err != nil {
		return err
	}
	if group, gErr := s.repo.FindGroupByTGID(tgGroupID); gErr == nil {
		_ = s.repo.CreateLog(group.ID, "join_verify_pass", 0, 0)
	}
	return nil
}

func (s *Service) CreateLotteryByTGGroupID(tgGroupID int64, title string, winners int) (*model.Lottery, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	if winners <= 0 {
		winners = 1
	}
	return s.repo.CreateLottery(group.ID, title, winners)
}

func (s *Service) JoinActiveLotteryByTGGroupID(tgGroupID int64, tgUser *tgbotapi.User) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	lottery, err := s.repo.GetActiveLottery(group.ID)
	if err != nil {
		return err
	}
	u, err := s.repo.UpsertUserFromTG(tgUser)
	if err != nil {
		return err
	}
	return s.repo.JoinLottery(lottery.ID, u.ID)
}

func (s *Service) DrawActiveLotteryByTGGroupID(tgGroupID int64) ([]model.User, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	lottery, err := s.repo.GetActiveLottery(group.ID)
	if err != nil {
		return nil, err
	}
	ids, err := s.repo.ListLotteryParticipantUserIDs(lottery.ID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errors.New("no participants")
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	count := lottery.WinnersCount
	if count > len(ids) {
		count = len(ids)
	}
	ids = ids[:count]
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	winners := make([]model.User, 0, count)
	for _, id := range ids {
		u, err := s.repo.FindUserByID(id)
		if err != nil {
			continue
		}
		winners = append(winners, *u)
	}
	_ = s.repo.CloseLottery(lottery.ID)
	return winners, nil
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

func (s *Service) SystemCleanViewByTGGroupID(tgGroupID int64) (*SystemCleanView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.getSystemCleanConfig(group.ID)
	if err != nil {
		return nil, err
	}
	return &SystemCleanView{
		Join:  cfg.Join,
		Leave: cfg.Leave,
		Pin:   cfg.Pin,
		Photo: cfg.Photo,
		Title: cfg.Title,
	}, nil
}

func (s *Service) ToggleSystemCleanByTGGroupID(tgGroupID int64, key string) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	cfg, err := s.getSystemCleanConfig(group.ID)
	if err != nil {
		return false, err
	}
	var v bool
	switch key {
	case "join":
		cfg.Join = !cfg.Join
		v = cfg.Join
	case "leave":
		cfg.Leave = !cfg.Leave
		v = cfg.Leave
	case "pin":
		cfg.Pin = !cfg.Pin
		v = cfg.Pin
	case "photo":
		cfg.Photo = !cfg.Photo
		v = cfg.Photo
	case "title":
		cfg.Title = !cfg.Title
		v = cfg.Title
	default:
		return false, errors.New("unknown system clean key")
	}
	if err := s.saveSystemCleanConfig(group.ID, cfg); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, "toggle_system_clean_"+key, 0, 0)
	return v, nil
}

func (s *Service) ApplySystemCleanPresetByTGGroupID(tgGroupID int64, preset string) (*SystemCleanView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	var cfg systemCleanConfig
	switch preset {
	case "strict":
		cfg = systemCleanConfig{Join: true, Leave: true, Pin: true, Photo: true, Title: true}
	case "off":
		cfg = systemCleanConfig{Join: false, Leave: false, Pin: false, Photo: false, Title: false}
	default:
		// recommended
		cfg = systemCleanConfig{Join: true, Leave: true, Pin: false, Photo: false, Title: false}
		preset = "recommended"
	}
	if err := s.saveSystemCleanConfig(group.ID, cfg); err != nil {
		return nil, err
	}
	_ = s.repo.CreateLog(group.ID, "apply_system_clean_preset_"+preset, 0, 0)
	return &SystemCleanView{
		Join: cfg.Join, Leave: cfg.Leave, Pin: cfg.Pin, Photo: cfg.Photo, Title: cfg.Title,
	}, nil
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
	if cfg.Type == "math" {
		cfg.Type = "button"
	} else {
		cfg.Type = "math"
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

func (s *Service) AddAutoReplyByTGGroupID(tgGroupID int64, keyword, reply, matchType string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	if matchType == "" {
		matchType = "exact"
	}
	return s.repo.CreateAutoReply(group.ID, keyword, reply, matchType)
}

func (s *Service) AddBannedWordByTGGroupID(tgGroupID int64, word string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.CreateBannedWord(group.ID, word)
}

func (s *Service) ListAutoRepliesByTGGroupID(tgGroupID int64, page, pageSize int) (*AutoReplyPage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListAutoRepliesPage(group.ID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &AutoReplyPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) DeleteAutoReplyByTGGroupID(tgGroupID int64, id uint) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.DeleteAutoReply(group.ID, id)
}

func (s *Service) UpdateAutoReplyByTGGroupID(tgGroupID int64, id uint, keyword, reply, matchType string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	if matchType == "" {
		matchType = "contains"
	}
	return s.repo.UpdateAutoReply(group.ID, id, keyword, reply, matchType)
}

func (s *Service) ListBannedWordsByTGGroupID(tgGroupID int64, page, pageSize int) (*BannedWordPage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListBannedWordsPage(group.ID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &BannedWordPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) DeleteBannedWordByTGGroupID(tgGroupID int64, id uint) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.DeleteBannedWord(group.ID, id)
}

func (s *Service) UpdateBannedWordByTGGroupID(tgGroupID int64, id uint, word string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.UpdateBannedWord(group.ID, id, word)
}

func (s *Service) CreateScheduledMessageByTGGroupID(tgGroupID int64, content, cronExpr string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.CreateScheduledMessage(group.ID, content, cronExpr)
}

func (s *Service) ListScheduledMessagesByTGGroupID(tgGroupID int64, page, pageSize int) (*ScheduledMessagePage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListScheduledMessagesPage(group.ID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &ScheduledMessagePage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) DeleteScheduledMessageByTGGroupID(tgGroupID int64, id uint) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.DeleteScheduledMessage(group.ID, id)
}

func (s *Service) ToggleScheduledMessageByTGGroupID(tgGroupID int64, id uint) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	return s.repo.ToggleScheduledMessage(group.ID, id)
}

func (s *Service) GroupStatsByTGGroupID(tgGroupID int64, limit int) (*GroupStats, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	top, err := s.repo.TopUsersByPoints(group.ID, limit)
	if err != nil {
		return nil, err
	}
	out := &GroupStats{GroupTitle: group.Title, GroupID: group.TGGroupID}
	for _, row := range top {
		user, err := s.repo.FindUserByID(row.UserID)
		if err != nil {
			continue
		}
		name := user.Username
		if name == "" {
			name = strings.TrimSpace(user.FirstName + " " + user.LastName)
		}
		if name == "" {
			name = fmt.Sprintf("uid:%d", user.TGUserID)
		}
		out.TopUsers = append(out.TopUsers, UserScore{DisplayName: name, Points: row.Points})
	}
	return out, nil
}

func (s *Service) ListLogsByTGGroupID(tgGroupID int64, page, pageSize int, action string) (*LogPage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListLogsPage(group.ID, page, pageSize, action)
	if err != nil {
		return nil, err
	}
	return &LogPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) ExportLogsCSVByTGGroupID(tgGroupID int64, action string) (string, []byte, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", nil, err
	}
	items, err := s.repo.ListLogsForExport(group.ID, action, 2000)
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "action", "operator_id", "target_id", "created_at"})
	for _, item := range items {
		_ = w.Write([]string{
			strconv.FormatUint(uint64(item.ID), 10),
			item.Action,
			strconv.FormatUint(uint64(item.OperatorID), 10),
			strconv.FormatUint(uint64(item.TargetID), 10),
			item.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", nil, err
	}
	file := fmt.Sprintf("logs_%d_%s.csv", tgGroupID, time.Now().Format("20060102150405"))
	return file, buf.Bytes(), nil
}

func (s *Service) ParseScheduledInput(raw string) (cronExpr, content string, err error) {
	parts := strings.SplitN(raw, "=>", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid format")
	}
	cronExpr = strings.TrimSpace(parts[0])
	content = strings.TrimSpace(parts[1])
	if cronExpr == "" || content == "" {
		return "", "", errors.New("empty field")
	}
	return cronExpr, content, nil
}

func (s *Service) applyNewbieLimit(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, group *model.Group) (bool, error) {
	enabled, err := s.IsFeatureEnabled(group.ID, featureNewbieLimit, false)
	if err != nil || !enabled {
		return false, err
	}
	if msg.From == nil {
		return false, nil
	}
	joinAt, ok := s.getJoinAt(group.TGGroupID, msg.From.ID)
	if !ok {
		return false, nil
	}
	minutes, _ := s.getNewbieLimitMinutes(group.ID)
	if time.Since(joinAt) > time.Duration(minutes)*time.Minute {
		s.clearJoinAt(group.TGGroupID, msg.From.ID)
		return false, nil
	}
	if !containsLink(msg.Text) && msg.Photo == nil && msg.Video == nil && msg.Document == nil {
		return false, nil
	}
	_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 新成员限制中，暂不可发链接或媒体", msg.From.UserName)))
	_ = s.repo.CreateLog(group.ID, "newbie_limit_delete", 0, 0)
	return true, nil
}

func (s *Service) isFlooding(tgGroupID, tgUserID int64) bool {
	now := time.Now().Unix()
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.flood[key]
	valid := make([]int64, 0, len(items)+1)
	for _, ts := range items {
		if now-ts <= 10 {
			valid = append(valid, ts)
		}
	}
	valid = append(valid, now)
	s.flood[key] = valid
	return len(valid) > 5
}

func containsLink(text string) bool {
	l := strings.ToLower(text)
	return strings.Contains(l, "http://") ||
		strings.Contains(l, "https://") ||
		strings.Contains(l, "t.me/") ||
		strings.Contains(l, "www.")
}

func (s *Service) markJoin(tgGroupID, tgUserID int64) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.joinAt[key] = time.Now()
}

func (s *Service) getJoinAt(tgGroupID, tgUserID int64) (time.Time, bool) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.joinAt[key]
	return t, ok
}

func (s *Service) clearJoinAt(tgGroupID, tgUserID int64) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.joinAt, key)
}

func (s *Service) addVerifyPending(tgGroupID, tgUserID int64, p verifyPending) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verify[key] = p
}

func (s *Service) getVerifyPending(tgGroupID, tgUserID int64) (verifyPending, bool) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.verify[key]
	return p, ok
}

func (s *Service) popVerifyPending(tgGroupID, tgUserID int64) bool {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.verify[key]
	if ok {
		delete(s.verify, key)
	}
	return ok
}

func (s *Service) verifyTimeoutKick(bot *tgbotapi.BotAPI, tgGroupID, tgUserID int64, after time.Duration) {
	time.Sleep(after)
	if !s.popVerifyPending(tgGroupID, tgUserID) {
		return
	}
	_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
		UntilDate:        time.Now().Add(24 * time.Hour).Unix(),
	})
	_, _ = bot.Send(tgbotapi.NewMessage(tgGroupID, fmt.Sprintf("用户 %d 验证超时，已移出群组", tgUserID)))
	if group, err := s.repo.FindGroupByTGID(tgGroupID); err == nil {
		_ = s.repo.CreateLog(group.ID, "join_verify_timeout_kick", 0, 0)
	}
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

func (s *Service) getJoinVerifyConfig(groupID uint) (joinVerifyConfig, error) {
	cfg := joinVerifyConfig{Type: "button", TimeoutSec: 120}
	setting, err := s.repo.GetGroupSetting(groupID, featureJoinVerify)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	if cfg.Type != "math" {
		cfg.Type = "button"
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 120
	}
	return cfg, nil
}

func (s *Service) saveJoinVerifyConfig(groupID uint, cfg joinVerifyConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureJoinVerify, string(b))
}

func (s *Service) getSystemCleanConfig(groupID uint) (systemCleanConfig, error) {
	cfg := systemCleanConfig{
		Join:  true,
		Leave: true,
		Pin:   false,
		Photo: false,
		Title: false,
	}
	setting, err := s.repo.GetGroupSetting(groupID, featureSystemClean)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if saveErr := s.saveSystemCleanConfig(groupID, cfg); saveErr != nil {
				return cfg, saveErr
			}
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) saveSystemCleanConfig(groupID uint, cfg systemCleanConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureSystemClean, string(b))
}

func (s *Service) getKeywordMonitorConfig(groupID uint) (keywordMonitorConfig, error) {
	cfg := keywordMonitorConfig{Keywords: []string{}}
	setting, err := s.repo.GetGroupSetting(groupID, featureKeywordMonitor)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) saveKeywordMonitorConfig(groupID uint, cfg keywordMonitorConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureKeywordMonitor, string(b))
}

func (s *Service) getChainConfig(groupID uint) (chainConfig, error) {
	cfg := chainConfig{Active: false, Title: "", Entries: []string{}}
	setting, err := s.repo.GetGroupSetting(groupID, featureChain)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) saveChainConfig(groupID uint, cfg chainConfig) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featureChain, string(b))
}

func (s *Service) getPollMeta(groupID uint) (pollMeta, error) {
	cfg := pollMeta{}
	setting, err := s.repo.GetGroupSetting(groupID, featurePollMeta)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
	}
	return cfg, nil
}

func (s *Service) savePollMeta(groupID uint, cfg pollMeta) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.repo.UpsertFeatureConfig(groupID, featurePollMeta, string(b))
}

func (s *Service) getRBACConfig(groupID uint) (rbacConfig, error) {
	cfg := rbacConfig{Roles: map[string]string{}, FeatureACL: map[string][]string{}}
	setting, err := s.repo.GetGroupSetting(groupID, featureRBAC)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
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
	return s.repo.UpsertFeatureConfig(groupID, featureRBAC, string(b))
}

func (s *Service) getNewbieLimitMinutes(groupID uint) (int, error) {
	cfg := newbieLimitConfig{Minutes: 10}
	setting, err := s.repo.GetGroupSetting(groupID, featureNewbieLimit)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cfg.Minutes, nil
		}
		return 10, err
	}
	if setting.Config != "" {
		_ = json.Unmarshal([]byte(setting.Config), &cfg)
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
	return s.repo.UpsertFeatureConfig(groupID, featureNewbieLimit, string(b))
}

func buildMathOptions(answer int) []int {
	opts := map[int]struct{}{answer: {}}
	for len(opts) < 4 {
		delta := rand.Intn(7) - 3
		if delta == 0 {
			continue
		}
		v := answer + delta
		if v > 0 {
			opts[v] = struct{}{}
		}
	}
	out := make([]int, 0, len(opts))
	for k := range opts {
		out = append(out, k)
	}
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func onOff(v bool) string {
	if v {
		return "开启"
	}
	return "关闭"
}
