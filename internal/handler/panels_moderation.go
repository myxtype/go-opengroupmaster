package handler

import (
	"fmt"
	"strings"
	"supervisor/internal/handler/keyboards"

	tgbot "github.com/go-telegram/bot"
)

func (h *Handler) sendSystemCleanPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	cfg, err := h.service.SystemCleanViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载系统消息清理失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"帮助您自动清理群组中的系统消息",
		"预设: 严格 / 推荐 / 关闭",
		"",
		fmt.Sprintf("进群: %s", onOffWithEmoji(cfg.Join)),
		fmt.Sprintf("退群: %s", onOffWithEmoji(cfg.Leave)),
		fmt.Sprintf("置顶: %s", onOffWithEmoji(cfg.Pin)),
		fmt.Sprintf("修改头像: %s", onOffWithEmoji(cfg.Photo)),
		fmt.Sprintf("修改名称: %s", onOffWithEmoji(cfg.Title)),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.SystemCleanKeyboard(tgGroupID, cfg))
}

func (h *Handler) sendAntiFloodPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反刷屏设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	status := "❌ 关闭"
	if view.Enabled {
		status = "✅ 启用"
	}
	lines := []string{
		"💬 反刷屏",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("当前设置:在%d秒内发送%d条消息触发反刷屏", view.WindowSec, view.MaxMessages),
		fmt.Sprintf("惩罚:%s", antiFloodPenaltyText(view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes)),
		fmt.Sprintf("删除提醒:%s", antiFloodAlertDeleteText(view.WarnDeleteSec)),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiFloodKeyboard(tgGroupID, view))
}

func (h *Handler) sendAntiFloodPenaltyPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反刷屏设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"💬 反刷屏 - 惩罚设置",
		"",
		fmt.Sprintf("当前惩罚:%s", antiFloodPenaltyText(view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes)),
		"",
		"可设置：惩罚方式、警告阈值、阈值后动作、禁言/封禁时长。",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiFloodPenaltyKeyboard(tgGroupID, view))
}

func (h *Handler) sendAntiFloodAlertDeletePanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反刷屏设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"💬 反刷屏 - 删除提醒",
		"",
		fmt.Sprintf("当前设置:%s", antiFloodAlertDeleteText(view.WarnDeleteSec)),
		"请选择提醒消息自动删除时间：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiFloodAlertDeleteKeyboard(tgGroupID, view.WarnDeleteSec))
}

func (h *Handler) sendAntiFloodCountPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反刷屏设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"💬 反刷屏 - 触发条数",
		"",
		fmt.Sprintf("当前设置:%d 条", view.MaxMessages),
		"请选择触发条数：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiFloodCountKeyboard(tgGroupID, view.MaxMessages))
}

func (h *Handler) sendAntiFloodWindowPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiFloodViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反刷屏设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"💬 反刷屏 - 检测间隔",
		"",
		fmt.Sprintf("当前设置:%d 秒", view.WindowSec),
		"请选择检测间隔：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiFloodWindowKeyboard(tgGroupID, view.WindowSec))
}

func (h *Handler) sendAntiSpamPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反垃圾设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	status := "❌ 关闭"
	if view.Enabled {
		status = "✅ 启用"
	}
	keywords := "无"
	if len(view.ExceptionKeywords) > 0 {
		show := view.ExceptionKeywords
		if len(show) > 5 {
			show = show[:5]
		}
		keywords = strings.Join(show, "、")
		if len(view.ExceptionKeywords) > len(show) {
			keywords += " ..."
		}
	}
	lines := []string{
		"📨 反垃圾",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("惩罚:%s", antiFloodPenaltyText(view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes)),
	}
	if view.AIAvailable {
		lines = append(lines,
			fmt.Sprintf("AI判定:%s", onOffWithEmoji(view.AIEnabled)),
			fmt.Sprintf("AI判定垃圾分:%d", view.AISpamScore),
			fmt.Sprintf("AI严格度:%s", antiSpamAIStrictnessText(view.AIStrictness)),
		)
	}
	lines = append(lines,
		"",
		fmt.Sprintf("已添加例外: %d条", view.ExceptionKeywordCount),
		fmt.Sprintf("例外关键词:%s", keywords),
	)
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiSpamKeyboard(tgGroupID, view))
}

func (h *Handler) sendAntiSpamPenaltyPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反垃圾设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"📨 反垃圾 - 惩罚设置",
		"",
		fmt.Sprintf("当前惩罚:%s", antiFloodPenaltyText(view.Penalty, view.WarnThreshold, view.WarnAction, view.WarnActionMuteMinutes, view.WarnActionBanMinutes, view.MuteMinutes, view.BanMinutes)),
		"",
		"可设置：惩罚方式、警告阈值、阈值后动作、禁言/封禁时长。",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiSpamPenaltyKeyboard(tgGroupID, view))
}

func (h *Handler) sendAntiSpamAlertDeletePanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载反垃圾设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{
		"📨 反垃圾 - 删除提醒",
		"",
		fmt.Sprintf("当前设置:%s", antiSpamAlertSettingText(view.WarnDeleteSec)),
		"请选择提醒策略：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiSpamAlertDeleteKeyboard(tgGroupID, view.WarnDeleteSec))
}

func (h *Handler) sendAntiSpamAIPanel(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	view, err := h.service.AntiSpamViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载AI反垃圾设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	status := "❌ 关闭"
	if view.AIEnabled {
		status = "✅ 启用"
	}
	if !view.AIAvailable {
		lines := []string{
			"🤖 AI智能反垃圾",
			"",
			"当前未配置 AI 模型（ANTI_SPAM_AI_MODEL），AI 功能不可用。",
		}
		h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiSpamKeyboard(tgGroupID, view))
		return
	}
	lines := []string{
		"🤖 AI智能反垃圾",
		"",
		fmt.Sprintf("状态:%s", status),
		fmt.Sprintf("AI判定垃圾分:%d", view.AISpamScore),
		fmt.Sprintf("严格度:%s", antiSpamAIStrictnessText(view.AIStrictness)),
		"",
		"说明：命中基础规则会直接按规则处理；规则未命中但可疑时，AI会按所选严格度做二分类判断。",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.AntiSpamAIKeyboard(tgGroupID, view))
}

func antiSpamAIStrictnessText(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "low":
		return "低（更宽松，减少误杀）"
	case "high":
		return "高（更严格，减少漏判）"
	default:
		return "中（平衡）"
	}
}
