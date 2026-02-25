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
		h.render(bot, target, "获取群列表失败", mainMenuKeyboard(bot.Self.UserName))
		return
	}
	if len(groups) == 0 {
		h.render(bot, target, "你当前没有可管理且机器人已加入的群", mainMenuKeyboard(bot.Self.UserName))
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
		h.render(bot, target, "加载群面板失败", mainMenuKeyboard(bot.Self.UserName))
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
		btnCount := buttonRowsCount(item.ButtonRows)
		lines = append(lines, fmt.Sprintf("#%d [%s] %s => %s（链接按钮:%d）", item.ID, autoReplyMatchTypeLabel(item.MatchType), item.Keyword, item.Reply, btnCount))
	}
	h.render(bot, target, strings.Join(lines, "\n"), autoReplyListKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendBannedWordList(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.BannedWordViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载违禁词设置失败", groupPanelKeyboard(tgGroupID))
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
	h.render(bot, target, strings.Join(lines, "\n"), bannedWordListKeyboard(tgGroupID, view, data.Items, data.Page, totalPages))
}

func (h *Handler) sendBannedWordPenaltyPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.BannedWordViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载违禁词设置失败", groupPanelKeyboard(tgGroupID))
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
	h.render(bot, target, strings.Join(lines, "\n"), bannedWordPenaltyKeyboard(tgGroupID, view))
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
		btnCount := buttonRowsCount(item.ButtonRows)
		lines = append(lines, fmt.Sprintf("#%d [%s] %s => %s（链接按钮:%d）", item.ID, status, item.CronExpr, item.Content, btnCount))
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

func (h *Handler) sendInvitePanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.InvitePanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载邀请链接设置失败", groupPanelKeyboard(tgGroupID))
		return
	}
	status := "❌ 关闭"
	if view.Enabled {
		status = "✅ 开启"
	}
	lines := []string{
		"邀请链接生成",
		"开启后群组中成员使用 /link 指令自动生成链接/查询邀请统计",
		"",
		"防作弊:",
		"└ 只有第一次进群视为有效邀请数，退群再用其他人的链接加群不计算邀请数",
		"",
		fmt.Sprintf("┌状态:%s", status),
		fmt.Sprintf("├总邀请人数:%d", view.TotalInvited),
		fmt.Sprintf("├链接过期时间:%s", inviteExpireText(view.ExpireDate)),
		fmt.Sprintf("├最大邀请人数:%s", inviteLimitText(view.MemberLimit)),
		fmt.Sprintf("└生成数量上限:%s     已生成数量:%d", inviteLimitText(view.GenerateLimit), view.GeneratedCount),
	}
	h.render(bot, target, strings.Join(lines, "\n"), inviteKeyboard(tgGroupID, view.Enabled))
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

func (h *Handler) sendAntiFloodPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反刷屏设置失败", groupPanelKeyboard(tgGroupID))
		return
	}
	status := "❌ 关闭"
	if view.Enabled {
		status = "✅ 启用"
	}
	lines := []string{
		"💬 反刷屏",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("当前设置:在%d秒内发送%d条消息触发反刷屏", view.WindowSec, view.MaxMessages),
		fmt.Sprintf("惩罚:%s", antiFloodPenaltyText(view.Penalty, view.MuteSec)),
		fmt.Sprintf("删除提醒:%s", antiFloodAlertDeleteText(view.WarnDeleteSec)),
	}
	h.render(bot, target, strings.Join(lines, "\n"), antiFloodKeyboard(tgGroupID, view))
}

func (h *Handler) sendAntiFloodAlertDeletePanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反刷屏设置失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"💬 反刷屏 - 删除提醒",
		"",
		fmt.Sprintf("当前设置:%s", antiFloodAlertDeleteText(view.WarnDeleteSec)),
		"请选择提醒消息自动删除时间：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), antiFloodAlertDeleteKeyboard(tgGroupID, view.WarnDeleteSec))
}

