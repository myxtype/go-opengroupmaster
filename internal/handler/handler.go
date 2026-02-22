package handler

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"supervisor/internal/model"
	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	groupPageSize  = 6
	rulesPageSize  = 5
	cbMenuGroups   = "menu:groups"
	cbMenuSettings = "menu:settings"
	cbGroupPrefix  = "group:"
	cbFeaturePref  = "feat:"
	cbGroupsPagePF = "menu:groups:page:"
	cbVerifyPF     = "verify:"
)

type renderTarget struct {
	ChatID    int64
	MessageID int
	Edit      bool
}

type pendingInput struct {
	Kind      string
	TGGroupID int64
	RuleID    uint
	Page      int
}

type Handler struct {
	service *service.Service
	logger  *log.Logger

	mu      sync.Mutex
	pending map[int64]pendingInput
}

func New(svc *service.Service, logger *log.Logger) *Handler {
	return &Handler{service: svc, logger: logger, pending: make(map[int64]pendingInput)}
}

func (h *Handler) HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message != nil {
		h.handleMessage(bot, update.Message)
	}
	if update.CallbackQuery != nil {
		h.handleCallback(bot, update.CallbackQuery)
	}
}

func (h *Handler) handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg == nil || msg.From == nil {
		return
	}

	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		group, _, err := h.service.RegisterGroupAndUser(msg)
		if err == nil {
			_ = h.service.SyncGroupAdmins(bot, group)
		}
		_ = h.service.HandleSystemMessageCleanup(bot, msg)
		if len(msg.NewChatMembers) > 0 {
			_ = h.service.OnNewMembers(bot, msg)
		}

		if msg.IsCommand() {
			h.handleGroupCommand(bot, msg)
			return
		}
		_ = h.service.CheckMessageAndRespond(bot, msg)
		return
	}

	if !msg.Chat.IsPrivate() {
		return
	}
	if msg.IsCommand() {
		h.handlePrivateCommand(bot, msg)
		return
	}
	if msg.Text != "" {
		h.handlePrivatePendingInput(bot, msg)
	}
}

func (h *Handler) handlePrivateCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	target := renderTarget{ChatID: msg.Chat.ID}
	switch msg.Command() {
	case "start":
		h.render(bot, target, "欢迎使用 GroupMaster Bot。\n请通过按钮管理群组。", mainMenuKeyboard())
	case "groups":
		h.sendGroupsMenu(bot, target, msg.From.ID, 1)
	case "settings":
		h.sendSettingsPanel(bot, target, msg.From.ID)
	default:
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "暂不支持该私聊命令"))
	}
}

func (h *Handler) handleGroupCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "lottery_create":
		args := strings.TrimSpace(msg.CommandArguments())
		title := "默认抽奖"
		winners := 1
		if args != "" {
			parts := strings.Split(args, "|")
			title = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				if n, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					winners = n
				}
			}
		}
		l, err := h.service.CreateLotteryByTGGroupID(msg.Chat.ID, title, winners)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建抽奖失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("抽奖已创建：%s（中奖人数:%d）\n发送 /lottery_join 参与", l.Title, l.WinnersCount)))
	case "lottery_join":
		if err := h.service.JoinActiveLotteryByTGGroupID(msg.Chat.ID, msg.From); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "参与失败：当前可能没有进行中的抽奖"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "参与成功"))
	case "lottery_draw":
		winners, err := h.service.DrawActiveLotteryByTGGroupID(msg.Chat.ID)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "开奖失败：没有足够参与者或无活动抽奖"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "开奖结果："+joinWinnerNames(winners)))
	}
}

