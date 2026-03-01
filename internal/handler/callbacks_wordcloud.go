package handler

import (
	"fmt"
	"strconv"
	"strings"

	"supervisor/internal/handler/keyboards"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) handleWordCloudFeature(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, target renderTarget, userID, tgGroupID int64, action string, parts []string) {
	switch action {
	case "noop":
		h.answerCallback(bot, cb.ID, "")
	case "view":
		h.answerCallback(bot, cb.ID, "加载词云")
		h.sendWordCloudPanel(bot, target, userID, tgGroupID)
	case "on":
		if _, err := h.service.SetWordCloudEnabledByTGGroupID(tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "词云已开启")
		h.sendWordCloudPanel(bot, target, userID, tgGroupID)
	case "off":
		if _, err := h.service.SetWordCloudEnabledByTGGroupID(tgGroupID, false); err != nil {
			h.answerCallback(bot, cb.ID, "设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "词云已关闭")
		h.sendWordCloudPanel(bot, target, userID, tgGroupID)
	case "gen":
		if err := h.service.SendWordCloudReportByTGGroupID(bot, tgGroupID, true); err != nil {
			h.answerCallback(bot, cb.ID, "生成失败（暂无数据或字体未配置）")
			return
		}
		h.answerCallback(bot, cb.ID, "已发送词云")
		h.sendWordCloudPanel(bot, target, userID, tgGroupID)
	case "settimeinput":
		h.answerCallback(bot, cb.ID, "请输入时间")
		h.setPending(userID, pendingInput{Kind: "wc_set_push_time", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入词云推送时间（HH:MM，24小时制）\n示例：18:00", keyboards.PendingCancelKeyboard(tgGroupID))
	case "settime":
		if len(parts) < 5 || len(parts[4]) != 4 {
			h.answerCallback(bot, cb.ID, "参数错误")
			return
		}
		raw := strings.TrimSpace(parts[4])
		hour := raw[:2]
		minute := raw[2:]
		if _, _, err := h.service.SetWordCloudPushTimeByTGGroupID(tgGroupID, fmt.Sprintf("%s:%s", hour, minute)); err != nil {
			h.answerCallback(bot, cb.ID, "时间设置失败")
			return
		}
		h.answerCallback(bot, cb.ID, "推送时间已更新")
		h.sendWordCloudPanel(bot, target, userID, tgGroupID)
	case "blacklist":
		page := 1
		if len(parts) >= 5 {
			if p, err := strconv.Atoi(parts[4]); err == nil {
				page = p
			}
		}
		h.answerCallback(bot, cb.ID, "加载黑名单")
		h.sendWordCloudBlacklistPanel(bot, target, userID, tgGroupID, page)
	case "blackadd":
		h.answerCallback(bot, cb.ID, "请输入词语")
		h.setPending(userID, pendingInput{Kind: "wc_black_add", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入要加入词云黑名单的词语（单个）", keyboards.PendingCancelKeyboard(tgGroupID))
	case "blackremove":
		h.answerCallback(bot, cb.ID, "请输入词语")
		h.setPending(userID, pendingInput{Kind: "wc_black_remove", TGGroupID: tgGroupID})
		h.render(bot, target, "请输入要移除的词云黑名单词语（单个）", keyboards.PendingCancelKeyboard(tgGroupID))
	default:
		h.answerCallback(bot, cb.ID, "未知操作")
	}
}
