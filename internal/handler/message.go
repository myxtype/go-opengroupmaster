package handler

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
	h.handlePrivatePendingInput(bot, msg)
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
		keyword := "参加"
		if args != "" {
			parts := strings.Split(args, "|")
			title = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				if n, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					winners = n
				}
			}
			if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
				keyword = strings.TrimSpace(parts[2])
			}
		}
		l, err := h.service.CreateLotteryByTGGroupIDWithKeyword(msg.Chat.ID, title, winners, keyword)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建抽奖失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("抽奖已创建：%s（中奖人数:%d）\n发送关键词「%s」即可参与", l.Title, l.WinnersCount, l.JoinKeyword)))
	case "lottery_join":
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请发送当前抽奖设置的关键词参与（不再使用 /lottery_join）"))
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
		keyword := "参加"
		parts := strings.Split(text, "|")
		if strings.TrimSpace(parts[0]) != "" {
			title = strings.TrimSpace(parts[0])
		}
		if len(parts) > 1 {
			n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err == nil && n > 0 {
				winners = n
			}
		}
		if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
			keyword = strings.TrimSpace(parts[2])
		}
		if _, err := h.service.CreateLotteryByTGGroupIDWithKeyword(pending.TGGroupID, title, winners, keyword); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建抽奖失败"))
			return
		}
		h.sendLotteryPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "sched_add":
		cronExpr, content, err := h.service.ParseScheduledInput(text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式错误，请按：cron表达式=>消息内容"))
			return
		}
		if err := h.service.CreateScheduledMessageByTGGroupID(pending.TGGroupID, content, cronExpr); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建定时消息失败："+err.Error()))
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
		h.sendPollPanel(bot, target, msg.From.ID, pending.TGGroupID)
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
	case "welcome_edit":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "欢迎文案不能为空"))
			return
		}
		if err := h.service.SetWelcomeTextByTGGroupID(pending.TGGroupID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "保存欢迎文案失败"))
			return
		}
		h.sendWelcomePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "welcome_edit_media":
		if text == "关闭" {
			if err := h.service.SetWelcomeMediaByTGGroupID(pending.TGGroupID, ""); err != nil {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "清空欢迎图片失败"))
				return
			}
			h.sendWelcomePanel(bot, target, msg.From.ID, pending.TGGroupID)
			break
		}
		if len(msg.Photo) == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请发送一张图片"))
			return
		}
		fileID := msg.Photo[len(msg.Photo)-1].FileID
		if err := h.service.SetWelcomeMediaByTGGroupID(pending.TGGroupID, fileID); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "保存欢迎图片失败"))
			return
		}
		h.sendWelcomePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "welcome_edit_button":
		if text == "关闭" {
			if err := h.service.SetWelcomeButtonByTGGroupID(pending.TGGroupID, "", ""); err != nil {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "清空欢迎按钮失败"))
				return
			}
			h.sendWelcomePanel(bot, target, msg.From.ID, pending.TGGroupID)
			break
		}
		parts := strings.SplitN(text, "|", 2)
		if len(parts) != 2 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式错误，请按：按钮文本|链接URL"))
			return
		}
		btnText := strings.TrimSpace(parts[0])
		btnURL := strings.TrimSpace(parts[1])
		if btnText == "" || btnURL == "" || !(strings.HasPrefix(btnURL, "http://") || strings.HasPrefix(btnURL, "https://")) {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入有效按钮，示例：官网|https://example.com"))
			return
		}
		if err := h.service.SetWelcomeButtonByTGGroupID(pending.TGGroupID, btnText, btnURL); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "保存欢迎按钮失败"))
			return
		}
		h.sendWelcomePanel(bot, target, msg.From.ID, pending.TGGroupID)
	default:
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "未识别输入态，请重新点击菜单操作"))
	}

	h.clearPending(msg.From.ID)
}
