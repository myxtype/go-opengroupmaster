package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"supervisor/internal/handler/keyboards"
	svc "supervisor/internal/service"
	"time"

	"slices"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) handleCallback(bot *tgbot.Bot, cb *models.CallbackQuery) {
	msg := callbackMessage(cb)
	if cb == nil || msg == nil {
		return
	}
	target := renderTarget{ChatID: msg.Chat.ID, MessageID: msg.ID, Edit: true}
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

func (h *Handler) handleVerifyCallback(bot *tgbot.Bot, cb *models.CallbackQuery) {
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
	// 入群验证挑战
	if err := h.service.PassVerification(bot, cb, tgGroupID, tgUserID, mode, answer); err != nil {
		if errors.Is(err, svc.ErrVerifyWrongAnswer) {
			h.answerCallbackAlert(bot, cb.ID, "答案错误，请重试")
			return
		}
		h.answerCallback(bot, cb.ID, "验证失败或已过期")
		return
	}
	h.answerCallback(bot, cb.ID, "验证通过")
}

func (h *Handler) handleFeatureCallback(bot *tgbot.Bot, cb *models.CallbackQuery, target renderTarget, userID int64, data string) {
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

	// reminder buttons only respond to admin clicks without re-rendering the panel.
	isReminderAction := slices.Contains([]string{"spamunlock", "spamwarnrevoke", "floodwarnrevoke", "bwwarnrevoke"}, action)
	if feature == "mod" && isReminderAction {
		ok, err := h.service.IsAdminByTGGroupID(tgGroupID, userID)
		if err != nil || !ok {
			h.answerCallback(bot, cb.ID, "你不是该群管理员，或机器人尚未同步该群权限")
			return
		}
	} else {
		if !h.ensureAdmin(bot, target, userID, tgGroupID) {
			h.answerCallback(bot, cb.ID, "无权限")
			return
		}
	}
	if perm := permissionFeatureKey(feature, action); perm != "" {
		ok, err := h.service.CanAccessFeatureByTGGroupID(tgGroupID, userID, perm)
		if err != nil || !ok {
			h.answerCallback(bot, cb.ID, "该功能无权限")
			return
		}
	}
	keepPending := (feature == "chain" && slices.Contains([]string{"limmode", "setdur"}, action)) ||
		(feature == "sched" && action == "pinset") ||
		(feature == "poll" && slices.Contains([]string{"submit", "reset"}, action))
	if !slices.Contains([]string{"pending"}, feature) && !slices.Contains([]string{"add", "edit"}, action) && !keepPending {
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
			h.render(bot, target, "请输入新的欢迎文案，支持占位符 {user}\n示例：欢迎 {user} 加入本群", keyboards.PendingCancelKeyboard(tgGroupID))
		case "media":
			h.answerCallback(bot, cb.ID, "请发送欢迎图片")
			h.setPending(userID, pendingInput{Kind: "welcome_edit_media", TGGroupID: tgGroupID})
			h.render(bot, target, "请发送一张图片作为欢迎媒体，或发送“关闭”清空媒体", keyboards.PendingCancelKeyboard(tgGroupID))
		case "button":
			h.answerCallback(bot, cb.ID, "请输入按钮")
			h.setPending(userID, pendingInput{Kind: "welcome_edit_button", TGGroupID: tgGroupID})
			h.render(bot, target, "支持多按钮配置，格式示例：\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n说明：\n- 按钮文字和网址中间用英文 - 分隔\n- 一行两个按钮用 && 分隔\n发送“关闭”可清空按钮", keyboards.PendingCancelKeyboard(tgGroupID))
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
	case "points":
		switch action {
		case "noop":
			h.answerCallback(bot, cb.ID, "")
		case "view":
			h.answerCallback(bot, cb.ID, "加载积分设置")
			h.sendPointsPanel(bot, target, userID, tgGroupID)
		case "checkin":
			h.answerCallback(bot, cb.ID, "加载签到规则")
			h.sendPointsCheckinPanel(bot, target, userID, tgGroupID)
		case "message":
			h.answerCallback(bot, cb.ID, "加载发言规则")
			h.sendPointsMessagePanel(bot, target, userID, tgGroupID)
		case "invite":
			h.answerCallback(bot, cb.ID, "加载邀请规则")
			h.sendPointsInvitePanel(bot, target, userID, tgGroupID)
		case "on":
			if _, err := h.service.SetPointsEnabledByTGGroupID(tgGroupID, true); err != nil {
				h.answerCallback(bot, cb.ID, "设置失败")
				return
			}
			h.answerCallback(bot, cb.ID, "积分系统已开启")
			h.sendPointsPanel(bot, target, userID, tgGroupID)
		case "off":
			if _, err := h.service.SetPointsEnabledByTGGroupID(tgGroupID, false); err != nil {
				h.answerCallback(bot, cb.ID, "设置失败")
				return
			}
			h.answerCallback(bot, cb.ID, "积分系统已关闭")
			h.sendPointsPanel(bot, target, userID, tgGroupID)
		case "checkinkey":
			h.answerCallback(bot, cb.ID, "请输入签到口令")
			h.setPending(userID, pendingInput{Kind: "points_checkin_keyword", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入签到口令（例如：签到）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "checkinreward":
			h.answerCallback(bot, cb.ID, "请输入签到奖励")
			h.setPending(userID, pendingInput{Kind: "points_checkin_reward", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入签到奖励积分（大于 0 的整数）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "msgreward":
			h.answerCallback(bot, cb.ID, "请输入发言奖励")
			h.setPending(userID, pendingInput{Kind: "points_message_reward", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入每条发言奖励积分（大于 0 的整数）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "msgdaily":
			h.answerCallback(bot, cb.ID, "请输入发言每日上限")
			h.setPending(userID, pendingInput{Kind: "points_message_daily", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入发言每日获取上限积分（0 表示无限制）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "msgmin":
			h.answerCallback(bot, cb.ID, "请输入最小字数")
			h.setPending(userID, pendingInput{Kind: "points_message_min_len", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入发言最小字数长度（0 表示无限制）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "invitereward":
			h.answerCallback(bot, cb.ID, "请输入邀请奖励")
			h.setPending(userID, pendingInput{Kind: "points_invite_reward", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入每次邀请奖励积分（>=0）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "invitedaily":
			h.answerCallback(bot, cb.ID, "请输入邀请每日上限")
			h.setPending(userID, pendingInput{Kind: "points_invite_daily", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入邀请每日获取上限积分（0 表示无限制）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "aliasbalance":
			h.answerCallback(bot, cb.ID, "请输入积分别名")
			h.setPending(userID, pendingInput{Kind: "points_balance_alias", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入查询个人积分的关键词（例如：积分）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "aliasrank":
			h.answerCallback(bot, cb.ID, "请输入排行别名")
			h.setPending(userID, pendingInput{Kind: "points_rank_alias", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入查询积分排行的关键词（例如：积分排行）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "add":
			h.answerCallback(bot, cb.ID, "请输入目标用户")
			h.setPending(userID, pendingInput{Kind: "points_admin_add", TGGroupID: tgGroupID})
			h.render(bot, target, "增加积分\n第1步：请输入用户名，用户ID，或转发成员消息到这里\n第2步：再输入要增加的积分数值（正整数）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "sub":
			h.answerCallback(bot, cb.ID, "请输入目标用户")
			h.setPending(userID, pendingInput{Kind: "points_admin_sub", TGGroupID: tgGroupID})
			h.render(bot, target, "扣除积分\n第1步：请输入用户名，用户ID，或转发成员消息到这里\n第2步：再输入要扣除的积分数值（正整数）", keyboards.PendingCancelKeyboard(tgGroupID))
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
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
			_, _ = sendDocumentBytes(bot, target.ChatID, name, content, "日志 CSV 导出")
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
			h.render(bot, target, "请输入：tg_user_id|role\nrole: super_admin 或 admin", keyboards.PendingCancelKeyboard(tgGroupID))
		case "setacl":
			h.answerCallback(bot, cb.ID, "请输入权限配置")
			h.setPending(userID, pendingInput{Kind: "rbac_set_acl", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入：feature|role1,role2\n示例：lottery|super_admin", keyboards.PendingCancelKeyboard(tgGroupID))
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "black":
		switch action {
		case "view":
			h.answerCallback(bot, cb.ID, "加载黑名单")
			h.sendBlacklistPanel(bot, target, userID, tgGroupID)
		case "add":
			h.answerCallback(bot, cb.ID, "请输入目标用户")
			h.setPending(userID, pendingInput{Kind: "black_add", TGGroupID: tgGroupID})
			h.render(bot, target, "添加黑名单\n第1步：请输入用户名，用户ID，或转发成员消息到这里\n第2步：再输入加入原因（可选，可发送“跳过”）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "remove":
			h.answerCallback(bot, cb.ID, "请输入目标用户")
			h.setPending(userID, pendingInput{Kind: "black_remove", TGGroupID: tgGroupID})
			h.render(bot, target, "移除黑名单\n请输入用户名，用户ID，或转发成员消息到这里", keyboards.PendingCancelKeyboard(tgGroupID))
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
			h.render(bot, target, "1. 配置过期时间\n👉 请回复链接过期时间(不限制请输入:0)\n\n注意:此设置仅应用在新生成的链接中，不会修改已生成的链接\n\n格式:年-月-日 时:分\n例如:2026-02-24 17:09", keyboards.InviteExpireInputKeyboard(tgGroupID))
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
			h.render(bot, target, "2. 最大邀请数配置\n\n👉 邀请达到设定人数后链接失效\n\n注意:此设置仅应用在新生成的链接中，不会修改已生成的链接\n\n请回复单个链接最大邀请人数(不限制请输入:0)", keyboards.InviteMemberInputKeyboard(tgGroupID))
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
			h.render(bot, target, "3. 生成数量限制配置\n\n👉 生成链接数量达到设定数量后，不再生成新的链接\n\n注意:此设置仅应用在新生成的链接中，不会修改已生成的链接\n\n请回复生成链接数量上限(不限制请输入:0)", keyboards.InviteGenerateInputKeyboard(tgGroupID))
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
			_, _ = sendDocumentBytes(bot, target.ChatID, name, content, "邀请数据 CSV 导出")
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
			h.render(bot, target, "第1步：请选择接龙限制方式", keyboards.ChainLimitModeKeyboard(tgGroupID))
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
				h.render(bot, target, "第2步：请输入接龙规则或介绍", keyboards.PendingCancelKeyboard(tgGroupID))
			case "people":
				h.answerCallback(bot, cb.ID, "请输入限制人数")
				h.setPending(userID, pendingInput{Kind: "chain_create_count", TGGroupID: tgGroupID, ChainMode: mode})
				h.render(bot, target, "第2步：请输入接龙人数上限（正整数）", keyboards.PendingCancelKeyboard(tgGroupID))
			case "time":
				h.answerCallback(bot, cb.ID, "请选择截止时间")
				h.setPending(userID, pendingInput{Kind: "chain_create_duration", TGGroupID: tgGroupID, ChainMode: mode})
				h.render(bot, target, "第2步：请选择多久后截止", keyboards.ChainDurationKeyboard(tgGroupID))
			case "both":
				h.answerCallback(bot, cb.ID, "请输入限制人数")
				h.setPending(userID, pendingInput{Kind: "chain_create_count", TGGroupID: tgGroupID, ChainMode: mode})
				h.render(bot, target, "第2步：请输入接龙人数上限（正整数）", keyboards.PendingCancelKeyboard(tgGroupID))
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
				h.render(bot, target, "第3步：请输入接龙规则或介绍\n截止时间："+chainDeadlineText(deadline), keyboards.PendingCancelKeyboard(tgGroupID))
			} else {
				h.answerCallback(bot, cb.ID, "已设置为无截止时间")
				h.render(bot, target, "第3步：请输入接龙规则或介绍\n截止时间：不限时", keyboards.PendingCancelKeyboard(tgGroupID))
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
			_, _ = sendDocumentBytes(bot, target.ChatID, name, content, "接龙数据 CSV 导出")
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
			h.answerCallback(bot, cb.ID, "开始创建投票")
			h.setPending(userID, pendingInput{Kind: "poll_create_question", TGGroupID: tgGroupID})
			h.render(bot, target, "第1步：请输入投票问题（1-300字）\n示例：今天开会吗？\n\n兼容写法：也可直接发送\n问题|选项1,选项2,选项3", keyboards.PendingCancelKeyboard(tgGroupID))
		case "submit":
			pending, ok := h.getPending(userID)
			if !ok || pending.TGGroupID != tgGroupID || pending.Kind != "poll_create_option" {
				h.answerCallback(bot, cb.ID, "请先开始创建投票")
				return
			}
			if strings.TrimSpace(pending.PollQuestion) == "" {
				h.answerCallback(bot, cb.ID, "缺少投票问题")
				return
			}
			if err := validatePollOptions(pending.PollOptions); err != nil {
				h.answerCallback(bot, cb.ID, err.Error())
				return
			}
			if _, err := h.service.CreatePollByTGGroupID(bot, tgGroupID, pending.PollQuestion, pending.PollOptions); err != nil {
				h.answerCallback(bot, cb.ID, "创建投票失败")
				return
			}
			h.clearPending(userID)
			h.answerCallback(bot, cb.ID, "投票已创建")
			h.sendPollPanel(bot, target, userID, tgGroupID)
		case "reset":
			pending, ok := h.getPending(userID)
			if !ok || pending.TGGroupID != tgGroupID || pending.Kind != "poll_create_option" {
				h.answerCallback(bot, cb.ID, "暂无可清空的投票草稿")
				return
			}
			pending.PollOptions = nil
			h.setPending(userID, pending)
			h.answerCallback(bot, cb.ID, "选项已清空")
			h.render(bot, target, pollDraftText(pending.PollQuestion, pending.PollOptions), keyboards.PollCreateDraftKeyboard(tgGroupID))
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
			h.render(bot, target, "请输入要监控的关键词（单条）", keyboards.PendingCancelKeyboard(tgGroupID))
		case "remove":
			h.answerCallback(bot, cb.ID, "请输入关键词")
			h.setPending(userID, pendingInput{Kind: "monitor_remove", TGGroupID: tgGroupID})
			h.render(bot, target, "请输入要移除的关键词（单条）", keyboards.PendingCancelKeyboard(tgGroupID))
		default:
			h.answerCallback(bot, cb.ID, "未知操作")
		}
	case "wc":
		if !h.service.WordCloudAvailable() {
			h.answerCallback(bot, cb.ID, "词云分词器未就绪，请检查 WORDCLOUD_JIEBA_DICT_DIR")
			h.sendGroupPanel(bot, target, userID, tgGroupID)
			return
		}
		h.handleWordCloudFeature(bot, cb, target, userID, tgGroupID, action, parts)
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
