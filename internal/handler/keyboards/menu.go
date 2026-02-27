package keyboards

import (
	"fmt"
	"strconv"
	"strings"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func MainMenuKeyboard(botUsername string) tgbotapi.InlineKeyboardMarkup {
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

func GroupsKeyboard(groups []model.Group, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
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

func GroupPanelKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
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
			tgbotapi.NewInlineKeyboardButtonData("💰 积分系统", fmt.Sprintf("feat:points:view:%s", id)),
			tgbotapi.NewInlineKeyboardButtonData("📊 数据统计", fmt.Sprintf("feat:stats:show:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📜 管理日志", fmt.Sprintf("feat:logs:list:%s:1:all", id)),
			tgbotapi.NewInlineKeyboardButtonData("📨 邀请链接", fmt.Sprintf("feat:invite:view:%s", id)),
		),
		tgbotapi.NewInlineKeyboardRow(
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

func PendingCancelKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%d", tgGroupID)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%d", tgGroupID)),
		),
	)
}

func RBACKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("设置角色", fmt.Sprintf("feat:rbac:setrole:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("设置功能权限", fmt.Sprintf("feat:rbac:setacl:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:rbac:view:%s", gid)),
	)
}

func BlacklistKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("添加", fmt.Sprintf("feat:black:add:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("移除", fmt.Sprintf("feat:black:remove:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:black:view:%s", gid)),
	)
}

func SettingsKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
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
