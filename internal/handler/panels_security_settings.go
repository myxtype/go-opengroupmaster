package handler

import (
	"fmt"
	"strings"
	"supervisor/internal/handler/keyboards"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) sendWelcomePanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	cfg, enabled, err := h.service.WelcomeViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载欢迎设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	modeText := "验证后欢迎"
	if cfg.Mode == "join" {
		modeText = "进群欢迎"
	}
	deleteText := "否"
	if cfg.DeleteMinutes > 0 {
		deleteText = fmt.Sprintf("%d", cfg.DeleteMinutes)
	}
	buttonCount := 0
	for _, row := range cfg.ButtonRows {
		buttonCount += len(row)
	}
	buttonText := onOffWithEmoji(buttonCount > 0)
	if buttonCount > 0 {
		buttonText = fmt.Sprintf("%s（%d个）", buttonText, buttonCount)
	}
	lines := []string{
		"🎉 进群欢迎",
		"",
		fmt.Sprintf("状态: %s", onOffWithEmoji(enabled)),
		fmt.Sprintf("模式: %s", modeText),
		fmt.Sprintf("删除消息(分钟): %s", deleteText),
		"",
		"自定义欢迎内容:",
		fmt.Sprintf("┌📸 媒体图片: %s", onOffWithEmoji(cfg.MediaFileID != "")),
		fmt.Sprintf("├🔠 链接按钮: %s", buttonText),
		fmt.Sprintf("└📄 文本内容: %s", onOffWithEmoji(strings.TrimSpace(cfg.Text) != "")),
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.WelcomeKeyboard(tgGroupID, enabled, cfg.Mode, cfg.DeleteMinutes))
}

func (h *Handler) sendWelcomeDeleteMinutesPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	cfg, _, err := h.service.WelcomeViewByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载欢迎设置失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	deleteText := "关闭"
	if cfg.DeleteMinutes > 0 {
		deleteText = fmt.Sprintf("%d分钟", cfg.DeleteMinutes)
	}
	lines := []string{
		"🎉 欢迎 - 删除消息",
		"",
		fmt.Sprintf("当前设置:%s", deleteText),
		"请选择欢迎消息自动删除时间：",
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.WelcomeDeleteMinutesKeyboard(tgGroupID, cfg.DeleteMinutes))
}

func (h *Handler) sendRBACPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	text, err := h.service.RBACSummaryByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载权限分级失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	h.render(bot, target, text, keyboards.RBACKeyboard(tgGroupID))
}

func (h *Handler) sendBlacklistPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) {
	if !h.ensureAdmin(bot, target, tgUserID, tgGroupID) {
		return
	}
	items, err := h.service.ListBlacklistByTGGroupID(tgGroupID)
	if err != nil {
		h.render(bot, target, "加载黑名单失败", keyboards.GroupPanelKeyboard(tgGroupID))
		return
	}
	lines := []string{"本群黑名单"}
	lines = append(lines, "支持按用户名、用户ID，或转发成员消息来添加/移除")
	if len(items) == 0 {
		lines = append(lines, "暂无黑名单用户")
	} else {
		for i, it := range items {
			lines = append(lines, fmt.Sprintf("%d. %d (%s)", i+1, it.TGUserID, it.Reason))
		}
	}
	h.render(bot, target, strings.Join(lines, "\n"), keyboards.BlacklistKeyboard(tgGroupID))
}

func (h *Handler) sendSettingsPanel(bot *tgbotapi.BotAPI, target renderTarget, tgUserID int64) {
	lang, _ := h.service.GetUserLanguage(tgUserID)
	text := "设置\n当前语言: " + lang + "\n可切换为中文/英文（逐步覆盖）"
	h.render(bot, target, text, keyboards.SettingsKeyboard(lang))
}
