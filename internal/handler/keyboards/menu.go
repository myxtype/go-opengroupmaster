package keyboards

import (
	"fmt"
	"strconv"
	"strings"

	"supervisor/internal/model"

	"github.com/go-telegram/bot/models"
)

func MainMenuKeyboard(botUsername string) models.InlineKeyboardMarkup {
	addToGroupURL := "https://t.me"
	if username := strings.TrimSpace(botUsername); username != "" {
		addToGroupURL = fmt.Sprintf("https://t.me/%s?startgroup=true", username)
	}
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("📊 我的群组", cbMenuGroups),
			inlineKeyboardButtonData("⚙️ 设置", cbMenuSettings),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonURL("➕ 拉机器人入群", addToGroupURL),
		),
	)
}

func GroupsKeyboard(groups []model.Group, page, totalPages int) models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0, len(groups)+3)
	for _, g := range groups {
		label := g.Title
		if label == "" {
			label = strconv.FormatInt(g.TGGroupID, 10)
		}
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData("🗂 "+label, cbGroupPrefix+strconv.FormatInt(g.TGGroupID, 10)),
		))
	}

	nav := make([]models.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, inlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("%s%d", cbGroupsPagePF, page-1)))
	}
	if page < totalPages {
		nav = append(nav, inlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("%s%d", cbGroupsPagePF, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, inlineKeyboardRow(nav...))
	}
	rows = append(rows, inlineKeyboardRow(inlineKeyboardButtonData("🔄 刷新", cbMenuGroups)))
	return inlineKeyboardMarkup(rows...)
}

func GroupPanelKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	return GroupPanelKeyboardWithWordCloud(tgGroupID, true)
}

func GroupPanelKeyboardWithWordCloud(tgGroupID int64, wordCloudAvailable bool) models.InlineKeyboardMarkup {
	id := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]models.InlineKeyboardButton, 0, 16)
	rows = append(rows,
		inlineKeyboardRow(
			inlineKeyboardButtonData("🤖 自动回复", fmt.Sprintf("feat:auto:view:%s", id)),
			inlineKeyboardButtonData("🚫 违禁词", fmt.Sprintf("feat:bw:view:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("👋 欢迎设置", fmt.Sprintf("feat:welcome:view:%s", id)),
			inlineKeyboardButtonData("🎯 抽奖", fmt.Sprintf("feat:lottery:view:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🗳 投票", fmt.Sprintf("feat:poll:view:%s", id)),
			inlineKeyboardButtonData("📋 接龙", fmt.Sprintf("feat:chain:view:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("👁 关键词监控", fmt.Sprintf("feat:monitor:view:%s", id)),
			inlineKeyboardButtonData("⏰ 定时消息", fmt.Sprintf("feat:sched:view:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("💰 积分系统", fmt.Sprintf("feat:points:view:%s", id)),
			inlineKeyboardButtonData("📊 数据统计", fmt.Sprintf("feat:stats:show:%s", id)),
		),
	)
	if wordCloudAvailable {
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData("☁️ 词云", fmt.Sprintf("feat:wc:view:%s", id)),
		))
	}
	rows = append(rows,
		inlineKeyboardRow(
			inlineKeyboardButtonData("📜 管理日志", fmt.Sprintf("feat:logs:list:%s:1:all", id)),
			inlineKeyboardButtonData("📨 邀请链接", fmt.Sprintf("feat:invite:view:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🧹 系统消息清理", fmt.Sprintf("feat:sys:view:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🚫 反垃圾设置", fmt.Sprintf("feat:mod:spamview:%s", id)),
			inlineKeyboardButtonData("⚡ 反刷屏设置", fmt.Sprintf("feat:mod:floodview:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🧩 验证设置", fmt.Sprintf("feat:mod:verifyview:%s", id)),
			inlineKeyboardButtonData("🔒 新成员限制设置", fmt.Sprintf("feat:mod:newbieview:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🌙 夜间模式", fmt.Sprintf("feat:mod:nightview:%s", id)),
			inlineKeyboardButtonData("🌐 群时区", fmt.Sprintf("feat:mod:grouptz:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🧭 权限分级", fmt.Sprintf("feat:rbac:view:%s", id)),
			inlineKeyboardButtonData("⛔ 黑名单", fmt.Sprintf("feat:black:view:%s", id)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群组列表", cbMenuGroups),
		),
	)
	return inlineKeyboardMarkup(rows...)
}

func PendingCancelKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%d", tgGroupID)),
			inlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%d", tgGroupID)),
		),
	)
}

func RBACKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("设置角色", fmt.Sprintf("feat:rbac:setrole:%s", gid)),
			inlineKeyboardButtonData("设置功能权限", fmt.Sprintf("feat:rbac:setacl:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:rbac:view:%s", gid)),
	)
}

func BlacklistKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("添加", fmt.Sprintf("feat:black:add:%s", gid)),
			inlineKeyboardButtonData("移除", fmt.Sprintf("feat:black:remove:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:black:view:%s", gid)),
	)
}

func SettingsKeyboard(lang string) models.InlineKeyboardMarkup {
	zhLabel := selectedLabel("中文", lang == "zh")
	enLabel := selectedLabel("English", lang == "en")
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(zhLabel, "feat:lang:set:0:zh"),
			inlineKeyboardButtonData(enLabel, "feat:lang:set:0:en"),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("返回群组", cbMenuGroups),
		),
	)
}