func (h *Handler) sendAntiFloodCountPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反刷屏设置失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"💬 反刷屏 - 触发条数",
		"",
		fmt.Sprintf("当前设置:%d 条", view.MaxMessages),
		"请选择触发条数：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), antiFloodCountKeyboard(tgGroupID, view.MaxMessages))
}

func (h *Handler) sendAntiFloodWindowPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反刷屏设置失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"💬 反刷屏 - 检测间隔",
		"",
		fmt.Sprintf("当前设置:%d 秒", view.WindowSec),
		"请选择检测间隔：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), antiFloodWindowKeyboard(tgGroupID, view.WindowSec))
}

func (h *Handler) sendAntiSpamPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反垃圾设置失败", groupPanelKeyboard(tgGroupID))
		return
	}
	status := "❌ 关闭"
	if view.Enabled {
		status = "✅ 启用"
	}
	keywords := "无"
	if len(view.ExceptionKeywords) > 0 {
		show := view.ExceptionKeywords
		if len(show) > 5 {
			show = show[:5]
		}
		keywords = strings.Join(show, "、")
		if len(view.ExceptionKeywords) > len(show) {
			keywords += " ..."
		}
	}
	lines := []string{
		"📨 反垃圾",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("惩罚:%s", antiFloodPenaltyText(view.Penalty, view.MuteSec)),
		"",
		fmt.Sprintf("1. 屏蔽图片: %s", onOffWithEmoji(view.BlockPhoto)),
		fmt.Sprintf("2. 屏蔽链接: %s", onOffWithEmoji(view.BlockLink)),
		fmt.Sprintf("3. 屏蔽频道马甲发言: %s", onOffWithEmoji(view.BlockChannelAlias)),
		fmt.Sprintf("4. 屏蔽来自频道转发: %s", onOffWithEmoji(view.BlockForwardFromChan)),
		fmt.Sprintf("5. 屏蔽来自用户转发: %s", onOffWithEmoji(view.BlockForwardFromUser)),
		fmt.Sprintf("6. 屏蔽@群组ID: %s", onOffWithEmoji(view.BlockAtGroupID)),
		fmt.Sprintf("7. 屏蔽@用户ID: %s", onOffWithEmoji(view.BlockAtUserID)),
		fmt.Sprintf("8. 屏蔽以太坊地址: %s", onOffWithEmoji(view.BlockEthAddress)),
		fmt.Sprintf("9. 屏蔽超长消息: %s", onOffWithEmoji(view.BlockLongMessage)),
		fmt.Sprintf("10. 当前设置最大消息长度: %d", view.MaxMessageLength),
		fmt.Sprintf("11. 屏蔽超长姓名: %s", onOffWithEmoji(view.BlockLongName)),
		fmt.Sprintf("12. 当前设置最大姓名长度: %d", view.MaxNameLength),
		fmt.Sprintf("13. 已添加例外: %d条", view.ExceptionKeywordCount),
		fmt.Sprintf("例外关键词:%s", keywords),
		fmt.Sprintf("14. 删除提醒: %s", antiFloodAlertDeleteText(view.WarnDeleteSec)),
	}
	h.render(bot, target, strings.Join(lines, "\n"), antiSpamKeyboard(tgGroupID, view))
}

func (h *Handler) sendVerifyPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.JoinVerifyViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载验证设置失败", groupPanelKeyboard(tgGroupID))
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
	h.render(bot, target, strings.Join(lines, "\n"), verifyKeyboard(tgGroupID, view))
}

func (h *Handler) sendVerifyTimeoutMinutesPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.JoinVerifyViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载验证设置失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"🤖 验证 - 验证时间",
		"",
		fmt.Sprintf("当前设置:%d分钟", view.TimeoutMinutes),
		"请选择验证时间：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), verifyTimeoutMinutesKeyboard(tgGroupID, view.TimeoutMinutes))
}

