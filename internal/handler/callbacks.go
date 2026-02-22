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
		switch action {
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
			h.sendGroupPanel(bot, target, userID, tgGroupID)
		case "set":
			h.answerCallback(bot, cb.ID, "请输入欢迎文案")
			h.setPending(userID, pendingInput{Kind: "welcome_edit", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入新的欢迎文案，支持占位符 {user}\n示例：欢迎 {user} 加入本群", pendingCancelKeyboard(tgGroupID))
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
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
