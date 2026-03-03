package handler

import (
	"errors"
	"fmt"
	"strconv"

	"supervisor/internal/handler/keyboards"
	svc "supervisor/internal/service"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) handleModerationFeature(bot *tgbot.Bot, cb *models.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	ensureAntiSpamAIAvailable := func() bool {
		view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "加载失败")
			return false
		}
		if view.AIAvailable {
			return true
		}
		h.answerCallback(bot, cb.ID, "未配置 ANTI_SPAM_AI_MODEL，AI 功能不可用")
		h.sendAntiSpamPanel(bot, target, userID, tgGroupID)
		return false
	}

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
	case "spampenaltycfg":
		h.answerCallback(bot, cb.ID, "加载惩罚设置")
		h.sendAntiSpamPenaltyPanel(bot, target, userID, tgGroupID)
		return
	case "spampenalty":
		h.handleModerationPenaltySetCallback(
			bot, cb, target, userID, tgGroupID, parts,
			h.service.SetAntiSpamPenaltyByTGGroupID,
			h.antiSpamPenaltySummaryByTGGroupID,
			h.sendAntiSpamPenaltyPanel,
		)
		return
	case "spamwarncount":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "spam_warn_threshold", "请输入警告次数", "请输入达到处罚前的警告次数（正整数，例如 3）")
		return
	case "spamwarnaction":
		h.handleModerationWarnActionSetCallback(bot, cb, target, userID, tgGroupID, parts, h.service.SetAntiSpamWarnActionByTGGroupID, h.sendAntiSpamPenaltyPanel)
		return
	case "spamwarnmuteinput":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "spam_warn_action_mute_minutes", "请输入阈值禁言时长", "请输入警告达到阈值后禁言时长（分钟，1-10080）")
		return
	case "spamwarnbaninput":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "spam_warn_action_ban_minutes", "请输入阈值封禁时长", "请输入警告达到阈值后封禁时长（分钟，1-10080）")
		return
	case "spammuteinput":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "spam_mute_minutes", "请输入禁言时长", "请输入禁言时长（分钟，1-10080）")
		return
	case "spambaninput":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "spam_ban_minutes", "请输入封禁时长", "请输入封禁时长（分钟，1-10080）")
		return
	case "spamaicfg":
		if !ensureAntiSpamAIAvailable() {
			return
		}
		h.answerCallback(bot, cb.ID, "加载AI反垃圾")
		h.sendAntiSpamAIPanel(bot, target, userID, tgGroupID)
		return
	case "spamaion":
		if !ensureAntiSpamAIAvailable() {
			return
		}
		if _, err := h.service.SetAntiSpamAIEnabledByTGGroupID(tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "AI反垃圾已开启")
		h.sendAntiSpamAIPanel(bot, target, userID, tgGroupID)
		return
	case "spamaioff":
		if _, err := h.service.SetAntiSpamAIEnabledByTGGroupID(tgGroupID, false); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "AI反垃圾已关闭")
		h.sendAntiSpamAIPanel(bot, target, userID, tgGroupID)
		return
	case "spamaiscore":
		if !ensureAntiSpamAIAvailable() {
			return
		}
		view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "加载失败")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入 AI 垃圾分阈值")
		h.setPending(userID, pendingInput{Kind: "spam_ai_spam_score", TGGroupID: tgGroupID})
		h.render(bot, target, fmt.Sprintf("当 AI 返回 score >= 该值时，判定为垃圾。\n当前阈值:%d\n👉 输入 1~100 的整数：", view.AISpamScore), keyboards.PendingCancelKeyboard(tgGroupID))
		return
	case "spamaistrict":
		if !ensureAntiSpamAIAvailable() {
			return
		}
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		strictness, err := h.service.SetAntiSpamAIStrictnessByTGGroupID(tgGroupID, parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "AI严格度已设为 "+antiSpamAIStrictnessText(strictness))
		h.sendAntiSpamAIPanel(bot, target, userID, tgGroupID)
		return
	case "spammsglen":
		view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "加载失败")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入消息最大长度")
		h.setPending(userID, pendingInput{Kind: "spam_msg_len", TGGroupID: tgGroupID})
		h.render(bot, target, fmt.Sprintf("检测到消息内容长度大于设定数时，将会判定为超长消息并处罚。\n当前设置最大长度:%d\n👉 输入允许的消息最大长度（例如:100）:", view.MaxMessageLength), keyboards.PendingCancelKeyboard(tgGroupID))
		return
	case "spamnamelen":
		view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "加载失败")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入姓名最大长度")
		h.setPending(userID, pendingInput{Kind: "spam_name_len", TGGroupID: tgGroupID})
		h.render(bot, target, fmt.Sprintf("检测到姓名长度大于设定数时，将会判定为超长姓名并处罚。\n当前设置最大长度:%d\n👉 输入允许的姓名最大长度（例如:32）:", view.MaxNameLength), keyboards.PendingCancelKeyboard(tgGroupID))
		return
	case "spamexadd":
		h.answerCallback(bot, cb.ID, "请输入例外关键词")
		h.setPending(userID, pendingInput{Kind: "spam_exception_add", TGGroupID: tgGroupID})
		h.render(bot, target, "输入一个例外关键词（命中后跳过反垃圾检测）", keyboards.PendingCancelKeyboard(tgGroupID))
		return
	case "spamexdel":
		h.answerCallback(bot, cb.ID, "请输入要移除的关键词")
		h.setPending(userID, pendingInput{Kind: "spam_exception_remove", TGGroupID: tgGroupID})
		h.render(bot, target, "输入要移除的例外关键词（精确匹配，不区分大小写）", keyboards.PendingCancelKeyboard(tgGroupID))
		return
	case "spamalertdel":
		h.answerCallback(bot, cb.ID, "请选择提醒策略")
		h.sendAntiSpamAlertDeletePanel(bot, target, userID, tgGroupID)
		return
	case "spamalertdelset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "请选择具体选项")
			h.sendAntiSpamAlertDeletePanel(bot, target, userID, tgGroupID)
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
		h.answerCallback(bot, cb.ID, "删除提醒已设置为 "+antiSpamAlertSettingText(sec))
		h.sendAntiSpamAlertDeletePanel(bot, target, userID, tgGroupID)
		return
	case "spamunlock":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		targetUserID, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		if err := h.service.ReleaseAntiSpamPenaltyByTGGroupID(bot, tgGroupID, targetUserID); err != nil {
			h.answerCallback(bot, cb.ID, "解禁失败")
			return
		}
		h.answerCallback(bot, cb.ID, "已解禁")
		return
	case "spamwarnrevoke":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		targetUserID, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		if err := h.service.RevokeAntiSpamWarnByTGGroupID(tgGroupID, userID, targetUserID); err != nil {
			if errors.Is(err, svc.ErrNoModerationWarnToRevoke) {
				h.answerCallback(bot, cb.ID, "暂无可撤销的警告")
				return
			}
			h.answerCallback(bot, cb.ID, "撤销失败")
			return
		}
		h.answerCallback(bot, cb.ID, "警告已撤销")
		return
	case "floodwarnrevoke":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		targetUserID, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		if err := h.service.RevokeAntiFloodWarnByTGGroupID(tgGroupID, userID, targetUserID); err != nil {
			if errors.Is(err, svc.ErrNoModerationWarnToRevoke) {
				h.answerCallback(bot, cb.ID, "暂无可撤销的警告")
				return
			}
			h.answerCallback(bot, cb.ID, "撤销失败")
			return
		}
		h.answerCallback(bot, cb.ID, "警告已撤销")
		return
	case "bwwarnrevoke":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		targetUserID, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		if err := h.service.RevokeBannedWordWarnByTGGroupID(tgGroupID, userID, targetUserID); err != nil {
			if errors.Is(err, svc.ErrNoModerationWarnToRevoke) {
				h.answerCallback(bot, cb.ID, "暂无可撤销的警告")
				return
			}
			h.answerCallback(bot, cb.ID, "撤销失败")
			return
		}
		h.answerCallback(bot, cb.ID, "警告已撤销")
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
		h.render(bot, target, fmt.Sprintf("当前时区:%s\n请输入时区（示例：+8、-5、+8:30、UTC+8）", view.TimezoneText), keyboards.PendingCancelKeyboard(tgGroupID))
		return
	case "nightwindow":
		view, err := h.service.NightModeViewByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "加载失败")
			return
		}
		h.answerCallback(bot, cb.ID, "请先输入开始小时")
		h.setPending(userID, pendingInput{Kind: "night_start_hour", TGGroupID: tgGroupID})
		h.render(bot, target, fmt.Sprintf("当前夜间时段:%s\n请输入开始小时（0-23）\n例如:22", view.NightWindow), keyboards.PendingCancelKeyboard(tgGroupID))
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
		h.handleModerationPenaltySetCallback(
			bot, cb, target, userID, tgGroupID, parts,
			h.service.SetAntiFloodPenaltyByTGGroupID,
			h.antiFloodPenaltySummaryByTGGroupID,
			h.sendAntiFloodPenaltyPanel,
		)
		return
	case "floodpenaltycfg":
		h.answerCallback(bot, cb.ID, "加载惩罚设置")
		h.sendAntiFloodPenaltyPanel(bot, target, userID, tgGroupID)
		return
	case "floodwarncount":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "flood_warn_threshold", "请输入警告次数", "请输入达到处罚前的警告次数（正整数，例如 3）")
		return
	case "floodwarnaction":
		h.handleModerationWarnActionSetCallback(bot, cb, target, userID, tgGroupID, parts, h.service.SetAntiFloodWarnActionByTGGroupID, h.sendAntiFloodPenaltyPanel)
		return
	case "floodwarnmuteinput":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "flood_warn_action_mute_minutes", "请输入阈值禁言时长", "请输入警告达到阈值后禁言时长（分钟，1-10080）")
		return
	case "floodwarnbaninput":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "flood_warn_action_ban_minutes", "请输入阈值封禁时长", "请输入警告达到阈值后封禁时长（分钟，1-10080）")
		return
	case "floodmuteinput":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "flood_mute_minutes", "请输入禁言时长", "请输入禁言时长（分钟，1-10080）")
		return
	case "floodbaninput":
		h.beginModerationPendingInput(bot, cb, target, userID, tgGroupID, "flood_ban_minutes", "请输入封禁时长", "请输入封禁时长（分钟，1-10080）")
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

