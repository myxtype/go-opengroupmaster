package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"supervisor/internal/handler/keyboards"
	svc "supervisor/internal/service"

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

func (h *Handler) handleEditedMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg == nil || msg.From == nil || msg.From.IsBot {
		return
	}
	if !msg.Chat.IsGroup() && !msg.Chat.IsSuperGroup() {
		return
	}
	// 编辑消息只做风控检测，避免重复触发命令/积分/自动回复等流程。
	_ = h.service.CheckEditedMessageAndModerate(bot, msg)
}

func (h *Handler) handlePrivateCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	target := renderTarget{ChatID: msg.Chat.ID}
	switch msg.Command() {
	case "start":
		args := strings.TrimSpace(msg.CommandArguments())
		if strings.HasPrefix(args, "chain_") {
			chainID64, err := strconv.ParseUint(strings.TrimPrefix(args, "chain_"), 10, 64)
			if err != nil || chainID64 == 0 {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "接龙参数错误，请回到群里重新点击按钮"))
				return
			}
			view, err := h.service.ChainViewByChainID(uint(chainID64))
			if err != nil || !view.Active {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "当前接龙已结束或不存在，请回到群里查看最新消息"))
				return
			}
			h.setPending(msg.From.ID, pendingInput{Kind: "chain_submit_entry", TGGroupID: view.TGGroupID, ChainID: view.ID})
			existing, ok, _ := h.service.UserChainEntryByChainID(view.ID, msg.From.ID)
			if ok && strings.TrimSpace(existing) != "" {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "👉 请输入您的接龙内容\n\n当前已提交内容：\n"+existing+"\n\n发送新内容即可覆盖修改。"))
			} else {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "👉 请输入您的接龙内容"))
			}
			return
		}
		h.render(bot, target, "欢迎使用 GroupMaster Bot。\n请通过按钮管理群组。", keyboards.MainMenuKeyboard(bot.Self.UserName))
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
		resultText, resultEntities := lotteryResultText(winners)
		result := tgbotapi.NewMessage(msg.Chat.ID, resultText)
		result.Entities = resultEntities
		resultMsg, sendErr := bot.Send(result)
		if sendErr == nil {
			_ = h.service.PinLotteryMessageByTGGroupID(bot, msg.Chat.ID, resultMsg.MessageID, "result")
		}
	case "link":
		sendLinkReply := func(text string) {
			out := tgbotapi.NewMessage(msg.Chat.ID, text)
			out.ReplyToMessageID = msg.MessageID
			_, _ = bot.Send(out)
		}
		res, err := h.service.CreateInviteLinkForUserByTGGroupID(bot, msg.Chat.ID, msg.From.ID)
		if err != nil {
			switch {
			case errors.Is(err, svc.ErrInviteFeatureDisabled):
				sendLinkReply("邀请链接功能未开启，请联系管理员在面板中开启")
			case errors.Is(err, svc.ErrInviteGenerateLimitReached):
				stats, statErr := h.service.InviteUserStatsByTGGroupID(msg.Chat.ID, msg.From.ID)
				if statErr != nil {
					sendLinkReply("当前生成数量已达到上限，暂时无法生成新链接")
					return
				}
				sendLinkReply(fmt.Sprintf("当前生成数量已达到上限，暂时无法生成新链接\n你的邀请统计：\n有效邀请人数：%d\n已生成链接数量：%d", stats.InvitedCount, stats.GeneratedCount))
			default:
				sendLinkReply("生成邀请链接失败")
			}
			return
		}
		lines := []string{
			"你的专属邀请链接：",
			res.Link,
			"",
			"你的邀请统计：",
			fmt.Sprintf("有效邀请人数：%d", res.UserStats.InvitedCount),
			fmt.Sprintf("已生成链接数量：%d", res.UserStats.GeneratedCount),
		}
		if res.GenerateLimit > 0 {
			lines = append(lines, fmt.Sprintf("群组生成总数：%d/%d", res.GroupGenerated, res.GenerateLimit))
		} else {
			lines = append(lines, fmt.Sprintf("群组生成总数：%d", res.GroupGenerated))
		}
		sendLinkReply(strings.Join(lines, "\n"))
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
	case "mute":
		ok, err := h.service.IsAdminByTGGroupID(msg.Chat.ID, msg.From.ID)
		if err != nil || !ok {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "仅群管理员可执行该命令"))
			return
		}
		tgUserID, arg, err := h.resolveModerationTargetAndArg(msg)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用法：/mute @用户名 [分钟]\n也可回复对方消息使用：/mute [分钟]\n默认 60 分钟"))
			return
		}
		minutes := 60
		if strings.TrimSpace(arg) != "" {
			v, pErr := strconv.Atoi(strings.TrimSpace(arg))
			if pErr != nil || v <= 0 {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "禁言分钟数需为大于 0 的整数"))
				return
			}
			minutes = v
		}
		if err := h.service.MuteMemberByTGGroupID(bot, msg.Chat.ID, tgUserID, minutes); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "禁言失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已禁言用户：%d（%d 分钟）", tgUserID, minutes)))
	case "unmute":
		ok, err := h.service.IsAdminByTGGroupID(msg.Chat.ID, msg.From.ID)
		if err != nil || !ok {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "仅群管理员可执行该命令"))
			return
		}
		tgUserID, _, err := h.resolveModerationTargetAndArg(msg)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用法：/unmute @用户名\n也可回复对方消息使用：/unmute"))
			return
		}
		if err := h.service.UnmuteMemberByTGGroupID(bot, msg.Chat.ID, tgUserID); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "解除禁言失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已解除禁言：%d", tgUserID)))
	case "ban":
		ok, err := h.service.IsAdminByTGGroupID(msg.Chat.ID, msg.From.ID)
		if err != nil || !ok {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "仅群管理员可执行该命令"))
			return
		}
		tgUserID, arg, err := h.resolveModerationTargetAndArg(msg)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用法：/ban @用户名 [分钟]\n也可回复对方消息使用：/ban [分钟]\n不填分钟为永久封禁"))
			return
		}
		minutes := 0
		if strings.TrimSpace(arg) != "" {
			v, pErr := strconv.Atoi(strings.TrimSpace(arg))
			if pErr != nil || v <= 0 {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "封禁分钟数需为大于 0 的整数"))
				return
			}
			minutes = v
		}
		if err := h.service.BanMemberByTGGroupID(bot, msg.Chat.ID, tgUserID, minutes); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "封禁失败"))
			return
		}
		if minutes > 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已封禁用户：%d（%d 分钟）", tgUserID, minutes)))
		} else {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已永久封禁用户：%d", tgUserID)))
		}
	case "unban":
		ok, err := h.service.IsAdminByTGGroupID(msg.Chat.ID, msg.From.ID)
		if err != nil || !ok {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "仅群管理员可执行该命令"))
			return
		}
		tgUserID, _, err := h.resolveModerationTargetAndArg(msg)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用法：/unban @用户名\n也可回复对方消息使用：/unban"))
			return
		}
		if err := h.service.UnbanMemberByTGGroupID(bot, msg.Chat.ID, tgUserID); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "解除封禁失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已解除封禁：%d", tgUserID)))
	case "kick":
		ok, err := h.service.IsAdminByTGGroupID(msg.Chat.ID, msg.From.ID)
		if err != nil || !ok {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "仅群管理员可执行该命令"))
			return
		}
		tgUserID, _, err := h.resolveModerationTargetAndArg(msg)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "用法：/kick @用户名\n也可回复对方消息使用：/kick"))
			return
		}
		if err := h.service.KickMemberByTGGroupID(bot, msg.Chat.ID, tgUserID); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "踢出失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已踢出用户：%d", tgUserID)))
	}
}