func (h *Handler) handlePrivatePendingInput(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	pending, ok := h.getPending(msg.From.ID)
	if !ok {
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请先点击菜单按钮选择操作。"))
		return
	}
	target := renderTarget{ChatID: msg.Chat.ID}
	if !h.ensureAdmin(bot, target, msg.From.ID, pending.TGGroupID) {
		h.clearPending(msg.From.ID)
		return
	}

	text := strings.TrimSpace(msg.Text)
	switch pending.Kind {
	case "auto_add":
		parts := strings.SplitN(text, "=>", 2)
		if len(parts) != 2 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式错误，请按：关键词=>回复内容"))
			return
		}
		keyword := strings.TrimSpace(parts[0])
		reply := strings.TrimSpace(parts[1])
		if keyword == "" || reply == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "关键词和回复都不能为空"))
			return
		}
		if err := h.service.AddAutoReplyByTGGroupID(pending.TGGroupID, keyword, reply, "contains"); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "新增自动回复失败"))
			return
		}
		h.sendAutoReplyList(bot, target, msg.From.ID, pending.TGGroupID, 1)
	case "bw_add":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "违禁词不能为空"))
			return
		}
		if err := h.service.AddBannedWordByTGGroupID(pending.TGGroupID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "新增违禁词失败"))
			return
		}
		h.sendBannedWordList(bot, target, msg.From.ID, pending.TGGroupID, 1)
	case "auto_edit":
		parts := strings.SplitN(text, "=>", 2)
		if len(parts) != 2 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式错误，请按：关键词=>回复内容"))
			return
		}
		keyword := strings.TrimSpace(parts[0])
		reply := strings.TrimSpace(parts[1])
		if keyword == "" || reply == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "关键词和回复都不能为空"))
			return
		}
		if err := h.service.UpdateAutoReplyByTGGroupID(pending.TGGroupID, pending.RuleID, keyword, reply, "contains"); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "更新自动回复失败"))
			return
		}
		h.sendAutoReplyList(bot, target, msg.From.ID, pending.TGGroupID, pending.Page)
	case "bw_edit":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "违禁词不能为空"))
			return
		}
		if err := h.service.UpdateBannedWordByTGGroupID(pending.TGGroupID, pending.RuleID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "更新违禁词失败"))
			return
		}
		h.sendBannedWordList(bot, target, msg.From.ID, pending.TGGroupID, pending.Page)
	case "lottery_create":
		title := "默认抽奖"
		winners := 1
		parts := strings.SplitN(text, "|", 2)
		if strings.TrimSpace(parts[0]) != "" {
			title = strings.TrimSpace(parts[0])
		}
		if len(parts) > 1 {
			n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err == nil && n > 0 {
				winners = n
			}
		}
		if _, err := h.service.CreateLotteryByTGGroupID(pending.TGGroupID, title, winners); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建抽奖失败"))
			return
		}
		h.sendGroupPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "sched_add":
		cronExpr, content, err := h.service.ParseScheduledInput(text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式错误，请按：cron表达式=>消息内容"))
			return
		}
		if err := h.service.CreateScheduledMessageByTGGroupID(pending.TGGroupID, content, cronExpr); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建定时消息失败"))
			return
		}
		h.sendScheduledList(bot, target, msg.From.ID, pending.TGGroupID, 1)
	case "invite_create":
		expireHours := 24
		memberLimit := 0
		parts := strings.SplitN(text, "|", 2)
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			if v, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
				expireHours = v
			}
		}
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			if v, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				memberLimit = v
			}
		}
		link, err := h.service.CreateInviteLinkByTGGroupID(bot, pending.TGGroupID, expireHours, memberLimit)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建邀请链接失败"))
			return
		}
		h.render(bot, target, "邀请链接已生成：\n"+link, groupPanelKeyboard(pending.TGGroupID))
	case "chain_start":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入接龙标题"))
			return
		}
		if err := h.service.StartChainByTGGroupID(pending.TGGroupID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建接龙失败"))
			return
		}
		h.sendChainPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "chain_add":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "接龙内容不能为空"))
			return
		}
		if err := h.service.AddChainEntryByTGGroupID(pending.TGGroupID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "接龙添加失败"))
			return
		}
		h.sendChainPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "poll_create":
		parts := strings.SplitN(text, "|", 2)
		if len(parts) != 2 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式错误，请按：问题|选项1,选项2,..."))
			return
		}
		question := strings.TrimSpace(parts[0])
		optTexts := strings.Split(parts[1], ",")
		options := make([]string, 0, len(optTexts))
		for _, o := range optTexts {
			if t := strings.TrimSpace(o); t != "" {
				options = append(options, t)
			}
		}
		if question == "" || len(options) < 2 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "至少需要 1 个问题和 2 个选项"))
			return
		}
		if _, err := h.service.CreatePollByTGGroupID(bot, pending.TGGroupID, question, options); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建投票失败"))
			return
		}
		h.render(bot, target, "投票已发送到群内", groupPanelKeyboard(pending.TGGroupID))
	case "monitor_add":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "关键词不能为空"))
			return
		}
		if err := h.service.AddMonitorKeywordByTGGroupID(pending.TGGroupID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "添加关键词失败"))
			return
		}
		h.sendMonitorPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "monitor_remove":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "关键词不能为空"))
			return
		}
		if err := h.service.RemoveMonitorKeywordByTGGroupID(pending.TGGroupID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "删除关键词失败"))
			return
		}
		h.sendMonitorPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "rbac_set_role":
		parts := strings.SplitN(text, "|", 2)
		if len(parts) != 2 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式错误，请按：tg_user_id|role"))
			return
		}
		tgUID, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用户ID格式错误"))
			return
		}
		role := strings.TrimSpace(parts[1])
		if err := h.service.SetRoleByTGGroupID(pending.TGGroupID, tgUID, role); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置角色失败"))
			return
		}
		h.sendRBACPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "rbac_set_acl":
		parts := strings.SplitN(text, "|", 2)
		if len(parts) != 2 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式错误，请按：feature|role1,role2"))
			return
		}
		feature := strings.TrimSpace(parts[0])
		roles := strings.Split(parts[1], ",")
		if err := h.service.SetFeatureACLByTGGroupID(pending.TGGroupID, feature, roles); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置权限失败"))
			return
		}
		h.sendRBACPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "black_add":
		parts := strings.SplitN(text, "|", 2)
		tgUID, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用户ID格式错误"))
			return
		}
		reason := ""
		if len(parts) == 2 {
			reason = strings.TrimSpace(parts[1])
		}
		if err := h.service.AddGlobalBlacklist(tgUID, reason); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "加入黑名单失败"))
			return
		}
		h.sendBlacklistPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "black_remove":
		tgUID, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用户ID格式错误"))
			return
		}
		if err := h.service.RemoveGlobalBlacklist(tgUID); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "移除黑名单失败"))
			return
		}
		h.sendBlacklistPanel(bot, target, msg.From.ID, pending.TGGroupID)
	default:
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "未识别输入态，请重新点击菜单操作"))
	}

	h.clearPending(msg.From.ID)
}

