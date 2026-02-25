package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
	mode := parts[1]
	if mode == "pass" {
		mode = "button"
	}
	tgGroupID, err1 := strconv.ParseInt(parts[2], 10, 64)
	tgUserID, err2 := strconv.ParseInt(parts[3], 10, 64)
	if err1 != nil || err2 != nil {
		h.answerCallback(bot, cb.ID, "参数错误")
		return
	}
	answer := ""
	if len(parts) >= 5 {
		answer = parts[4]
	}
	if err := h.service.PassVerification(bot, tgGroupID, tgUserID, cb.From.ID, mode, answer); err != nil {
		h.answerCallback(bot, cb.ID, "验证失败或已过期")
		return
	}
	h.answerCallback(bot, cb.ID, "验证通过")
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
	keepPending := feature == "chain" && (action == "limmode" || action == "setdur")
	if feature != "pending" && action != "add" && action != "edit" && !keepPending {
		h.clearPending(userID)
	}

	switch feature {
	case "pending":
		switch action {
		case "cancel":
			h.clearPending(userID)
			h.answerCallback(bot, cb.ID, "已取消")
			h.sendGroupPanel(bot, target, userID, tgGroupID)
		case "back":
			pending, ok := h.getPending(userID)
			h.clearPending(userID)
			if !ok {
				h.answerCallback(bot, cb.ID, "无可返回的上级面板")
				h.sendGroupPanel(bot, target, userID, tgGroupID)
				return
			}
			h.answerCallback(bot, cb.ID, "已返回上级面板")
			h.sendPendingParentPanel(bot, target, userID, pending)
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "welcome":
		switch action {
		case "noop":
			h.answerCallback(bot, cb.ID, "")
			return
		case "view":
			h.answerCallback(bot, cb.ID, "加载欢迎设置")
			h.sendWelcomePanel(bot, target, userID, tgGroupID)
		case "on":
			if _, err := h.service.SetWelcomeEnabledByTGGroupID(tgGroupID, true); err != nil {
				h.answerCallback(bot, cb.ID, "切换失败")
				return
			}
			h.answerCallback(bot, cb.ID, "欢迎消息已开启")
			h.sendWelcomePanel(bot, target, userID, tgGroupID)
		case "off":
			if _, err := h.service.SetWelcomeEnabledByTGGroupID(tgGroupID, false); err != nil {
				h.answerCallback(bot, cb.ID, "切换失败")
				return
			}
			h.answerCallback(bot, cb.ID, "欢迎消息已关闭")
			h.sendWelcomePanel(bot, target, userID, tgGroupID)
		case "toggle":
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
			h.sendWelcomePanel(bot, target, userID, tgGroupID)
		case "mode":
			mode, err := h.service.ToggleWelcomeModeByTGGroupID(tgGroupID)
			if err != nil {
				h.answerCallback(bot, cb.ID, "切换失败")
				return
			}
			if mode == "verify" {
				h.answerCallback(bot, cb.ID, "模式：验证后欢迎")
			} else {
				h.answerCallback(bot, cb.ID, "模式：进群欢迎")
			}
			h.sendWelcomePanel(bot, target, userID, tgGroupID)
		case "delmins":
			h.answerCallback(bot, cb.ID, "请选择删除时间")
			h.sendWelcomeDeleteMinutesPanel(bot, target, userID, tgGroupID)
		case "delminsset":
			if len(parts) < 5 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			v, err := strconv.Atoi(parts[4])
			if err != nil {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			mins, err := h.service.SetWelcomeDeleteMinutesByTGGroupID(tgGroupID, v)
			if err != nil {
				h.answerCallback(bot, cb.ID, "设置失败")
				return
			}
			if mins == 0 {
				h.answerCallback(bot, cb.ID, "删除消息：否")
			} else {
				h.answerCallback(bot, cb.ID, fmt.Sprintf("删除消息：%d 分钟", mins))
			}
			h.sendWelcomeDeleteMinutesPanel(bot, target, userID, tgGroupID)
		case "set":
			h.answerCallback(bot, cb.ID, "请输入欢迎文案")
			h.setPending(userID, pendingInput{Kind: "welcome_edit", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入新的欢迎文案，支持占位符 {user}\n示例：欢迎 {user} 加入本群", pendingCancelKeyboard(tgGroupID))
		case "media":
			h.answerCallback(bot, cb.ID, "请发送欢迎图片")
			h.setPending(userID, pendingInput{Kind: "welcome_edit_media", TGGroupID: tgGroupID})
			h.render(bot, target, "请发送一张图片作为欢迎媒体，或发送“关闭”清空媒体", pendingCancelKeyboard(tgGroupID))
		case "button":
			h.answerCallback(bot, cb.ID, "请输入按钮")
			h.setPending(userID, pendingInput{Kind: "welcome_edit_button", TGGroupID: tgGroupID})
			h.render(bot, target, "支持多按钮配置，格式示例：\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n说明：\n- 按钮文字和网址中间用英文 - 分隔\n- 一行两个按钮用 && 分隔\n发送“关闭”可清空按钮", pendingCancelKeyboard(tgGroupID))
		case "preview":
			if err := h.service.SendWelcomePreviewByTGGroupID(bot, tgGroupID, target.ChatID, userID); err != nil {
				h.answerCallback(bot, cb.ID, "预览失败")
				return
			}
			h.answerCallback(bot, cb.ID, "已发送预览")
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "auto":
		h.handleAutoReplyFeature(bot, cb, target, userID, tgGroupID, action, parts)
	case "bw":
		h.handleBannedWordFeature(bot, cb, target, userID, tgGroupID, action, parts)
	case "lottery":
		h.handleLotteryFeature(bot, cb, target, tgGroupID, action, parts)
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
		h.handleModerationFeature(bot, cb, target, userID, tgGroupID, action, parts)
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
			h.render(bot, target, "请输入：tg_user_id|原因(可选)\n将只加入当前群黑名单", pendingCancelKeyboard(tgGroupID))
		case "remove":
			h.answerCallback(bot, cb.ID, "请输入用户ID")
			h.setPending(userID, pendingInput{Kind: "black_remove", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入要移除的 tg_user_id\n将只影响当前群黑名单", pendingCancelKeyboard(tgGroupID))
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "invite":
		switch action {
		case "noop":
			h.answerCallback(bot, cb.ID, "")
		case "view":
			h.answerCallback(bot, cb.ID, "加载邀请设置")
			h.sendInvitePanel(bot, target, userID, tgGroupID)
		case "on":
			if _, err := h.service.SetInviteEnabledByTGGroupID(tgGroupID, true); err != nil {
				h.answerCallback(bot, cb.ID, "设置失败")
				return
			}
			h.answerCallback(bot, cb.ID, "邀请链接已开启")
			h.sendInvitePanel(bot, target, userID, tgGroupID)
		case "off":
			if _, err := h.service.SetInviteEnabledByTGGroupID(tgGroupID, false); err != nil {
				h.answerCallback(bot, cb.ID, "设置失败")
				return
			}
			h.answerCallback(bot, cb.ID, "邀请链接已关闭")
			h.sendInvitePanel(bot, target, userID, tgGroupID)
		case "expire":
			h.answerCallback(bot, cb.ID, "请输入过期时间")
			h.setPending(userID, pendingInput{Kind: "invite_set_expire", TGGroupID: tgGroupID})
			h.render(bot, target, "1. 配置过期时间\n👉 请回复链接过期时间(不限制请输入:0)\n\n注意:此设置仅应用在新生成的链接中，不会修改已生成的链接\n\n格式:年-月-日 时:分\n例如:2026-02-24 17:09", inviteExpireInputKeyboard(tgGroupID))
		case "expireunlimit":
			if _, err := h.service.SetInviteExpireDateByTGGroupID(tgGroupID, 0); err != nil {
				h.answerCallback(bot, cb.ID, "设置失败")
				return
			}
			h.answerCallback(bot, cb.ID, "链接过期时间：无限制")
			h.sendInvitePanel(bot, target, userID, tgGroupID)
		case "member":
			h.answerCallback(bot, cb.ID, "请输入最大邀请人数")
			h.setPending(userID, pendingInput{Kind: "invite_set_member_limit", TGGroupID: tgGroupID})
			h.render(bot, target, "2. 最大邀请数配置\n\n👉 邀请达到设定人数后链接失效\n\n注意:此设置仅应用在新生成的链接中，不会修改已生成的链接\n\n请回复单个链接最大邀请人数(不限制请输入:0)", inviteMemberInputKeyboard(tgGroupID))
		case "memberunlimit":
			if _, err := h.service.SetInviteMemberLimitByTGGroupID(tgGroupID, 0); err != nil {
				h.answerCallback(bot, cb.ID, "设置失败")
				return
			}
			h.answerCallback(bot, cb.ID, "最大邀请人数：无限制")
			h.sendInvitePanel(bot, target, userID, tgGroupID)
		case "gen":
			h.answerCallback(bot, cb.ID, "请输入生成数量上限")
			h.setPending(userID, pendingInput{Kind: "invite_set_generate_limit", TGGroupID: tgGroupID})
			h.render(bot, target, "3. 生成数量限制配置\n\n👉 生成链接数量达到设定数量后，不再生成新的链接\n\n注意:此设置仅应用在新生成的链接中，不会修改已生成的链接\n\n请回复生成链接数量上限(不限制请输入:0)", inviteGenerateInputKeyboard(tgGroupID))
		case "genunlimit":
			if _, err := h.service.SetInviteGenerateLimitByTGGroupID(tgGroupID, 0); err != nil {
				h.answerCallback(bot, cb.ID, "设置失败")
				return
			}
			h.answerCallback(bot, cb.ID, "生成数量上限：无限制")
			h.sendInvitePanel(bot, target, userID, tgGroupID)
		case "export":
			name, content, err := h.service.ExportInviteCSVByTGGroupID(tgGroupID)
			if err != nil {
				h.answerCallback(bot, cb.ID, "导出失败")
				return
			}
			doc := tgbotapi.NewDocument(target.ChatID, tgbotapi.FileBytes{Name: name, Bytes: content})
			doc.Caption = "邀请数据 CSV 导出"
			_, _ = bot.Send(doc)
			h.answerCallback(bot, cb.ID, "已导出")
			h.sendInvitePanel(bot, target, userID, tgGroupID)
		case "clear":
			if err := h.service.ClearInviteDataByTGGroupID(tgGroupID); err != nil {
				h.answerCallback(bot, cb.ID, "清空失败")
				return
			}
			h.answerCallback(bot, cb.ID, "邀请数据已清空")
			h.sendInvitePanel(bot, target, userID, tgGroupID)
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "chain":
		switch action {
		case "view":
			h.answerCallback(bot, cb.ID, "加载接龙")
			h.sendChainPanel(bot, target, userID, tgGroupID)
		case "start":
			h.answerCallback(bot, cb.ID, "请选择限制方式")
			h.setPending(userID, pendingInput{Kind: "chain_create_mode", TGGroupID: tgGroupID})
			h.render(bot, target, "第1步：请选择接龙限制方式", chainLimitModeKeyboard(tgGroupID))
		case "limmode":
			if len(parts) < 5 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			mode := strings.TrimSpace(parts[4])
			switch mode {
			case "none":
				h.answerCallback(bot, cb.ID, "已设置为不限制")
				h.setPending(userID, pendingInput{Kind: "chain_create_intro", TGGroupID: tgGroupID, ChainMode: mode, Count: 0, Deadline: 0})
				h.render(bot, target, "第2步：请输入接龙规则或介绍", pendingCancelKeyboard(tgGroupID))
			case "people":
				h.answerCallback(bot, cb.ID, "请输入限制人数")
				h.setPending(userID, pendingInput{Kind: "chain_create_count", TGGroupID: tgGroupID, ChainMode: mode})
				h.render(bot, target, "第2步：请输入接龙人数上限（正整数）", pendingCancelKeyboard(tgGroupID))
			case "time":
				h.answerCallback(bot, cb.ID, "请选择截止时间")
				h.setPending(userID, pendingInput{Kind: "chain_create_duration", TGGroupID: tgGroupID, ChainMode: mode})
				h.render(bot, target, "第2步：请选择多久后截止", chainDurationKeyboard(tgGroupID))
			case "both":
				h.answerCallback(bot, cb.ID, "请输入限制人数")
				h.setPending(userID, pendingInput{Kind: "chain_create_count", TGGroupID: tgGroupID, ChainMode: mode})
				h.render(bot, target, "第2步：请输入接龙人数上限（正整数）", pendingCancelKeyboard(tgGroupID))
			default:
				h.answerCallback(bot, cb.ID, "未知限制方式")
			}
		case "setdur":
			if len(parts) < 5 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			pending, ok := h.getPending(userID)
			if !ok || pending.TGGroupID != tgGroupID || pending.Kind != "chain_create_duration" {
				h.answerCallback(bot, cb.ID, "创建流程已过期，请重新开始")
				h.sendChainPanel(bot, target, userID, tgGroupID)
				return
			}
			sec, err := strconv.ParseInt(parts[4], 10, 64)
			if err != nil || sec < 0 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			deadline := int64(0)
			if sec > 0 {
				deadline = time.Now().Unix() + sec
			}
			pending.Kind = "chain_create_intro"
			pending.Deadline = deadline
			h.setPending(userID, pending)
			if deadline > 0 {
				h.answerCallback(bot, cb.ID, "截止时间已设置")
				h.render(bot, target, "第3步：请输入接龙规则或介绍\n截止时间："+chainDeadlineText(deadline), pendingCancelKeyboard(tgGroupID))
			} else {
				h.answerCallback(bot, cb.ID, "已设置为无截止时间")
				h.render(bot, target, "第3步：请输入接龙规则或介绍\n截止时间：不限时", pendingCancelKeyboard(tgGroupID))
			}
		case "export":
			if len(parts) < 5 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			chainID64, err := strconv.ParseUint(parts[4], 10, 64)
			if err != nil || chainID64 == 0 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			name, content, err := h.service.ExportChainCSVByTGGroupIDAndChainID(tgGroupID, uint(chainID64))
			if err != nil {
				h.answerCallback(bot, cb.ID, "导出失败")
				return
			}
			doc := tgbotapi.NewDocument(target.ChatID, tgbotapi.FileBytes{Name: name, Bytes: content})
			doc.Caption = "接龙数据 CSV 导出"
			_, _ = bot.Send(doc)
			h.answerCallback(bot, cb.ID, "已导出")
			h.sendChainPanel(bot, target, userID, tgGroupID)
		case "close":
			if len(parts) < 5 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			chainID64, err := strconv.ParseUint(parts[4], 10, 64)
			if err != nil || chainID64 == 0 {
				h.answerCallback(bot, cb.ID, "参数错误")
				return
			}
			if err := h.service.CloseChainByTGGroupIDAndChainID(tgGroupID, uint(chainID64)); err != nil {
				h.answerCallback(bot, cb.ID, "关闭失败")
				return
			}
			h.answerCallback(bot, cb.ID, "接龙已关闭")
			h.syncChainAnnouncementByID(bot, uint(chainID64))
			h.sendChainPanel(bot, target, userID, tgGroupID)
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "poll":
		switch action {
		case "view":
			h.answerCallback(bot, cb.ID, "加载投票")
			h.sendPollPanel(bot, target, userID, tgGroupID)
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
			h.sendPollPanel(bot, target, userID, tgGroupID)
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
	case "view":
		h.answerCallback(bot, cb.ID, "加载自动回复")
		h.sendAutoReplyList(bot, target, userID, tgGroupID, 1)
	case "add":
		h.answerCallback(bot, cb.ID, "请选择触发方式")
		h.setPending(userID, pendingInput{Kind: "auto_add_mode", TGGroupID: tgGroupID, Page: 1})
		h.render(bot, target, "第1步：请选择触发方式\n精准触发：消息内容与关键词完全相同才触发\n包含触发：消息内容中包含关键词就触发", autoReplyMatchTypeKeyboard(tgGroupID, fmt.Sprintf("feat:auto:addmode:%d", tgGroupID)))
	case "addmode":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		matchType := strings.TrimSpace(parts[4])
		if matchType != "exact" && matchType != "contains" {
			h.answerCallback(bot, cb.ID, "触发方式错误")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入关键词")
		h.setPending(userID, pendingInput{
			Kind:      "auto_add_keyword",
			TGGroupID: tgGroupID,
			Page:      1,
			MatchType: matchType,
		})
		h.render(bot, target, "第2步：请输入自动回复关键词", pendingCancelKeyboard(tgGroupID))
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
		h.answerCallback(bot, cb.ID, "请选择触发方式")
		h.setPending(userID, pendingInput{Kind: "auto_edit_mode", TGGroupID: tgGroupID, RuleID: uint(id), Page: page})
		h.render(bot, target, "第1步：请选择新触发方式\n精准触发：消息内容与关键词完全相同才触发\n包含触发：消息内容中包含关键词就触发", autoReplyMatchTypeKeyboard(tgGroupID, fmt.Sprintf("feat:auto:editmode:%d:%d:%d", tgGroupID, id, page)))
	case "editmode":
		if len(parts) < 7 {
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
		matchType := strings.TrimSpace(parts[6])
		if matchType != "exact" && matchType != "contains" {
			h.answerCallback(bot, cb.ID, "触发方式错误")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入关键词")
		h.setPending(userID, pendingInput{
			Kind:      "auto_edit_keyword",
			TGGroupID: tgGroupID,
			RuleID:    uint(id),
			Page:      page,
			MatchType: matchType,
		})
		h.render(bot, target, "第2步：请输入新的关键词", pendingCancelKeyboard(tgGroupID))
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleBannedWordFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "noop":
		h.answerCallback(bot, cb.ID, "")
	case "view":
		h.answerCallback(bot, cb.ID, "加载违禁词")
		h.sendBannedWordList(bot, target, userID, tgGroupID, 1)
	case "on":
		if _, err := h.service.SetBannedWordEnabledByTGGroupID(tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "违禁词已开启")
		h.sendBannedWordList(bot, target, userID, tgGroupID, 1)
	case "off":
		if _, err := h.service.SetBannedWordEnabledByTGGroupID(tgGroupID, false); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "违禁词已关闭")
		h.sendBannedWordList(bot, target, userID, tgGroupID, 1)
	case "penalty":
		h.answerCallback(bot, cb.ID, "加载惩罚设置")
		h.sendBannedWordPenaltyPanel(bot, target, userID, tgGroupID)
	case "warn", "mute":
		h.answerCallback(bot, cb.ID, "请在惩罚面板设置")
		h.sendBannedWordPenaltyPanel(bot, target, userID, tgGroupID)
	case "penaltyset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		if _, err := h.service.SetBannedWordPenaltyByTGGroupID(tgGroupID, parts[4]); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "惩罚已更新")
		h.sendBannedWordPenaltyPanel(bot, target, userID, tgGroupID)
	case "warncount":
		h.answerCallback(bot, cb.ID, "请输入警告次数")
		h.setPending(userID, pendingInput{Kind: "bw_warn_threshold", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入达到处罚前的警告次数（正整数，例如 3）", pendingCancelKeyboard(tgGroupID))
	case "warnaction":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		if _, err := h.service.SetBannedWordWarnActionByTGGroupID(tgGroupID, parts[4]); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "阈值后动作已更新")
		h.sendBannedWordPenaltyPanel(bot, target, userID, tgGroupID)
	case "warnmuteinput":
		h.answerCallback(bot, cb.ID, "请输入阈值禁言时长")
		h.setPending(userID, pendingInput{Kind: "bw_warn_action_mute_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入警告达到阈值后禁言时长（分钟，1-10080）", pendingCancelKeyboard(tgGroupID))
	case "warnbaninput":
		h.answerCallback(bot, cb.ID, "请输入阈值封禁时长")
		h.setPending(userID, pendingInput{Kind: "bw_warn_action_ban_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入警告达到阈值后封禁时长（分钟，1-10080）", pendingCancelKeyboard(tgGroupID))
	case "muteinput":
		h.answerCallback(bot, cb.ID, "请输入禁言时长")
		h.setPending(userID, pendingInput{Kind: "bw_mute_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入禁言时长（分钟，1-10080）", pendingCancelKeyboard(tgGroupID))
	case "baninput":
		h.answerCallback(bot, cb.ID, "请输入封禁时长")
		h.setPending(userID, pendingInput{Kind: "bw_ban_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入封禁时长（分钟，1-10080）", pendingCancelKeyboard(tgGroupID))
	case "delwarninput":
		h.answerCallback(bot, cb.ID, "请输入删除提醒时长")
		h.setPending(userID, pendingInput{Kind: "bw_warn_delete_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入提醒消息自动删除时长（分钟，0-1440；0 表示不自动删除）", pendingCancelKeyboard(tgGroupID))
	case "delwarn":
		h.answerCallback(bot, cb.ID, "请输入删除提醒时长")
		h.setPending(userID, pendingInput{Kind: "bw_warn_delete_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入提醒消息自动删除时长（分钟，0-1440；0 表示不自动删除）", pendingCancelKeyboard(tgGroupID))
	case "delwarnset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		v, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		mins, err := h.service.SetBannedWordWarnDeleteMinutesByTGGroupID(tgGroupID, v)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "删除提醒："+bannedWordDeleteText(mins))
		h.sendBannedWordList(bot, target, userID, tgGroupID, 1)
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

func (h *Handler) handleLotteryFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, tgGroupID int64, action string, parts []string) {
	switch action {
	case "view":
		h.answerCallback(bot, cb.ID, "加载抽奖")
		h.sendLotteryPanel(bot, target, cb.From.ID, tgGroupID)
	case "create":
		h.answerCallback(bot, cb.ID, "请发送抽奖配置")
		h.setPending(cb.From.ID, pendingInput{Kind: "lottery_create", TGGroupID: tgGroupID})
		h.render(bot, target, "请发送：抽奖标题|中奖人数|参与关键词\n示例：周末福利|3|参加", pendingCancelKeyboard(tgGroupID))
	case "draw":
		winners, err := h.service.DrawActiveLotteryByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "开奖失败")
			view, viewErr := h.service.LotteryPanelViewByTGGroupID(tgGroupID)
			if viewErr != nil {
				h.render(bot, target, "开奖失败：没有可开奖的活动抽奖", groupPanelKeyboard(tgGroupID))
				return
			}
			h.render(bot, target, "开奖失败：没有可开奖的活动抽奖", lotteryKeyboard(tgGroupID, view.PublishPin, view.ResultPin, view.DeleteKeywordMins))
			return
		}
		h.answerCallback(bot, cb.ID, "开奖完成")
		resultText, resultEntities := lotteryResultText(winners)
		result := tgbotapi.NewMessage(tgGroupID, resultText)
		result.Entities = resultEntities
		resultMsg, sendErr := bot.Send(result)
		if sendErr == nil {
			_ = h.service.PinLotteryMessageByTGGroupID(bot, tgGroupID, resultMsg.MessageID, "result")
		}
		h.sendLotteryPanel(bot, target, cb.From.ID, tgGroupID)
	case "records":
		page := 1
		if len(parts) >= 5 {
			if p, err := strconv.Atoi(parts[4]); err == nil {
				page = p
			}
		}
		h.answerCallback(bot, cb.ID, "加载抽奖记录")
		h.sendLotteryRecordsPanel(bot, target, cb.From.ID, tgGroupID, page)
	case "cancel":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		lotteryID, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		ok, err := h.service.CancelLotteryByTGGroupID(tgGroupID, uint(lotteryID))
		if err != nil {
			h.answerCallback(bot, cb.ID, "取消失败")
			return
		}
		if !ok {
			h.answerCallback(bot, cb.ID, "仅可取消未开奖活动")
		} else {
			h.answerCallback(bot, cb.ID, "已取消抽奖活动")
		}
		h.sendLotteryRecordsPanel(bot, target, cb.From.ID, tgGroupID, page)
	case "toggle":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		on, err := h.service.ToggleLotteryConfigByTGGroupID(tgGroupID, parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		if on {
			h.answerCallback(bot, cb.ID, "已开启")
		} else {
			h.answerCallback(bot, cb.ID, "已关闭")
		}
		h.sendLotteryPanel(bot, target, cb.From.ID, tgGroupID)
	case "delmins":
		h.answerCallback(bot, cb.ID, "请选择删除时长")
		h.sendLotteryDeleteMinutesPanel(bot, target, cb.From.ID, tgGroupID)
	case "delminsset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		v, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		mins, err := h.service.SetLotteryDeleteKeywordMinutesByTGGroupID(tgGroupID, v)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		if mins > 0 {
			h.answerCallback(bot, cb.ID, fmt.Sprintf("口令和参与成功提示消息将于 %d 分钟后删除", mins))
		} else {
			h.answerCallback(bot, cb.ID, "已关闭自动删除口令和参与成功提示消息")
		}
		h.sendLotteryDeleteMinutesPanel(bot, target, cb.From.ID, tgGroupID)
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleScheduleFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "view":
		h.answerCallback(bot, cb.ID, "加载定时消息")
		h.sendScheduledList(bot, target, userID, tgGroupID, 1)
	case "add":
		h.answerCallback(bot, cb.ID, "请发送定时消息")
		h.setPending(userID, pendingInput{Kind: "sched_add_cron", TGGroupID: tgGroupID, Page: 1})
		h.render(bot, target, "第1步：请输入 cron 表达式\n含义：分钟 小时 日 月 星期（共5段，用空格分隔）\n示例：\n- 0 9 * * *  （每天 09:00）\n- */30 * * * *（每30分钟）\n- 0 21 * * 1-5（工作日 21:00）\n输入后将进入第2步填写消息内容（支持换行），第3步可选设置链接按钮", pendingCancelKeyboard(tgGroupID))
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

func (h *Handler) handleModerationFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "noop":
		h.answerCallback(bot, cb.ID, "")
		return
	case "spam", "spamview":
		h.answerCallback(bot, cb.ID, "加载反垃圾")
		h.sendAntiSpamPanel(bot, target, userID, tgGroupID)
		return
	case "spamon":
		if _, err := h.service.SetAntiSpamEnabledByTGGroupID(tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "反垃圾已开启")
		h.sendAntiSpamPanel(bot, target, userID, tgGroupID)
		return
	case "spamoff":
		if _, err := h.service.SetAntiSpamEnabledByTGGroupID(tgGroupID, false); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "反垃圾已关闭")
		h.sendAntiSpamPanel(bot, target, userID, tgGroupID)
		return
	case "spamopt":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		enabled, err := h.service.ToggleAntiSpamOptionByTGGroupID(tgGroupID, parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		if enabled {
			h.answerCallback(bot, cb.ID, "已启用")
		} else {
			h.answerCallback(bot, cb.ID, "已关闭")
		}
		h.sendAntiSpamPanel(bot, target, userID, tgGroupID)
		return
	case "spampenalty":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		penalty, err := h.service.SetAntiSpamPenaltyByTGGroupID(tgGroupID, parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "惩罚已设为 "+antiFloodPenaltyText(penalty, 60))
		h.sendAntiSpamPanel(bot, target, userID, tgGroupID)
		return
	case "spammsglen":
		view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "加载失败")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入消息最大长度")
		h.setPending(userID, pendingInput{Kind: "spam_msg_len", TGGroupID: tgGroupID})
		h.render(bot, target, fmt.Sprintf("检测到消息内容长度大于设定数时，将会判定为超长消息并处罚。\n当前设置最大长度:%d\n👉 输入允许的消息最大长度（例如:100）:", view.MaxMessageLength), pendingCancelKeyboard(tgGroupID))
		return
	case "spamnamelen":
		view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "加载失败")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入姓名最大长度")
		h.setPending(userID, pendingInput{Kind: "spam_name_len", TGGroupID: tgGroupID})
		h.render(bot, target, fmt.Sprintf("检测到姓名长度大于设定数时，将会判定为超长姓名并处罚。\n当前设置最大长度:%d\n👉 输入允许的姓名最大长度（例如:32）:", view.MaxNameLength), pendingCancelKeyboard(tgGroupID))
		return
	case "spamexadd":
		h.answerCallback(bot, cb.ID, "请输入例外关键词")
		h.setPending(userID, pendingInput{Kind: "spam_exception_add", TGGroupID: tgGroupID})
		h.render(bot, target, "输入一个例外关键词（命中后跳过反垃圾检测）", pendingCancelKeyboard(tgGroupID))
		return
	case "spamexdel":
		h.answerCallback(bot, cb.ID, "请输入要移除的关键词")
		h.setPending(userID, pendingInput{Kind: "spam_exception_remove", TGGroupID: tgGroupID})
		h.render(bot, target, "输入要移除的例外关键词（精确匹配，不区分大小写）", pendingCancelKeyboard(tgGroupID))
		return
	case "spamalertdel":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "请选择具体秒数")
			h.sendAntiSpamPanel(bot, target, userID, tgGroupID)
			return
		}
		secValue, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		sec, err := h.service.SetAntiSpamWarnDeleteSecByTGGroupID(tgGroupID, secValue)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		if sec <= 0 {
			h.answerCallback(bot, cb.ID, "提醒自动删除：关闭")
		} else {
			h.answerCallback(bot, cb.ID, fmt.Sprintf("提醒自动删除：%d 秒", sec))
		}
		h.sendAntiSpamPanel(bot, target, userID, tgGroupID)
		return
	case "flood", "floodview":
		h.answerCallback(bot, cb.ID, "加载反刷屏")
		h.sendAntiFloodPanel(bot, target, userID, tgGroupID)
		return
	case "floodon":
		if _, err := h.service.SetAntiFloodEnabledByTGGroupID(tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "反刷屏已开启")
		h.sendAntiFloodPanel(bot, target, userID, tgGroupID)
		return
	case "verify", "verifyview":
		h.answerCallback(bot, cb.ID, "加载验证设置")
		h.sendVerifyPanel(bot, target, userID, tgGroupID)
		return
	case "verifyon":
		if _, err := h.service.SetJoinVerifyEnabledByTGGroupID(tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "进群验证已开启")
		h.sendVerifyPanel(bot, target, userID, tgGroupID)
		return
	case "verifyoff":
		if _, err := h.service.SetJoinVerifyEnabledByTGGroupID(tgGroupID, false); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "进群验证已关闭")
		h.sendVerifyPanel(bot, target, userID, tgGroupID)
		return
	case "verifytime":
		h.answerCallback(bot, cb.ID, "请选择验证时间")
		h.sendVerifyTimeoutMinutesPanel(bot, target, userID, tgGroupID)
		return
	case "verifytimeset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		v, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		mins, err := h.service.SetJoinVerifyTimeoutMinutesByTGGroupID(tgGroupID, v)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, fmt.Sprintf("验证时间已设为 %d 分钟", mins))
		h.sendVerifyTimeoutMinutesPanel(bot, target, userID, tgGroupID)
		return
	case "verifytimeout":
		actionName, err := h.service.ToggleJoinVerifyTimeoutActionByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "验证超时已设为 "+verifyTimeoutActionLabel(actionName))
		h.sendVerifyPanel(bot, target, userID, tgGroupID)
		return
	case "verifymethod":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		mode, err := h.service.SetJoinVerifyTypeByTGGroupID(tgGroupID, parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "验证方式已设为 "+verifyTypeLabel(mode))
		h.sendVerifyPanel(bot, target, userID, tgGroupID)
		return
	case "verifytype":
		mode, err := h.service.CycleJoinVerifyTypeByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "验证方式已切换为 "+verifyTypeLabel(mode))
		h.sendVerifyPanel(bot, target, userID, tgGroupID)
		return
	case "newbie", "newbieview":
		h.answerCallback(bot, cb.ID, "加载新成员限制")
		h.sendNewbieLimitPanel(bot, target, userID, tgGroupID)
		return
	case "night", "nightview":
		h.answerCallback(bot, cb.ID, "加载夜间模式")
		h.sendNightModePanel(bot, target, userID, tgGroupID)
		return
	case "nighton":
		if _, err := h.service.SetNightModeEnabledByTGGroupID(tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "夜间模式已开启")
		h.sendNightModePanel(bot, target, userID, tgGroupID)
		return
	case "nightoff":
		if _, err := h.service.SetNightModeEnabledByTGGroupID(tgGroupID, false); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "夜间模式已关闭")
		h.sendNightModePanel(bot, target, userID, tgGroupID)
		return
	case "nighttz":
		view, err := h.service.NightModeViewByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "加载失败")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入时区")
		h.setPending(userID, pendingInput{Kind: "night_tz", TGGroupID: tgGroupID})
		h.render(bot, target, fmt.Sprintf("当前时区:%s\n请输入时区（示例：+8、-5、+8:30、UTC+8）", view.TimezoneText), pendingCancelKeyboard(tgGroupID))
		return
	case "nightmode":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		mode, err := h.service.SetNightModeModeByTGGroupID(tgGroupID, parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "处理方式已设为 "+nightModeActionLabel(mode))
		h.sendNightModePanel(bot, target, userID, tgGroupID)
		return
	case "newbieon":
		if _, err := h.service.SetNewbieLimitEnabledByTGGroupID(tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "新成员限制已开启")
		h.sendNewbieLimitPanel(bot, target, userID, tgGroupID)
		return
	case "newbieoff":
		if _, err := h.service.SetNewbieLimitEnabledByTGGroupID(tgGroupID, false); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "新成员限制已关闭")
		h.sendNewbieLimitPanel(bot, target, userID, tgGroupID)
		return
	case "newbietime":
		h.answerCallback(bot, cb.ID, "请选择限制时长")
		h.sendNewbieLimitMinutesPanel(bot, target, userID, tgGroupID)
		return
	case "newbietimeset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		v, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		mins, err := h.service.SetNewbieLimitMinutesByTGGroupID(tgGroupID, v)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, fmt.Sprintf("限制时长已设为 %d 分钟", mins))
		h.sendNewbieLimitMinutesPanel(bot, target, userID, tgGroupID)
		return
	case "floodoff":
		if _, err := h.service.SetAntiFloodEnabledByTGGroupID(tgGroupID, false); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "反刷屏已关闭")
		h.sendAntiFloodPanel(bot, target, userID, tgGroupID)
		return
	case "floodcount":
		h.answerCallback(bot, cb.ID, "请选择触发条数")
		h.sendAntiFloodCountPanel(bot, target, userID, tgGroupID)
		return
	case "floodcountset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		v, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		n, err := h.service.SetAntiFloodMaxMessagesByTGGroupID(tgGroupID, v)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, fmt.Sprintf("触发条数已设为 %d", n))
		h.sendAntiFloodCountPanel(bot, target, userID, tgGroupID)
		return
	case "floodwindow":
		h.answerCallback(bot, cb.ID, "请选择检测间隔")
		h.sendAntiFloodWindowPanel(bot, target, userID, tgGroupID)
		return
	case "floodwindowset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		v, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		sec, err := h.service.SetAntiFloodWindowSecByTGGroupID(tgGroupID, v)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, fmt.Sprintf("检测间隔已设为 %d 秒", sec))
		h.sendAntiFloodWindowPanel(bot, target, userID, tgGroupID)
		return
	case "floodpenalty":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		penalty, err := h.service.SetAntiFloodPenaltyByTGGroupID(tgGroupID, parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "惩罚已设为 "+antiFloodPenaltyText(penalty, 60))
		h.sendAntiFloodPanel(bot, target, userID, tgGroupID)
		return
	case "floodalertdel":
		h.answerCallback(bot, cb.ID, "请选择删除提醒时长")
		h.sendAntiFloodAlertDeletePanel(bot, target, userID, tgGroupID)
		return
	case "floodalertset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		secValue, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		sec, err := h.service.SetAntiFloodWarnDeleteSecByTGGroupID(tgGroupID, secValue)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		if sec <= 0 {
			h.answerCallback(bot, cb.ID, "提醒自动删除：关闭")
		} else {
			h.answerCallback(bot, cb.ID, fmt.Sprintf("提醒自动删除：%d 秒", sec))
		}
		h.sendAntiFloodAlertDeletePanel(bot, target, userID, tgGroupID)
		return
	}

	h.answerCallback(bot, cb.ID, "未知操作")
}

func (h *Handler) sendPendingParentPanel(bot *tgbotapi.BotAPI, target renderTarget, userID int64, pending pendingInput) {
	switch pending.Kind {
	case "auto_add", "auto_add_mode", "auto_add_keyword", "auto_add_reply", "auto_add_buttons", "auto_edit", "auto_edit_mode", "auto_edit_keyword", "auto_edit_reply", "auto_edit_buttons":
		page := pending.Page
		if page < 1 {
			page = 1
		}
		h.sendAutoReplyList(bot, target, userID, pending.TGGroupID, page)
	case "bw_add", "bw_edit":
		page := pending.Page
		if page < 1 {
			page = 1
		}
		h.sendBannedWordList(bot, target, userID, pending.TGGroupID, page)
	case "lottery_create":
		h.sendLotteryPanel(bot, target, userID, pending.TGGroupID)
	case "sched_add_cron", "sched_add_content", "sched_add_buttons":
		page := pending.Page
		if page < 1 {
			page = 1
		}
		h.sendScheduledList(bot, target, userID, pending.TGGroupID, page)
	case "chain_create_mode", "chain_create_count", "chain_create_duration", "chain_create_intro":
		h.sendChainPanel(bot, target, userID, pending.TGGroupID)
	case "poll_create":
		h.sendPollPanel(bot, target, userID, pending.TGGroupID)
	case "monitor_add", "monitor_remove":
		h.sendMonitorPanel(bot, target, userID, pending.TGGroupID)
	case "rbac_set_role", "rbac_set_acl":
		h.sendRBACPanel(bot, target, userID, pending.TGGroupID)
	case "black_add", "black_remove":
		h.sendBlacklistPanel(bot, target, userID, pending.TGGroupID)
	case "welcome_edit", "welcome_edit_media", "welcome_edit_button":
		h.sendWelcomePanel(bot, target, userID, pending.TGGroupID)
	case "spam_msg_len", "spam_name_len", "spam_exception_add", "spam_exception_remove":
		h.sendAntiSpamPanel(bot, target, userID, pending.TGGroupID)
	case "night_tz":
		h.sendNightModePanel(bot, target, userID, pending.TGGroupID)
	case "invite_set_expire", "invite_set_member_limit", "invite_set_generate_limit":
		h.sendInvitePanel(bot, target, userID, pending.TGGroupID)
	default:
		h.sendGroupPanel(bot, target, userID, pending.TGGroupID)
	}
}
