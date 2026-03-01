package handler

import (
	"fmt"
	"strings"
	"supervisor/internal/handler/keyboards"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) sendGroupsMenu(bot *tgbotapi.BotAPI, target renderTarget, tgUserID int64, page int) {
	groups, err := h.service.ListManageableGroups(tgUserID)
	if err != nil {
		h.render(bot, target, "获取群列表失败", keyboards.MainMenuKeyboard(bot.Self.UserName))
		return
	}
	if len(groups) == 0 {
		h.render(bot, target, "你当前没有可管理且机器人已加入的群", keyboards.MainMenuKeyboard(bot.Self.UserName))
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
	h.render(bot, target, text, keyboards.GroupsKeyboard(current, page, totalPages))
}

func (h *Handler) sendGroupPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	summary, err := h.service.GroupPanelSummary(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载群面板失败", keyboards.MainMenuKeyboard(bot.Self.UserName))
		return
	}
	h.render(bot, target, summary, keyboards.GroupPanelKeyboardWithWordCloud(tgGroupID, h.service.WordCloudAvailable()))
}

func (h *Handler) sendAutoReplyList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListAutoRepliesByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载自动回复失败", keyboards.GroupPanelKeyboard(tgGroupID))
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
		btnCount := buttonRowsCount(item.ButtonRows)
		lines = append(lines, fmt.Sprintf("#%d [%s] %s => %s（链接按钮:%d）", item.ID, autoReplyMatchTypeLabel(item.MatchType), item.Keyword, item.Reply, btnCount))
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AutoReplyListKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendBannedWordList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.BannedWordViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载违禁词设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	data, err := h.service.ListBannedWordsByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载违禁词失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	totalPages := maxPages(data.Total, rulesPageSize)
	if data.Page < 1 {
		data.Page = 1
	}
	if data.Page > totalPages {
		data.Page = totalPages
	}
	lines := []string{
		fmt.Sprintf("违禁词列表（第 %d/%d 页，总 %d 条）", data.Page, totalPages, data.Total),
		fmt.Sprintf("状态:%s", onOffWithEmoji(view.Enabled)),
		fmt.Sprintf("惩罚:%s", bannedWordPenaltyText(view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes)),
		fmt.Sprintf("删除提醒:%s", bannedWordDeleteText(view.WarnDeleteMinutes)),
	}
	if len(data.Items) == 0 {
		lines = append(lines, "暂无词条")
	}
	for _, item := range data.Items {
		lines = append(lines, fmt.Sprintf("#%d %s", item.ID, item.Word))
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.BannedWordListKeyboard(tgGroupID, view, data.Items, data.Page, totalPages))
}

func (h *Handler) sendBannedWordPenaltyPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.BannedWordViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载违禁词设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"🚫 违禁词 - 惩罚设置",
		"",
		fmt.Sprintf("当前惩罚:%s", bannedWordPenaltyText(view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes)),
		"",
		"说明:",
		"1) 警告：可设置警告次数，达到后执行禁言/踢出/踢出+封禁",
		"2) 禁言：可设置禁言时长",
		"3) 踢出：直接踢出",
		"4) 踢出+封禁：可设置封禁时长",
		"5) 仅撤回消息+不惩罚",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.BannedWordPenaltyKeyboard(tgGroupID, view))
}

func (h *Handler) sendScheduledList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListScheduledMessagesByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载定时消息失败", keyboards.GroupPanelKeyboard(tgGroupID))
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
		btnCount := buttonRowsCount(item.ButtonRows)
		pin := "不置顶"
		if item.PinMessage {
			pin = "置顶"
		}
		lines = append(lines, fmt.Sprintf("#%d [%s] %s => %s（类型:%s，链接按钮:%d，%s）", item.ID, status, item.CronExpr, scheduledContentPreview(item.Content, 24), scheduledMediaTypeLabel(item.MediaType), btnCount, pin))
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.ScheduledListKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendScheduledEditPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, id uint, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	item, err := h.service.GetScheduledMessageByTGGroupID(tgGroupID, id)
	if err != nil || item == nil {
		h.render(bot, target, "加载定时任务失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	status := "关闭"
	if item.Enabled {
		status = "启用"
	}
	pin := "否"
	if item.PinMessage {
		pin = "是"
	}
	lines := []string{
		fmt.Sprintf("定时任务编辑 #%d", item.ID),
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("Cron:%s", item.CronExpr),
		fmt.Sprintf("文本:%s", scheduledContentPreview(item.Content, 60)),
		fmt.Sprintf("媒体:%s", scheduledMediaTypeLabel(item.MediaType)),
		fmt.Sprintf("链接按钮:%d", buttonRowsCount(item.ButtonRows)),
		fmt.Sprintf("发送后置顶:%s", pin),
	}
	if page < 1 {
		page = 1
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.ScheduledEditKeyboard(tgGroupID, item.ID, page, item.Enabled, item.PinMessage))
}