func (h *Handler) resolveModerationTargetAndArg(msg *tgbotapi.Message) (int64, string, error) {
	if msg == nil {
		return 0, "", fmt.Errorf("invalid message")
	}
	args := strings.TrimSpace(msg.CommandArguments())
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		return msg.ReplyToMessage.From.ID, args, nil
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return 0, "", fmt.Errorf("missing target")
	}
	first := strings.TrimSpace(fields[0])
	var target int64
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
	} else {
		id, err := strconv.ParseInt(first, 10, 64)
		if err != nil {
			return 0, "", fmt.Errorf("invalid target")
		}
		target = id
	}
	extra := ""
	if len(fields) > 1 {
		extra = strings.TrimSpace(strings.Join(fields[1:], " "))
	}
	return target, extra, nil
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

func (h *Handler) resolvePointTarget(msg *tgbotapi.Message, text string) (int64, string, error) {
	if msg != nil && msg.ForwardFrom != nil && !msg.ForwardFrom.IsBot {
		name := displayNameFromUser(msg.ForwardFrom)
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("%d", msg.ForwardFrom.ID)
		}
		return msg.ForwardFrom.ID, name, nil
	}
	raw := strings.TrimSpace(text)
	if raw == "" {
		return 0, "", fmt.Errorf("empty target")
	}
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return id, raw, nil
	}
	username := strings.TrimPrefix(raw, "@")
	if strings.TrimSpace(username) == "" {
		return 0, "", fmt.Errorf("invalid username")
	}
	user, err := h.service.Repo().FindUserByUsername(username)
	if err != nil {
		return 0, "", err
	}
	if strings.TrimSpace(user.Username) != "" {
		return user.TGUserID, "@" + user.Username, nil
	}
	return user.TGUserID, fmt.Sprintf("%d", user.TGUserID), nil
}

