package keyboards

import (
	"fmt"
	"strconv"

	"supervisor/internal/model"
	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ScheduledListKeyboard(tgGroupID int64, items []model.ScheduledMessage, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(items)+5)
	for _, item := range items {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("✏️ 编辑 #%d", item.ID),
				fmt.Sprintf("feat:sched:edit:%s:%d:%d", gid, item.ID, page),
			),
		))
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
		pinText := "置顶 ❌"
		if item.PinMessage {
			pinText = "置顶 ✅"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s #%d", pinText, item.ID),
				fmt.Sprintf("feat:sched:pin:%s:%d:%d", gid, item.ID, page),
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

func ScheduledPinSelectKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("置顶", fmt.Sprintf("feat:sched:pinset:%s:on", gid)),
			tgbotapi.NewInlineKeyboardButtonData("不置顶", fmt.Sprintf("feat:sched:pinset:%s:off", gid)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			tgbotapi.NewInlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func ScheduledEditKeyboard(tgGroupID int64, id uint, page int, enabled bool, pin bool) tgbotapi.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	statusText := "启用"
	if enabled {
		statusText = "关闭"
	}
	pinText := "置顶 ❌"
	if pin {
		pinText = "置顶 ✅"
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("状态："+statusText, fmt.Sprintf("feat:sched:edittoggle:%s:%d:%d", gid, id, page)),
			tgbotapi.NewInlineKeyboardButtonData(pinText, fmt.Sprintf("feat:sched:editpin:%s:%d:%d", gid, id, page)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("修改文本", fmt.Sprintf("feat:sched:edittext:%s:%d:%d", gid, id, page)),
			tgbotapi.NewInlineKeyboardButtonData("修改媒体", fmt.Sprintf("feat:sched:editmedia:%s:%d:%d", gid, id, page)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("修改按钮", fmt.Sprintf("feat:sched:editbuttons:%s:%d:%d", gid, id, page)),
			tgbotapi.NewInlineKeyboardButtonData("修改 Cron", fmt.Sprintf("feat:sched:editcron:%s:%d:%d", gid, id, page)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ 返回定时列表", fmt.Sprintf("feat:sched:list:%s:%d", gid, page)),
		),
	)
}

func LogListKeyboard(tgGroupID int64, page, totalPages int, filter string) tgbotapi.InlineKeyboardMarkup {
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

func SystemCleanKeyboard(tgGroupID int64, cfg *service.SystemCleanView) tgbotapi.InlineKeyboardMarkup {
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
