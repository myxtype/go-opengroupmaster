package keyboards

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

const (
	cbMenuGroups   = "menu:groups"
	cbMenuSettings = "menu:settings"
	cbGroupPrefix  = "group:"
	cbGroupsPagePF = "menu:groups:page:"
)

func inlineKeyboardButtonData(text, data string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         text,
		CallbackData: data,
	}
}

func inlineKeyboardButtonURL(text, link string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text: text,
		URL:  link,
	}
}

func inlineKeyboardRow(buttons ...models.InlineKeyboardButton) []models.InlineKeyboardButton {
	return buttons
}

func inlineKeyboardMarkup(rows ...[]models.InlineKeyboardButton) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func statusControlRow(enabled bool, labelData, onData, offData string) []models.InlineKeyboardButton {
	onLabel := "启用"
	offLabel := "关闭"
	if enabled {
		onLabel = "✅启用"
	} else {
		offLabel = "✅关闭"
	}
	return inlineKeyboardRow(
		inlineKeyboardButtonData("状态：", labelData),
		inlineKeyboardButtonData(onLabel, onData),
		inlineKeyboardButtonData(offLabel, offData),
	)
}

func panelRefreshBackRow(gid string, refreshData string) []models.InlineKeyboardButton {
	return inlineKeyboardRow(
		inlineKeyboardButtonData("🔄 刷新", refreshData),
		inlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	)
}

func selectedLabel(label string, selected bool) string {
	if selected {
		return "✅" + label
	}
	return label
}

func onOffWithEmoji(v bool) string {
	if v {
		return "✅"
	}
	return "❌"
}

func boolIcon(v bool) string {
	if v {
		return "✅"
	}
	return "❌"
}

func bannedWordDeleteText(minutes int) string {
	if minutes <= 0 {
		return "关闭"
	}
	return fmt.Sprintf("%d分钟", minutes)
}

func antiFloodAlertDeleteText(seconds int) string {
	if seconds <= 0 {
		return "不自动删除"
	}
	return fmt.Sprintf("%d秒", seconds)
}

func verifyTimeoutActionLabel(v string) string {
	if v == "kick" {
		return "踢出"
	}
	return "永久禁言"
}

func moderationPenaltyRows(gid, scope string, penalty string, warnThreshold int, warnAction string, warnActionMuteMinutes int, warnActionBanMinutes int, muteMinutes int, banMinutes int) [][]models.InlineKeyboardButton {
	return moderationPenaltyRowsWithSpec(
		gid,
		penalty,
		warnThreshold,
		warnAction,
		warnActionMuteMinutes,
		warnActionBanMinutes,
		muteMinutes,
		banMinutes,
		moderationPenaltyRowSpec{
			PenaltySet:    fmt.Sprintf("feat:mod:%spenalty:%%s:%%s", scope),
			WarnCount:     fmt.Sprintf("feat:mod:%swarncount:%%s", scope),
			WarnAction:    fmt.Sprintf("feat:mod:%swarnaction:%%s:%%s", scope),
			WarnMuteInput: fmt.Sprintf("feat:mod:%swarnmuteinput:%%s", scope),
			WarnBanInput:  fmt.Sprintf("feat:mod:%swarnbaninput:%%s", scope),
			MuteInput:     fmt.Sprintf("feat:mod:%smuteinput:%%s", scope),
			BanInput:      fmt.Sprintf("feat:mod:%sbaninput:%%s", scope),
		},
	)
}

type moderationPenaltyRowSpec struct {
	PenaltySet    string
	WarnCount     string
	WarnAction    string
	WarnMuteInput string
	WarnBanInput  string
	MuteInput     string
	BanInput      string
}

func moderationPenaltyRowsWithSpec(
	gid string,
	penalty string,
	warnThreshold int,
	warnAction string,
	warnActionMuteMinutes int,
	warnActionBanMinutes int,
	muteMinutes int,
	banMinutes int,
	spec moderationPenaltyRowSpec,
) [][]models.InlineKeyboardButton {
	const (
		penaltyWarn       = "warn"
		penaltyMute       = "mute"
		penaltyKick       = "kick"
		penaltyKickBan    = "kick_ban"
		penaltyDeleteOnly = "delete_only"
	)

	warnLabel := selectedLabel("警告", penalty == penaltyWarn)
	muteLabel := selectedLabel("禁言", penalty == penaltyMute)
	kickLabel := selectedLabel("踢出", penalty == penaltyKick)
	kickBanLabel := selectedLabel("踢出+封禁", penalty == penaltyKickBan)
	deleteOnlyLabel := selectedLabel("仅撤回", penalty == penaltyDeleteOnly)
	warnMuteLabel := selectedLabel("阈值后禁言", warnAction == penaltyMute)
	warnKickLabel := selectedLabel("阈值后踢出", warnAction == penaltyKick)
	warnKickBanLabel := selectedLabel("阈值后封禁", warnAction == penaltyKickBan)

	rows := [][]models.InlineKeyboardButton{
		inlineKeyboardRow(
			inlineKeyboardButtonData(warnLabel, fmt.Sprintf(spec.PenaltySet, gid, penaltyWarn)),
			inlineKeyboardButtonData(muteLabel, fmt.Sprintf(spec.PenaltySet, gid, penaltyMute)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(kickLabel, fmt.Sprintf(spec.PenaltySet, gid, penaltyKick)),
			inlineKeyboardButtonData(kickBanLabel, fmt.Sprintf(spec.PenaltySet, gid, penaltyKickBan)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(deleteOnlyLabel, fmt.Sprintf(spec.PenaltySet, gid, penaltyDeleteOnly)),
		),
	}

	if penalty == penaltyWarn {
		rows = append(rows,
			inlineKeyboardRow(
				inlineKeyboardButtonData(fmt.Sprintf("警告次数：%d（输入）", warnThreshold), fmt.Sprintf(spec.WarnCount, gid)),
			),
			inlineKeyboardRow(
				inlineKeyboardButtonData(warnMuteLabel, fmt.Sprintf(spec.WarnAction, gid, penaltyMute)),
				inlineKeyboardButtonData(warnKickLabel, fmt.Sprintf(spec.WarnAction, gid, penaltyKick)),
				inlineKeyboardButtonData(warnKickBanLabel, fmt.Sprintf(spec.WarnAction, gid, penaltyKickBan)),
			),
		)
		if warnAction == penaltyMute {
			rows = append(rows, inlineKeyboardRow(
				inlineKeyboardButtonData(fmt.Sprintf("阈值禁言时长：%d分钟（输入）", warnActionMuteMinutes), fmt.Sprintf(spec.WarnMuteInput, gid)),
			))
		}
		if warnAction == penaltyKickBan {
			rows = append(rows, inlineKeyboardRow(
				inlineKeyboardButtonData(fmt.Sprintf("阈值封禁时长：%d分钟（输入）", warnActionBanMinutes), fmt.Sprintf(spec.WarnBanInput, gid)),
			))
		}
	}

	if penalty == penaltyMute {
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("禁言时长：%d分钟（输入）", muteMinutes), fmt.Sprintf(spec.MuteInput, gid)),
		))
	}

	if penalty == penaltyKickBan {
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("封禁时长：%d分钟（输入）", banMinutes), fmt.Sprintf(spec.BanInput, gid)),
		))
	}
	return rows
}