func (h *Handler) handlePrivatePendingInput(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	pending, ok := h.getPending(msg.From.ID)
	if !ok {
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请先点击菜单按钮选择操作。"))
		return
	}
	target := renderTarget{ChatID: msg.Chat.ID}
	if pending.Kind != "chain_submit_entry" && !h.ensureAdmin(bot, target, msg.From.ID, pending.TGGroupID) {
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
		h.render(bot, target, "第3步：请输入自动回复内容（支持换行）", keyboards.PendingCancelKeyboard(pending.TGGroupID))
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
		h.render(bot, target, "第4步（可选）：请输入链接按钮配置。\n支持格式示例：\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n说明：\n- 按钮文字和网址中间用英文 - 分隔\n- 一行两个按钮用 && 分隔\n发送“跳过”表示不设置按钮，发送“关闭”清空按钮", keyboards.PendingCancelKeyboard(pending.TGGroupID))
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
		h.render(bot, target, "第3步：请输入新的回复内容（支持换行）", keyboards.PendingCancelKeyboard(pending.TGGroupID))
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
		h.render(bot, target, "第4步（可选）：请输入新的链接按钮配置。\n支持格式示例：\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n说明：\n- 按钮文字和网址中间用英文 - 分隔\n- 一行两个按钮用 && 分隔\n发送“跳过”保持当前按钮，发送“关闭”清空按钮", keyboards.PendingCancelKeyboard(pending.TGGroupID))
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
	case "bw_warn_threshold":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		if _, err := h.service.SetBannedWordWarnThresholdByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置警告次数失败"))
			return
		}
		h.sendBannedWordPenaltyPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "bw_warn_action_mute_minutes":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		if _, err := h.service.SetBannedWordWarnActionMuteMinutesByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置阈值后禁言时长失败"))
			return
		}
		h.sendBannedWordPenaltyPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "bw_warn_action_ban_minutes":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		if _, err := h.service.SetBannedWordWarnActionBanMinutesByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置阈值后封禁时长失败"))
			return
		}
		h.sendBannedWordPenaltyPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "bw_mute_minutes":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		if _, err := h.service.SetBannedWordMuteMinutesByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置禁言时长失败"))
			return
		}
		h.sendBannedWordPenaltyPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "bw_ban_minutes":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		if _, err := h.service.SetBannedWordBanMinutesByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置封禁时长失败"))
			return
		}
		h.sendBannedWordPenaltyPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "bw_warn_delete_minutes":
		v, err := strconv.Atoi(text)
		if err != nil || v < 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于等于 0 的整数"))
			return
		}
		if _, err := h.service.SetBannedWordWarnDeleteMinutesByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置删除提醒失败"))
			return
		}
		h.sendBannedWordList(bot, target, msg.From.ID, pending.TGGroupID, 1)
	case "lottery_create":
		parts := strings.Split(text, "|")
		if len(parts) != 3 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式错误，请按：抽奖标题|中奖人数|参与关键词"))
			return
		}
		title := strings.TrimSpace(parts[0])
		winners, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		keyword := strings.TrimSpace(parts[2])
		if title == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "抽奖标题不能为空"))
			return
		}
		if err != nil || winners <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "中奖人数需为大于 0 的整数"))
			return
		}
		if keyword == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "参与关键词不能为空"))
			return
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
	case "lottery_create_title":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "抽奖标题不能为空，请重新输入"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:         "lottery_create_winners",
			TGGroupID:    pending.TGGroupID,
			LotteryTitle: text,
		})
		h.render(bot, target, "第2步：请输入中奖人数（大于 0 的整数）\n示例：3", keyboards.PendingCancelKeyboard(pending.TGGroupID))
		return
	case "lottery_create_winners":
		winners, err := strconv.Atoi(text)
		if err != nil || winners <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "中奖人数需为大于 0 的整数，请重新输入"))
			return
		}
		if strings.TrimSpace(pending.LotteryTitle) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "抽奖标题已丢失，请重新创建抽奖"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:           "lottery_create_keyword",
			TGGroupID:      pending.TGGroupID,
			LotteryTitle:   pending.LotteryTitle,
			LotteryWinners: winners,
		})
		h.render(bot, target, "第3步：请输入参与关键词\n示例：参加", keyboards.PendingCancelKeyboard(pending.TGGroupID))
		return
	case "lottery_create_keyword":
		if strings.TrimSpace(pending.LotteryTitle) == "" || pending.LotteryWinners <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "抽奖配置已丢失，请重新创建抽奖"))
			return
		}
		keyword := text
		if keyword == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "参与关键词不能为空，请重新输入"))
			return
		}
		if _, err := h.service.CreateLotteryByTGGroupIDWithKeyword(pending.TGGroupID, pending.LotteryTitle, pending.LotteryWinners, keyword); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建抽奖失败"))
			return
		}
		publishMsg, sendErr := bot.Send(tgbotapi.NewMessage(
			pending.TGGroupID,
			fmt.Sprintf("抽奖已创建：%s（中奖人数:%d）\n发送关键词「%s」即可参与", pending.LotteryTitle, pending.LotteryWinners, keyword),
		))
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
		h.render(bot, target, "第2步：请输入要发送的消息内容。\n支持：\n- 纯文本（支持换行）\n- 图片/视频/文件/动图（可带文字说明）", keyboards.PendingCancelKeyboard(pending.TGGroupID))
		return
	case "sched_add_content":
		if strings.TrimSpace(pending.CronExpr) == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少 cron 表达式，请重新创建定时消息"))
			return
		}
		content := strings.TrimSpace(msg.Text)
		mediaType := ""
		mediaFileID := ""
		switch {
		case len(msg.Photo) > 0:
			mediaType = "photo"
			mediaFileID = msg.Photo[len(msg.Photo)-1].FileID
			content = strings.TrimSpace(msg.Caption)
		case msg.Video != nil:
			mediaType = "video"
			mediaFileID = msg.Video.FileID
			content = strings.TrimSpace(msg.Caption)
		case msg.Document != nil:
			mediaType = "document"
			mediaFileID = msg.Document.FileID
			content = strings.TrimSpace(msg.Caption)
		case msg.Animation != nil:
			mediaType = "animation"
			mediaFileID = msg.Animation.FileID
			content = strings.TrimSpace(msg.Caption)
		}
		if content == "" && mediaType == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "消息内容不能为空；可发送文本，或发送图片/视频/文件/动图（可选文字说明）"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:        "sched_add_buttons",
			TGGroupID:   pending.TGGroupID,
			Page:        pending.Page,
			CronExpr:    pending.CronExpr,
			Content:     content,
			MediaType:   mediaType,
			MediaFileID: mediaFileID,
		})
		h.render(bot, target, "第3步（可选）：请输入链接按钮配置。\n支持格式示例：\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n说明：\n- 按钮文字和网址中间用英文 - 分隔\n- 一行两个按钮用 && 分隔\n发送“跳过”表示不设置按钮，发送“关闭”清空按钮", keyboards.PendingCancelKeyboard(pending.TGGroupID))
		return
	case "sched_add_buttons":
		if strings.TrimSpace(pending.CronExpr) == "" || (strings.TrimSpace(pending.Content) == "" && strings.TrimSpace(pending.MediaType) == "") {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少定时消息内容，请重新创建"))
			return
		}
		rawButtons := strings.TrimSpace(msg.Text)
		if rawButtons == "" {
			rawButtons = "跳过"
		}
		pending.Kind = "sched_add_pin"
		pending.RawButtons = rawButtons
		h.setPending(msg.From.ID, pending)
		h.render(bot, target, "第4步：请选择发送后是否自动置顶", keyboards.ScheduledPinSelectKeyboard(pending.TGGroupID))
		return
	case "sched_edit_text":
		if pending.RuleID == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少任务 ID，请重新进入编辑面板"))
			return
		}
		if err := h.service.UpdateScheduledTextByTGGroupID(pending.TGGroupID, pending.RuleID, msg.Text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "修改文本失败："+err.Error()))
			return
		}
		h.sendScheduledEditPanel(bot, target, msg.From.ID, pending.TGGroupID, pending.RuleID, pending.Page)
	case "sched_edit_cron":
		if pending.RuleID == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少任务 ID，请重新进入编辑面板"))
			return
		}
		if err := h.service.UpdateScheduledCronByTGGroupID(pending.TGGroupID, pending.RuleID, msg.Text); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "修改 Cron 失败："+err.Error()))
			return
		}
		h.sendScheduledEditPanel(bot, target, msg.From.ID, pending.TGGroupID, pending.RuleID, pending.Page)
	case "sched_edit_buttons":
		if pending.RuleID == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少任务 ID，请重新进入编辑面板"))
			return
		}
		rawButtons := strings.TrimSpace(msg.Text)
		if rawButtons == "" {
			rawButtons = "关闭"
		}
		if err := h.service.UpdateScheduledButtonsByTGGroupID(pending.TGGroupID, pending.RuleID, rawButtons); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "修改按钮失败："+err.Error()))
			return
		}
		h.sendScheduledEditPanel(bot, target, msg.From.ID, pending.TGGroupID, pending.RuleID, pending.Page)
	case "sched_edit_media":
		if pending.RuleID == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "缺少任务 ID，请重新进入编辑面板"))
			return
		}
		if strings.TrimSpace(text) == "关闭" {
			if err := h.service.UpdateScheduledMediaByTGGroupID(pending.TGGroupID, pending.RuleID, "", ""); err != nil {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "清空媒体失败："+err.Error()))
				return
			}
			h.sendScheduledEditPanel(bot, target, msg.From.ID, pending.TGGroupID, pending.RuleID, pending.Page)
			break
		}
		mediaType := ""
		mediaFileID := ""
		caption := ""
		switch {
		case len(msg.Photo) > 0:
			mediaType = "photo"
			mediaFileID = msg.Photo[len(msg.Photo)-1].FileID
			caption = msg.Caption
		case msg.Video != nil:
			mediaType = "video"
			mediaFileID = msg.Video.FileID
			caption = msg.Caption
		case msg.Document != nil:
			mediaType = "document"
			mediaFileID = msg.Document.FileID
			caption = msg.Caption
		case msg.Animation != nil:
			mediaType = "animation"
			mediaFileID = msg.Animation.FileID
			caption = msg.Caption
		default:
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请发送图片/视频/文件/动图，或发送“关闭”清空媒体"))
			return
		}
		if err := h.service.UpdateScheduledMediaByTGGroupID(pending.TGGroupID, pending.RuleID, mediaType, mediaFileID); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "修改媒体失败："+err.Error()))
			return
		}
		if strings.TrimSpace(caption) != "" {
			_ = h.service.UpdateScheduledTextByTGGroupID(pending.TGGroupID, pending.RuleID, caption)
		}
		h.sendScheduledEditPanel(bot, target, msg.From.ID, pending.TGGroupID, pending.RuleID, pending.Page)
	case "invite_set_expire":
		if text == "0" {
			if _, err := h.service.SetInviteExpireDateByTGGroupID(pending.TGGroupID, 0); err != nil {
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置失败："+err.Error()))
				return
			}
			h.sendInvitePanel(bot, target, msg.From.ID, pending.TGGroupID)
			break
		}
		expireAt, err := time.ParseInLocation("2006-01-02 15:04", text, time.Local)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "时间格式错误，请按格式输入：2026-02-24 17:09"))
			return
		}
		if _, err := h.service.SetInviteExpireDateByTGGroupID(pending.TGGroupID, expireAt.Unix()); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置失败："+err.Error()))
			return
		}
		h.sendInvitePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "invite_set_member_limit":
		v, err := strconv.Atoi(text)
		if err != nil || v < 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入有效数字，0 表示不限制"))
			return
		}
		if _, err := h.service.SetInviteMemberLimitByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置失败："+err.Error()))
			return
		}
		h.sendInvitePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "invite_set_generate_limit":
		v, err := strconv.Atoi(text)
		if err != nil || v < 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入有效数字，0 表示不限制"))
			return
		}
		if _, err := h.service.SetInviteGenerateLimitByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置失败："+err.Error()))
			return
		}
		h.sendInvitePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "chain_create_count":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		pending.Count = v
		if pending.ChainMode == "both" {
			pending.Kind = "chain_create_duration"
			h.setPending(msg.From.ID, pending)
			h.render(bot, target, "第3步：请选择多久后截止", keyboards.ChainDurationKeyboard(pending.TGGroupID))
		} else {
			pending.Kind = "chain_create_intro"
			h.setPending(msg.From.ID, pending)
			h.render(bot, target, "第3步：请输入接龙规则或介绍", keyboards.PendingCancelKeyboard(pending.TGGroupID))
		}
		return
	case "chain_create_intro":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入接龙规则或介绍"))
			return
		}
		chainID, err := h.service.StartChainByTGGroupID(pending.TGGroupID, text, pending.Count, pending.Deadline)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "创建接龙失败："+err.Error()))
			return
		}
		h.syncChainAnnouncementByID(bot, chainID)
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "接龙创建成功，已自动发布到群里"))
		h.sendChainPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "chain_submit_entry":
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "接龙内容不能为空，请重新输入"))
			return
		}
		if pending.ChainID == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "接龙参数缺失，请回到群里重新点击参与按钮"))
			return
		}
		if err := h.service.SubmitChainEntryByChainID(pending.ChainID, msg.From.ID, displayNameFromUser(msg.From), text); err != nil {
			switch {
			case errors.Is(err, svc.ErrChainNotActive):
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "当前接龙已结束"))
			case errors.Is(err, svc.ErrChainDeadlineReached):
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "当前接龙已截止"))
			case errors.Is(err, svc.ErrChainParticipantLimitReached):
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "当前接龙人数已满"))
			default:
				_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "提交失败："+err.Error()))
			}
			return
		}
		h.syncChainAnnouncementByID(bot, pending.ChainID)
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "接龙成功！"))
		h.setPending(msg.From.ID, pending)
		return
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
	case "spam_warn_threshold":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置警告次数失败", h.service.SetAntiSpamWarnThresholdByTGGroupID, h.sendAntiSpamPenaltyPanel)
	case "spam_warn_action_mute_minutes":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置阈值后禁言时长失败", h.service.SetAntiSpamWarnActionMuteMinutesByTGGroupID, h.sendAntiSpamPenaltyPanel)
	case "spam_warn_action_ban_minutes":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置阈值后封禁时长失败", h.service.SetAntiSpamWarnActionBanMinutesByTGGroupID, h.sendAntiSpamPenaltyPanel)
	case "spam_mute_minutes":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置禁言时长失败", h.service.SetAntiSpamMuteMinutesByTGGroupID, h.sendAntiSpamPenaltyPanel)
	case "spam_ban_minutes":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置封禁时长失败", h.service.SetAntiSpamBanMinutesByTGGroupID, h.sendAntiSpamPenaltyPanel)
	case "spam_ai_spam_score":
		v, err := strconv.Atoi(text)
		if err != nil || v < 1 || v > 100 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入 1~100 的整数"))
			return
		}
		if _, err := h.service.SetAntiSpamAISpamScoreByTGGroupID(pending.TGGroupID, v); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置 AI 垃圾分失败"))
			return
		}
		h.sendAntiSpamAIPanel(bot, target, msg.From.ID, pending.TGGroupID)
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
	case "flood_warn_threshold":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置警告次数失败", h.service.SetAntiFloodWarnThresholdByTGGroupID, h.sendAntiFloodPenaltyPanel)
	case "flood_warn_action_mute_minutes":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置阈值后禁言时长失败", h.service.SetAntiFloodWarnActionMuteMinutesByTGGroupID, h.sendAntiFloodPenaltyPanel)
	case "flood_warn_action_ban_minutes":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置阈值后封禁时长失败", h.service.SetAntiFloodWarnActionBanMinutesByTGGroupID, h.sendAntiFloodPenaltyPanel)
	case "flood_mute_minutes":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置禁言时长失败", h.service.SetAntiFloodMuteMinutesByTGGroupID, h.sendAntiFloodPenaltyPanel)
	case "flood_ban_minutes":
		h.handleModerationDurationInput(bot, msg, target, pending, text, "设置封禁时长失败", h.service.SetAntiFloodBanMinutesByTGGroupID, h.sendAntiFloodPenaltyPanel)
	case "night_tz":
		tz, err := h.service.SetNightModeTimezoneByTGGroupID(pending.TGGroupID, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "时区格式错误，请输入如 +8、-5、+8:30、UTC+8"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "夜间模式时区已设置为 "+tz))
		h.sendNightModePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "night_start_hour":
		hour, err := h.service.SetNightModeStartHourByTGGroupID(pending.TGGroupID, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "开始小时格式错误，请输入 0-23 的整数"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{Kind: "night_end_hour", TGGroupID: pending.TGGroupID})
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("开始小时已设置为 %02d:00\n请继续输入结束小时（0-23）", hour)))
		return
	case "night_end_hour":
		hour, err := h.service.SetNightModeEndHourByTGGroupID(pending.TGGroupID, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "结束小时格式错误，请输入 0-23 的整数"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("结束小时已设置为 %02d:00", hour)))
		h.sendNightModePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_checkin_keyword":
		keyword, err := h.service.SetPointsCheckinKeywordByTGGroupID(pending.TGGroupID, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "签到口令不能为空"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "签到口令已设置为："+keyword))
		h.sendPointsCheckinPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_checkin_reward":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		reward, err := h.service.SetPointsCheckinRewardByTGGroupID(pending.TGGroupID, v)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置签到奖励失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("签到奖励已设置为：%d", reward)))
		h.sendPointsCheckinPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_message_reward":
		v, err := strconv.Atoi(text)
		if err != nil || v <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		reward, err := h.service.SetPointsMessageRewardByTGGroupID(pending.TGGroupID, v)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置发言奖励失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("发言奖励已设置为：%d", reward)))
		h.sendPointsMessagePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_message_daily":
		v, err := strconv.Atoi(text)
		if err != nil || v < 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于等于 0 的整数"))
			return
		}
		limit, err := h.service.SetPointsMessageDailyLimitByTGGroupID(pending.TGGroupID, v)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置发言每日上限失败"))
			return
		}
		if limit <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "发言每日上限：无限制"))
		} else {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("发言每日上限已设置为：%d", limit)))
		}
		h.sendPointsMessagePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_message_min_len":
		v, err := strconv.Atoi(text)
		if err != nil || v < 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于等于 0 的整数"))
			return
		}
		minLen, err := h.service.SetPointsMessageMinLenByTGGroupID(pending.TGGroupID, v)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置最小字数失败"))
			return
		}
		if minLen <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "最小字数长度：无限制"))
		} else {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("最小字数长度已设置为：%d", minLen)))
		}
		h.sendPointsMessagePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_invite_reward":
		v, err := strconv.Atoi(text)
		if err != nil || v < 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于等于 0 的整数"))
			return
		}
		reward, err := h.service.SetPointsInviteRewardByTGGroupID(pending.TGGroupID, v)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置邀请奖励失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("邀请奖励已设置为：%d", reward)))
		h.sendPointsInvitePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_invite_daily":
		v, err := strconv.Atoi(text)
		if err != nil || v < 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于等于 0 的整数"))
			return
		}
		limit, err := h.service.SetPointsInviteDailyLimitByTGGroupID(pending.TGGroupID, v)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "设置邀请每日上限失败"))
			return
		}
		if limit <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "邀请每日上限：无限制"))
		} else {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("邀请每日上限已设置为：%d", limit)))
		}
		h.sendPointsInvitePanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_balance_alias":
		alias, err := h.service.SetPointsBalanceAliasByTGGroupID(pending.TGGroupID, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "积分别名不能为空"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "积分别名已设置为："+alias))
		h.sendPointsPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_rank_alias":
		alias, err := h.service.SetPointsRankAliasByTGGroupID(pending.TGGroupID, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "排行别名不能为空"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "排行别名已设置为："+alias))
		h.sendPointsPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_admin_add":
		targetTGUserID, display, err := h.resolvePointTarget(msg, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "目标用户解析失败，请输入用户名、用户ID，或转发成员消息"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:        "points_admin_add_value",
			TGGroupID:   pending.TGGroupID,
			TargetTGUID: targetTGUserID,
			TargetLabel: display,
		})
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("目标用户：%s\n请输入要增加的积分数值（正整数）", display)))
		return
	case "points_admin_sub":
		targetTGUserID, display, err := h.resolvePointTarget(msg, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "目标用户解析失败，请输入用户名、用户ID，或转发成员消息"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:        "points_admin_sub_value",
			TGGroupID:   pending.TGGroupID,
			TargetTGUID: targetTGUserID,
			TargetLabel: display,
		})
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("目标用户：%s\n请输入要扣除的积分数值（正整数）", display)))
		return
	case "points_admin_add_value":
		if pending.TargetTGUID == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "目标用户已失效，请重新点击“增加积分”"))
			return
		}
		value, err := strconv.Atoi(text)
		if err != nil || value <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		applied, current, err := h.service.AdjustPointsByTargetTGUserID(pending.TGGroupID, pending.TargetTGUID, value)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "增加积分失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已为 %s 增加积分：%d\n当前积分：%d", pending.TargetLabel, applied, current)))
		h.sendPointsPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "points_admin_sub_value":
		if pending.TargetTGUID == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "目标用户已失效，请重新点击“扣除积分”"))
			return
		}
		value, err := strconv.Atoi(text)
		if err != nil || value <= 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
			return
		}
		applied, current, err := h.service.AdjustPointsByTargetTGUserID(pending.TGGroupID, pending.TargetTGUID, -value)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "扣除积分失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已为 %s 扣除积分：%d\n当前积分：%d", pending.TargetLabel, -applied, current)))
		h.sendPointsPanel(bot, target, msg.From.ID, pending.TGGroupID)
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
		targetTGUserID, display, err := h.resolvePointTarget(msg, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "目标用户解析失败，请输入用户名、用户ID，或转发成员消息"))
			return
		}
		h.setPending(msg.From.ID, pendingInput{
			Kind:        "black_add_reason",
			TGGroupID:   pending.TGGroupID,
			TargetTGUID: targetTGUserID,
			TargetLabel: display,
		})
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("目标用户：%s\n请输入加入黑名单原因（可选，发送“跳过”使用默认原因）", display)))
		return
	case "black_add_reason":
		if pending.TargetTGUID == 0 {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "目标用户已失效，请重新点击“添加”"))
			return
		}
		reason := strings.TrimSpace(msg.Text)
		if reason == "" || reason == "跳过" {
			reason = "panel_manual_add"
		}
		if err := h.service.AddBlacklistByTGGroupID(pending.TGGroupID, pending.TargetTGUID, reason); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "加入黑名单失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已加入黑名单：%s", pending.TargetLabel)))
		h.sendBlacklistPanel(bot, target, msg.From.ID, pending.TGGroupID)
	case "black_remove":
		targetTGUserID, display, err := h.resolvePointTarget(msg, text)
		if err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "目标用户解析失败，请输入用户名、用户ID，或转发成员消息"))
			return
		}
		if err := h.service.RemoveBlacklistByTGGroupID(pending.TGGroupID, targetTGUserID); err != nil {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "移除黑名单失败"))
			return
		}
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("已移除黑名单：%s", display)))
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

