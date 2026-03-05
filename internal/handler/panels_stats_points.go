package handler

import (
	"fmt"
	"strconv"
	"strings"
	"supervisor/internal/handler/keyboards"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) sendStatsPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	stats, err := h.service.GroupStatsByTGGroupID(tgGroupID, 10)
	if err != nil {
		h.render(bot, target, "加载统计失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		fmt.Sprintf("群统计：%s (%d)", stats.GroupTitle, stats.GroupID),
		"",
		"累计数据:",
		fmt.Sprintf("├ 积分用户数: %d", stats.PointsUsersTotal),
		fmt.Sprintf("├ 积分总额: %d", stats.PointsTotal),
		fmt.Sprintf("├ 有效邀请总人数: %d", stats.InviteTotal),
		fmt.Sprintf("├ 发言积分总额: %d", stats.MessagePointsTotal),
		fmt.Sprintf("├ 发言事件总数: %d", stats.MessageEventsTotal),
		fmt.Sprintf("└ 发言贡献人数: %d", stats.MessageUsersTotal),
		"",
		fmt.Sprintf("今日数据（%s %s）:", stats.TimezoneText, stats.DayKey),
		fmt.Sprintf("├ 今日发言积分: %d", stats.TodayMessagePoints),
		fmt.Sprintf("├ 今日发言人数: %d", stats.TodayMessageUsers),
		fmt.Sprintf("└ 今日签到次数: %d", stats.TodayCheckins),
		"",
		fmt.Sprintf("近7日（含今日，按%s）:", stats.TimezoneText),
		fmt.Sprintf("├ 发言积分: %d", stats.Recent7MessagePoints),
		fmt.Sprintf("├ 发言人数: %d", stats.Recent7MessageUsers),
		fmt.Sprintf("├ 发言事件: %d", stats.Recent7MessageEvents),
		fmt.Sprintf("├ 签到次数: %d", stats.Recent7Checkins),
		fmt.Sprintf("└ 有效邀请: %d", stats.Recent7Invites),
		"",
		fmt.Sprintf("近30日（含今日，按%s）:", stats.TimezoneText),
		fmt.Sprintf("├ 发言积分: %d", stats.Recent30MessagePoints),
		fmt.Sprintf("├ 发言人数: %d", stats.Recent30MessageUsers),
		fmt.Sprintf("├ 发言事件: %d", stats.Recent30MessageEvents),
		fmt.Sprintf("├ 签到次数: %d", stats.Recent30Checkins),
		fmt.Sprintf("└ 有效邀请: %d", stats.Recent30Invites),
		"",
	}
	if len(stats.TopUsers) == 0 {
		lines = append(lines, "暂无活跃数据")
	} else {
		lines = append(lines, "活跃榜（按消息积分）:")
		for i, u := range stats.TopUsers {
			lines = append(lines, fmt.Sprintf("%d. %s - %d", i+1, u.DisplayName, u.Points))
		}
	}
	markup := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "刷新统计", CallbackData: fmt.Sprintf("feat:stats:show:%d", tgGroupID)},
				{Text: "返回群面板", CallbackData: cbGroupPrefix + strconv.FormatInt(tgGroupID, 10)},
			},
		},
	}
	h.render(bot, target, strings.Join(lines, "\n"), markup)
}

func (h *Handler) sendPointsPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.PointsPanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载积分设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	status := "❌ 关闭"
	if view.Enabled {
		status = "✅ 开启"
	}
	msgDaily := "无限制"
	if view.Config.MessageDaily > 0 {
		msgDaily = strconv.Itoa(view.Config.MessageDaily)
	}
	msgMinLen := "无限制"
	if view.Config.MessageMinLen > 0 {
		msgMinLen = strconv.Itoa(view.Config.MessageMinLen)
	}
	inviteDaily := "无限制"
	if view.Config.InviteDaily > 0 {
		inviteDaily = strconv.Itoa(view.Config.InviteDaily)
	}
	lines := []string{
		"💰 积分系统",
		"",
		fmt.Sprintf("状态: %s", status),
		"规则配置入口:",
		"└ 签到规则 / 发言规则 / 邀请规则（独立面板）",
		"",
		"签到规则摘要:",
		fmt.Sprintf("└ 口令:%s  奖励:%d 积分", view.Config.CheckinKeyword, view.Config.CheckinReward),
		"发言规则摘要:",
		fmt.Sprintf("└ 单次奖励:%d  每日上限:%s  最小字数:%s", view.Config.MessageReward, msgDaily, msgMinLen),
		"邀请规则摘要:",
		fmt.Sprintf("└ 单次奖励:%d  每日上限:%s", view.Config.InviteReward, inviteDaily),
		"积分别名:",
		fmt.Sprintf("└ 群组中发送“%s”查询自己的积分（可配置）", view.Config.BalanceAlias),
		"排行别名：",
		fmt.Sprintf("└ 群组中发送“%s”查询积分排名（可配置）", view.Config.RankAlias),
		fmt.Sprintf("抽奖消耗:\n└ 参与抽奖消耗:%d 积分", view.Config.LotteryCost),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.PointsKeyboard(tgGroupID, view))
}

func (h *Handler) sendPointsCheckinPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.PointsPanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载签到规则失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"💰 积分系统 - 签到规则",
		"",
		fmt.Sprintf("签到口令:%s", view.Config.CheckinKeyword),
		fmt.Sprintf("签到奖励:%d 积分", view.Config.CheckinReward),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.PointsCheckinKeyboard(tgGroupID, view))
}

func (h *Handler) sendPointsMessagePanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.PointsPanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载发言规则失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	msgDaily := "无限制"
	if view.Config.MessageDaily > 0 {
		msgDaily = strconv.Itoa(view.Config.MessageDaily)
	}
	msgMinLen := "无限制"
	if view.Config.MessageMinLen > 0 {
		msgMinLen = strconv.Itoa(view.Config.MessageMinLen)
	}
	lines := []string{
		"💰 积分系统 - 发言规则",
		"",
		fmt.Sprintf("单次发言奖励:%d 积分", view.Config.MessageReward),
		fmt.Sprintf("每日获取上限:%s 积分", msgDaily),
		fmt.Sprintf("最小字数限制:%s", msgMinLen),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.PointsMessageKeyboard(tgGroupID, view))
}

func (h *Handler) sendPointsInvitePanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.PointsPanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载邀请规则失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	inviteDaily := "无限制"
	if view.Config.InviteDaily > 0 {
		inviteDaily = strconv.Itoa(view.Config.InviteDaily)
	}
	lines := []string{
		"💰 积分系统 - 邀请规则",
		"",
		fmt.Sprintf("每次邀请奖励:%d 积分", view.Config.InviteReward),
		fmt.Sprintf("每日获取上限:%s 积分", inviteDaily),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.PointsInviteKeyboard(tgGroupID, view))
}

func (h *Handler) sendInvitePanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.InvitePanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载邀请链接设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
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
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.InviteKeyboard(tgGroupID, view.Enabled))
}

func (h *Handler) sendLogPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64, page int, filter string) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListLogsByTGGroupID(tgGroupID, page, rulesPageSize, filter)
	if err != nil {
		h.render(bot, target, "加载管理日志失败", keyboards.GroupPanelKeyboard(tgGroupID))
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
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.LogListKeyboard(tgGroupID, data.Page, totalPages, filter))
}
