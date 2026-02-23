package handler

import (
	"fmt"
	"strconv"

	"supervisor/internal/model"
	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func mainMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 我的群组", cbMenuGroups),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ 设置", cbMenuSettings),
		),
	)
}

func groupsKeyboard(groups []model.Group, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(groups)+3)
	for _, g := range groups {
		label := g.Title
		if label == "" {
			label = strconv.FormatInt(g.TGGroupID, 10)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗂 "+label, cbGroupPrefix+strconv.FormatInt(g.TGGroupID, 10)),
		))
	}

	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("%s%d", cbGroupsPagePF, page-1)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("%s%d", cbGroupsPagePF, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔄 刷新", cbMenuGroups)))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func groupPanelKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	id := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤖 自动回复", fmt.Sprintf("feat:auto:view:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("🚫 违禁词", fmt.Sprintf("feat:bw:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👋 欢迎设置", fmt.Sprintf("feat:welcome:view:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("🎯 抽奖", fmt.Sprintf("feat:lottery:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗳 投票", fmt.Sprintf("feat:poll:view:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("📋 接龙", fmt.Sprintf("feat:chain:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👁 关键词监控", fmt.Sprintf("feat:monitor:view:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("⏰ 定时消息", fmt.Sprintf("feat:sched:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 数据统计", fmt.Sprintf("feat:stats:show:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("📜 管理日志", fmt.Sprintf("feat:logs:list:%s:1:all", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📨 邀请链接", fmt.Sprintf("feat:invite:create:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("🧹 系统消息清理", fmt.Sprintf("feat:sys:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 反垃圾开关", fmt.Sprintf("feat:mod:spam:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("⚡ 反刷屏设置", fmt.Sprintf("feat:mod:floodview:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧩 验证设置", fmt.Sprintf("feat:mod:verifyview:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("🔒 新成员限制设置", fmt.Sprintf("feat:mod:newbieview:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧭 权限分级", fmt.Sprintf("feat:rbac:view:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("⛔ 黑名单", fmt.Sprintf("feat:black:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群组列表", cbMenuGroups),
		),
	)
}

func pendingCancelKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("取消并返回群面板", fmt.Sprintf("feat:pending:cancel:%d", tgGroupID)),
		),
	)
}

func autoReplyListKeyboard(tgGroupID int64, items []model.AutoReply, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+4)
	for _, item := range items {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("✏️ 编辑 #%d", item.ID),
				fmt.Sprintf("feat:auto:edit:%s:%d:%d", gid, item.ID, page),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🗑 删除 #%d", item.ID),
				fmt.Sprintf("feat:auto:del:%s:%d:%d", gid, item.ID, page),
			),
		))
	}
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:auto:list:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:auto:list:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ 新增自动回复", fmt.Sprintf("feat:auto:add:%s", gid)),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func bannedWordListKeyboard(tgGroupID int64, items []model.BannedWord, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+4)
	for _, item := range items {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("✏️ 编辑 #%d", item.ID),
				fmt.Sprintf("feat:bw:edit:%s:%d:%d", gid, item.ID, page),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🗑 删除 #%d", item.ID),
				fmt.Sprintf("feat:bw:del:%s:%d:%d", gid, item.ID, page),
			),
		))
	}
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:bw:list:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:bw:list:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ 新增违禁词", fmt.Sprintf("feat:bw:add:%s", gid)),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func scheduledListKeyboard(tgGroupID int64, items []model.ScheduledMessage, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+5)
	for _, item := range items {
		toggleLabel := fmt.Sprintf("启用 #%d", item.ID)
		if item.Enabled {
			toggleLabel = fmt.Sprintf("停用 #%d", item.ID)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				toggleLabel,
				fmt.Sprintf("feat:sched:toggle:%s:%d:%d", gid, item.ID, page),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🗑 删除 #%d", item.ID),
				fmt.Sprintf("feat:sched:del:%s:%d:%d", gid, item.ID, page),
			),
		))
	}
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:sched:list:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:sched:list:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ 新建定时", fmt.Sprintf("feat:sched:add:%s", gid)),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func logListKeyboard(tgGroupID int64, page, totalPages int, filter string) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, 3)
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:logs:list:%s:%d:%s", gid, page-1, filter)))
	}
	if page < totalPages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:logs:list:%s:%d:%s", gid, page+1, filter)))
	}
	if len(nav) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("全部", fmt.Sprintf("feat:logs:list:%s:1:all", gid)),
		tgbotapi.NewInlineKeyboardButtonData("审核", fmt.Sprintf("feat:logs:list:%s:1:anti_spam_delete", gid)),
		tgbotapi.NewInlineKeyboardButtonData("验证", fmt.Sprintf("feat:logs:list:%s:1:join_verify_pass", gid)),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("导出 CSV", fmt.Sprintf("feat:logs:export:%s:%s", gid, filter)),
		tgbotapi.NewInlineKeyboardButtonData("刷新日志", fmt.Sprintf("feat:logs:list:%s:%d:%s", gid, page, filter)),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func systemCleanKeyboard(tgGroupID int64, cfg *service.SystemCleanView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("严格", fmt.Sprintf("feat:sys:preset:%s:strict", gid)),
			tgbotapi.NewInlineKeyboardButtonData("推荐", fmt.Sprintf("feat:sys:preset:%s:recommended", gid)),
			tgbotapi.NewInlineKeyboardButtonData("关闭", fmt.Sprintf("feat:sys:preset:%s:off", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("进群 "+onOffWithEmoji(cfg.Join), fmt.Sprintf("feat:sys:toggle:%s:join", gid)),
			tgbotapi.NewInlineKeyboardButtonData("退群 "+onOffWithEmoji(cfg.Leave), fmt.Sprintf("feat:sys:toggle:%s:leave", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("置顶 "+onOffWithEmoji(cfg.Pin), fmt.Sprintf("feat:sys:toggle:%s:pin", gid)),
			tgbotapi.NewInlineKeyboardButtonData("头像 "+onOffWithEmoji(cfg.Photo), fmt.Sprintf("feat:sys:toggle:%s:photo", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("名称 "+onOffWithEmoji(cfg.Title), fmt.Sprintf("feat:sys:toggle:%s:title", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func antiFloodKeyboard(tgGroupID int64, view *service.AntiFloodView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("状态：✅启用", fmt.Sprintf("feat:mod:floodon:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("状态：❌关闭", fmt.Sprintf("feat:mod:floodoff:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("触发条数：%d", view.MaxMessages), fmt.Sprintf("feat:mod:floodcount:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("检测间隔：%d秒", view.WindowSec), fmt.Sprintf("feat:mod:floodwindow:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("惩罚：警告", fmt.Sprintf("feat:mod:floodpenalty:%s:warn", gid)),
			tgbotapi.NewInlineKeyboardButtonData("惩罚：禁言", fmt.Sprintf("feat:mod:floodpenalty:%s:mute", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("惩罚：踢出", fmt.Sprintf("feat:mod:floodpenalty:%s:kick", gid)),
			tgbotapi.NewInlineKeyboardButtonData("惩罚：踢出+封禁", fmt.Sprintf("feat:mod:floodpenalty:%s:kick_ban", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("惩罚：撤回+不处罚", fmt.Sprintf("feat:mod:floodpenalty:%s:delete_only", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("删除提醒："+antiFloodAlertDeleteText(view.WarnDeleteSec), fmt.Sprintf("feat:mod:floodalertdel:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:mod:floodview:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func verifyKeyboard(tgGroupID int64, view *service.JoinVerifyView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("状态：✅启用", fmt.Sprintf("feat:mod:verifyon:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("状态：❌关闭", fmt.Sprintf("feat:mod:verifyoff:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("验证时间：%d分钟", view.TimeoutMinutes), fmt.Sprintf("feat:mod:verifytime:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("超时处理："+verifyTimeoutActionLabel(view.TimeoutAction), fmt.Sprintf("feat:mod:verifytimeout:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("方式：按钮", fmt.Sprintf("feat:mod:verifymethod:%s:button", gid)),
			tgbotapi.NewInlineKeyboardButtonData("方式：数学题", fmt.Sprintf("feat:mod:verifymethod:%s:math", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("方式：验证码", fmt.Sprintf("feat:mod:verifymethod:%s:captcha", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:mod:verifyview:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func newbieLimitKeyboard(tgGroupID int64, view *service.NewbieLimitView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("状态：✅启用", fmt.Sprintf("feat:mod:newbieon:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("状态：❌关闭", fmt.Sprintf("feat:mod:newbieoff:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("限制时长：%d分钟", view.Minutes), fmt.Sprintf("feat:mod:newbietime:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:mod:newbieview:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func chainKeyboard(tgGroupID int64, active bool) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("新建接龙", fmt.Sprintf("feat:chain:start:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("添加条目", fmt.Sprintf("feat:chain:add:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("关闭接龙", fmt.Sprintf("feat:chain:close:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:chain:view:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	}
	_ = active
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func monitorKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("新增关键词", fmt.Sprintf("feat:monitor:add:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("移除关键词", fmt.Sprintf("feat:monitor:remove:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:monitor:view:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func pollKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("创建投票", fmt.Sprintf("feat:poll:create:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("结束投票", fmt.Sprintf("feat:poll:stop:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:poll:view:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func lotteryKeyboard(tgGroupID int64, publishPin bool, resultPin bool, deleteKeywordMins int) tgbotapi.InlineKeyboardMarkup {
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
			tgbotapi.NewInlineKeyboardButtonData("发布置顶 "+boolIcon(publishPin), fmt.Sprintf("feat:lottery:toggle:%s:publish_pin", gid)),
			tgbotapi.NewInlineKeyboardButtonData("结果置顶 "+boolIcon(resultPin), fmt.Sprintf("feat:lottery:toggle:%s:result_pin", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("删除口令 "+deleteText, fmt.Sprintf("feat:lottery:delmins:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:lottery:view:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func welcomeKeyboard(tgGroupID int64, enabled bool, mode string, deleteMinutes int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	statusText := "关闭"
	if enabled {
		statusText = "启用"
	}
	modeText := "验证后欢迎"
	if mode == "join" {
		modeText = "进群欢迎"
	}
	deleteText := "否"
	if deleteMinutes > 0 {
		deleteText = strconv.Itoa(deleteMinutes)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("状态："+statusText, fmt.Sprintf("feat:welcome:toggle:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("模式："+modeText, fmt.Sprintf("feat:welcome:mode:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("删除消息（分钟）："+deleteText, fmt.Sprintf("feat:welcome:delmins:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("修改文本", fmt.Sprintf("feat:welcome:set:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("修改媒体", fmt.Sprintf("feat:welcome:media:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("修改按钮", fmt.Sprintf("feat:welcome:button:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:welcome:view:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func rbacKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("设置角色", fmt.Sprintf("feat:rbac:setrole:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("设置功能权限", fmt.Sprintf("feat:rbac:setacl:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:rbac:view:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func blacklistKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("添加", fmt.Sprintf("feat:black:add:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("移除", fmt.Sprintf("feat:black:remove:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("刷新", fmt.Sprintf("feat:black:view:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
		),
	)
}

func settingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("中文", "feat:lang:set:0:zh"),
			tgbotapi.NewInlineKeyboardButtonData("English", "feat:lang:set:0:en"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("返回群组", cbMenuGroups),
		),
	)
}