func (h *Handler) antiSpamPenaltySummaryByTGGroupID(tgGroupID int64) (string, error) {
	view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
	if err != nil {
		return "", err
	}
	return antiFloodPenaltyText(view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes), nil
}

func (h *Handler) antiFloodPenaltySummaryByTGGroupID(tgGroupID int64) (string, error) {
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		return "", err
	}
	return antiFloodPenaltyText(view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes), nil
}

func (h *Handler) handleModerationPenaltySetCallback(
	bot *tgbot.Bot,
	cb *models.CallbackQuery,
	target renderTarget,
	userID int64,
	tgGroupID int64,
	parts []string,
	setFn func(int64, string) (string, error),
	summaryFn func(int64) (string, error),
	sendPanelFn func(*tgbot.Bot, renderTarget, int64, int64),
) {
	if len(parts) < 5 {
		h.answerCallback(bot, cb.ID, "参数错误")
		return
	}
	if _, err := setFn(tgGroupID, parts[4]); err != nil {
		h.answerCallback(bot, cb.ID, "设置失败")
		return
	}
	summary, err := summaryFn(tgGroupID)
	if err != nil {
		h.answerCallback(bot, cb.ID, "设置失败")
		return
	}
	h.answerCallback(bot, cb.ID, "惩罚已设为 "+summary)
	sendPanelFn(bot, target, userID, tgGroupID)
}

func (h *Handler) handleModerationWarnActionSetCallback(
	bot *tgbot.Bot,
	cb *models.CallbackQuery,
	target renderTarget,
	userID int64,
	tgGroupID int64,
	parts []string,
	setFn func(int64, string) (string, error),
	sendPanelFn func(*tgbot.Bot, renderTarget, int64, int64),
) {
	if len(parts) < 5 {
		h.answerCallback(bot, cb.ID, "参数错误")
		return
	}
	if _, err := setFn(tgGroupID, parts[4]); err != nil {
		h.answerCallback(bot, cb.ID, "设置失败")
		return
	}
	h.answerCallback(bot, cb.ID, "阈值后动作已更新")
	sendPanelFn(bot, target, userID, tgGroupID)
}

func (h *Handler) beginModerationPendingInput(
	bot *tgbot.Bot,
	cb *models.CallbackQuery,
	target renderTarget,
	userID int64,
	tgGroupID int64,
	kind string,
	tip string,
	prompt string,
) {
	h.answerCallback(bot, cb.ID, tip)
	h.setPending(userID, pendingInput{Kind: kind, TGGroupID: tgGroupID})
	h.render(bot, target, prompt, keyboards.PendingCancelKeyboard(tgGroupID))
}
