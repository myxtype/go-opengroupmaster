package handler

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg == nil || msg.From == nil || msg.From.IsBot {
		return
	}

	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		group, _, err := h.service.RegisterGroupAndUser(msg)
		if err == nil {
			_ = h.service.SyncGroupAdmins(bot, group)
		}
		// 处理系统消息清理
		_ = h.service.HandleSystemMessageCleanup(bot, msg)
		// 处理新成员加入
		if len(msg.NewChatMembers) > 0 {
			_ = h.service.OnNewMembers(bot, msg)
		}
		// 处理群组命令
		if msg.IsCommand() {
			h.handleGroupCommand(bot, msg)
			return
		}
		// 处理消息检查
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
	case "help":
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, privateHelpText()))
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
	case "help":
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, groupHelpText()))
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
		publishMsg, sendErr := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("抽奖已创建：%s（中奖人数:%d）\n发送关键词「%s」即可参与", l.Title, l.WinnersCount, l.JoinKeyword)))
		if sendErr == nil {
			_ = h.service.PinLotteryMessageByTGGroupID(bot, msg.Chat.ID, publishMsg.MessageID, "publish")
		}
	case "lottery_join":
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请发送当前抽奖设置的关键词参与（不再使用 /lottery_join）"))
	case "lottery_draw":
		winners, err := h.service.DrawActiveLotteryByTGGroupID(msg.Chat.ID)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "开奖失败：没有足够参与者或无活动抽奖"))
			return
		}
		resultMsg, sendErr := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "开奖结果："+joinWinnerNames(winners)))
		if sendErr == nil {
			_ = h.service.PinLotteryMessageByTGGroupID(bot, msg.Chat.ID, resultMsg.MessageID, "result")
		}
	case "black_add":
		ok, err := h.service.IsAdminByTGGroupID(msg.Chat.ID, msg.From.ID)
		if err != nil || !ok {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "仅群管理员可执行该命令"))
			return
		}
		tgUserID, reason, err := h.resolveBlacklistTargetAndReason(msg)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用法：/black_add @用户名 原因(可选)\n也可回复对方消息使用：/black_add 原因(可选)"))
			return
		}
		if reason == "" {
			reason = "group_admin_command"
		}
		if err := h.service.AddBlacklistByTGGroupID(msg.Chat.ID, tgUserID, reason); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "加入黑名单失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已加入本群黑名单：%d", tgUserID)))
	case "black_remove":
		ok, err := h.service.IsAdminByTGGroupID(msg.Chat.ID, msg.From.ID)
		if err != nil || !ok {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "仅群管理员可执行该命令"))
			return
		}
		tgUserID, _, err := h.resolveBlacklistTargetAndReason(msg)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用法：/black_remove @用户名\n也可回复对方消息使用：/black_remove"))
			return
		}
		if err := h.service.RemoveBlacklistByTGGroupID(msg.Chat.ID, tgUserID); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "移除黑名单失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已移除本群黑名单：%d", tgUserID)))
	}
}