func (h *Handler) handleModerationDurationInput(
	bot *tgbotapi.BotAPI,
	msg *tgbotapi.Message,
	target renderTarget,
	pending pendingInput,
	text string,
	failMessage string,
	setFn func(int64, int) (int, error),
	sendPanelFn func(*tgbotapi.BotAPI, renderTarget, int64, int64),
) {
	v, err := strconv.Atoi(text)
	if err != nil || v <= 0 {
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "请输入大于 0 的整数"))
		return
	}
	if _, err := setFn(pending.TGGroupID, v); err != nil {
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, failMessage))
		return
	}
	sendPanelFn(bot, target, msg.From.ID, pending.TGGroupID)
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
		"/link - 生成专属邀请链接并查看邀请统计",
		"/black_add @用户名 原因(可选) - 加入本群黑名单（管理员）",
		"/black_remove @用户名 - 移除本群黑名单（管理员）",
		"/mute @用户名 [分钟] - 禁言用户（管理员，默认60分钟）",
		"/unmute @用户名 - 解除禁言（管理员）",
		"/ban @用户名 [分钟] - 封禁用户（管理员，不填为永久）",
		"/unban @用户名 - 解除封禁（管理员）",
		"/kick @用户名 - 踢出用户（管理员）",
		"回复用户消息发送 /black_add - 直接拉黑该用户（管理员）",
		"回复用户消息发送 /black_remove - 直接移除该用户黑名单（管理员）",
		"回复用户消息发送 /mute [分钟] / /ban [分钟] / /kick（管理员）",
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
		"/link - 生成专属邀请链接并查看邀请统计",
		"发送“签到” - 每日签到获取积分（可配置）",
		"发送“积分” - 查询个人积分（可配置）",
		"发送“积分排行” - 查询积分排行（可配置）",
		"/black_add @用户名 原因(可选) - 加入本群黑名单（管理员）",
		"/black_remove @用户名 - 移除本群黑名单（管理员）",
		"/mute @用户名 [分钟] - 禁言用户（管理员，默认60分钟）",
		"/unmute @用户名 - 解除禁言（管理员）",
		"/ban @用户名 [分钟] - 封禁用户（管理员，不填为永久）",
		"/unban @用户名 - 解除封禁（管理员）",
		"/kick @用户名 - 踢出用户（管理员）",
		"回复用户消息发送 /black_add - 直接拉黑该用户（管理员）",
		"回复用户消息发送 /black_remove - 直接移除该用户黑名单（管理员）",
		"回复用户消息发送 /mute [分钟] / /ban [分钟] / /kick（管理员）",
		"",
		"更多功能可私聊机器人后通过按钮面板管理：/start、/groups。",
	}
	return strings.Join(lines, "\n")
}
