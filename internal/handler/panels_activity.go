package handler

import (
	"fmt"
	"strings"
	"supervisor/internal/handler/keyboards"

	tgbot "github.com/go-telegram/bot"
)

func (h *Handler) sendMonitorPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	items, err := h.service.ListMonitorKeywordsByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载关键词监控失败", keyboards.GroupPanelKeyboard(tgGroupID))
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
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.MonitorKeyboard(tgGroupID))
}

func (h *Handler) sendWordCloudPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.WordCloudPanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载词云面板失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	status := "❌ 关闭"
	if view.Enabled {
		status = "✅ 开启"
	}
	lines := []string{
		"☁️ 词云统计",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("定时推送时间:%02d:%02d", view.PushHour, view.PushMinute),
		fmt.Sprintf("黑名单词语:%d 个", view.BlacklistCount),
		"",
		"说明：",
		"1) 开启后，机器人会对群内消息分词并持久化词频",
		"2) 到达推送时间会自动发送今日词云统计",
		"3) 管理员可在群内使用 /wordcloud 立即生成",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.WordCloudKeyboard(tgGroupID, view))
}

func (h *Handler) sendWordCloudBlacklistPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListWordCloudBlacklistByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载词云黑名单失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	totalPages := maxPages(data.Total, rulesPageSize)
	if data.Page < 1 {
		data.Page = 1
	}
	if data.Page > totalPages {
		data.Page = totalPages
	}
	lines := []string{fmt.Sprintf("词云黑名单（第 %d/%d 页，总 %d 条）", data.Page, totalPages, data.Total)}
	if len(data.Items) == 0 {
		lines = append(lines, "暂无黑名单词")
	} else {
		for _, item := range data.Items {
			lines = append(lines, fmt.Sprintf("#%d %s", item.ID, item.Word))
		}
	}
	lines = append(lines, "", "新增/移除时请输入单个词语（建议小写）")
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.WordCloudBlacklistKeyboard(tgGroupID, data.Page, totalPages))
}

func (h *Handler) sendPollPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	text := strings.Join([]string{
		"投票管理",
		"创建流程：",
		"第1步：输入投票问题",
		"第2步：逐条或批量输入选项（每条消息可输入1个或多个）",
		"第3步：点击“完成创建”发布",
		"",
		"兼容快捷格式：问题|选项1,选项2,...",
	}, "\n")
	h.render(bot, target, text, keyboards.PollKeyboard(tgGroupID))
}

func (h *Handler) sendLotteryPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.LotteryPanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载抽奖面板失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}

	lines := []string{
		"抽奖管理",
		"创建流程：",
		"第1步：输入抽奖标题",
		"第2步：输入中奖人数",
		"第3步：输入参与关键词",
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

	h.render(bot, target, strings.Join(lines, "\n"), keyboards.LotteryKeyboard(tgGroupID, view.PublishPin, view.ResultPin, view.DeleteKeywordMins))
}

func (h *Handler) sendLotteryRecordsPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64, page int) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	data, err := h.service.ListLotteryRecordsByTGGroupID(tgGroupID, page, rulesPageSize)
	if err != nil {
		h.render(bot, target, "加载抽奖记录失败", keyboards.GroupPanelKeyboard(tgGroupID))
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
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.LotteryRecordsKeyboard(tgGroupID, data.Items, data.Page, totalPages))
}

func (h *Handler) sendLotteryDeleteMinutesPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.LotteryPanelViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载抽奖面板失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"🎯 抽奖 - 删除口令",
		"",
		fmt.Sprintf("当前设置:%s", lotteryDeleteDesc(view.DeleteKeywordMins)),
		"请选择自动删除口令消息时长：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.LotteryDeleteMinutesKeyboard(tgGroupID, view.DeleteKeywordMins))
}