func (h *Handler) sendNewbieLimitPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.NewbieLimitViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载新成员限制失败", groupPanelKeyboard(tgGroupID))
		return
	}
	status := "关闭❌"
	if view.Enabled {
		status = "启用✅"
	}
	lines := []string{
		"🔒 新成员限制",
		"启用后，新成员在限制时长内不能发送链接或媒体",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("限制时长:%d分钟", view.Minutes),
	}
	h.render(bot, target, strings.Join(lines, "\n"), newbieLimitKeyboard(tgGroupID, view))
}

func (h *Handler) sendNewbieLimitMinutesPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.NewbieLimitViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载新成员限制失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"🔒 新成员限制 - 限制时长",
		"",
		fmt.Sprintf("当前设置:%d分钟", view.Minutes),
		"请选择限制时长：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), newbieLimitMinutesKeyboard(tgGroupID, view.Minutes))
}

func (h *Handler) sendNightModePanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.NightModeViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载夜间模式失败", groupPanelKeyboard(tgGroupID))
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
	h.render(bot, target, strings.Join(lines, "\n"), nightModeKeyboard(tgGroupID, view))
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
	activeItems, err := h.service.ListActiveChainSummariesByTGGroupID(tgGroupID, 8)
	if err != nil {
		h.render(bot, target, "加载接龙失败", groupPanelKeyboard(tgGroupID))
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
	h.render(bot, target, strings.Join(lines, "\n"), chainKeyboard(tgGroupID, activeItems))
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

func (h *Handler) sendLotteryPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.LotteryPanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载抽奖面板失败", groupPanelKeyboard(tgGroupID))
		return
	}

	lines := []string{
		"抽奖管理",
		"创建格式：抽奖标题|中奖人数|参与关键词",
		"示例：周末福利|3|参加",
		"成员在群内发送“参与关键词”即可参与",
		fmt.Sprintf("创建的抽奖次数:%d    已开奖:%d    未开奖:%d    取消:%d", view.CreatedTotal, view.DrawnTotal, view.PendingTotal, view.CanceledTotal),
		"",
		"⚙ 抽奖设置",
		fmt.Sprintf("%s 发布置顶:", boolIcon(view.PublishPin)),
		"└ 发布抽奖消息群内置顶",
		fmt.Sprintf("%s 结果置顶:", boolIcon(view.ResultPin)),
		"└ 中奖结果消息群内置顶",
		fmt.Sprintf("%s 删除口令:", boolIcon(view.DeleteKeywordMins > 0)),
		fmt.Sprintf("└ %s", lotteryDeleteDesc(view.DeleteKeywordMins)),
		"",
	}
	if view.ActiveID > 0 {
		lines = append(lines,
			fmt.Sprintf("进行中：#%d %s", view.ActiveID, view.ActiveTitle),
			fmt.Sprintf("参与关键词：%s", view.ActiveJoinKeyword),
			fmt.Sprintf("中奖人数：%d", view.ActiveWinnersCount),
			fmt.Sprintf("参与人数：%d", view.ActiveParticipants),
		)
	} else {
		lines = append(lines, "进行中：无")
	}
	if view.LatestID > 0 {
		lines = append(lines, "", fmt.Sprintf("最近一期：#%d %s [%s]", view.LatestID, view.LatestTitle, lotteryStatusLabel(view.LatestStatus)), fmt.Sprintf("关键词：%s", view.LatestJoinKeyword))
	}

	h.render(bot, target, strings.Join(lines, "\n"), lotteryKeyboard(tgGroupID, view.PublishPin, view.ResultPin, view.DeleteKeywordMins))
}

