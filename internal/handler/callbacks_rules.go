package handler

import (
	"fmt"
	"strconv"
	"strings"
	"supervisor/internal/handler/keyboards"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) handleAutoReplyFeature(bot *tgbot.Bot, cb *models.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "view":
		h.answerCallback(bot, cb.ID, "加载自动回复")
		h.sendAutoReplyList(bot, target, userID, tgGroupID, 1)
	case "add":
		h.answerCallback(bot, cb.ID, "请选择触发方式")
		h.setPending(userID, pendingInput{Kind: "auto_add_mode", TGGroupID: tgGroupID, Page: 1})
		h.render(bot, target, "第1步：请选择触发方式\n精准触发：消息内容与关键词完全相同才触发\n包含触发：消息内容中包含关键词就触发\n正则触发：关键词按正则表达式匹配消息", keyboards.AutoReplyMatchTypeKeyboard(tgGroupID, fmt.Sprintf("feat:auto:addmode:%d", tgGroupID)))
	case "addmode":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		matchType := strings.TrimSpace(parts[4])
		if !isAutoReplyMatchType(matchType) {
			h.answerCallback(bot, cb.ID, "触发方式错误")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入关键词")
		keywordPrompt := "第2步：请输入自动回复关键词"
		if matchType == "regex" {
			keywordPrompt = "第2步：请输入正则表达式（例如：(?i)hello|hi）"
		}
		h.setPending(userID, pendingInput{
			Kind:      "auto_add_keyword",
			TGGroupID: tgGroupID,
			Page:      1,
			MatchType: matchType,
		})
		h.render(bot, target, keywordPrompt, keyboards.PendingCancelKeyboard(tgGroupID))
	case "list":
		page := 1
		if len(parts) >= 5 {
			if p, err := strconv.Atoi(parts[4]); err == nil {
				page = p
			}
		}
		h.answerCallback(bot, cb.ID, "加载自动回复")
		h.sendAutoReplyList(bot, target, userID, tgGroupID, page)
	case "del":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		if err := h.service.DeleteAutoReplyByTGGroupID(tgGroupID, uint(id)); err != nil {
			h.answerCallback(bot, cb.ID, "删除失败")
			return
		}
		h.answerCallback(bot, cb.ID, "已删除")
		h.sendAutoReplyList(bot, target, userID, tgGroupID, page)
	case "edit":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		h.answerCallback(bot, cb.ID, "请选择触发方式")
		h.setPending(userID, pendingInput{Kind: "auto_edit_mode", TGGroupID: tgGroupID, RuleID: uint(id), Page: page})
		h.render(bot, target, "第1步：请选择新触发方式\n精准触发：消息内容与关键词完全相同才触发\n包含触发：消息内容中包含关键词就触发\n正则触发：关键词按正则表达式匹配消息", keyboards.AutoReplyMatchTypeKeyboard(tgGroupID, fmt.Sprintf("feat:auto:editmode:%d:%d:%d", tgGroupID, id, page)))
	case "editmode":
		if len(parts) < 7 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		matchType := strings.TrimSpace(parts[6])
		if !isAutoReplyMatchType(matchType) {
			h.answerCallback(bot, cb.ID, "触发方式错误")
			return
		}
		h.answerCallback(bot, cb.ID, "请输入关键词")
		keywordPrompt := "第2步：请输入新的关键词"
		if matchType == "regex" {
			keywordPrompt = "第2步：请输入新的正则表达式（例如：(?i)hello|hi）"
		}
		h.setPending(userID, pendingInput{
			Kind:      "auto_edit_keyword",
			TGGroupID: tgGroupID,
			RuleID:    uint(id),
			Page:      page,
			MatchType: matchType,
		})
		h.render(bot, target, keywordPrompt, keyboards.PendingCancelKeyboard(tgGroupID))
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleBannedWordFeature(bot *tgbot.Bot, cb *models.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "noop":
		h.answerCallback(bot, cb.ID, "")
	case "view":
		h.answerCallback(bot, cb.ID, "加载违禁词")
		h.sendBannedWordList(bot, target, userID, tgGroupID, 1)
	case "on":
		if _, err := h.service.SetBannedWordEnabledByTGGroupID(tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "违禁词已开启")
		h.sendBannedWordList(bot, target, userID, tgGroupID, 1)
	case "off":
		if _, err := h.service.SetBannedWordEnabledByTGGroupID(tgGroupID, false); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "违禁词已关闭")
		h.sendBannedWordList(bot, target, userID, tgGroupID, 1)
	case "penalty":
		h.answerCallback(bot, cb.ID, "加载惩罚设置")
		h.sendBannedWordPenaltyPanel(bot, target, userID, tgGroupID)
	case "warn", "mute":
		h.answerCallback(bot, cb.ID, "请在惩罚面板设置")
		h.sendBannedWordPenaltyPanel(bot, target, userID, tgGroupID)
	case "penaltyset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		if _, err := h.service.SetBannedWordPenaltyByTGGroupID(tgGroupID, parts[4]); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "惩罚已更新")
		h.sendBannedWordPenaltyPanel(bot, target, userID, tgGroupID)
	case "warncount":
		h.answerCallback(bot, cb.ID, "请输入警告次数")
		h.setPending(userID, pendingInput{Kind: "bw_warn_threshold", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入达到处罚前的警告次数（正整数，例如 3）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "warnaction":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		if _, err := h.service.SetBannedWordWarnActionByTGGroupID(tgGroupID, parts[4]); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "阈值后动作已更新")
		h.sendBannedWordPenaltyPanel(bot, target, userID, tgGroupID)
	case "warnmuteinput":
		h.answerCallback(bot, cb.ID, "请输入阈值禁言时长")
		h.setPending(userID, pendingInput{Kind: "bw_warn_action_mute_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入警告达到阈值后禁言时长（分钟，1-10080）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "warnbaninput":
		h.answerCallback(bot, cb.ID, "请输入阈值封禁时长")
		h.setPending(userID, pendingInput{Kind: "bw_warn_action_ban_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入警告达到阈值后封禁时长（分钟，1-10080）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "muteinput":
		h.answerCallback(bot, cb.ID, "请输入禁言时长")
		h.setPending(userID, pendingInput{Kind: "bw_mute_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入禁言时长（分钟，1-10080）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "baninput":
		h.answerCallback(bot, cb.ID, "请输入封禁时长")
		h.setPending(userID, pendingInput{Kind: "bw_ban_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入封禁时长（分钟，1-10080）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "delwarninput":
		h.answerCallback(bot, cb.ID, "请输入删除提醒时长")
		h.setPending(userID, pendingInput{Kind: "bw_warn_delete_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入提醒消息自动删除时长（分钟，0-1440；0 表示不自动删除）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "delwarn":
		h.answerCallback(bot, cb.ID, "请输入删除提醒时长")
		h.setPending(userID, pendingInput{Kind: "bw_warn_delete_minutes", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入提醒消息自动删除时长（分钟，0-1440；0 表示不自动删除）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "delwarnset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		v, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		mins, err := h.service.SetBannedWordWarnDeleteMinutesByTGGroupID(tgGroupID, v)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "删除提醒："+bannedWordDeleteText(mins))
		h.sendBannedWordList(bot, target, userID, tgGroupID, 1)
	case "add":
		h.answerCallback(bot, cb.ID, "请发送违禁词")
		h.setPending(userID, pendingInput{Kind: "bw_add", TGGroupID: tgGroupID, Page: 1})
		h.render(bot, target, "请直接发送要新增的违禁词（支持多条，一行一个）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "remove":
		h.answerCallback(bot, cb.ID, "请发送要删除的违禁词")
		h.setPending(userID, pendingInput{Kind: "bw_remove", TGGroupID: tgGroupID, Page: 1})
		h.render(bot, target, "请直接发送要删除的违禁词（支持多条，一行一个）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "list":
		page := 1
		if len(parts) >= 5 {
			if p, err := strconv.Atoi(parts[4]); err == nil {
				page = p
			}
		}
		h.answerCallback(bot, cb.ID, "加载违禁词")
		h.sendBannedWordList(bot, target, userID, tgGroupID, page)
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleLotteryFeature(bot *tgbot.Bot, cb *models.CallbackQuery, target renderTarget, tgGroupID int64, action string, parts []string) {
	switch action {
	case "view":
		h.answerCallback(bot, cb.ID, "加载抽奖")
		h.sendLotteryPanel(bot, target, cb.From.ID, tgGroupID)
	case "create":
		h.answerCallback(bot, cb.ID, "开始创建抽奖")
		h.setPending(cb.From.ID, pendingInput{Kind: "lottery_create_title", TGGroupID: tgGroupID})
		h.render(bot, target, "第1步：请输入抽奖标题\n示例：周末福利", keyboards.PendingCancelKeyboard(tgGroupID))
	case "draw":
		winners, err := h.service.DrawActiveLotteryByTGGroupID(tgGroupID)
		if err != nil {
			h.answerCallback(bot, cb.ID, "开奖失败")
			view, viewErr := h.service.LotteryPanelViewByTGGroupID(tgGroupID)
			if viewErr != nil {
				h.render(bot, target, "开奖失败：没有可开奖的活动抽奖", keyboards.GroupPanelKeyboard(tgGroupID))
				return
			}
			h.render(bot, target, "开奖失败：没有可开奖的活动抽奖", keyboards.LotteryKeyboard(tgGroupID, view.PublishPin, view.ResultPin, view.DeleteKeywordMins))
			return
		}
		h.answerCallback(bot, cb.ID, "开奖完成")
		resultText, resultEntities := lotteryResultText(winners)
		resultMsg, sendErr := sendTextWithEntities(bot, tgGroupID, resultText, resultEntities)
		if sendErr == nil {
			_ = h.service.PinLotteryMessageByTGGroupID(bot, tgGroupID, resultMsg.ID, "result")
		}
		h.sendLotteryPanel(bot, target, cb.From.ID, tgGroupID)
	case "records":
		page := 1
		if len(parts) >= 5 {
			if p, err := strconv.Atoi(parts[4]); err == nil {
				page = p
			}
		}
		h.answerCallback(bot, cb.ID, "加载抽奖记录")
		h.sendLotteryRecordsPanel(bot, target, cb.From.ID, tgGroupID, page)
	case "cancel":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		lotteryID, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		ok, err := h.service.CancelLotteryByTGGroupID(tgGroupID, uint(lotteryID))
		if err != nil {
			h.answerCallback(bot, cb.ID, "取消失败")
			return
		}
		if !ok {
			h.answerCallback(bot, cb.ID, "仅可取消未开奖活动")
		} else {
			h.answerCallback(bot, cb.ID, "已取消抽奖活动")
		}
		h.sendLotteryRecordsPanel(bot, target, cb.From.ID, tgGroupID, page)
	case "toggle":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		on, err := h.service.ToggleLotteryConfigByTGGroupID(tgGroupID, parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		if on {
			h.answerCallback(bot, cb.ID, "已开启")
		} else {
			h.answerCallback(bot, cb.ID, "已关闭")
		}
		h.sendLotteryPanel(bot, target, cb.From.ID, tgGroupID)
	case "delmins":
		h.answerCallback(bot, cb.ID, "请选择删除时长")
		h.sendLotteryDeleteMinutesPanel(bot, target, cb.From.ID, tgGroupID)
	case "delminsset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		v, err := strconv.Atoi(parts[4])
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		mins, err := h.service.SetLotteryDeleteKeywordMinutesByTGGroupID(tgGroupID, v)
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		if mins > 0 {
			h.answerCallback(bot, cb.ID, fmt.Sprintf("口令和参与成功提示消息将于 %d 分钟后删除", mins))
		} else {
			h.answerCallback(bot, cb.ID, "已关闭自动删除口令和参与成功提示消息")
		}
		h.sendLotteryDeleteMinutesPanel(bot, target, cb.From.ID, tgGroupID)
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}

func (h *Handler) handleScheduleFeature(bot *tgbot.Bot, cb *models.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "view":
		h.answerCallback(bot, cb.ID, "加载定时消息")
		h.sendScheduledList(bot, target, userID, tgGroupID, 1)
	case "add":
		h.answerCallback(bot, cb.ID, "请发送定时消息")
		h.setPending(userID, pendingInput{Kind: "sched_add_cron", TGGroupID: tgGroupID, Page: 1})
		h.render(bot, target, "第1步：请输入 cron 表达式\n含义：分钟 小时 日 月 星期（共5段，用空格分隔）\n示例：\n- 0 9 * * *  （每天 09:00）\n- */30 * * * *（每30分钟）\n- 0 21 * * 1-5（工作日 21:00）\n输入后将进入第2步填写消息内容（支持文本或媒体），第3步可选设置链接按钮，第4步设置是否置顶", keyboards.PendingCancelKeyboard(tgGroupID))
	case "list":
		page := 1
		if len(parts) >= 5 {
			if p, err := strconv.Atoi(parts[4]); err == nil {
				page = p
			}
		}
		h.answerCallback(bot, cb.ID, "加载定时消息")
		h.sendScheduledList(bot, target, userID, tgGroupID, page)
	case "del":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		if err := h.service.DeleteScheduledMessageByTGGroupID(tgGroupID, uint(id)); err != nil {
			h.answerCallback(bot, cb.ID, "删除失败")
			return
		}
		h.answerCallback(bot, cb.ID, "已删除")
		h.sendScheduledList(bot, target, userID, tgGroupID, page)
	case "toggle":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		enabled, err := h.service.ToggleScheduledMessageByTGGroupID(tgGroupID, uint(id))
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		if enabled {
			h.answerCallback(bot, cb.ID, "已启用")
		} else {
			h.answerCallback(bot, cb.ID, "已停用")
		}
		h.sendScheduledList(bot, target, userID, tgGroupID, page)
	case "edit":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		h.answerCallback(bot, cb.ID, "编辑定时任务")
		h.sendScheduledEditPanel(bot, target, userID, tgGroupID, uint(id), page)
	case "edittoggle":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		enabled, err := h.service.ToggleScheduledMessageByTGGroupID(tgGroupID, uint(id))
		if err != nil {
			h.answerCallback(bot, cb.ID, "切换失败")
			return
		}
		if enabled {
			h.answerCallback(bot, cb.ID, "已启用")
		} else {
			h.answerCallback(bot, cb.ID, "已关闭")
		}
		h.sendScheduledEditPanel(bot, target, userID, tgGroupID, uint(id), page)
	case "editpin":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		pin, err := h.service.ToggleScheduledPinByTGGroupID(tgGroupID, uint(id))
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		if pin {
			h.answerCallback(bot, cb.ID, "已设为置顶")
		} else {
			h.answerCallback(bot, cb.ID, "已设为不置顶")
		}
		h.sendScheduledEditPanel(bot, target, userID, tgGroupID, uint(id), page)
	case "edittext":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		h.answerCallback(bot, cb.ID, "请输入新的文本内容")
		h.setPending(userID, pendingInput{Kind: "sched_edit_text", TGGroupID: tgGroupID, RuleID: uint(id), Page: page})
		h.render(bot, target, "请输入新的定时消息文本。\n注意：若当前没有媒体，文本不能为空。", keyboards.PendingCancelKeyboard(tgGroupID))
	case "editmedia":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		h.answerCallback(bot, cb.ID, "请发送媒体")
		h.setPending(userID, pendingInput{Kind: "sched_edit_media", TGGroupID: tgGroupID, RuleID: uint(id), Page: page})
		h.render(bot, target, "请发送图片/视频/文件/动图作为定时媒体（可带文字说明）。\n发送“关闭”可清空媒体。", keyboards.PendingCancelKeyboard(tgGroupID))
	case "editbuttons":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		h.answerCallback(bot, cb.ID, "请输入按钮配置")
		h.setPending(userID, pendingInput{Kind: "sched_edit_buttons", TGGroupID: tgGroupID, RuleID: uint(id), Page: page})
		h.render(bot, target, "请输入链接按钮配置，格式示例：\n官网 - link.com\n电报 - t.me/GroupName\n官网 - link.com && 电报 - t.me/GroupName\n发送“关闭”可清空按钮。", keyboards.PendingCancelKeyboard(tgGroupID))
	case "editcron":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		h.answerCallback(bot, cb.ID, "请输入新的 cron")
		h.setPending(userID, pendingInput{Kind: "sched_edit_cron", TGGroupID: tgGroupID, RuleID: uint(id), Page: page})
		h.render(bot, target, "请输入新的 cron 表达式（5段）：分钟 小时 日 月 星期\n示例：0 9 * * *", keyboards.PendingCancelKeyboard(tgGroupID))
	case "pin":
		if len(parts) < 6 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		id, err := strconv.ParseUint(parts[4], 10, 64)
		if err != nil {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		page, _ := strconv.Atoi(parts[5])
		if page < 1 {
			page = 1
		}
		pin, err := h.service.ToggleScheduledPinByTGGroupID(tgGroupID, uint(id))
		if err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		if pin {
			h.answerCallback(bot, cb.ID, "已设为置顶")
		} else {
			h.answerCallback(bot, cb.ID, "已设为不置顶")
		}
		h.sendScheduledList(bot, target, userID, tgGroupID, page)
	case "pinset":
		if len(parts) < 5 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		pending, ok := h.getPending(userID)
		if !ok || pending.Kind != "sched_add_pin" || pending.TGGroupID != tgGroupID {
			h.answerCallback(bot, cb.ID, "创建流程已过期，请重新开始")
			h.sendScheduledList(bot, target, userID, tgGroupID, 1)
			return
		}
		pin := strings.TrimSpace(parts[4]) == "on"
		if err := h.service.CreateScheduledMessageByTGGroupIDAdvanced(
			pending.TGGroupID,
			pending.Content,
			pending.CronExpr,
			pending.RawButtons,
			pending.MediaType,
			pending.MediaFileID,
			pin,
		); err != nil {
			h.answerCallback(bot, cb.ID, "创建失败")
			h.render(bot, target, "创建定时消息失败："+err.Error(), keyboards.PendingCancelKeyboard(tgGroupID))
			return
		}
		h.clearPending(userID)
		if pin {
			h.answerCallback(bot, cb.ID, "创建成功（已设为置顶）")
		} else {
			h.answerCallback(bot, cb.ID, "创建成功（不置顶）")
		}
		h.sendScheduledList(bot, target, userID, tgGroupID, 1)
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}