func (h *Handler) resolveBlacklistTargetAndReason(msg *tgbotapi.Message) (int64, string, error) {
	replyTarget := int64(0)
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		replyTarget = msg.ReplyToMessage.From.ID
	}

	args := strings.TrimSpace(msg.CommandArguments())
	if args == "" {
		if replyTarget != 0 {
			return replyTarget, "", nil
		}
		return 0, "", fmt.Errorf("missing target")
	}

	fields := strings.Fields(args)
	if len(fields) == 0 {
		if replyTarget != 0 {
			return replyTarget, "", nil
		}
		return 0, "", fmt.Errorf("missing target")
	}

	target := int64(0)
	reasonStart := 1
	first := strings.TrimSpace(fields[0])
	if strings.HasPrefix(first, "@") {
		username := strings.TrimSpace(strings.TrimPrefix(first, "@"))
		if username == "" {
			return 0, "", fmt.Errorf("invalid username")
		}
		u, err := h.service.Repo().FindUserByUsername(username)
		if err != nil {
			return 0, "", err
		}
		target = u.TGUserID
	} else if id, err := strconv.ParseInt(first, 10, 64); err == nil {
		target = id
	} else if replyTarget != 0 {
		target = replyTarget
		reasonStart = 0
	} else {
		return 0, "", fmt.Errorf("invalid target")
	}

	reason := ""
	if reasonStart < len(fields) {
		reason = strings.TrimSpace(strings.Join(fields[reasonStart:], " "))
	}
	return target, reason, nil
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
	case "auto_add_keyword":
		keyword := strings.TrimSpace(msg.Text)
		if keyword == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "关键词不能为空"))
			return
		}
		matchType := strings.TrimSpace(pending.MatchType)
		if matchType != "exact" && matchType != "contains" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少触发方式，请重新新增自动回复"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:      "auto_add_reply",
			TGGroupID: pending.TGGroupID,
			Page:      pending.Page,
			Keyword:   keyword,
			MatchType: matchType,
		})
		h.render(bot, target, "第3步：请输入自动回复内容（支持换行）", pendingCancelKeyboard(pending.TGGroupID))
		return
	case "auto_add_reply":
		reply := msg.Text
		if strings.TrimSpace(pending.Keyword) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少关键词，请重新新增自动回复"))
			return
		}
		matchType := strings.TrimSpace(pending.MatchType)
		if matchType != "exact" && matchType != "contains" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少触发方式，请重新新增自动回复"))
			return
		}
		if strings.TrimSpace(reply) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "回复内容不能为空"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:      "auto_add_buttons",
			TGGroupID: pending.TGGroupID,
			Page:      pending.Page,
			Keyword:   pending.Keyword,
			MatchType: matchType,
			Content:   reply,
		})
		h.render(bot, target, "第4步（可选）：请输入链接按钮配置。\n支持格式示例：\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n说明：\n- 按钮文字和网址中间用英文 - 分隔\n- 一行两个按钮用 && 分隔\n发送“跳过”表示不设置按钮，发送“关闭”清空按钮", pendingCancelKeyboard(pending.TGGroupID))
		return
	case "auto_add_buttons":
		if strings.TrimSpace(pending.Keyword) == "" || strings.TrimSpace(pending.Content) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少自动回复内容，请重新新增自动回复"))
			return
		}
		matchType := strings.TrimSpace(pending.MatchType)
		if matchType != "exact" && matchType != "contains" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少触发方式，请重新新增自动回复"))
			return
		}
		if err := h.service.AddAutoReplyByTGGroupIDWithButtons(pending.TGGroupID, pending.Keyword, pending.Content, matchType, msg.Text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "新增自动回复失败："+err.Error()))
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
	case "auto_edit_keyword":
		keyword := strings.TrimSpace(msg.Text)
		if keyword == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "关键词不能为空"))
			return
		}
		matchType := strings.TrimSpace(pending.MatchType)
		if matchType != "exact" && matchType != "contains" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少触发方式，请重新编辑自动回复"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:      "auto_edit_reply",
			TGGroupID: pending.TGGroupID,
			RuleID:    pending.RuleID,
			Page:      pending.Page,
			Keyword:   keyword,
			MatchType: matchType,
		})
		h.render(bot, target, "第3步：请输入新的回复内容（支持换行）", pendingCancelKeyboard(pending.TGGroupID))
		return
	case "auto_edit_reply":
		reply := msg.Text
		if strings.TrimSpace(pending.Keyword) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少关键词，请重新编辑自动回复"))
			return
		}
		matchType := strings.TrimSpace(pending.MatchType)
		if matchType != "exact" && matchType != "contains" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少触发方式，请重新编辑自动回复"))
			return
		}
		if strings.TrimSpace(reply) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "回复内容不能为空"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:      "auto_edit_buttons",
			TGGroupID: pending.TGGroupID,
			RuleID:    pending.RuleID,
			Page:      pending.Page,
			Keyword:   pending.Keyword,
			MatchType: matchType,
			Content:   reply,
		})
		h.render(bot, target, "第4步（可选）：请输入新的链接按钮配置。\n支持格式示例：\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n说明：\n- 按钮文字和网址中间用英文 - 分隔\n- 一行两个按钮用 && 分隔\n发送“跳过”保持当前按钮，发送“关闭”清空按钮", pendingCancelKeyboard(pending.TGGroupID))
		return
	case "auto_edit_buttons":
		if strings.TrimSpace(pending.Keyword) == "" || strings.TrimSpace(pending.Content) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少自动回复内容，请重新编辑"))
			return
		}
		matchType := strings.TrimSpace(pending.MatchType)
		if matchType != "exact" && matchType != "contains" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少触发方式，请重新编辑自动回复"))
			return
		}
		rawButtons := strings.TrimSpace(msg.Text)
		var err error
		if rawButtons == "" || rawButtons == "跳过" {
			err = h.service.UpdateAutoReplyByTGGroupID(pending.TGGroupID, pending.RuleID, pending.Keyword, pending.Content, matchType)
		} else {
			err = h.service.UpdateAutoReplyByTGGroupIDWithButtons(pending.TGGroupID, pending.RuleID, pending.Keyword, pending.Content, matchType, msg.Text)
		}
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "更新自动回复失败："+err.Error()))
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
		publishMsg, sendErr := bot.Send(tgbotapi.NewMessage(pending.TGGroupID, fmt.Sprintf("抽奖已创建：%s（中奖人数:%d）\n发送关键词「%s」即可参与", title, winners, keyword)))
		if sendErr == nil {
			_ = h.service.PinLotteryMessageByTGGroupID(bot, pending.TGGroupID, publishMsg.MessageID, "publish")
		}
		h.sendLotteryPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "sched_add_cron":
		cronExpr := strings.TrimSpace(msg.Text)
		if cronExpr == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "cron 表达式不能为空"))
			return
		}
		if err := h.service.ValidateCronExpr(cronExpr); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "cron 表达式格式错误，请按 5 段格式：分钟 小时 日 月 星期\n示例：0 9 * * *"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:      "sched_add_content",
			TGGroupID: pending.TGGroupID,
			Page:      pending.Page,
			CronExpr:  cronExpr,
		})
		h.render(bot, target, "第2步：请输入要发送的消息内容（支持换行）", pendingCancelKeyboard(pending.TGGroupID))
		return
	case "sched_add_content":
		content := msg.Text
		if strings.TrimSpace(content) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "消息内容不能为空"))
			return
		}
		if strings.TrimSpace(pending.CronExpr) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少 cron 表达式，请重新创建定时消息"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:      "sched_add_buttons",
			TGGroupID: pending.TGGroupID,
			Page:      pending.Page,
			CronExpr:  pending.CronExpr,
			Content:   content,
		})
		h.render(bot, target, "第3步（可选）：请输入链接按钮配置。\n支持格式示例：\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n说明：\n- 按钮文字和网址中间用英文 - 分隔\n- 一行两个按钮用 && 分隔\n发送“跳过”表示不设置按钮，发送“关闭”清空按钮", pendingCancelKeyboard(pending.TGGroupID))
		return
	case "sched_add_buttons":
		if strings.TrimSpace(pending.CronExpr) == "" || strings.TrimSpace(pending.Content) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少定时消息内容，请重新创建"))
			return
		}
		if err := h.service.CreateScheduledMessageByTGGroupIDWithButtons(pending.TGGroupID, pending.Content, pending.CronExpr, msg.Text); err != nil {
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
	case "spam_msg_len":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		if _, err := h.service.SetAntiSpamMaxMessageLengthByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置超长消息长度失败"))
			return
		}
		h.sendAntiSpamPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "spam_name_len":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		if _, err := h.service.SetAntiSpamMaxNameLengthByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置超长姓名长度失败"))
			return
		}
		h.sendAntiSpamPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "spam_exception_add":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "关键词不能为空"))
			return
		}
		if _, err := h.service.AddAntiSpamExceptionByTGGroupID(pending.TGGroupID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "添加例外失败"))
			return
		}
		h.sendAntiSpamPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "spam_exception_remove":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "关键词不能为空"))
			return
		}
		if _, err := h.service.RemoveAntiSpamExceptionByTGGroupID(pending.TGGroupID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "移除例外失败"))
			return
		}
		h.sendAntiSpamPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "night_tz":
		tz, err := h.service.SetNightModeTimezoneByTGGroupID(pending.TGGroupID, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "时区格式错误，请输入如 +8、-5、+8:30、UTC+8"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "夜间模式时区已设置为 "+tz))
		h.sendNightModePanel(bot, target, msg.From.ID, pending.TGGroupID)
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
		if err := h.service.AddBlacklistByTGGroupID(pending.TGGroupID, tgUID, reason); err != nil {
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
		if err := h.service.RemoveBlacklistByTGGroupID(pending.TGGroupID, tgUID); err != nil {
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
			if err := h.service.ClearWelcomeButtonsByTGGroupID(pending.TGGroupID); err != nil {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "清空欢迎按钮失败"))
				return
			}
			h.sendWelcomePanel(bot, target, msg.From.ID, pending.TGGroupID)
			break
		}
		if err := h.service.SetWelcomeButtonsByTGGroupID(pending.TGGroupID, text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "按钮格式错误："+
				err.Error()+"\n\n示例:\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n说明:\n- 按钮文字和网址用英文 - 分隔\n- 一行两个按钮用 && 分隔"))
			return
		}
		h.sendWelcomePanel(bot, target, msg.From.ID, pending.TGGroupID)
	default:
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "未识别输入态，请重新点击菜单操作"))
	}

	h.clearPending(msg.From.ID)
}

