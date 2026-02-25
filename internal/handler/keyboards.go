package handler

import (
	"fmt"
	"strconv"
	"strings"

	"supervisor/internal/model"
	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func statusControlRow(enabled bool, labelData, onData, offData string) []tgbotapi.InlineKeyboardButton {
	onLabel := "启用"
	offLabel := "关闭"
	if enabled {
		onLabel = "✅启用"
	} else {
		offLabel = "✅关闭"
	}
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("状态：", labelData),
		tgbotapi.NewInlineKeyboardButtonData(onLabel, onData),
		tgbotapi.NewInlineKeyboardButtonData(offLabel, offData),
	)
}

func panelRefreshBackRow(gid string, refreshData string) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 刷新", refreshData),
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	)
}

func selectedLabel(label string, selected bool) string {
	if selected {
		return "✅" + label
	}
	return label
}

func mainMenuKeyboard(botUsername string) tgbotapi.InlineKeyboardMarkup {
	addToGroupURL := "https://t.me"
	if username := strings.TrimSpace(botUsername); username != "" {
		addToGroupURL = fmt.Sprintf("https://t.me/%s?startgroup=true", username)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 我的群组", cbMenuGroups),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ 设置", cbMenuSettings),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("➕ 拉机器人入群", addToGroupURL),
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
			tgbotapi.NewInlineKeyboardButtonData("📨 邀请链接", fmt.Sprintf("feat:invite:view:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("🧹 系统消息清理", fmt.Sprintf("feat:sys:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 反垃圾设置", fmt.Sprintf("feat:mod:spamview:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("⚡ 反刷屏设置", fmt.Sprintf("feat:mod:floodview:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧩 验证设置", fmt.Sprintf("feat:mod:verifyview:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("🔒 新成员限制设置", fmt.Sprintf("feat:mod:newbieview:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌙 夜间模式", fmt.Sprintf("feat:mod:nightview:%s", id)),
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

func inviteKeyboard(tgGroupID int64, enabled bool) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		statusControlRow(
			enabled,
			fmt.Sprintf("feat:invite:noop:%s", gid),
			fmt.Sprintf("feat:invite:on:%s", gid),
			fmt.Sprintf("feat:invite:off:%s", gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("配置过期时间", fmt.Sprintf("feat:invite:expire:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("最大邀请数配置", fmt.Sprintf("feat:invite:member:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("生成数量限制配置", fmt.Sprintf("feat:invite:gen:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("导出", fmt.Sprintf("feat:invite:export:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("清空数据", fmt.Sprintf("feat:invite:clear:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:invite:view:%s", gid)),
	)
}

func inviteExpireInputKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 无限制", fmt.Sprintf("feat:invite:expireunlimit:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func inviteMemberInputKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 无限制", fmt.Sprintf("feat:invite:memberunlimit:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func inviteGenerateInputKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 无限制", fmt.Sprintf("feat:invite:genunlimit:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func pendingCancelKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%d", tgGroupID)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%d", tgGroupID)),
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

func autoReplyMatchTypeKeyboard(tgGroupID int64, modeSelectPrefix string) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("精准触发", fmt.Sprintf("%s:exact", modeSelectPrefix)),
			tgbotapi.NewInlineKeyboardButtonData("包含触发", fmt.Sprintf("%s:contains", modeSelectPrefix)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func bannedWordListKeyboard(tgGroupID int64, view *service.BannedWordView, items []model.BannedWord, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+8)
	rows = append(rows,
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:bw:noop:%s", gid),
			fmt.Sprintf("feat:bw:on:%s", gid),
			fmt.Sprintf("feat:bw:off:%s", gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("惩罚设置", fmt.Sprintf("feat:bw:penalty:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("删除提醒："+bannedWordDeleteText(view.WarnDeleteMinutes), fmt.Sprintf("feat:bw:delwarninput:%s", gid)),
		),
	)
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

func bannedWordPenaltyKeyboard(tgGroupID int64, view *service.BannedWordView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	const (
		bwPenaltyWarn       = "warn"
		bwPenaltyMute       = "mute"
		bwPenaltyKick       = "kick"
		bwPenaltyKickBan    = "kick_ban"
		bwPenaltyDeleteOnly = "delete_only"
	)
	warnLabel := selectedLabel("警告", view.Penalty == bwPenaltyWarn)
	muteLabel := selectedLabel("禁言", view.Penalty == bwPenaltyMute)
	kickLabel := selectedLabel("踢出", view.Penalty == bwPenaltyKick)
	kickBanLabel := selectedLabel("踢出+封禁", view.Penalty == bwPenaltyKickBan)
	deleteOnlyLabel := selectedLabel("仅撤回", view.Penalty == bwPenaltyDeleteOnly)

	warnMuteLabel := selectedLabel("阈值后禁言", view.WarnAction == bwPenaltyMute)
	warnKickLabel := selectedLabel("阈值后踢出", view.WarnAction == bwPenaltyKick)
	warnKickBanLabel := selectedLabel("阈值后封禁", view.WarnAction == bwPenaltyKickBan)

	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(warnLabel, fmt.Sprintf("feat:bw:penaltyset:%s:%s", gid, bwPenaltyWarn)),
			tgbotapi.NewInlineKeyboardButtonData(muteLabel, fmt.Sprintf("feat:bw:penaltyset:%s:%s", gid, bwPenaltyMute)),
			tgbotapi.NewInlineKeyboardButtonData(kickLabel, fmt.Sprintf("feat:bw:penaltyset:%s:%s", gid, bwPenaltyKick)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(kickBanLabel, fmt.Sprintf("feat:bw:penaltyset:%s:%s", gid, bwPenaltyKickBan)),
			tgbotapi.NewInlineKeyboardButtonData(deleteOnlyLabel, fmt.Sprintf("feat:bw:penaltyset:%s:%s", gid, bwPenaltyDeleteOnly)),
		),
	}

	if view.Penalty == bwPenaltyWarn {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("警告次数：%d（输入）", view.WarnThreshold), fmt.Sprintf("feat:bw:warncount:%s", gid)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(warnMuteLabel, fmt.Sprintf("feat:bw:warnaction:%s:%s", gid, bwPenaltyMute)),
				tgbotapi.NewInlineKeyboardButtonData(warnKickLabel, fmt.Sprintf("feat:bw:warnaction:%s:%s", gid, bwPenaltyKick)),
				tgbotapi.NewInlineKeyboardButtonData(warnKickBanLabel, fmt.Sprintf("feat:bw:warnaction:%s:%s", gid, bwPenaltyKickBan)),
			),
		)
		if view.WarnAction == bwPenaltyMute {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("阈值禁言时长：%d分钟（输入）", view.WarnActionMuteMinutes), fmt.Sprintf("feat:bw:warnmuteinput:%s", gid)),
			))
		}
		if view.WarnAction == bwPenaltyKickBan {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("阈值封禁时长：%d分钟（输入）", view.WarnActionBanMinutes), fmt.Sprintf("feat:bw:warnbaninput:%s", gid)),
			))
		}
	}

	if view.Penalty == bwPenaltyMute {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("禁言时长：%d分钟（输入）", view.MuteMinutes), fmt.Sprintf("feat:bw:muteinput:%s", gid)),
		))
	}

	if view.Penalty == bwPenaltyKickBan {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("封禁时长：%d分钟（输入）", view.BanMinutes), fmt.Sprintf("feat:bw:baninput:%s", gid)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回违禁词面板", fmt.Sprintf("feat:bw:view:%s", gid)),
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
	allLabel := selectedLabel("全部", filter == "all")
	spamLabel := selectedLabel("审核", filter == "anti_spam*")
	verifyLabel := selectedLabel("验证", filter == "join_verify_pass")
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
		tgbotapi.NewInlineKeyboardButtonData(allLabel, fmt.Sprintf("feat:logs:list:%s:1:all", gid)),
		tgbotapi.NewInlineKeyboardButtonData(spamLabel, fmt.Sprintf("feat:logs:list:%s:1:anti_spam*", gid)),
		tgbotapi.NewInlineKeyboardButtonData(verifyLabel, fmt.Sprintf("feat:logs:list:%s:1:join_verify_pass", gid)),
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
	strictSelected := cfg.Join && cfg.Leave && cfg.Pin && cfg.Photo && cfg.Title
	offSelected := !cfg.Join && !cfg.Leave && !cfg.Pin && !cfg.Photo && !cfg.Title
	recommendedSelected := cfg.Join && cfg.Leave && !cfg.Pin && !cfg.Photo && !cfg.Title
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(selectedLabel("严格", strictSelected), fmt.Sprintf("feat:sys:preset:%s:strict", gid)),
			tgbotapi.NewInlineKeyboardButtonData(selectedLabel("推荐", recommendedSelected), fmt.Sprintf("feat:sys:preset:%s:recommended", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(selectedLabel("关闭", offSelected), fmt.Sprintf("feat:sys:preset:%s:off", gid)),
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
		panelRefreshBackRow(gid, fmt.Sprintf("feat:sys:view:%s", gid)),
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func antiFloodKeyboard(tgGroupID int64, view *service.AntiFloodView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	warnLabel := selectedLabel("惩罚：警告", view.Penalty == "warn")
	muteLabel := selectedLabel("惩罚：禁言", view.Penalty == "mute")
	kickLabel := selectedLabel("惩罚：踢出", view.Penalty == "kick")
	kickBanLabel := selectedLabel("惩罚：踢出+封禁", view.Penalty == "kick_ban")
	deleteOnlyLabel := selectedLabel("惩罚：撤回+不处罚", view.Penalty == "delete_only")
	return tgbotapi.NewInlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:floodon:%s", gid),
			fmt.Sprintf("feat:mod:floodoff:%s", gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("触发条数：%d", view.MaxMessages), fmt.Sprintf("feat:mod:floodcount:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("检测间隔：%d秒", view.WindowSec), fmt.Sprintf("feat:mod:floodwindow:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(warnLabel, fmt.Sprintf("feat:mod:floodpenalty:%s:warn", gid)),
			tgbotapi.NewInlineKeyboardButtonData(muteLabel, fmt.Sprintf("feat:mod:floodpenalty:%s:mute", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(kickLabel, fmt.Sprintf("feat:mod:floodpenalty:%s:kick", gid)),
			tgbotapi.NewInlineKeyboardButtonData(kickBanLabel, fmt.Sprintf("feat:mod:floodpenalty:%s:kick_ban", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(deleteOnlyLabel, fmt.Sprintf("feat:mod:floodpenalty:%s:delete_only", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("删除提醒："+antiFloodAlertDeleteText(view.WarnDeleteSec), fmt.Sprintf("feat:mod:floodalertdel:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:floodview:%s", gid)),
	)
}

func antiFloodAlertDeleteKeyboard(tgGroupID int64, currentSec int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	offLabel := selectedLabel("关闭", currentSec <= 0)
	sec5Label := selectedLabel("5秒", currentSec == 5)
	sec10Label := selectedLabel("10秒", currentSec == 10)
	sec20Label := selectedLabel("20秒", currentSec == 20)
	sec30Label := selectedLabel("30秒", currentSec == 30)
	sec60Label := selectedLabel("60秒", currentSec == 60)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(offLabel, fmt.Sprintf("feat:mod:floodalertset:%s:0", gid)),
			tgbotapi.NewInlineKeyboardButtonData(sec5Label, fmt.Sprintf("feat:mod:floodalertset:%s:5", gid)),
			tgbotapi.NewInlineKeyboardButtonData(sec10Label, fmt.Sprintf("feat:mod:floodalertset:%s:10", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(sec20Label, fmt.Sprintf("feat:mod:floodalertset:%s:20", gid)),
			tgbotapi.NewInlineKeyboardButtonData(sec30Label, fmt.Sprintf("feat:mod:floodalertset:%s:30", gid)),
			tgbotapi.NewInlineKeyboardButtonData(sec60Label, fmt.Sprintf("feat:mod:floodalertset:%s:60", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回反刷屏面板", fmt.Sprintf("feat:mod:floodview:%s", gid)),
		),
	)
}

func antiFloodCountKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	c3Label := selectedLabel("3条", current == 3)
	c5Label := selectedLabel("5条", current == 5)
	c8Label := selectedLabel("8条", current == 8)
	c10Label := selectedLabel("10条", current == 10)
	c15Label := selectedLabel("15条", current == 15)
	c20Label := selectedLabel("20条", current == 20)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c3Label, fmt.Sprintf("feat:mod:floodcountset:%s:3", gid)),
			tgbotapi.NewInlineKeyboardButtonData(c5Label, fmt.Sprintf("feat:mod:floodcountset:%s:5", gid)),
			tgbotapi.NewInlineKeyboardButtonData(c8Label, fmt.Sprintf("feat:mod:floodcountset:%s:8", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c10Label, fmt.Sprintf("feat:mod:floodcountset:%s:10", gid)),
			tgbotapi.NewInlineKeyboardButtonData(c15Label, fmt.Sprintf("feat:mod:floodcountset:%s:15", gid)),
			tgbotapi.NewInlineKeyboardButtonData(c20Label, fmt.Sprintf("feat:mod:floodcountset:%s:20", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回反刷屏面板", fmt.Sprintf("feat:mod:floodview:%s", gid)),
		),
	)
}

func antiFloodWindowKeyboard(tgGroupID int64, currentSec int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	sec3Label := selectedLabel("3秒", currentSec == 3)
	sec5Label := selectedLabel("5秒", currentSec == 5)
	sec10Label := selectedLabel("10秒", currentSec == 10)
	sec15Label := selectedLabel("15秒", currentSec == 15)
	sec20Label := selectedLabel("20秒", currentSec == 20)
	sec30Label := selectedLabel("30秒", currentSec == 30)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(sec3Label, fmt.Sprintf("feat:mod:floodwindowset:%s:3", gid)),
			tgbotapi.NewInlineKeyboardButtonData(sec5Label, fmt.Sprintf("feat:mod:floodwindowset:%s:5", gid)),
			tgbotapi.NewInlineKeyboardButtonData(sec10Label, fmt.Sprintf("feat:mod:floodwindowset:%s:10", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(sec15Label, fmt.Sprintf("feat:mod:floodwindowset:%s:15", gid)),
			tgbotapi.NewInlineKeyboardButtonData(sec20Label, fmt.Sprintf("feat:mod:floodwindowset:%s:20", gid)),
			tgbotapi.NewInlineKeyboardButtonData(sec30Label, fmt.Sprintf("feat:mod:floodwindowset:%s:30", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回反刷屏面板", fmt.Sprintf("feat:mod:floodview:%s", gid)),
		),
	)
}

func antiSpamKeyboard(tgGroupID int64, view *service.AntiSpamView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	warnLabel := selectedLabel("惩罚：警告", view.Penalty == "warn")
	muteLabel := selectedLabel("惩罚：禁言", view.Penalty == "mute")
	kickLabel := selectedLabel("惩罚：踢出", view.Penalty == "kick")
	kickBanLabel := selectedLabel("惩罚：踢出+封禁", view.Penalty == "kick_ban")
	deleteOnlyLabel := selectedLabel("惩罚：撤回+不处罚", view.Penalty == "delete_only")
	warnDeleteOffLabel := selectedLabel("关闭", view.WarnDeleteSec <= 0)
	warnDelete5Label := selectedLabel("5秒", view.WarnDeleteSec == 5)
	warnDelete10Label := selectedLabel("10秒", view.WarnDeleteSec == 10)
	warnDelete20Label := selectedLabel("20秒", view.WarnDeleteSec == 20)
	warnDelete30Label := selectedLabel("30秒", view.WarnDeleteSec == 30)
	warnDelete60Label := selectedLabel("60秒", view.WarnDeleteSec == 60)
	return tgbotapi.NewInlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:spamon:%s", gid),
			fmt.Sprintf("feat:mod:spamoff:%s", gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(warnLabel, fmt.Sprintf("feat:mod:spampenalty:%s:warn", gid)),
			tgbotapi.NewInlineKeyboardButtonData(muteLabel, fmt.Sprintf("feat:mod:spampenalty:%s:mute", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(kickLabel, fmt.Sprintf("feat:mod:spampenalty:%s:kick", gid)),
			tgbotapi.NewInlineKeyboardButtonData(kickBanLabel, fmt.Sprintf("feat:mod:spampenalty:%s:kick_ban", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(deleteOnlyLabel, fmt.Sprintf("feat:mod:spampenalty:%s:delete_only", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("屏蔽图片 "+onOffWithEmoji(view.BlockPhoto), fmt.Sprintf("feat:mod:spamopt:%s:photo", gid)),
			tgbotapi.NewInlineKeyboardButtonData("屏蔽链接 "+onOffWithEmoji(view.BlockLink), fmt.Sprintf("feat:mod:spamopt:%s:link", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("屏蔽频道马甲 "+onOffWithEmoji(view.BlockChannelAlias), fmt.Sprintf("feat:mod:spamopt:%s:senderchat", gid)),
			tgbotapi.NewInlineKeyboardButtonData("屏蔽频道转发 "+onOffWithEmoji(view.BlockForwardFromChan), fmt.Sprintf("feat:mod:spamopt:%s:fwdchan", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("屏蔽用户转发 "+onOffWithEmoji(view.BlockForwardFromUser), fmt.Sprintf("feat:mod:spamopt:%s:fwduser", gid)),
			tgbotapi.NewInlineKeyboardButtonData("屏蔽@群组ID "+onOffWithEmoji(view.BlockAtGroupID), fmt.Sprintf("feat:mod:spamopt:%s:atgroup", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("屏蔽@用户ID "+onOffWithEmoji(view.BlockAtUserID), fmt.Sprintf("feat:mod:spamopt:%s:atuser", gid)),
			tgbotapi.NewInlineKeyboardButtonData("屏蔽ETH地址 "+onOffWithEmoji(view.BlockEthAddress), fmt.Sprintf("feat:mod:spamopt:%s:eth", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("屏蔽超长消息 "+onOffWithEmoji(view.BlockLongMessage), fmt.Sprintf("feat:mod:spamopt:%s:longmsg", gid)),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("消息长度:%d", view.MaxMessageLength), fmt.Sprintf("feat:mod:spammsglen:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("屏蔽超长姓名 "+onOffWithEmoji(view.BlockLongName), fmt.Sprintf("feat:mod:spamopt:%s:longname", gid)),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("姓名长度:%d", view.MaxNameLength), fmt.Sprintf("feat:mod:spamnamelen:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("例外+（%d）", view.ExceptionKeywordCount), fmt.Sprintf("feat:mod:spamexadd:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("例外-（按关键词）", fmt.Sprintf("feat:mod:spamexdel:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("删除提醒：", fmt.Sprintf("feat:mod:noop:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData(warnDeleteOffLabel, fmt.Sprintf("feat:mod:spamalertdel:%s:0", gid)),
			tgbotapi.NewInlineKeyboardButtonData(warnDelete5Label, fmt.Sprintf("feat:mod:spamalertdel:%s:5", gid)),
			tgbotapi.NewInlineKeyboardButtonData(warnDelete10Label, fmt.Sprintf("feat:mod:spamalertdel:%s:10", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(warnDelete20Label, fmt.Sprintf("feat:mod:spamalertdel:%s:20", gid)),
			tgbotapi.NewInlineKeyboardButtonData(warnDelete30Label, fmt.Sprintf("feat:mod:spamalertdel:%s:30", gid)),
			tgbotapi.NewInlineKeyboardButtonData(warnDelete60Label, fmt.Sprintf("feat:mod:spamalertdel:%s:60", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:spamview:%s", gid)),
	)
}

func verifyKeyboard(tgGroupID int64, view *service.JoinVerifyView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	buttonLabel := "方式：按钮"
	mathLabel := "方式：数学题"
	captchaLabel := "方式：验证码"
	zhcharLabel := "方式：中文字符验证码"
	switch view.Type {
	case "button":
		buttonLabel = "✅" + buttonLabel
	case "math":
		mathLabel = "✅" + mathLabel
	case "captcha":
		captchaLabel = "✅" + captchaLabel
	case "zhchar":
		zhcharLabel = "✅" + zhcharLabel
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:verifyon:%s", gid),
			fmt.Sprintf("feat:mod:verifyoff:%s", gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("验证时间：%d分钟", view.TimeoutMinutes), fmt.Sprintf("feat:mod:verifytime:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("超时处理："+verifyTimeoutActionLabel(view.TimeoutAction), fmt.Sprintf("feat:mod:verifytimeout:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonLabel, fmt.Sprintf("feat:mod:verifymethod:%s:button", gid)),
			tgbotapi.NewInlineKeyboardButtonData(mathLabel, fmt.Sprintf("feat:mod:verifymethod:%s:math", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(captchaLabel, fmt.Sprintf("feat:mod:verifymethod:%s:captcha", gid)),
			tgbotapi.NewInlineKeyboardButtonData(zhcharLabel, fmt.Sprintf("feat:mod:verifymethod:%s:zhchar", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:verifyview:%s", gid)),
	)
}

func newbieLimitKeyboard(tgGroupID int64, view *service.NewbieLimitView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:newbieon:%s", gid),
			fmt.Sprintf("feat:mod:newbieoff:%s", gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("限制时长：%d分钟", view.Minutes), fmt.Sprintf("feat:mod:newbietime:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:newbieview:%s", gid)),
	)
}

func verifyTimeoutMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	m1Label := selectedLabel("1分钟", current == 1)
	m5Label := selectedLabel("5分钟", current == 5)
	m10Label := selectedLabel("10分钟", current == 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(m1Label, fmt.Sprintf("feat:mod:verifytimeset:%s:1", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m5Label, fmt.Sprintf("feat:mod:verifytimeset:%s:5", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m10Label, fmt.Sprintf("feat:mod:verifytimeset:%s:10", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回验证面板", fmt.Sprintf("feat:mod:verifyview:%s", gid)),
		),
	)
}

func newbieLimitMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	m10Label := selectedLabel("10分钟", current == 10)
	m30Label := selectedLabel("30分钟", current == 30)
	m60Label := selectedLabel("60分钟", current == 60)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(m10Label, fmt.Sprintf("feat:mod:newbietimeset:%s:10", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m30Label, fmt.Sprintf("feat:mod:newbietimeset:%s:30", gid)),
			tgbotapi.NewInlineKeyboardButtonData(m60Label, fmt.Sprintf("feat:mod:newbietimeset:%s:60", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回新成员限制面板", fmt.Sprintf("feat:mod:newbieview:%s", gid)),
		),
	)
}

func nightModeKeyboard(tgGroupID int64, view *service.NightModeView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	deleteMediaLabel := "删除媒体"
	globalMuteLabel := "全局禁言"
	if view.Mode == "global_mute" {
		globalMuteLabel = "✅全局禁言"
	} else {
		deleteMediaLabel = "✅删除媒体"
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:nighton:%s", gid),
			fmt.Sprintf("feat:mod:nightoff:%s", gid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("时区："+view.TimezoneText, fmt.Sprintf("feat:mod:nighttz:%s", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("处理方式：", fmt.Sprintf("feat:mod:noop:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData(deleteMediaLabel, fmt.Sprintf("feat:mod:nightmode:%s:delete_media", gid)),
			tgbotapi.NewInlineKeyboardButtonData(globalMuteLabel, fmt.Sprintf("feat:mod:nightmode:%s:global_mute", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:nightview:%s", gid)),
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
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:chain:view:%s", gid)),
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
		panelRefreshBackRow(gid, fmt.Sprintf("feat:monitor:view:%s", gid)),
	)
}

func pollKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("创建投票", fmt.Sprintf("feat:poll:create:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("结束投票", fmt.Sprintf("feat:poll:stop:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:poll:view:%s", gid)),
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

func lotteryRecordsKeyboard(tgGroupID int64, items []service.LotteryRecordItem, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
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

func lotteryDeleteMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
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

func welcomeKeyboard(tgGroupID int64, enabled bool, mode string, deleteMinutes int) tgbotapi.InlineKeyboardMarkup {
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

func welcomeDeleteMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
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

func rbacKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("设置角色", fmt.Sprintf("feat:rbac:setrole:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("设置功能权限", fmt.Sprintf("feat:rbac:setacl:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:rbac:view:%s", gid)),
	)
}

func blacklistKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("添加", fmt.Sprintf("feat:black:add:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("移除", fmt.Sprintf("feat:black:remove:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:black:view:%s", gid)),
	)
}

func settingsKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	zhLabel := selectedLabel("中文", lang == "zh")
	enLabel := selectedLabel("English", lang == "en")
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(zhLabel, "feat:lang:set:0:zh"),
			tgbotapi.NewInlineKeyboardButtonData(enLabel, "feat:lang:set:0:en"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("返回群组", cbMenuGroups),
		),
	)
}
