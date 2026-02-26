package keyboards

import (
	"fmt"
	"strconv"

	"supervisor/internal/model"
	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func InviteKeyboard(tgGroupID int64, enabled bool) tgbotapi.InlineKeyboardMarkup {
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

func InviteExpireInputKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
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

func InviteMemberInputKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
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

func InviteGenerateInputKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
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

func AutoReplyListKeyboard(tgGroupID int64, items []model.AutoReply, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
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

func AutoReplyMatchTypeKeyboard(tgGroupID int64, modeSelectPrefix string) tgbotapi.InlineKeyboardMarkup {
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

func BannedWordListKeyboard(tgGroupID int64, view *service.BannedWordView, items []model.BannedWord, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
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

func BannedWordPenaltyKeyboard(tgGroupID int64, view *service.BannedWordView) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := moderationPenaltyRowsWithSpec(
		gid,
		view.Penalty,
		view.WarnThreshold,
		view.WarnAction,
		view.WarnActionMuteMinutes,
		view.WarnActionBanMinutes,
		view.MuteMinutes,
		view.BanMinutes,
		moderationPenaltyRowSpec{
			WarnLabelPrefix: false,
			DeleteOnlyLabel: "仅撤回",
			PenaltySet:      "feat:bw:penaltyset:%s:%s",
			WarnCount:       "feat:bw:warncount:%s",
			WarnAction:      "feat:bw:warnaction:%s:%s",
			WarnMuteInput:   "feat:bw:warnmuteinput:%s",
			WarnBanInput:    "feat:bw:warnbaninput:%s",
			MuteInput:       "feat:bw:muteinput:%s",
			BanInput:        "feat:bw:baninput:%s",
		},
	)

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀ 返回违禁词面板", fmt.Sprintf("feat:bw:view:%s", gid)),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
