package keyboards

import (
	"fmt"
	"strconv"

	"supervisor/internal/model"
	"supervisor/internal/service"

	"github.com/go-telegram/bot/models"
)

func ScheduledListKeyboard(tgGroupID int64, items []model.ScheduledMessage, page, totalPages int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := make([][]models.InlineKeyboardButton, 0, len(items)+5)
	for _, item := range items {
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData(
				fmt.Sprintf("✏️ 编辑 #%d", item.ID),
				fmt.Sprintf("feat:sched:edit:%s:%d:%d", gid, item.ID, page),
			),
		))
		toggleLabel := fmt.Sprintf("启用 #%d", item.ID)
		if item.Enabled {
			toggleLabel = fmt.Sprintf("停用 #%d", item.ID)
		}
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData(
				toggleLabel,
				fmt.Sprintf("feat:sched:toggle:%s:%d:%d", gid, item.ID, page),
			),
			inlineKeyboardButtonData(
				fmt.Sprintf("🗑 删除 #%d", item.ID),
				fmt.Sprintf("feat:sched:del:%s:%d:%d", gid, item.ID, page),
			),
		))
		pinText := "置顶 ❌"
		if item.PinMessage {
			pinText = "置顶 ✅"
		}
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData(
				fmt.Sprintf("%s #%d", pinText, item.ID),
				fmt.Sprintf("feat:sched:pin:%s:%d:%d", gid, item.ID, page),
			),
		))
	}
	nav := make([]models.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, inlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:sched:list:%s:%d", gid, page-1)))
	}
	if page < totalPages {
		nav = append(nav, inlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:sched:list:%s:%d", gid, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, inlineKeyboardRow(nav...))
	}
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("➕ 新建定时", fmt.Sprintf("feat:sched:add:%s", gid)),
		inlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return inlineKeyboardMarkup(rows...)
}

func ScheduledPinSelectKeyboard(tgGroupID int64) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("置顶", fmt.Sprintf("feat:sched:pinset:%s:on", gid)),
			inlineKeyboardButtonData("不置顶", fmt.Sprintf("feat:sched:pinset:%s:off", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回群面板", fmt.Sprintf("feat:pending:cancel:%s", gid)),
			inlineKeyboardButtonData("返回上级", fmt.Sprintf("feat:pending:back:%s", gid)),
		),
	)
}

func ScheduledEditKeyboard(tgGroupID int64, id uint, page int, enabled bool, pin bool) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	statusText := "启用"
	if enabled {
		statusText = "关闭"
	}
	pinText := "置顶 ❌"
	if pin {
		pinText = "置顶 ✅"
	}
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData("状态："+statusText, fmt.Sprintf("feat:sched:edittoggle:%s:%d:%d", gid, id, page)),
			inlineKeyboardButtonData(pinText, fmt.Sprintf("feat:sched:editpin:%s:%d:%d", gid, id, page)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("修改文本", fmt.Sprintf("feat:sched:edittext:%s:%d:%d", gid, id, page)),
			inlineKeyboardButtonData("修改媒体", fmt.Sprintf("feat:sched:editmedia:%s:%d:%d", gid, id, page)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("修改按钮", fmt.Sprintf("feat:sched:editbuttons:%s:%d:%d", gid, id, page)),
			inlineKeyboardButtonData("修改 Cron", fmt.Sprintf("feat:sched:editcron:%s:%d:%d", gid, id, page)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回定时列表", fmt.Sprintf("feat:sched:list:%s:%d", gid, page)),
		),
	)
}

func LogListKeyboard(tgGroupID int64, page, totalPages int, filter string) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	allLabel := selectedLabel("全部", filter == "all")
	spamLabel := selectedLabel("审核", filter == "anti_spam*")
	verifyLabel := selectedLabel("验证", filter == "join_verify_pass")
	rows := make([][]models.InlineKeyboardButton, 0, 3)
	nav := make([]models.InlineKeyboardButton, 0, 2)
	if page > 1 {
		nav = append(nav, inlineKeyboardButtonData("⬅ 上一页", fmt.Sprintf("feat:logs:list:%s:%d:%s", gid, page-1, filter)))
	}
	if page < totalPages {
		nav = append(nav, inlineKeyboardButtonData("下一页 ➡", fmt.Sprintf("feat:logs:list:%s:%d:%s", gid, page+1, filter)))
	}
	if len(nav) > 0 {
		rows = append(rows, inlineKeyboardRow(nav...))
	}
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData(allLabel, fmt.Sprintf("feat:logs:list:%s:1:all", gid)),
		inlineKeyboardButtonData(spamLabel, fmt.Sprintf("feat:logs:list:%s:1:anti_spam*", gid)),
		inlineKeyboardButtonData(verifyLabel, fmt.Sprintf("feat:logs:list:%s:1:join_verify_pass", gid)),
	))
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("导出 CSV", fmt.Sprintf("feat:logs:export:%s:%s", gid, filter)),
		inlineKeyboardButtonData("刷新日志", fmt.Sprintf("feat:logs:list:%s:%d:%s", gid, page, filter)),
		inlineKeyboardButtonData("◀ 返回群面板", cbGroupPrefix+gid),
	))
	return inlineKeyboardMarkup(rows...)
}

func SystemCleanKeyboard(tgGroupID int64, cfg *service.SystemCleanView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	strictSelected := cfg.Join && cfg.Leave && cfg.Pin && cfg.Photo && cfg.Title
	offSelected := !cfg.Join && !cfg.Leave && !cfg.Pin && !cfg.Photo && !cfg.Title
	recommendedSelected := cfg.Join && cfg.Leave && !cfg.Pin && !cfg.Photo && !cfg.Title
	rows := [][]models.InlineKeyboardButton{
		inlineKeyboardRow(
			inlineKeyboardButtonData(selectedLabel("严格", strictSelected), fmt.Sprintf("feat:sys:preset:%s:strict", gid)),
			inlineKeyboardButtonData(selectedLabel("推荐", recommendedSelected), fmt.Sprintf("feat:sys:preset:%s:recommended", gid)),
			inlineKeyboardButtonData(selectedLabel("关闭", offSelected), fmt.Sprintf("feat:sys:preset:%s:off", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("进群 "+onOffWithEmoji(cfg.Join), fmt.Sprintf("feat:sys:toggle:%s:join", gid)),
			inlineKeyboardButtonData("退群 "+onOffWithEmoji(cfg.Leave), fmt.Sprintf("feat:sys:toggle:%s:leave", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("置顶 "+onOffWithEmoji(cfg.Pin), fmt.Sprintf("feat:sys:toggle:%s:pin", gid)),
			inlineKeyboardButtonData("头像 "+onOffWithEmoji(cfg.Photo), fmt.Sprintf("feat:sys:toggle:%s:photo", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("名称 "+onOffWithEmoji(cfg.Title), fmt.Sprintf("feat:sys:toggle:%s:title", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:sys:view:%s", gid)),
	}
	return inlineKeyboardMarkup(rows...)
}