func (h *Handler) sendLotteryRecordsPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListLotteryRecordsByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载抽奖记录失败", groupPanelKeyboard(tgGroupID))
		return
	}
	totalPages := maxPages(data.Total, rulesPageSize)
	if data.Page < 1 {
		data.Page = 1
	}
	if data.Page > totalPages {
		data.Page = totalPages
	}
	lines := []string{fmt.Sprintf("创建的抽奖记录（第 %d/%d 页，总 %d 条）", data.Page, totalPages, data.Total)}
	if len(data.Items) == 0 {
		lines = append(lines, "暂无抽奖记录")
	}
	for _, item := range data.Items {
		keyword := strings.TrimSpace(item.Lottery.JoinKeyword)
		if keyword == "" {
			keyword = "参加"
		}
		lines = append(lines, fmt.Sprintf("#%d %s", item.Lottery.ID, item.Lottery.Title))
		lines = append(lines, fmt.Sprintf("状态:%s  中奖人数:%d  参与人数:%d", lotteryStatusLabel(item.Lottery.Status), item.Lottery.WinnersCount, item.Participants))
		lines = append(lines, fmt.Sprintf("口令:%s", keyword))
	}
	h.render(bot, target, strings.Join(lines, "\n"), lotteryRecordsKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendLotteryDeleteMinutesPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.LotteryPanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载抽奖面板失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"🎯 抽奖 - 删除口令",
		"",
		fmt.Sprintf("当前设置:%s", lotteryDeleteDesc(view.DeleteKeywordMins)),
		"请选择自动删除口令消息时长：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), lotteryDeleteMinutesKeyboard(tgGroupID, view.DeleteKeywordMins))
}

func (h *Handler) sendWelcomePanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	cfg, enabled, err := h.service.WelcomeViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载欢迎设置失败", groupPanelKeyboard(tgGroupID))
		return
	}
	modeText := "验证后欢迎"
	if cfg.Mode == "join" {
		modeText = "进群欢迎"
	}
	deleteText := "否"
	if cfg.DeleteMinutes > 0 {
		deleteText = fmt.Sprintf("%d", cfg.DeleteMinutes)
	}
	buttonCount := 0
	for _, row := range cfg.ButtonRows {
		buttonCount += len(row)
	}
	buttonText := onOffWithEmoji(buttonCount > 0)
	if buttonCount > 0 {
		buttonText = fmt.Sprintf("%s（%d个）", buttonText, buttonCount)
	}
	lines := []string{
		"🎉 进群欢迎",
		"",
		fmt.Sprintf("状态: %s", onOffWithEmoji(enabled)),
		fmt.Sprintf("模式: %s", modeText),
		fmt.Sprintf("删除消息(分钟): %s", deleteText),
		"",
		"自定义欢迎内容:",
		fmt.Sprintf("┌📸 媒体图片: %s", onOffWithEmoji(cfg.MediaFileID != "")),
		fmt.Sprintf("├🔠 链接按钮: %s", buttonText),
		fmt.Sprintf("└📄 文本内容: %s", onOffWithEmoji(strings.TrimSpace(cfg.Text) != "")),
	}
	h.render(bot, target, strings.Join(lines, "\n"), welcomeKeyboard(tgGroupID, enabled, cfg.Mode, cfg.DeleteMinutes))
}

func (h *Handler) sendWelcomeDeleteMinutesPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	cfg, _, err := h.service.WelcomeViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载欢迎设置失败", groupPanelKeyboard(tgGroupID))
		return
	}
	deleteText := "关闭"
	if cfg.DeleteMinutes > 0 {
		deleteText = fmt.Sprintf("%d分钟", cfg.DeleteMinutes)
	}
	lines := []string{
		"🎉 欢迎 - 删除消息",
		"",
		fmt.Sprintf("当前设置:%s", deleteText),
		"请选择欢迎消息自动删除时间：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), welcomeDeleteMinutesKeyboard(tgGroupID, cfg.DeleteMinutes))
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
	items, err := h.service.ListBlacklistByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载黑名单失败", groupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{"本群黑名单"}
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
	h.render(bot, target, text, settingsKeyboard(lang))
}