func privateHelpText() string {
	lines := []string{
		"常用命令（私聊）：",
		"/start - 打开主菜单",
		"/groups - 查看并管理你的群组",
		"/settings - 打开设置面板",
		"/help - 查看帮助",
		"",
		"常用命令（群内）：",
		"/help - 查看群内命令列表",
		"/lottery_create 标题|人数|口令 - 创建抽奖",
		"/lottery_draw - 立即开奖",
		"/black_add @用户名 原因(可选) - 加入本群黑名单（管理员）",
		"/black_remove @用户名 - 移除本群黑名单（管理员）",
		"",
		"提示：也可以通过私聊面板按钮进行大多数管理操作。",
	}
	return strings.Join(lines, "\n")
}

func groupHelpText() string {
	lines := []string{
		"常用群命令：",
		"/help - 显示帮助",
		"/lottery_create 标题|人数|口令 - 创建抽奖（口令可省略，默认“参加”）",
		"/lottery_draw - 立即开奖",
		"/black_add @用户名 原因(可选) - 加入本群黑名单（管理员）",
		"/black_remove @用户名 - 移除本群黑名单（管理员）",
		"",
		"更多功能可私聊机器人后通过按钮面板管理：/start、/groups。",
	}
	return strings.Join(lines, "\n")
}
