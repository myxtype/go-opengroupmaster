package keyboards

import (
	"fmt"
	"strconv"

	"supervisor/internal/service"

	"github.com/go-telegram/bot/models"
)

func AntiFloodKeyboard(tgGroupID int64, view *service.AntiFloodView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := [][]models.InlineKeyboardButton{
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:floodon:%s", gid),
			fmt.Sprintf("feat:mod:floodoff:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("触发条数：%d", view.MaxMessages), fmt.Sprintf("feat:mod:floodcount:%s", gid)),
			inlineKeyboardButtonData(fmt.Sprintf("检测间隔：%d秒", view.WindowSec), fmt.Sprintf("feat:mod:floodwindow:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("惩罚设置", fmt.Sprintf("feat:mod:floodpenaltycfg:%s", gid)),
		),
	}

	rows = append(rows,
		inlineKeyboardRow(
			inlineKeyboardButtonData("删除提醒："+antiFloodAlertDeleteText(view.WarnDeleteSec), fmt.Sprintf("feat:mod:floodalertdel:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:floodview:%s", gid)),
	)
	return inlineKeyboardMarkup(rows...)
}

func AntiFloodAlertDeleteKeyboard(tgGroupID int64, currentSec int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	offLabel := selectedLabel("关闭", currentSec <= 0)
	sec5Label := selectedLabel("5秒", currentSec == 5)
	sec10Label := selectedLabel("10秒", currentSec == 10)
	sec20Label := selectedLabel("20秒", currentSec == 20)
	sec30Label := selectedLabel("30秒", currentSec == 30)
	sec60Label := selectedLabel("60秒", currentSec == 60)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(offLabel, fmt.Sprintf("feat:mod:floodalertset:%s:0", gid)),
			inlineKeyboardButtonData(sec5Label, fmt.Sprintf("feat:mod:floodalertset:%s:5", gid)),
			inlineKeyboardButtonData(sec10Label, fmt.Sprintf("feat:mod:floodalertset:%s:10", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(sec20Label, fmt.Sprintf("feat:mod:floodalertset:%s:20", gid)),
			inlineKeyboardButtonData(sec30Label, fmt.Sprintf("feat:mod:floodalertset:%s:30", gid)),
			inlineKeyboardButtonData(sec60Label, fmt.Sprintf("feat:mod:floodalertset:%s:60", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回反刷屏面板", fmt.Sprintf("feat:mod:floodview:%s", gid)),
		),
	)
}

func AntiFloodCountKeyboard(tgGroupID int64, current int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	c3Label := selectedLabel("3条", current == 3)
	c5Label := selectedLabel("5条", current == 5)
	c8Label := selectedLabel("8条", current == 8)
	c10Label := selectedLabel("10条", current == 10)
	c15Label := selectedLabel("15条", current == 15)
	c20Label := selectedLabel("20条", current == 20)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(c3Label, fmt.Sprintf("feat:mod:floodcountset:%s:3", gid)),
			inlineKeyboardButtonData(c5Label, fmt.Sprintf("feat:mod:floodcountset:%s:5", gid)),
			inlineKeyboardButtonData(c8Label, fmt.Sprintf("feat:mod:floodcountset:%s:8", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(c10Label, fmt.Sprintf("feat:mod:floodcountset:%s:10", gid)),
			inlineKeyboardButtonData(c15Label, fmt.Sprintf("feat:mod:floodcountset:%s:15", gid)),
			inlineKeyboardButtonData(c20Label, fmt.Sprintf("feat:mod:floodcountset:%s:20", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回反刷屏面板", fmt.Sprintf("feat:mod:floodview:%s", gid)),
		),
	)
}

func AntiFloodWindowKeyboard(tgGroupID int64, currentSec int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	sec3Label := selectedLabel("3秒", currentSec == 3)
	sec5Label := selectedLabel("5秒", currentSec == 5)
	sec10Label := selectedLabel("10秒", currentSec == 10)
	sec15Label := selectedLabel("15秒", currentSec == 15)
	sec20Label := selectedLabel("20秒", currentSec == 20)
	sec30Label := selectedLabel("30秒", currentSec == 30)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(sec3Label, fmt.Sprintf("feat:mod:floodwindowset:%s:3", gid)),
			inlineKeyboardButtonData(sec5Label, fmt.Sprintf("feat:mod:floodwindowset:%s:5", gid)),
			inlineKeyboardButtonData(sec10Label, fmt.Sprintf("feat:mod:floodwindowset:%s:10", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(sec15Label, fmt.Sprintf("feat:mod:floodwindowset:%s:15", gid)),
			inlineKeyboardButtonData(sec20Label, fmt.Sprintf("feat:mod:floodwindowset:%s:20", gid)),
			inlineKeyboardButtonData(sec30Label, fmt.Sprintf("feat:mod:floodwindowset:%s:30", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回反刷屏面板", fmt.Sprintf("feat:mod:floodview:%s", gid)),
		),
	)
}

func AntiSpamKeyboard(tgGroupID int64, view *service.AntiSpamView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := [][]models.InlineKeyboardButton{
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:spamon:%s", gid),
			fmt.Sprintf("feat:mod:spamoff:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("惩罚设置", fmt.Sprintf("feat:mod:spampenaltycfg:%s", gid)),
		),
	}

	if view != nil && view.AIAvailable {
		rows = append(rows, inlineKeyboardRow(
			inlineKeyboardButtonData("🤖 AI智能反垃圾设置", fmt.Sprintf("feat:mod:spamaicfg:%s", gid)),
		))
	}

	rows = append(rows,
		inlineKeyboardRow(
			inlineKeyboardButtonData("屏蔽图片 "+onOffWithEmoji(view.BlockPhoto), fmt.Sprintf("feat:mod:spamopt:%s:photo", gid)),
			inlineKeyboardButtonData("屏蔽链接 "+onOffWithEmoji(view.BlockLink), fmt.Sprintf("feat:mod:spamopt:%s:link", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("屏蔽频道马甲 "+onOffWithEmoji(view.BlockChannelAlias), fmt.Sprintf("feat:mod:spamopt:%s:senderchat", gid)),
			inlineKeyboardButtonData("屏蔽频道转发 "+onOffWithEmoji(view.BlockForwardFromChan), fmt.Sprintf("feat:mod:spamopt:%s:fwdchan", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("屏蔽用户转发 "+onOffWithEmoji(view.BlockForwardFromUser), fmt.Sprintf("feat:mod:spamopt:%s:fwduser", gid)),
			inlineKeyboardButtonData("屏蔽@群组ID "+onOffWithEmoji(view.BlockAtGroupID), fmt.Sprintf("feat:mod:spamopt:%s:atgroup", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("屏蔽联系人分享 "+onOffWithEmoji(view.BlockContactShare), fmt.Sprintf("feat:mod:spamopt:%s:contact", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("屏蔽@用户ID "+onOffWithEmoji(view.BlockAtUserID), fmt.Sprintf("feat:mod:spamopt:%s:atuser", gid)),
			inlineKeyboardButtonData("屏蔽ETH地址 "+onOffWithEmoji(view.BlockEthAddress), fmt.Sprintf("feat:mod:spamopt:%s:eth", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("屏蔽超长消息 "+onOffWithEmoji(view.BlockLongMessage), fmt.Sprintf("feat:mod:spamopt:%s:longmsg", gid)),
			inlineKeyboardButtonData(fmt.Sprintf("消息长度:%d", view.MaxMessageLength), fmt.Sprintf("feat:mod:spammsglen:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("屏蔽超长姓名 "+onOffWithEmoji(view.BlockLongName), fmt.Sprintf("feat:mod:spamopt:%s:longname", gid)),
			inlineKeyboardButtonData(fmt.Sprintf("姓名长度:%d", view.MaxNameLength), fmt.Sprintf("feat:mod:spamnamelen:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("例外+（%d）", view.ExceptionKeywordCount), fmt.Sprintf("feat:mod:spamexadd:%s", gid)),
			inlineKeyboardButtonData("例外-（按关键词）", fmt.Sprintf("feat:mod:spamexdel:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("删除提醒设置", fmt.Sprintf("feat:mod:spamalertdel:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:spamview:%s", gid)),
	)
	return inlineKeyboardMarkup(rows...)
}

func AntiSpamPenaltyKeyboard(tgGroupID int64, view *service.AntiSpamView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := moderationPenaltyRows(gid, "spam", view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes)
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("◀ 返回反垃圾面板", fmt.Sprintf("feat:mod:spamview:%s", gid)),
	))
	return inlineKeyboardMarkup(rows...)
}

func AntiFloodPenaltyKeyboard(tgGroupID int64, view *service.AntiFloodView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	rows := moderationPenaltyRows(gid, "flood", view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes)
	rows = append(rows, inlineKeyboardRow(
		inlineKeyboardButtonData("◀ 返回反刷屏面板", fmt.Sprintf("feat:mod:floodview:%s", gid)),
	))
	return inlineKeyboardMarkup(rows...)
}

func AntiSpamAlertDeleteKeyboard(tgGroupID int64, currentSec int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	s10 := selectedLabel("10秒", currentSec == 10)
	s30 := selectedLabel("30秒", currentSec == 30)
	s60 := selectedLabel("60秒", currentSec == 60)
	m5 := selectedLabel("5分钟", currentSec == 300)
	m10 := selectedLabel("10分钟", currentSec == 600)
	m30 := selectedLabel("30分钟", currentSec == 1800)
	h1 := selectedLabel("1小时", currentSec == 3600)
	h6 := selectedLabel("6小时", currentSec == 21600)
	h12 := selectedLabel("12小时", currentSec == 43200)
	noDelete := selectedLabel("不删除", currentSec == 0)
	noAlert := selectedLabel("不提醒", currentSec == -1)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(s10, fmt.Sprintf("feat:mod:spamalertdelset:%s:10", gid)),
			inlineKeyboardButtonData(s30, fmt.Sprintf("feat:mod:spamalertdelset:%s:30", gid)),
			inlineKeyboardButtonData(s60, fmt.Sprintf("feat:mod:spamalertdelset:%s:60", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(m5, fmt.Sprintf("feat:mod:spamalertdelset:%s:300", gid)),
			inlineKeyboardButtonData(m10, fmt.Sprintf("feat:mod:spamalertdelset:%s:600", gid)),
			inlineKeyboardButtonData(m30, fmt.Sprintf("feat:mod:spamalertdelset:%s:1800", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(h1, fmt.Sprintf("feat:mod:spamalertdelset:%s:3600", gid)),
			inlineKeyboardButtonData(h6, fmt.Sprintf("feat:mod:spamalertdelset:%s:21600", gid)),
			inlineKeyboardButtonData(h12, fmt.Sprintf("feat:mod:spamalertdelset:%s:43200", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(noDelete, fmt.Sprintf("feat:mod:spamalertdelset:%s:0", gid)),
			inlineKeyboardButtonData(noAlert, fmt.Sprintf("feat:mod:spamalertdelset:%s:-1", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回反垃圾面板", fmt.Sprintf("feat:mod:spamview:%s", gid)),
		),
	)
}

func AntiSpamAIKeyboard(tgGroupID int64, view *service.AntiSpamView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	strictness := view.AIStrictness
	lowLabel := selectedLabel("低", strictness == "low")
	mediumLabel := selectedLabel("中", strictness == "medium")
	highLabel := selectedLabel("高", strictness == "high")
	return inlineKeyboardMarkup(
		statusControlRow(
			view.AIEnabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:spamaion:%s", gid),
			fmt.Sprintf("feat:mod:spamaioff:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("AI垃圾分:%d", view.AISpamScore), fmt.Sprintf("feat:mod:spamaiscore:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("严格度", fmt.Sprintf("feat:mod:noop:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(lowLabel, fmt.Sprintf("feat:mod:spamaistrict:%s:low", gid)),
			inlineKeyboardButtonData(mediumLabel, fmt.Sprintf("feat:mod:spamaistrict:%s:medium", gid)),
			inlineKeyboardButtonData(highLabel, fmt.Sprintf("feat:mod:spamaistrict:%s:high", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("🔄 刷新", fmt.Sprintf("feat:mod:spamaicfg:%s", gid)),
			inlineKeyboardButtonData("◀ 返回反垃圾面板", fmt.Sprintf("feat:mod:spamview:%s", gid)),
		),
	)
}

func VerifyKeyboard(tgGroupID int64, view *service.JoinVerifyView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	buttonLabel := "按钮"
	mathLabel := "数学题"
	captchaLabel := "验证码"
	zhcharLabel := "中文字符验证码"
	zhvoiceLabel := "中文语音验证码"
	switch view.Type {
	case "button":
		buttonLabel = "✅" + buttonLabel
	case "math":
		mathLabel = "✅" + mathLabel
	case "captcha":
		captchaLabel = "✅" + captchaLabel
	case "zhchar":
		zhcharLabel = "✅" + zhcharLabel
	case "zhvoice":
		zhvoiceLabel = "✅" + zhvoiceLabel
	}
	return inlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:verifyon:%s", gid),
			fmt.Sprintf("feat:mod:verifyoff:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("验证时间：%d分钟", view.TimeoutMinutes), fmt.Sprintf("feat:mod:verifytime:%s", gid)),
			inlineKeyboardButtonData("超时处理："+verifyTimeoutActionLabel(view.TimeoutAction), fmt.Sprintf("feat:mod:verifytimeout:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("⬇️验证方式⬇️", fmt.Sprintf("feat:mod:noop:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(buttonLabel, fmt.Sprintf("feat:mod:verifymethod:%s:button", gid)),
			inlineKeyboardButtonData(mathLabel, fmt.Sprintf("feat:mod:verifymethod:%s:math", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(captchaLabel, fmt.Sprintf("feat:mod:verifymethod:%s:captcha", gid)),
			inlineKeyboardButtonData(zhcharLabel, fmt.Sprintf("feat:mod:verifymethod:%s:zhchar", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(zhvoiceLabel, fmt.Sprintf("feat:mod:verifymethod:%s:zhvoice", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:verifyview:%s", gid)),
	)
}

func NewbieLimitKeyboard(tgGroupID int64, view *service.NewbieLimitView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	return inlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:newbieon:%s", gid),
			fmt.Sprintf("feat:mod:newbieoff:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData(fmt.Sprintf("限制时长：%d分钟", view.Minutes), fmt.Sprintf("feat:mod:newbietime:%s", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:newbieview:%s", gid)),
	)
}

func VerifyTimeoutMinutesKeyboard(tgGroupID int64, current int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	m1Label := selectedLabel("1分钟", current == 1)
	m5Label := selectedLabel("5分钟", current == 5)
	m10Label := selectedLabel("10分钟", current == 10)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(m1Label, fmt.Sprintf("feat:mod:verifytimeset:%s:1", gid)),
			inlineKeyboardButtonData(m5Label, fmt.Sprintf("feat:mod:verifytimeset:%s:5", gid)),
			inlineKeyboardButtonData(m10Label, fmt.Sprintf("feat:mod:verifytimeset:%s:10", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回验证面板", fmt.Sprintf("feat:mod:verifyview:%s", gid)),
		),
	)
}

func NewbieLimitMinutesKeyboard(tgGroupID int64, current int) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	m10Label := selectedLabel("10分钟", current == 10)
	m30Label := selectedLabel("30分钟", current == 30)
	m60Label := selectedLabel("60分钟", current == 60)
	return inlineKeyboardMarkup(
		inlineKeyboardRow(
			inlineKeyboardButtonData(m10Label, fmt.Sprintf("feat:mod:newbietimeset:%s:10", gid)),
			inlineKeyboardButtonData(m30Label, fmt.Sprintf("feat:mod:newbietimeset:%s:30", gid)),
			inlineKeyboardButtonData(m60Label, fmt.Sprintf("feat:mod:newbietimeset:%s:60", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("◀ 返回新成员限制面板", fmt.Sprintf("feat:mod:newbieview:%s", gid)),
		),
	)
}

func NightModeKeyboard(tgGroupID int64, view *service.NightModeView) models.InlineKeyboardMarkup {
	gid := strconv.FormatInt(tgGroupID, 10)
	deleteMediaLabel := "删除媒体"
	globalMuteLabel := "全局禁言"
	if view.Mode == "global_mute" {
		globalMuteLabel = "✅全局禁言"
	} else {
		deleteMediaLabel = "✅删除媒体"
	}
	return inlineKeyboardMarkup(
		statusControlRow(
			view.Enabled,
			fmt.Sprintf("feat:mod:noop:%s", gid),
			fmt.Sprintf("feat:mod:nighton:%s", gid),
			fmt.Sprintf("feat:mod:nightoff:%s", gid),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("时区："+view.TimezoneText, fmt.Sprintf("feat:mod:nighttz:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("夜间时段："+view.NightWindow, fmt.Sprintf("feat:mod:nightwindow:%s", gid)),
		),
		inlineKeyboardRow(
			inlineKeyboardButtonData("处理方式：", fmt.Sprintf("feat:mod:noop:%s", gid)),
			inlineKeyboardButtonData(deleteMediaLabel, fmt.Sprintf("feat:mod:nightmode:%s:delete_media", gid)),
			inlineKeyboardButtonData(globalMuteLabel, fmt.Sprintf("feat:mod:nightmode:%s:global_mute", gid)),
		),
		panelRefreshBackRow(gid, fmt.Sprintf("feat:mod:nightview:%s", gid)),
	)
}