func (h *Handler) handleCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery) {
	if cb == nil || cb.Message == nil || cb.From == nil {
		return
	}
	target := renderTarget{ChatID: cb.Message.Chat.ID, MessageID: cb.Message.MessageID, Edit: true}
	userID := cb.From.ID
	data := cb.Data

	switch {
	case strings.HasPrefix(data, cbVerifyPF):
		h.handleVerifyCallback(bot, cb)
	case data == cbMenuSettings:
		h.answerCallback(bot, cb.ID, "加载设置")
		h.sendSettingsPanel(bot, target, userID)
	case data == cbMenuGroups:
		h.answerCallback(bot, cb.ID, "加载群组")
		h.sendGroupsMenu(bot, target, userID, 1)
	case strings.HasPrefix(data, cbGroupsPagePF):
		h.answerCallback(bot, cb.ID, "翻页")
		page, err := parseIntSuffix(data, cbGroupsPagePF)
		if err != nil {
			return
		}
		h.sendGroupsMenu(bot, target, userID, page)
	case strings.HasPrefix(data, cbGroupPrefix):
		h.answerCallback(bot, cb.ID, "进入群管理")
		tgGroupID, err := parseInt64Suffix(data, cbGroupPrefix)
		if err != nil {
			return
		}
		h.sendGroupPanel(bot, target, userID, tgGroupID)
	case strings.HasPrefix(data, cbFeaturePref):
		h.handleFeatureCallback(bot, cb, target, userID, data)
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleVerifyCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery) {
	parts := strings.Split(cb.Data, ":")
	if len(parts) < 4 {
		h.answerCallback(bot, cb.ID, "参数错误")
		return
	}
	tgGroupID, err1 := strconv.ParseInt(parts[2], 10, 64)
	tgUserID, err2 := strconv.ParseInt(parts[3], 10, 64)
	if err1 != nil || err2 != nil {
		h.answerCallback(bot, cb.ID, "参数错误")
		return
	}
	var answer *int
	if len(parts) >= 5 {
		if v, err := strconv.Atoi(parts[4]); err == nil {
			answer = &v
		}
	}
	if err := h.service.PassVerification(bot, tgGroupID, tgUserID, cb.From.ID, answer); err != nil {
		h.answerCallback(bot, cb.ID, "验证失败或已过期")
		return
	}
	h.answerCallback(bot, cb.ID, "验证通过")
	if cb.Message != nil {
		target := renderTarget{ChatID: cb.Message.Chat.ID, MessageID: cb.Message.MessageID, Edit: true}
		h.render(bot, target, "✅ 验证通过，可正常发言", tgbotapi.NewInlineKeyboardMarkup())
	}
}

func (h *Handler) handleFeatureCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, userID int64, data string) {
	parts := strings.Split(data, ":")
	if len(parts) < 4 {
		h.answerCallback(bot, cb.ID, "参数错误")
		return
	}
	feature := parts[1]
	action := parts[2]

	if feature == "lang" {
		if action != "set" || len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		if err := h.service.SetUserLanguage(userID, parts[4]); err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		h.answerCallback(bot, cb.ID, "语言已切换")
		h.sendSettingsPanel(bot, target, userID)
		return
	}

	tgGroupID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		h.answerCallback(bot, cb.ID, "群参数错误")
		return
	}

	if !h.ensureAdmin(bot, target, userID, tgGroupID) {
		h.answerCallback(bot, cb.ID, "无权限")
		return
	}
	if perm := permissionFeatureKey(feature, action); perm != "" {
		ok, err := h.service.CanAccessFeatureByTGGroupID(tgGroupID, userID, perm)
		if err != nil || !ok {
			h.answerCallback(bot, cb.ID, "该功能无权限")
			return
		}
	}
	if action != "add" && action != "edit" {
		h.clearPending(userID)
	}

	switch feature {
	case "pending":
		if action != "cancel" {
			h.answerCallback(bot, cb.ID, "未知操作")
			return
		}
		h.clearPending(userID)
		h.answerCallback(bot, cb.ID, "已取消")
		h.sendGroupPanel(bot, target, userID, tgGroupID)
	case "welcome":
		if action != "toggle" {
			h.answerCallback(bot, cb.ID, "未知操作")
			return
		}
		enabled, err := h.service.ToggleWelcomeByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		if enabled {
			h.answerCallback(bot, cb.ID, "欢迎消息已开启")
		} else {
			h.answerCallback(bot, cb.ID, "欢迎消息已关闭")
		}
		h.sendGroupPanel(bot, target, userID, tgGroupID)
	case "auto":
		h.handleAutoReplyFeature(bot, cb, target, userID, tgGroupID, action, parts)
	case "bw":
		h.handleBannedWordFeature(bot, cb, target, userID, tgGroupID, action, parts)
	case "lottery":
		h.handleLotteryFeature(bot, cb, target, tgGroupID, action)
	case "sched":
		h.handleScheduleFeature(bot, cb, target, userID, tgGroupID, action, parts)
	case "stats":
		if action != "show" {
			h.answerCallback(bot, cb.ID, "未知操作")
			return
		}
		h.answerCallback(bot, cb.ID, "加载统计")
		h.sendStatsPanel(bot, target, userID, tgGroupID)
	case "logs":
		switch action {
		case "list":
			page := 1
			filter := "all"
			if len(parts) >= 5 {
				if p, err := strconv.Atoi(parts[4]); err == nil {
					page = p
				}
			}
			if len(parts) >= 6 && parts[5] != "" {
				filter = parts[5]
			}
			h.answerCallback(bot, cb.ID, "加载日志")
			h.sendLogPanel(bot, target, userID, tgGroupID, page, filter)
		case "export":
			filter := "all"
			if len(parts) >= 5 && parts[4] != "" {
				filter = parts[4]
			}
			name, content, err := h.service.ExportLogsCSVByTGGroupID(tgGroupID, filter)
			if err != nil {
				h.answerCallback(bot, cb.ID, "导出失败")
				return
			}
			doc := tgbotapi.NewDocument(target.ChatID, tgbotapi.FileBytes{Name: name, Bytes: content})
			doc.Caption = "日志 CSV 导出"
			_, _ = bot.Send(doc)
			h.answerCallback(bot, cb.ID, "已导出")
			h.sendLogPanel(bot, target, userID, tgGroupID, 1, filter)
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "mod":
		h.handleModerationFeature(bot, cb, target, userID, tgGroupID, action)
	case "rbac":
		switch action {
		case "view":
			h.answerCallback(bot, cb.ID, "加载权限分级")
			h.sendRBACPanel(bot, target, userID, tgGroupID)
		case "setrole":
			h.answerCallback(bot, cb.ID, "请输入角色配置")
			h.setPending(userID, pendingInput{Kind: "rbac_set_role", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入：tg_user_id|role\nrole: super_admin 或 admin", pendingCancelKeyboard(tgGroupID))
		case "setacl":
			h.answerCallback(bot, cb.ID, "请输入权限配置")
			h.setPending(userID, pendingInput{Kind: "rbac_set_acl", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入：feature|role1,role2\n示例：lottery|super_admin", pendingCancelKeyboard(tgGroupID))
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "black":
		switch action {
		case "view":
			h.answerCallback(bot, cb.ID, "加载黑名单")
			h.sendBlacklistPanel(bot, target, userID, tgGroupID)
		case "add":
			h.answerCallback(bot, cb.ID, "请输入用户ID")
			h.setPending(userID, pendingInput{Kind: "black_add", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入：tg_user_id|原因(可选)", pendingCancelKeyboard(tgGroupID))
		case "remove":
			h.answerCallback(bot, cb.ID, "请输入用户ID")
			h.setPending(userID, pendingInput{Kind: "black_remove", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入要移除的 tg_user_id", pendingCancelKeyboard(tgGroupID))
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "invite":
		switch action {
		case "create":
			h.answerCallback(bot, cb.ID, "请输入邀请参数")
			h.setPending(userID, pendingInput{Kind: "invite_create", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入：过期小时|人数上限\n示例：24|100（人数上限可省略）", pendingCancelKeyboard(tgGroupID))
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "chain":
		switch action {
		case "view":
			h.answerCallback(bot, cb.ID, "加载接龙")
			h.sendChainPanel(bot, target, userID, tgGroupID)
		case "start":
			h.answerCallback(bot, cb.ID, "请输入接龙标题")
			h.setPending(userID, pendingInput{Kind: "chain_start", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入接龙标题", pendingCancelKeyboard(tgGroupID))
		case "add":
			h.answerCallback(bot, cb.ID, "请输入接龙内容")
			h.setPending(userID, pendingInput{Kind: "chain_add", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入接龙内容（如：1. 张三）", pendingCancelKeyboard(tgGroupID))
		case "close":
			if err := h.service.CloseChainByTGGroupID(tgGroupID); err != nil {
				h.answerCallback(bot, cb.ID, "关闭失败")
				return
			}
			h.answerCallback(bot, cb.ID, "接龙已关闭")
			h.sendChainPanel(bot, target, userID, tgGroupID)
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "poll":
		switch action {
		case "create":
			h.answerCallback(bot, cb.ID, "请输入投票内容")
			h.setPending(userID, pendingInput{Kind: "poll_create", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入：问题|选项1,选项2,选项3", pendingCancelKeyboard(tgGroupID))
		case "stop":
			if err := h.service.StopPollByTGGroupID(bot, tgGroupID); err != nil {
				h.answerCallback(bot, cb.ID, "结束投票失败")
				return
			}
			h.answerCallback(bot, cb.ID, "投票已结束")
			h.sendGroupPanel(bot, target, userID, tgGroupID)
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "monitor":
		switch action {
		case "view":
			h.answerCallback(bot, cb.ID, "加载监控词")
			h.sendMonitorPanel(bot, target, userID, tgGroupID)
		case "add":
			h.answerCallback(bot, cb.ID, "请输入关键词")
			h.setPending(userID, pendingInput{Kind: "monitor_add", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入要监控的关键词（单条）", pendingCancelKeyboard(tgGroupID))
		case "remove":
			h.answerCallback(bot, cb.ID, "请输入关键词")
			h.setPending(userID, pendingInput{Kind: "monitor_remove", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入要移除的关键词（单条）", pendingCancelKeyboard(tgGroupID))
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "sys":
		switch action {
		case "view":
			h.answerCallback(bot, cb.ID, "加载系统消息清理")
			h.sendSystemCleanPanel(bot, target, userID, tgGroupID)
		case "preset":
			if len(parts) < 5 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			_, err := h.service.ApplySystemCleanPresetByTGGroupID(tgGroupID, parts[4])
			if err != nil {
				h.answerCallback(bot, cb.ID, "应用失败")
				return
			}
			h.answerCallback(bot, cb.ID, "预设已应用")
			h.sendSystemCleanPanel(bot, target, userID, tgGroupID)
		case "toggle":
			if len(parts) < 5 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			_, err := h.service.ToggleSystemCleanByTGGroupID(tgGroupID, parts[4])
			if err != nil {
				h.answerCallback(bot, cb.ID, "切换失败")
				return
			}
			h.answerCallback(bot, cb.ID, "已切换")
			h.sendSystemCleanPanel(bot, target, userID, tgGroupID)
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	default:
		h.answerCallback(bot, cb.ID, "未实现功能")
	}
}

func (h *Handler) handleAutoReplyFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "add":
		h.answerCallback(bot, cb.ID, "请发送自动回复配置")
		h.setPending(userID, pendingInput{Kind: "auto_add", TGGroupID: tgGroupID, Page: 1})
		h.render(bot, target, "请发送：关键词=>回复内容\n示例：官网=>https://example.com", pendingCancelKeyboard(tgGroupID))
	case "list":
		page := 1
		if len(parts) >= 5 {
			if p, err := strconv.Atoi(parts[4]); err == nil {
				page = p
			}
		}
		h.answerCallback(bot, cb.ID, "加载自动回复")
		h.sendAutoReplyList(bot, target, userID, tgGroupID, page)
	case "del":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		if err := h.service.DeleteAutoReplyByTGGroupID(tgGroupID, uint(id)); err != nil {
			h.answerCallback(bot, cb.ID, "删除失败")
			return
		}
		h.answerCallback(bot, cb.ID, "已删除")
		h.sendAutoReplyList(bot, target, userID, tgGroupID, page)
	case "edit":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		h.answerCallback(bot, cb.ID, "请输入新规则")
		h.setPending(userID, pendingInput{Kind: "auto_edit", TGGroupID: tgGroupID, RuleID: uint(id), Page: page})
		h.render(bot, target, "请发送新内容：关键词=>回复内容", pendingCancelKeyboard(tgGroupID))
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleBannedWordFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "add":
		h.answerCallback(bot, cb.ID, "请发送违禁词")
		h.setPending(userID, pendingInput{Kind: "bw_add", TGGroupID: tgGroupID, Page: 1})
		h.render(bot, target, "请直接发送要新增的违禁词（单条）", pendingCancelKeyboard(tgGroupID))
	case "list":
		page := 1
		if len(parts) >= 5 {
			if p, err := strconv.Atoi(parts[4]); err == nil {
				page = p
			}
		}
		h.answerCallback(bot, cb.ID, "加载违禁词")
		h.sendBannedWordList(bot, target, userID, tgGroupID, page)
	case "del":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		if err := h.service.DeleteBannedWordByTGGroupID(tgGroupID, uint(id)); err != nil {
			h.answerCallback(bot, cb.ID, "删除失败")
			return
		}
		h.answerCallback(bot, cb.ID, "已删除")
		h.sendBannedWordList(bot, target, userID, tgGroupID, page)
	case "edit":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		h.answerCallback(bot, cb.ID, "请输入新违禁词")
		h.setPending(userID, pendingInput{Kind: "bw_edit", TGGroupID: tgGroupID, RuleID: uint(id), Page: page})
		h.render(bot, target, "请发送新的违禁词内容", pendingCancelKeyboard(tgGroupID))
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleLotteryFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, tgGroupID int64, action string) {
	switch action {
	case "create":
		h.answerCallback(bot, cb.ID, "请发送抽奖配置")
		h.setPending(cb.From.ID, pendingInput{Kind: "lottery_create", TGGroupID: tgGroupID})
		h.render(bot, target, "请发送：抽奖标题|中奖人数\n示例：周末福利|3", pendingCancelKeyboard(tgGroupID))
	case "draw":
		winners, err := h.service.DrawActiveLotteryByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "开奖失败")
			h.render(bot, target, "开奖失败：没有可开奖的活动抽奖", groupPanelKeyboard(tgGroupID))
			return
		}
		h.answerCallback(bot, cb.ID, "开奖完成")
		_, _ = bot.Send(tgbotapi.NewMessage(tgGroupID, "开奖结果："+joinWinnerNames(winners)))
		h.sendGroupPanel(bot, target, cb.From.ID, tgGroupID)
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleScheduleFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "add":
		h.answerCallback(bot, cb.ID, "请发送定时消息")
		h.setPending(userID, pendingInput{Kind: "sched_add", TGGroupID: tgGroupID, Page: 1})
		h.render(bot, target, "请发送：cron表达式=>消息内容\n示例：0 9 * * *=>早上好", pendingCancelKeyboard(tgGroupID))
	case "list":
		page := 1
		if len(parts) >= 5 {
			if p, err := strconv.Atoi(parts[4]); err == nil {
				page = p
			}
		}
		h.answerCallback(bot, cb.ID, "加载定时消息")
		h.sendScheduledList(bot, target, userID, tgGroupID, page)
	case "del":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		if err := h.service.DeleteScheduledMessageByTGGroupID(tgGroupID, uint(id)); err != nil {
			h.answerCallback(bot, cb.ID, "删除失败")
			return
		}
		h.answerCallback(bot, cb.ID, "已删除")
		h.sendScheduledList(bot, target, userID, tgGroupID, page)
	case "toggle":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		enabled, err := h.service.ToggleScheduledMessageByTGGroupID(tgGroupID, uint(id))
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		if enabled {
			h.answerCallback(bot, cb.ID, "已启用")
		} else {
			h.answerCallback(bot, cb.ID, "已停用")
		}
		h.sendScheduledList(bot, target, userID, tgGroupID, page)
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleModerationFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string) {
	var (
		featureKey string
		label      string
	)
	switch action {
	case "spam":
		featureKey = "anti_spam"
		label = "反垃圾"
	case "flood":
		featureKey = "anti_flood"
		label = "反刷屏"
	case "verify":
		featureKey = "join_verify"
		label = "进群验证"
	case "newbie":
		featureKey = "newbie_limit"
		label = "新成员限制"
	case "verifytype":
		mode, err := h.service.ToggleJoinVerifyTypeByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		h.answerCallback(bot, cb.ID, "验证类型已切换为 "+mode)
		h.sendGroupPanel(bot, target, userID, tgGroupID)
		return
	case "newbietime":
		mins, err := h.service.CycleNewbieLimitMinutesByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		h.answerCallback(bot, cb.ID, fmt.Sprintf("新人限制时长已设为 %d 分钟", mins))
		h.sendGroupPanel(bot, target, userID, tgGroupID)
		return
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
		return
	}
	enabled, err := h.service.ToggleFeatureByTGGroupID(tgGroupID, featureKey)
	if err != nil {
		h.answerCallback(bot, cb.ID, "切换失败")
		return
	}
	if enabled {
		h.answerCallback(bot, cb.ID, label+"已开启")
	} else {
		h.answerCallback(bot, cb.ID, label+"已关闭")
	}
	h.sendGroupPanel(bot, target, userID, tgGroupID)
}

func (h *Handler) sendGroupsMenu(bot *tgbotapi.BotAPI, target renderTarget, tgUserID int64, page int) {
	groups, err := h.service.ListManageableGroups(tgUserID)
	if err != nil {
		h.render(bot, target, "获取群列表失败", mainMenuKeyboard())
		return
	}
	if len(groups) == 0 {
		h.render(bot, target, "你当前没有可管理且机器人已加入的群", mainMenuKeyboard())
		return
	}
	totalPages := (len(groups) + groupPageSize - 1) / groupPageSize
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * groupPageSize
	end := start + groupPageSize
	if end > len(groups) {
		end = len(groups)
	}
	current := groups[start:end]
	text := fmt.Sprintf("请选择要管理的群组（第 %d/%d 页）：", page, totalPages)
	h.render(bot, target, text, groupsKeyboard(current, page, totalPages))
}

func (h *Handler) sendGroupPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	summary, err := h.service.GroupPanelSummary(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载群面板失败", mainMenuKeyboard())
		return
	}
	h.render(bot, target, summary, groupPanelKeyboard(tgGroupID))
}

func (h *Handler) sendAutoReplyList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListAutoRepliesByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载自动回复失败", groupPanelKeyboard(tgGroupID))
		return
	}
	totalPages := maxPages(data.Total, rulesPageSize)
	if data.Page < 1 {
		data.Page = 1
	}
	if data.Page > totalPages {
		data.Page = totalPages
	}
	lines := []string{fmt.Sprintf("自动回复列表（第 %d/%d 页，总 %d 条）", data.Page, totalPages, data.Total)}
	if len(data.Items) == 0 {
		lines = append(lines, "暂无规则")
	}
	for _, item := range data.Items {
		lines = append(lines, fmt.Sprintf("#%d [%s] %s => %s", item.ID, item.MatchType, item.Keyword, item.Reply))
	}
	h.render(bot, target, strings.Join(lines, "\n"), autoReplyListKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendBannedWordList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListBannedWordsByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载违禁词失败", groupPanelKeyboard(tgGroupID))
		return
	}
	totalPages := maxPages(data.Total, rulesPageSize)
	if data.Page < 1 {
		data.Page = 1
	}
	if data.Page > totalPages {
		data.Page = totalPages
	}
	lines := []string{fmt.Sprintf("违禁词列表（第 %d/%d 页，总 %d 条）", data.Page, totalPages, data.Total)}
	if len(data.Items) == 0 {
		lines = append(lines, "暂无词条")
	}
	for _, item := range data.Items {
		lines = append(lines, fmt.Sprintf("#%d %s", item.ID, item.Word))
	}
	h.render(bot, target, strings.Join(lines, "\n"), bannedWordListKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendScheduledList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListScheduledMessagesByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载定时消息失败", groupPanelKeyboard(tgGroupID))
		return
	}
	totalPages := maxPages(data.Total, rulesPageSize)
	if data.Page < 1 {
		data.Page = 1
	}
	if data.Page > totalPages {
		data.Page = totalPages
	}
	lines := []string{fmt.Sprintf("定时消息（第 %d/%d 页，总 %d 条）", data.Page, totalPages, data.Total)}
	if len(data.Items) == 0 {
		lines = append(lines, "暂无任务")
	}
	for _, item := range data.Items {
		status := "停用"
		if item.Enabled {
			status = "启用"
		}
		lines = append(lines, fmt.Sprintf("#%d [%s] %s => %s", item.ID, status, item.CronExpr, item.Content))
	}
	h.render(bot, target, strings.Join(lines, "\n"), scheduledListKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendStatsPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	stats, err := h.service.GroupStatsByTGGroupID(tgGroupID, 10)
	if err != nil {
		h.render(bot, target, "加载统计失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{fmt.Sprintf("群统计：%s (%d)", stats.GroupTitle, stats.GroupID)}
	if len(stats.TopUsers) == 0 {
		lines = append(lines, "暂无活跃数据")
	} else {
		lines = append(lines, "活跃榜（按消息积分）:")
		for i, u := range stats.TopUsers {
			lines = append(lines, fmt.Sprintf("%d. %s - %d", i+1, u.DisplayName, u.Points))
		}
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新统计", fmt.Sprintf("feat:stats:show:%d", tgGroupID)),
			tgbotapi.NewInlineKeyboardButtonData("返回群面板", cbGroupPrefix+strconv.FormatInt(tgGroupID, 10)),
		),
	)
	h.render(bot, target, strings.Join(lines, "\n"), markup)
}

func (h *Handler) sendLogPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int, filter string) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListLogsByTGGroupID(tgGroupID, page, rulesPageSize, filter)
	if err != nil {
		h.render(bot, target, "加载管理日志失败", groupPanelKeyboard(tgGroupID))
		return
	}
	totalPages := maxPages(data.Total, rulesPageSize)
	if data.Page < 1 {
		data.Page = 1
	}
	if data.Page > totalPages {
		data.Page = totalPages
	}
	lines := []string{fmt.Sprintf("管理日志（第 %d/%d 页，总 %d 条）", data.Page, totalPages, data.Total)}
	if len(data.Items) == 0 {
		lines = append(lines, "暂无日志")
	}
	for _, item := range data.Items {
		lines = append(lines, fmt.Sprintf("#%d %s @ %s", item.ID, item.Action, item.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	h.render(bot, target, strings.Join(lines, "\n"), logListKeyboard(tgGroupID, data.Page, totalPages, filter))
}

func (h *Handler) sendSystemCleanPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	cfg, err := h.service.SystemCleanViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载系统消息清理失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"帮助您自动清理群组中的系统消息",
		"预设: 严格 / 推荐 / 关闭",
		"",
		fmt.Sprintf("进群: %s", onOffWithEmoji(cfg.Join)),
		fmt.Sprintf("退群: %s", onOffWithEmoji(cfg.Leave)),
		fmt.Sprintf("置顶: %s", onOffWithEmoji(cfg.Pin)),
		fmt.Sprintf("修改头像: %s", onOffWithEmoji(cfg.Photo)),
		fmt.Sprintf("修改名称: %s", onOffWithEmoji(cfg.Title)),
	}
	h.render(bot, target, strings.Join(lines, "\n"), systemCleanKeyboard(tgGroupID, cfg))
}

func (h *Handler) sendChainPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.ChainViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载接龙失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{"接龙管理"}
	if !view.Active {
		lines = append(lines, "状态：未开始")
	} else {
		lines = append(lines, "状态：进行中")
		lines = append(lines, "标题："+view.Title)
		if len(view.Entries) == 0 {
			lines = append(lines, "暂无条目")
		} else {
			lines = append(lines, "条目：")
			for i, e := range view.Entries {
				lines = append(lines, fmt.Sprintf("%d. %s", i+1, e))
			}
		}
	}
	h.render(bot, target, strings.Join(lines, "\n"), chainKeyboard(tgGroupID, view.Active))
}

func (h *Handler) sendMonitorPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	items, err := h.service.ListMonitorKeywordsByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载关键词监控失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{"关键词监控"}
	if len(items) == 0 {
		lines = append(lines, "暂无关键词")
	} else {
		lines = append(lines, "当前关键词：")
		for i, k := range items {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, k))
		}
		lines = append(lines, "", "命中后将私聊通知群管理员")
	}
	h.render(bot, target, strings.Join(lines, "\n"), monitorKeyboard(tgGroupID))
}

func (h *Handler) sendRBACPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	text, err := h.service.RBACSummaryByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载权限分级失败", groupPanelKeyboard(tgGroupID))
		return
	}
	h.render(bot, target, text, rbacKeyboard(tgGroupID))
}

func (h *Handler) sendBlacklistPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	items, err := h.service.ListGlobalBlacklist()
	if err != nil {
		h.render(bot, target, "加载黑名单失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{"全局黑名单（跨群生效）"}
	if len(items) == 0 {
		lines = append(lines, "暂无黑名单用户")
	} else {
		for i, it := range items {
			lines = append(lines, fmt.Sprintf("%d. %d (%s)", i+1, it.TGUserID, it.Reason))
		}
	}
	h.render(bot, target, strings.Join(lines, "\n"), blacklistKeyboard(tgGroupID))
}

func (h *Handler) sendSettingsPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID int64) {
	lang, _ := h.service.GetUserLanguage(tgUserID)
	text := "设置\n当前语言: " + lang + "\n可切换为中文/英文（逐步覆盖）"
	h.render(bot, target, text, settingsKeyboard())
}

func (h *Handler) ensureAdmin(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) bool {
	ok, err := h.service.IsAdminByTGGroupID(tgGroupID, tgUserID)
	if err != nil || !ok {
		h.render(bot, target, "你不是该群管理员，或机器人尚未同步该群权限", mainMenuKeyboard())
		return false
	}
	return true
}

func (h *Handler) render(bot *tgbotapi.BotAPI, target renderTarget, text string, markup tgbotapi.InlineKeyboardMarkup) {
	if target.Edit && target.MessageID > 0 {
		edit := tgbotapi.NewEditMessageTextAndMarkup(target.ChatID, target.MessageID, text, markup)
		if _, err := bot.Send(edit); err == nil {
			return
		}
	}
	msg := tgbotapi.NewMessage(target.ChatID, text)
	msg.ReplyMarkup = markup
	_, _ = bot.Send(msg)
}

func (h *Handler) setPending(userID int64, input pendingInput) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending[userID] = input
}

func (h *Handler) getPending(userID int64) (pendingInput, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	in, ok := h.pending[userID]
	return in, ok
}

func (h *Handler) clearPending(userID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.pending, userID)
}

func (h *Handler) answerCallback(bot *tgbotapi.BotAPI, callbackID, text string) {
	_, _ = bot.Request(tgbotapi.NewCallback(callbackID, text))
}

func mainMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 我的群组", cbMenuGroups),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ 设置", cbMenuSettings),
		),
	)
}

func groupsKeyboard(groups []model.Group, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(groups)+3)
	for _, g := range groups {
		label := g.Title
		if label == "" {
			label = strconv.FormatInt(g.TGGroupID, 10)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗂 "+label, cbGroupPrefix+strconv.FormatInt(g.TGGroupID, 10)),
		))
	}

	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("%s%d", cbGroupsPagePF, page-1)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("%s%d", cbGroupsPagePF, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔄 刷新", cbMenuGroups)))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func groupPanelKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	id := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤖 新增自动回复", fmt.Sprintf("feat:auto:add:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("📄 自动回复列表", fmt.Sprintf("feat:auto:list:%s:1", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📨 邀请链接", fmt.Sprintf("feat:invite:create:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("📋 接龙", fmt.Sprintf("feat:chain:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗳 投票", fmt.Sprintf("feat:poll:create:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("👁 关键词监控", fmt.Sprintf("feat:monitor:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 新增违禁词", fmt.Sprintf("feat:bw:add:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("📄 违禁词列表", fmt.Sprintf("feat:bw:list:%s:1", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👋 欢迎开关", fmt.Sprintf("feat:welcome:toggle:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("🎯 创建抽奖", fmt.Sprintf("feat:lottery:create:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏆 立即开奖", fmt.Sprintf("feat:lottery:draw:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("⏰ 定时消息", fmt.Sprintf("feat:sched:list:%s:1", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ 新建定时", fmt.Sprintf("feat:sched:add:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("📊 数据统计", fmt.Sprintf("feat:stats:show:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 反垃圾开关", fmt.Sprintf("feat:mod:spam:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("⚡ 反刷屏开关", fmt.Sprintf("feat:mod:flood:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧩 进群验证开关", fmt.Sprintf("feat:mod:verify:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("🔒 新成员限制开关", fmt.Sprintf("feat:mod:newbie:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧠 验证类型切换", fmt.Sprintf("feat:mod:verifytype:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("⏱ 新人时长切换", fmt.Sprintf("feat:mod:newbietime:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧹 系统消息清理", fmt.Sprintf("feat:sys:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📜 管理日志", fmt.Sprintf("feat:logs:list:%s:1:all", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧭 权限分级", fmt.Sprintf("feat:rbac:view:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("⛔ 黑名单", fmt.Sprintf("feat:black:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群组列表", cbMenuGroups),
		),
	)
}

func pendingCancelKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("取消并返回群面板", fmt.Sprintf("feat:pending:cancel:%d", tgGroupID)),
		),
	)
}

func autoReplyListKeyboard(tgGroupID int64, items []model.AutoReply, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+4)
	for _, item := range items {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("✏️ 编辑 #%d", item.ID),
				fmt.Sprintf("feat:auto:edit:%s:%d:%d", gid, item.ID, page),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🗑 删除 #%d", item.ID),
				fmt.Sprintf("feat:auto:del:%s:%d:%d", gid, item.ID, page),
			),
		))
	}
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:auto:list:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:auto:list:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ 新增自动回复", fmt.Sprintf("feat:auto:add:%s", gid)),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func bannedWordListKeyboard(tgGroupID int64, items []model.BannedWord, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+4)
	for _, item := range items {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("✏️ 编辑 #%d", item.ID),
				fmt.Sprintf("feat:bw:edit:%s:%d:%d", gid, item.ID, page),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🗑 删除 #%d", item.ID),
				fmt.Sprintf("feat:bw:del:%s:%d:%d", gid, item.ID, page),
			),
		))
	}
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:bw:list:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:bw:list:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ 新增违禁词", fmt.Sprintf("feat:bw:add:%s", gid)),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func scheduledListKeyboard(tgGroupID int64, items []model.ScheduledMessage, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+5)
	for _, item := range items {
		toggleLabel := fmt.Sprintf("启用 #%d", item.ID)
		if item.Enabled {
			toggleLabel = fmt.Sprintf("停用 #%d", item.ID)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				toggleLabel,
				fmt.Sprintf("feat:sched:toggle:%s:%d:%d", gid, item.ID, page),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🗑 删除 #%d", item.ID),
				fmt.Sprintf("feat:sched:del:%s:%d:%d", gid, item.ID, page),
			),
		))
	}
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:sched:list:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:sched:list:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ 新建定时", fmt.Sprintf("feat:sched:add:%s", gid)),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func logListKeyboard(tgGroupID int64, page, totalPages int, filter string) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, 3)
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:logs:list:%s:%d:%s", gid, page-1, filter)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:logs:list:%s:%d:%s", gid, page+1, filter)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("全部", fmt.Sprintf("feat:logs:list:%s:1:all", gid)),
		tgbotapi.NewInlineKeyboardButtonData("审核", fmt.Sprintf("feat:logs:list:%s:1:anti_spam_delete", gid)),
		tgbotapi.NewInlineKeyboardButtonData("验证", fmt.Sprintf("feat:logs:list:%s:1:join_verify_pass", gid)),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("导出 CSV", fmt.Sprintf("feat:logs:export:%s:%s", gid, filter)),
		tgbotapi.NewInlineKeyboardButtonData("刷新日志", fmt.Sprintf("feat:logs:list:%s:%d:%s", gid, page, filter)),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func systemCleanKeyboard(tgGroupID int64, cfg *service.SystemCleanView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("严格", fmt.Sprintf("feat:sys:preset:%s:strict", gid)),
			tgbotapi.NewInlineKeyboardButtonData("推荐", fmt.Sprintf("feat:sys:preset:%s:recommended", gid)),
			tgbotapi.NewInlineKeyboardButtonData("关闭", fmt.Sprintf("feat:sys:preset:%s:off", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("进群 "+onOffWithEmoji(cfg.Join), fmt.Sprintf("feat:sys:toggle:%s:join", gid)),
			tgbotapi.NewInlineKeyboardButtonData("退群 "+onOffWithEmoji(cfg.Leave), fmt.Sprintf("feat:sys:toggle:%s:leave", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("置顶 "+onOffWithEmoji(cfg.Pin), fmt.Sprintf("feat:sys:toggle:%s:pin", gid)),
			tgbotapi.NewInlineKeyboardButtonData("头像 "+onOffWithEmoji(cfg.Photo), fmt.Sprintf("feat:sys:toggle:%s:photo", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("名称 "+onOffWithEmoji(cfg.Title), fmt.Sprintf("feat:sys:toggle:%s:title", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func chainKeyboard(tgGroupID int64, active bool) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("新建接龙", fmt.Sprintf("feat:chain:start:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("添加条目", fmt.Sprintf("feat:chain:add:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("关闭接龙", fmt.Sprintf("feat:chain:close:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:chain:view:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	}
	_ = active
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func monitorKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("新增关键词", fmt.Sprintf("feat:monitor:add:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("移除关键词", fmt.Sprintf("feat:monitor:remove:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:monitor:view:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("结束当前投票", fmt.Sprintf("feat:poll:stop:%s", gid)),
		),
	)
}

func rbacKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("设置角色", fmt.Sprintf("feat:rbac:setrole:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("设置功能权限", fmt.Sprintf("feat:rbac:setacl:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:rbac:view:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func blacklistKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("添加", fmt.Sprintf("feat:black:add:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("移除", fmt.Sprintf("feat:black:remove:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:black:view:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func settingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("中文", "feat:lang:set:0:zh"),
			tgbotapi.NewInlineKeyboardButtonData("English", "feat:lang:set:0:en"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("返回群组", cbMenuGroups),
		),
	)
}

func permissionFeatureKey(feature, action string) string {
	switch feature {
	case "rbac", "black":
		return "security"
	case "invite":
		return "invite"
	case "poll":
		return "poll"
	case "chain":
		return "chain"
	case "monitor":
		return "monitor"
	case "sched":
		return "schedule"
	case "auto":
		return "auto_reply"
	case "bw":
		return "banned_words"
	case "logs":
		return "logs"
	case "stats":
		return "stats"
	case "mod":
		return "moderation"
	case "sys":
		return "system_clean"
	case "lottery":
		return "lottery"
	case "welcome":
		return "welcome"
	}
	_ = action
	return ""
}

func parseInt64Suffix(data, prefix string) (int64, error) {
	if !strings.HasPrefix(data, prefix) {
		return 0, errors.New("invalid prefix")
	}
	return strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
}

func parseIntSuffix(data, prefix string) (int, error) {
	if !strings.HasPrefix(data, prefix) {
		return 0, errors.New("invalid prefix")
	}
	return strconv.Atoi(strings.TrimPrefix(data, prefix))
}

func maxPages(total int64, pageSize int) int {
	if total <= 0 {
		return 1
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if pages < 1 {
		return 1
	}
	return pages
}

func joinWinnerNames(winners []model.User) string {
	if len(winners) == 0 {
		return "无"
	}
	names := make([]string, 0, len(winners))
	for _, w := range winners {
		if w.Username != "" {
			names = append(names, "@"+w.Username)
			continue
		}
		n := strings.TrimSpace(w.FirstName + " " + w.LastName)
		if n == "" {
			n = fmt.Sprintf("uid:%d", w.TGUserID)
		}
		names = append(names, n)
	}
	return strings.Join(names, ", ")
}

func onOffWithEmoji(v bool) string {
	if v {
		return "启用✅"
	}
	return "关闭❌"
}
