package keyboards

import (
	"fmt"
	"strconv"
	"strings"

	"supervisor/internal/service"

	"github.com/go-telegram/bot/models"
)

func ChainKeyboard(tgGroupID int64, items []service.ChainSummary) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]models.InlineKeyboardButton, 0, len(items)+3)
	rows = append(rows,
		inlineKeyboardRow(
			inlineKeyboardButtonData("创建接龙", fmt.Sprintf("feat:chain:start:%s", gid)),
		),
	)
	for _, item := range items {
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData(
				fmt.Sprintf("导出 #%d", item.ID),
				fmt.Sprintf("feat:chain:export:%s:%d", gid, item.ID),
			),
			inlineKeyboardButtonData(
				fmt.Sprintf("关闭 #%d", item.ID),
				fmt.Sprintf("feat:chain:close:%s:%d", gid, item.ID),
			),
		))
	}
	rows = append(rows, panelRefreshBackRow(gid, fmt.Sprintf("feat:chain:view:%s", gid)))
	return inlineKeyboardMarkup(rows...)
}

func ChainLimitModeKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("不限制", fmt.Sprintf("feat:chain:limmode:%s:none", gid)),
			inlineKeyboardButtonData("限制人数", fmt.Sprintf("feat:chain:limmode:%s:people", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("限制时间", fmt.Sprintf("feat:chain:limmode:%s:time", gid)),
			inlineKeyboardButtonData("人数+时间", fmt.Sprintf("feat:chain:limmode:%s:both", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			inlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func ChainDurationKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("30分钟", fmt.Sprintf("feat:chain:setdur:%s:1800", gid)),
			inlineKeyboardButtonData("1小时", fmt.Sprintf("feat:chain:setdur:%s:3600", gid)),
			inlineKeyboardButtonData("2小时", fmt.Sprintf("feat:chain:setdur:%s:7200", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("6小时", fmt.Sprintf("feat:chain:setdur:%s:21600", gid)),
			inlineKeyboardButtonData("12小时", fmt.Sprintf("feat:chain:setdur:%s:43200", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("1天", fmt.Sprintf("feat:chain:setdur:%s:86400", gid)),
			inlineKeyboardButtonData("3天", fmt.Sprintf("feat:chain:setdur:%s:259200", gid)),
			inlineKeyboardButtonData("7天", fmt.Sprintf("feat:chain:setdur:%s:604800", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("无截止", fmt.Sprintf("feat:chain:setdur:%s:0", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			inlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func ChainPublicJoinKeyboard(joinURL string, active bool) models.InlineKeyboardMarkup {
	if !active || strings.TrimSpace(joinURL) == "" {
		return inlineKeyboardMarkup()
	}
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonURL("点击参加接龙", joinURL),
		),
	)
}

func MonitorKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("新增关键词", fmt.Sprintf("feat:monitor:add:%s", gid)),
			inlineKeyboardButtonData("移除关键词", fmt.Sprintf("feat:monitor:remove:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:monitor:view:%s", gid)),
	)
}

func WordCloudKeyboard(tgGroupID int64, view *service.WordCloudPanelView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	pushText := fmt.Sprintf("%02d:%02d", view.PushHour, view.PushMinute)
	return inlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:wc:noop:%s", gid),
			fmt.Sprintf("feat:wc:on:%s", gid),
			fmt.Sprintf("feat:wc:off:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("立即生成词云", fmt.Sprintf("feat:wc:gen:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("推送时间："+pushText, fmt.Sprintf("feat:wc:settimeinput:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("09:00", fmt.Sprintf("feat:wc:settime:%s:0900", gid)),
			inlineKeyboardButtonData("12:00", fmt.Sprintf("feat:wc:settime:%s:1200", gid)),
			inlineKeyboardButtonData("18:00", fmt.Sprintf("feat:wc:settime:%s:1800", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("22:00", fmt.Sprintf("feat:wc:settime:%s:2200", gid)),
			inlineKeyboardButtonData("黑名单词语", fmt.Sprintf("feat:wc:blacklist:%s:1", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:wc:view:%s", gid)),
	)
}

func WordCloudBlacklistKeyboard(tgGroupID int64, page, totalPages int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]models.InlineKeyboardButton, 0, 5)
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("新增黑名单词", fmt.Sprintf("feat:wc:blackadd:%s", gid)),
		inlineKeyboardButtonData("移除黑名单词", fmt.Sprintf("feat:wc:blackremove:%s", gid)),
	))
	nav := make([]models.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, inlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:wc:blacklist:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, inlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:wc:blacklist:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, inlineKeyboardRow(nav...))
	}
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("🔄 刷新", fmt.Sprintf("feat:wc:blacklist:%s:%d", gid, page)),
		inlineKeyboardButtonData("◀ 返回词云面板", fmt.Sprintf("feat:wc:view:%s", gid)),
	))
	return inlineKeyboardMarkup(rows...)
}

func PollKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("创建投票", fmt.Sprintf("feat:poll:create:%s", gid)),
			inlineKeyboardButtonData("结束投票", fmt.Sprintf("feat:poll:stop:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:poll:view:%s", gid)),
	)
}

func PollCreateDraftKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("完成创建", fmt.Sprintf("feat:poll:submit:%s", gid)),
			inlineKeyboardButtonData("清空选项", fmt.Sprintf("feat:poll:reset:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			inlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func LotteryKeyboard(tgGroupID int64, publishPin bool, resultPin bool, deleteKeywordMins int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	deleteText := "关闭"
	if deleteKeywordMins > 0 {
		deleteText = fmt.Sprintf("%d分钟", deleteKeywordMins)
	}
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("创建抽奖", fmt.Sprintf("feat:lottery:create:%s", gid)),
			inlineKeyboardButtonData("立即开奖", fmt.Sprintf("feat:lottery:draw:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("创建的抽奖记录", fmt.Sprintf("feat:lottery:records:%s:1", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("发布置顶 "+boolIcon(publishPin), fmt.Sprintf("feat:lottery:toggle:%s:publish_pin", gid)),
			inlineKeyboardButtonData("结果置顶 "+boolIcon(resultPin), fmt.Sprintf("feat:lottery:toggle:%s:result_pin", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("删除口令 "+deleteText, fmt.Sprintf("feat:lottery:delmins:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:lottery:view:%s", gid)),
	)
}

func LotteryRecordsKeyboard(tgGroupID int64, items []service.LotteryRecordItem, page, totalPages int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]models.InlineKeyboardButton, 0, len(items)+3)
	for _, item := range items {
		if item.Lottery.Status != "active" {
			continue
		}
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData(
				fmt.Sprintf("取消 #%d", item.Lottery.ID),
				fmt.Sprintf("feat:lottery:cancel:%s:%d:%d", gid, item.Lottery.ID, page),
			),
		))
	}
	nav := make([]models.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, inlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:lottery:records:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, inlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:lottery:records:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, inlineKeyboardRow(nav...))
	}
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("🔄 刷新记录", fmt.Sprintf("feat:lottery:records:%s:%d", gid, page)),
		inlineKeyboardButtonData("◀ 返回抽奖面板", fmt.Sprintf("feat:lottery:view:%s", gid)),
	))
	return inlineKeyboardMarkup(rows...)
}

func LotteryDeleteMinutesKeyboard(tgGroupID int64, current int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	offLabel := selectedLabel("关闭", current <= 0)
	m1Label := selectedLabel("1分钟", current == 1)
	m3Label := selectedLabel("3分钟", current == 3)
	m5Label := selectedLabel("5分钟", current == 5)
	m10Label := selectedLabel("10分钟", current == 10)
	m30Label := selectedLabel("30分钟", current == 30)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(offLabel, fmt.Sprintf("feat:lottery:delminsset:%s:0", gid)),
			inlineKeyboardButtonData(m1Label, fmt.Sprintf("feat:lottery:delminsset:%s:1", gid)),
			inlineKeyboardButtonData(m3Label, fmt.Sprintf("feat:lottery:delminsset:%s:3", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(m5Label, fmt.Sprintf("feat:lottery:delminsset:%s:5", gid)),
			inlineKeyboardButtonData(m10Label, fmt.Sprintf("feat:lottery:delminsset:%s:10", gid)),
			inlineKeyboardButtonData(m30Label, fmt.Sprintf("feat:lottery:delminsset:%s:30", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回抽奖面板", fmt.Sprintf("feat:lottery:view:%s", gid)),
		),
	)
}

func PointsKeyboard(tgGroupID int64, view *service.PointsPanelView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:points:noop:%s", gid),
			fmt.Sprintf("feat:points:on:%s", gid),
			fmt.Sprintf("feat:points:off:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("签到规则", fmt.Sprintf("feat:points:checkin:%s", gid)),
			inlineKeyboardButtonData("发言规则", fmt.Sprintf("feat:points:message:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("邀请规则", fmt.Sprintf("feat:points:invite:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("积分别名："+view.Config.BalanceAlias, fmt.Sprintf("feat:points:aliasbalance:%s", gid)),
			inlineKeyboardButtonData("排行别名："+view.Config.RankAlias, fmt.Sprintf("feat:points:aliasrank:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("增加积分(自定义)", fmt.Sprintf("feat:points:add:%s", gid)),
			inlineKeyboardButtonData("扣除积分(自定义)", fmt.Sprintf("feat:points:sub:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:points:view:%s", gid)),
	)
}

func PointsCheckinKeyboard(tgGroupID int64, view *service.PointsPanelView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("签到口令："+view.Config.CheckinKeyword, fmt.Sprintf("feat:points:checkinkey:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("签到奖励：%d", view.Config.CheckinReward), fmt.Sprintf("feat:points:checkinreward:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🔄 刷新", fmt.Sprintf("feat:points:checkin:%s", gid)),
			inlineKeyboardButtonData("◀ 返回积分面板", fmt.Sprintf("feat:points:view:%s", gid)),
		),
	)
}

func PointsMessageKeyboard(tgGroupID int64, view *service.PointsPanelView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("单次奖励：%d", view.Config.MessageReward), fmt.Sprintf("feat:points:msgreward:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("发言上限："+pointsLimitText(view.Config.MessageDaily), fmt.Sprintf("feat:points:msgdaily:%s", gid)),
			inlineKeyboardButtonData("最小字数："+pointsLimitText(view.Config.MessageMinLen), fmt.Sprintf("feat:points:msgmin:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🔄 刷新", fmt.Sprintf("feat:points:message:%s", gid)),
			inlineKeyboardButtonData("◀ 返回积分面板", fmt.Sprintf("feat:points:view:%s", gid)),
		),
	)
}

func PointsInviteKeyboard(tgGroupID int64, view *service.PointsPanelView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("邀请奖励：%d", view.Config.InviteReward), fmt.Sprintf("feat:points:invitereward:%s", gid)),
			inlineKeyboardButtonData("邀请上限："+pointsLimitText(view.Config.InviteDaily), fmt.Sprintf("feat:points:invitedaily:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🔄 刷新", fmt.Sprintf("feat:points:invite:%s", gid)),
			inlineKeyboardButtonData("◀ 返回积分面板", fmt.Sprintf("feat:points:view:%s", gid)),
		),
	)
}

func pointsLimitText(v int) string {
	if v <= 0 {
		return "无限制"
	}
	return strconv.Itoa(v)
}

func WelcomeKeyboard(tgGroupID int64, enabled bool, mode string, deleteMinutes int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	_ = enabled
	modeText := "验证后欢迎"
	if mode == "join" {
		modeText = "进群欢迎"
	}
	deleteText := "否"
	if deleteMinutes > 0 {
		deleteText = strconv.Itoa(deleteMinutes)
	}
	return inlineKeyboardMarkup(
		statusControlRow(
			enabled,
			fmt.Sprintf("feat:welcome:noop:%s", gid),
			fmt.Sprintf("feat:welcome:on:%s", gid),
			fmt.Sprintf("feat:welcome:off:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("模式："+modeText, fmt.Sprintf("feat:welcome:mode:%s", gid)),
			inlineKeyboardButtonData("删除消息（分钟）："+deleteText, fmt.Sprintf("feat:welcome:delmins:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("修改文本", fmt.Sprintf("feat:welcome:set:%s", gid)),
			inlineKeyboardButtonData("修改媒体", fmt.Sprintf("feat:welcome:media:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("修改按钮", fmt.Sprintf("feat:welcome:button:%s", gid)),
			inlineKeyboardButtonData("预览", fmt.Sprintf("feat:welcome:preview:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:welcome:view:%s", gid)),
	)
}

func WelcomeDeleteMinutesKeyboard(tgGroupID int64, current int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	offLabel := selectedLabel("关闭", current <= 0)
	m1Label := selectedLabel("1分钟", current == 1)
	m5Label := selectedLabel("5分钟", current == 5)
	m10Label := selectedLabel("10分钟", current == 10)
	m30Label := selectedLabel("30分钟", current == 30)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(offLabel, fmt.Sprintf("feat:welcome:delminsset:%s:0", gid)),
			inlineKeyboardButtonData(m1Label, fmt.Sprintf("feat:welcome:delminsset:%s:1", gid)),
			inlineKeyboardButtonData(m5Label, fmt.Sprintf("feat:welcome:delminsset:%s:5", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(m10Label, fmt.Sprintf("feat:welcome:delminsset:%s:10", gid)),
			inlineKeyboardButtonData(m30Label, fmt.Sprintf("feat:welcome:delminsset:%s:30", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回欢迎面板", fmt.Sprintf("feat:welcome:view:%s", gid)),
		),
	)
}
