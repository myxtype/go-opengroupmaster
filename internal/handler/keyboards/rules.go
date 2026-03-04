package keyboards

import (
	"fmt"
	"strconv"

	"supervisor/internal/model"
	"supervisor/internal/service"

	"github.com/go-telegram/bot/models"
)

func InviteKeyboard(tgGroupID int64, enabled bool) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		statusControlRow(
			enabled,
			fmt.Sprintf("feat:invite:noop:%s", gid),
			fmt.Sprintf("feat:invite:on:%s", gid),
			fmt.Sprintf("feat:invite:off:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("配置过期时间", fmt.Sprintf("feat:invite:expire:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("最大邀请数配置", fmt.Sprintf("feat:invite:member:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("生成数量限制配置", fmt.Sprintf("feat:invite:gen:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("导出", fmt.Sprintf("feat:invite:export:%s", gid)),
			inlineKeyboardButtonData("清空数据", fmt.Sprintf("feat:invite:clear:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:invite:view:%s", gid)),
	)
}

func InviteExpireInputKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("🚫 无限制", fmt.Sprintf("feat:invite:expireunlimit:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			inlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func InviteMemberInputKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("🚫 无限制", fmt.Sprintf("feat:invite:memberunlimit:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			inlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func InviteGenerateInputKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("🚫 无限制", fmt.Sprintf("feat:invite:genunlimit:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			inlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func AutoReplyListKeyboard(tgGroupID int64, items []model.AutoReply, page, totalPages int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]models.InlineKeyboardButton, 0, len(items)+4)
	for _, item := range items {
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData(
				fmt.Sprintf("✏️ 编辑 #%d", item.ID),
				fmt.Sprintf("feat:auto:edit:%s:%d:%d", gid, item.ID, page),
			),
			inlineKeyboardButtonData(
				fmt.Sprintf("🗑 删除 #%d", item.ID),
				fmt.Sprintf("feat:auto:del:%s:%d:%d", gid, item.ID, page),
			),
		))
	}
	nav := make([]models.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, inlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:auto:list:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, inlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:auto:list:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, inlineKeyboardRow(nav...))
	}
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("➕ 新增自动回复", fmt.Sprintf("feat:auto:add:%s", gid)),
		inlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return inlineKeyboardMarkup(rows...)
}

func AutoReplyMatchTypeKeyboard(tgGroupID int64, modeSelectPrefix string) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("精准触发", fmt.Sprintf("%s:exact", modeSelectPrefix)),
			inlineKeyboardButtonData("包含触发", fmt.Sprintf("%s:contains", modeSelectPrefix)),
			inlineKeyboardButtonData("正则触发", fmt.Sprintf("%s:regex", modeSelectPrefix)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			inlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func BannedWordListKeyboard(tgGroupID int64, view *service.BannedWordView, items []model.BannedWord, page, totalPages int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]models.InlineKeyboardButton, 0, len(items)+8)
	rows = append(rows,
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:bw:noop:%s", gid),
			fmt.Sprintf("feat:bw:on:%s", gid),
			fmt.Sprintf("feat:bw:off:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("惩罚设置", fmt.Sprintf("feat:bw:penalty:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("删除提醒："+bannedWordDeleteText(view.WarnDeleteMinutes), fmt.Sprintf("feat:bw:delwarninput:%s", gid)),
		),
	)
	nav := make([]models.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, inlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:bw:list:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, inlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:bw:list:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, inlineKeyboardRow(nav...))
	}
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("➕ 新增违禁词", fmt.Sprintf("feat:bw:add:%s", gid)),
		inlineKeyboardButtonData("🗑 批量删除", fmt.Sprintf("feat:bw:remove:%s", gid)),
	))
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return inlineKeyboardMarkup(rows...)
}

func BannedWordPenaltyKeyboard(tgGroupID int64, view *service.BannedWordView) models.InlineKeyboardMarkup {
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
			PenaltySet:    "feat:bw:penaltyset:%s:%s",
			WarnCount:     "feat:bw:warncount:%s",
			WarnAction:    "feat:bw:warnaction:%s:%s",
			WarnMuteInput: "feat:bw:warnmuteinput:%s",
			WarnBanInput:  "feat:bw:warnbaninput:%s",
			MuteInput:     "feat:bw:muteinput:%s",
			BanInput:      "feat:bw:baninput:%s",
		},
	)

	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("◀ 返回违禁词面板", fmt.Sprintf("feat:bw:view:%s", gid)),
	))
	return inlineKeyboardMarkup(rows...)
}
