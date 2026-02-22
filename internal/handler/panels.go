package handler

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) sendGroupsMenu(bot *tgbotapi.BotAPI, target renderTarget, tgUserID int64, page int) {
	groups, err := h.service.ListManageableGroups(tgUserID)
	if err != nil {
		h.render(bot, target, "获取群列表失败", mainMenuKeyboard())
		return
	}
	if len(groups) == 0 {
		h.render(bot, target, "你当前没有可管理且机器人已加入的群", mainMenuKeyboard())
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
	h.render(bot, target, text, groupsKeyboard(current, page, totalPages))
}

func (h *Handler) sendGroupPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	summary, err := h.service.GroupPanelSummary(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载群面板失败", mainMenuKeyboard())
		return
	}
	h.render(bot, target, summary, groupPanelKeyboard(tgGroupID))
}

func (h *Handler) sendAutoReplyList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListAutoRepliesByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载自动回复失败", groupPanelKeyboard(tgGroupID))
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
		lines = append(lines, fmt.Sprintf("#%d [%s] %s => %s", item.ID, item.MatchType, item.Keyword, item.Reply))
	}
	h.render(bot, target, strings.Join(lines, "\n"), autoReplyListKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendBannedWordList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListBannedWordsByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载违禁词失败", groupPanelKeyboard(tgGroupID))
		return
	}
	totalPages := maxPages(data.Total, rulesPageSize)
	if data.Page < 1 {
		data.Page = 1
	}
	if data.Page > totalPages {
		data.Page = totalPages
	}
	lines := []string{fmt.Sprintf("违禁词列表（第 %d/%d 页，总 %d 条）", data.Page, totalPages, data.Total)}
	if len(data.Items) == 0 {
		lines = append(lines, "暂无词条")
	}
	for _, item := range data.Items {
		lines = append(lines, fmt.Sprintf("#%d %s", item.ID, item.Word))
	}
	h.render(bot, target, strings.Join(lines, "\n"), bannedWordListKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendScheduledList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListScheduledMessagesByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载定时消息失败", groupPanelKeyboard(tgGroupID))
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
		lines = append(lines, fmt.Sprintf("#%d [%s] %s => %s", item.ID, status, item.CronExpr, item.Content))
	}
	h.render(bot, target, strings.Join(lines, "\n"), scheduledListKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendStatsPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	stats, err := h.service.GroupStatsByTGGroupID(tgGroupID, 10)
	if err != nil {
		h.render(bot, target, "加载统计失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{fmt.Sprintf("群统计：%s (%d)", stats.GroupTitle, stats.GroupID)}
	if len(stats.TopUsers) == 0 {
		lines = append(lines, "暂无活跃数据")
	} else {
		lines = append(lines, "活跃榜（按消息积分）:")
		for i, u := range stats.TopUsers {
			lines = append(lines, fmt.Sprintf("%d. %s - %d", i+1, u.DisplayName, u.Points))
		}
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新统计", fmt.Sprintf("feat:stats:show:%d", tgGroupID)),
			tgbotapi.NewInlineKeyboardButtonData("返回群面板", cbGroupPrefix+strconv.FormatInt(tgGroupID, 10)),
		),
	)
	h.render(bot, target, strings.Join(lines, "\n"), markup)
}

func (h *Handler) sendLogPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int, filter string) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListLogsByTGGroupID(tgGroupID, page, rulesPageSize, filter)
	if err != nil {
		h.render(bot, target, "加载管理日志失败", groupPanelKeyboard(tgGroupID))
		return
	}
	totalPages := maxPages(data.Total, rulesPageSize)
	if data.Page < 1 {
		data.Page = 1
	}
	if data.Page > totalPages {
		data.Page = totalPages
	}
	lines := []string{fmt.Sprintf("管理日志（第 %d/%d 页，总 %d 条）", data.Page, totalPages, data.Total)}
	if len(data.Items) == 0 {
		lines = append(lines, "暂无日志")
	}
	for _, item := range data.Items {
		lines = append(lines, fmt.Sprintf("#%d %s @ %s", item.ID, item.Action, item.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	h.render(bot, target, strings.Join(lines, "\n"), logListKeyboard(tgGroupID, data.Page, totalPages, filter))
}

func (h *Handler) sendSystemCleanPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	cfg, err := h.service.SystemCleanViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载系统消息清理失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"帮助您自动清理群组中的系统消息",
		"预设: 严格 / 推荐 / 关闭",
		"",
		fmt.Sprintf("进群: %s", onOffWithEmoji(cfg.Join)),
		fmt.Sprintf("退群: %s", onOffWithEmoji(cfg.Leave)),
		fmt.Sprintf("置顶: %s", onOffWithEmoji(cfg.Pin)),
		fmt.Sprintf("修改头像: %s", onOffWithEmoji(cfg.Photo)),
		fmt.Sprintf("修改名称: %s", onOffWithEmoji(cfg.Title)),
	}
	h.render(bot, target, strings.Join(lines, "\n"), systemCleanKeyboard(tgGroupID, cfg))
}

func (h *Handler) sendChainPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.ChainViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载接龙失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{"接龙管理"}
	if !view.Active {
		lines = append(lines, "状态：未开始")
	} else {
		lines = append(lines, "状态：进行中")
		lines = append(lines, "标题："+view.Title)
		if len(view.Entries) == 0 {
			lines = append(lines, "暂无条目")
		} else {
			lines = append(lines, "条目：")
			for i, e := range view.Entries {
				lines = append(lines, fmt.Sprintf("%d. %s", i+1, e))
			}
		}
	}
	h.render(bot, target, strings.Join(lines, "\n"), chainKeyboard(tgGroupID, view.Active))
}

func (h *Handler) sendMonitorPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	items, err := h.service.ListMonitorKeywordsByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载关键词监控失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{"关键词监控"}
	if len(items) == 0 {
		lines = append(lines, "暂无关键词")
	} else {
		lines = append(lines, "当前关键词：")
		for i, k := range items {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, k))
		}
		lines = append(lines, "", "命中后将私聊通知群管理员")
	}
	h.render(bot, target, strings.Join(lines, "\n"), monitorKeyboard(tgGroupID))
}

func (h *Handler) sendPollPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	text := strings.Join([]string{
		"投票管理",
		"创建格式：问题|选项1,选项2,...",
		"示例：今天开会吗？|开,不开,待定",
	}, "\n")
	h.render(bot, target, text, pollKeyboard(tgGroupID))
}

func (h *Handler) sendRBACPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	text, err := h.service.RBACSummaryByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载权限分级失败", groupPanelKeyboard(tgGroupID))
		return
	}
	h.render(bot, target, text, rbacKeyboard(tgGroupID))
}

func (h *Handler) sendBlacklistPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	items, err := h.service.ListGlobalBlacklist()
	if err != nil {
		h.render(bot, target, "加载黑名单失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{"全局黑名单（跨群生效）"}
	if len(items) == 0 {
		lines = append(lines, "暂无黑名单用户")
	} else {
		for i, it := range items {
			lines = append(lines, fmt.Sprintf("%d. %d (%s)", i+1, it.TGUserID, it.Reason))
		}
	}
	h.render(bot, target, strings.Join(lines, "\n"), blacklistKeyboard(tgGroupID))
}

func (h *Handler) sendSettingsPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID int64) {
	lang, _ := h.service.GetUserLanguage(tgUserID)
	text := "设置\n当前语言: " + lang + "\n可切换为中文/英文（逐步覆盖）"
	h.render(bot, target, text, settingsKeyboard())
}
