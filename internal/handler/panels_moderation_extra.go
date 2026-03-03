package handler

import (
	"fmt"
	"strings"
	"supervisor/internal/handler/keyboards"

	tgbot "github.com/go-telegram/bot"
)

func (h *Handler) sendVerifyPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.JoinVerifyViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载验证设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	status := "关闭❌"
	if view.Enabled {
		status = "启用✅"
	}
	lines := []string{
		"🤖 验证",
		"启用后，用户进入群组需要验证才能发送消息",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("验证时间:%d分钟", view.TimeoutMinutes),
		fmt.Sprintf("验证超时:%s", verifyTimeoutActionLabel(view.TimeoutAction)),
		fmt.Sprintf("验证方式:%s", verifyTypeLabel(view.Type)),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.VerifyKeyboard(tgGroupID, view))
}

func (h *Handler) sendVerifyTimeoutMinutesPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.JoinVerifyViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载验证设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"🤖 验证 - 验证时间",
		"",
		fmt.Sprintf("当前设置:%d分钟", view.TimeoutMinutes),
		"请选择验证时间：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.VerifyTimeoutMinutesKeyboard(tgGroupID, view.TimeoutMinutes))
}

func (h *Handler) sendNewbieLimitPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.NewbieLimitViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载新成员限制失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	status := "关闭❌"
	if view.Enabled {
		status = "启用✅"
	}
	lines := []string{
		"🔒 新成员限制",
		"启用后，新成员在限制时长内不能发送任何消息",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("限制时长:%d分钟", view.Minutes),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.NewbieLimitKeyboard(tgGroupID, view))
}

func (h *Handler) sendNewbieLimitMinutesPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.NewbieLimitViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载新成员限制失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"🔒 新成员限制 - 限制时长",
		"",
		fmt.Sprintf("当前设置:%d分钟", view.Minutes),
		"请选择限制时长：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.NewbieLimitMinutesKeyboard(tgGroupID, view.Minutes))
}

func (h *Handler) sendNightModePanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.NightModeViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载夜间模式失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	status := "关闭❌"
	if view.Enabled {
		status = "启用✅"
	}
	lines := []string{
		"🌙 夜间模式",
		"夜间时段内按配置自动处理群成员消息",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("时区:%s", view.TimezoneText),
		fmt.Sprintf("夜间时段:%s", view.NightWindow),
		fmt.Sprintf("处理方式:%s", nightModeActionLabel(view.Mode)),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.NightModeKeyboard(tgGroupID, view))
}

func (h *Handler) sendChainPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.ChainViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载接龙失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	activeItems, err := h.service.ListActiveChainSummariesByTGGroupID(tgGroupID, 8)
	if err != nil {
		h.render(bot, target, "加载接龙失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{"接龙管理"}
	lines = append(lines, fmt.Sprintf("进行中接龙：%d", len(activeItems)))
	if len(activeItems) == 0 {
		lines = append(lines, "状态：当前无进行中接龙")
	} else {
		lines = append(lines, "进行中列表：")
		for _, item := range activeItems {
			lines = append(lines, fmt.Sprintf("#%d %s", item.ID, item.Intro))
			lines = append(lines, "人数限制："+chainLimitText(item.MaxParticipants))
			lines = append(lines, "截止时间："+chainDeadlineText(item.DeadlineUnix))
			lines = append(lines, fmt.Sprintf("已参与：%d", item.Participants))
			lines = append(lines, "")
		}
	}
	if view.ID > 0 {
		lines = append(lines, "最近一次接龙：")
		lines = append(lines, fmt.Sprintf("#%d %s", view.ID, view.Intro))
		lines = append(lines, "状态："+onOffWithEmoji(view.Active))
	}
	lines = append(lines, "")
	lines = append(lines, "创建后机器人会自动发送群公告，成员点击按钮进入私聊提交内容")
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.ChainKeyboard(tgGroupID, activeItems))
}
