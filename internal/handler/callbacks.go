package handler

import (
	"fmt"
	"strconv"
	"strings"

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
	if feature != "pending" && action != "add" && action != "edit" {
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
			mins, err := h.service.CycleWelcomeDeleteMinutesByTGGroupID(tgGroupID)
			if err != nil {
				h.answerCallback(bot, cb.ID, "切换失败")
				return
			}
			if mins == 0 {
				h.answerCallback(bot, cb.ID, "删除消息：否")
			} else {
				h.answerCallback(bot, cb.ID, fmt.Sprintf("删除消息：%d 分钟", mins))
			}
			h.sendWelcomePanel(bot, target, userID, tgGroupID)
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
			h.render(bot, target, "支持多按钮配置，格式示例：\n官网 - link.com\n电报 - t.me/WeGroupRobot\n官网 - link.com && 电报 - t.me/WeGroupRobot\n说明：\n- 按钮文字和网址中间用英文 - 分隔\n- 一行两个按钮用 && 分隔\n发送“关闭”可清空按钮", pendingCancelKeyboard(tgGroupID))
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
	case "view":
		h.answerCallback(bot, cb.ID, "加载违禁词")
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
		resultMsg, sendErr := bot.Send(tgbotapi.NewMessage(tgGroupID, "开奖结果："+joinWinnerNames(winners)))
		if sendErr == nil {
			_ = h.service.PinLotteryMessageByTGGroupID(bot, tgGroupID, resultMsg.MessageID, "result")
		}
		h.sendLotteryPanel(bot, target, cb.From.ID, tgGroupID)
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
		mins, err := h.service.CycleLotteryDeleteKeywordMinutesByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		if mins > 0 {
			h.answerCallback(bot, cb.ID, fmt.Sprintf("口令消息将于 %d 分钟后删除", mins))
		} else {
			h.answerCallback(bot, cb.ID, "已关闭自动删除口令消息")
		}
		h.sendLotteryPanel(bot, target, cb.From.ID, tgGroupID)
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
		sec, err := h.service.CycleAntiSpamWarnDeleteSecByTGGroupID(tgGroupID)
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
		mins, err := h.service.CycleJoinVerifyTimeoutMinutesByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, fmt.Sprintf("验证时间已设为 %d 分钟", mins))
		h.sendVerifyPanel(bot, target, userID, tgGroupID)
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
		mins, err := h.service.CycleNewbieLimitMinutesByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		h.answerCallback(bot, cb.ID, fmt.Sprintf("限制时长已设为 %d 分钟", mins))
		h.sendNewbieLimitPanel(bot, target, userID, tgGroupID)
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
		n, err := h.service.CycleAntiFloodMaxMessagesByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, fmt.Sprintf("触发条数已设为 %d", n))
		h.sendAntiFloodPanel(bot, target, userID, tgGroupID)
		return
	case "floodwindow":
		sec, err := h.service.CycleAntiFloodWindowSecByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, fmt.Sprintf("检测间隔已设为 %d 秒", sec))
		h.sendAntiFloodPanel(bot, target, userID, tgGroupID)
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
		sec, err := h.service.CycleAntiFloodWarnDeleteSecByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		if sec <= 0 {
			h.answerCallback(bot, cb.ID, "提醒自动删除：关闭")
		} else {
			h.answerCallback(bot, cb.ID, fmt.Sprintf("提醒自动删除：%d 秒", sec))
		}
		h.sendAntiFloodPanel(bot, target, userID, tgGroupID)
		return
	}

	h.answerCallback(bot, cb.ID, "未知操作")
}

func (h *Handler) sendPendingParentPanel(bot *tgbotapi.BotAPI, target renderTarget, userID int64, pending pendingInput) {
	switch pending.Kind {
	case "auto_add", "auto_edit":
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
	case "sched_add":
		page := pending.Page
		if page < 1 {
			page = 1
		}
		h.sendScheduledList(bot, target, userID, pending.TGGroupID, page)
	case "chain_start", "chain_add":
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
	case "invite_create":
		h.sendGroupPanel(bot, target, userID, pending.TGGroupID)
	default:
		h.sendGroupPanel(bot, target, userID, pending.TGGroupID)
	}
}
