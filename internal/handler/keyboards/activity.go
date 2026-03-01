package keyboards

import (
	"fmt"
	"strconv"
	"strings"

	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ChainKeyboard(tgGroupID int64, items []service.ChainSummary) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+3)
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("创建接龙", fmt.Sprintf("feat:chain:start:%s", gid)),
		),
	)
	for _, item := range items {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("导出 #%d", item.ID),
				fmt.Sprintf("feat:chain:export:%s:%d", gid, item.ID),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("关闭 #%d", item.ID),
				fmt.Sprintf("feat:chain:close:%s:%d", gid, item.ID),
			),
		))
	}
	rows = append(rows, panelRefreshBackRow(gid, fmt.Sprintf("feat:chain:view:%s", gid)))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func ChainLimitModeKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("不限制", fmt.Sprintf("feat:chain:limmode:%s:none", gid)),
			tgbotapi.NewInlineKeyboardButtonData("限制人数", fmt.Sprintf("feat:chain:limmode:%s:people", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("限制时间", fmt.Sprintf("feat:chain:limmode:%s:time", gid)),
			tgbotapi.NewInlineKeyboardButtonData("人数+时间", fmt.Sprintf("feat:chain:limmode:%s:both", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func ChainDurationKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("30分钟", fmt.Sprintf("feat:chain:setdur:%s:1800", gid)),
			tgbotapi.NewInlineKeyboardButtonData("1小时", fmt.Sprintf("feat:chain:setdur:%s:3600", gid)),
			tgbotapi.NewInlineKeyboardButtonData("2小时", fmt.Sprintf("feat:chain:setdur:%s:7200", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("6小时", fmt.Sprintf("feat:chain:setdur:%s:21600", gid)),
			tgbotapi.NewInlineKeyboardButtonData("12小时", fmt.Sprintf("feat:chain:setdur:%s:43200", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1天", fmt.Sprintf("feat:chain:setdur:%s:86400", gid)),
			tgbotapi.NewInlineKeyboardButtonData("3天", fmt.Sprintf("feat:chain:setdur:%s:259200", gid)),
			tgbotapi.NewInlineKeyboardButtonData("7天", fmt.Sprintf("feat:chain:setdur:%s:604800", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("无截止", fmt.Sprintf("feat:chain:setdur:%s:0", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func ChainPublicJoinKeyboard(joinURL string, active bool) tgbotapi.InlineKeyboardMarkup {
	if !active || strings.TrimSpace(joinURL) == "" {
		return tgbotapi.NewInlineKeyboardMarkup()
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("点击参加接龙", joinURL),
		),
	)
}

func MonitorKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("新增关键词", fmt.Sprintf("feat:monitor:add:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("移除关键词", fmt.Sprintf("feat:monitor:remove:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:monitor:view:%s", gid)),
	)
}

func PollKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("创建投票", fmt.Sprintf("feat:poll:create:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("结束投票", fmt.Sprintf("feat:poll:stop:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:poll:view:%s", gid)),
	)
}

func PollCreateDraftKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("完成创建", fmt.Sprintf("feat:poll:submit:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("清空选项", fmt.Sprintf("feat:poll:reset:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func LotteryKeyboard(tgGroupID int64, publishPin bool, resultPin bool, deleteKeywordMins int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	deleteText := "关闭"
	if deleteKeywordMins > 0 {
		deleteText = fmt.Sprintf("%d分钟", deleteKeywordMins)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("创建抽奖", fmt.Sprintf("feat:lottery:create:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("立即开奖", fmt.Sprintf("feat:lottery:draw:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("创建的抽奖记录", fmt.Sprintf("feat:lottery:records:%s:1", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("发布置顶 "+boolIcon(publishPin), fmt.Sprintf("feat:lottery:toggle:%s:publish_pin", gid)),
			tgbotapi.NewInlineKeyboardButtonData("结果置顶 "+boolIcon(resultPin), fmt.Sprintf("feat:lottery:toggle:%s:result_pin", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("删除口令 "+deleteText, fmt.Sprintf("feat:lottery:delmins:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:lottery:view:%s", gid)),
	)
}

func LotteryRecordsKeyboard(tgGroupID int64, items []service.LotteryRecordItem, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+3)
	for _, item := range items {
		if item.Lottery.Status != "active" {
			continue
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("取消 #%d", item.Lottery.ID),
				fmt.Sprintf("feat:lottery:cancel:%s:%d:%d", gid, item.Lottery.ID, page),
			),
		))
	}
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:lottery:records:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:lottery:records:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 刷新记录", fmt.Sprintf("feat:lottery:records:%s:%d", gid, page)),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回抽奖面板", fmt.Sprintf("feat:lottery:view:%s", gid)),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func LotteryDeleteMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	offLabel := selectedLabel("关闭", current <= 0)
	m1Label := selectedLabel("1分钟", current == 1)
	m3Label := selectedLabel("3分钟", current == 3)
	m5Label := selectedLabel("5分钟", current == 5)
	m10Label := selectedLabel("10分钟", current == 10)
	m30Label := selectedLabel("30分钟", current == 30)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(offLabel, fmt.Sprintf("feat:lottery:delminsset:%s:0", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m1Label, fmt.Sprintf("feat:lottery:delminsset:%s:1", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m3Label, fmt.Sprintf("feat:lottery:delminsset:%s:3", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(m5Label, fmt.Sprintf("feat:lottery:delminsset:%s:5", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m10Label, fmt.Sprintf("feat:lottery:delminsset:%s:10", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m30Label, fmt.Sprintf("feat:lottery:delminsset:%s:30", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回抽奖面板", fmt.Sprintf("feat:lottery:view:%s", gid)),
		),
	)
}

func PointsKeyboard(tgGroupID int64, view *service.PointsPanelView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:points:noop:%s", gid),
			fmt.Sprintf("feat:points:on:%s", gid),
			fmt.Sprintf("feat:points:off:%s", gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("签到规则", fmt.Sprintf("feat:points:checkin:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("发言规则", fmt.Sprintf("feat:points:message:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("邀请规则", fmt.Sprintf("feat:points:invite:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("积分别名："+view.Config.BalanceAlias, fmt.Sprintf("feat:points:aliasbalance:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("排行别名："+view.Config.RankAlias, fmt.Sprintf("feat:points:aliasrank:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("增加积分(自定义)", fmt.Sprintf("feat:points:add:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("扣除积分(自定义)", fmt.Sprintf("feat:points:sub:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:points:view:%s", gid)),
	)
}

func PointsCheckinKeyboard(tgGroupID int64, view *service.PointsPanelView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("签到口令："+view.Config.CheckinKeyword, fmt.Sprintf("feat:points:checkinkey:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("签到奖励：%d", view.Config.CheckinReward), fmt.Sprintf("feat:points:checkinreward:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 刷新", fmt.Sprintf("feat:points:checkin:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回积分面板", fmt.Sprintf("feat:points:view:%s", gid)),
		),
	)
}

func PointsMessageKeyboard(tgGroupID int64, view *service.PointsPanelView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("单次奖励：%d", view.Config.MessageReward), fmt.Sprintf("feat:points:msgreward:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("发言上限："+pointsLimitText(view.Config.MessageDaily), fmt.Sprintf("feat:points:msgdaily:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("最小字数："+pointsLimitText(view.Config.MessageMinLen), fmt.Sprintf("feat:points:msgmin:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 刷新", fmt.Sprintf("feat:points:message:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回积分面板", fmt.Sprintf("feat:points:view:%s", gid)),
		),
	)
}

func PointsInviteKeyboard(tgGroupID int64, view *service.PointsPanelView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("邀请奖励：%d", view.Config.InviteReward), fmt.Sprintf("feat:points:invitereward:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("邀请上限："+pointsLimitText(view.Config.InviteDaily), fmt.Sprintf("feat:points:invitedaily:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 刷新", fmt.Sprintf("feat:points:invite:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回积分面板", fmt.Sprintf("feat:points:view:%s", gid)),
		),
	)
}

func pointsLimitText(v int) string {
	if v <= 0 {
		return "无限制"
	}
	return strconv.Itoa(v)
}

func WelcomeKeyboard(tgGroupID int64, enabled bool, mode string, deleteMinutes int) tgbotapi.InlineKeyboardMarkup {
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
	return tgbotapi.NewInlineKeyboardMarkup(
		statusControlRow(
			enabled,
			fmt.Sprintf("feat:welcome:noop:%s", gid),
			fmt.Sprintf("feat:welcome:on:%s", gid),
			fmt.Sprintf("feat:welcome:off:%s", gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("模式："+modeText, fmt.Sprintf("feat:welcome:mode:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("删除消息（分钟）："+deleteText, fmt.Sprintf("feat:welcome:delmins:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("修改文本", fmt.Sprintf("feat:welcome:set:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("修改媒体", fmt.Sprintf("feat:welcome:media:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("修改按钮", fmt.Sprintf("feat:welcome:button:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("预览", fmt.Sprintf("feat:welcome:preview:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:welcome:view:%s", gid)),
	)
}

func WelcomeDeleteMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	offLabel := selectedLabel("关闭", current <= 0)
	m1Label := selectedLabel("1分钟", current == 1)
	m5Label := selectedLabel("5分钟", current == 5)
	m10Label := selectedLabel("10分钟", current == 10)
	m30Label := selectedLabel("30分钟", current == 30)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(offLabel, fmt.Sprintf("feat:welcome:delminsset:%s:0", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m1Label, fmt.Sprintf("feat:welcome:delminsset:%s:1", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m5Label, fmt.Sprintf("feat:welcome:delminsset:%s:5", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(m10Label, fmt.Sprintf("feat:welcome:delminsset:%s:10", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m30Label, fmt.Sprintf("feat:welcome:delminsset:%s:30", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回欢迎面板", fmt.Sprintf("feat:welcome:view:%s", gid)),
		),
	)
}
