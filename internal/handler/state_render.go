package handler

import (
	"supervisor/internal/handler/keyboards"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ensureAdmin 确保用户是目标群的管理员
func (h *Handler) ensureAdmin(bot *tgbotapi.BotAPI, target renderTarget, tgUserID, tgGroupID int64) bool {
	ok, err := h.service.IsAdminByTGGroupID(tgGroupID, tgUserID)
	if err != nil || !ok {
		h.render(bot, target, "你不是该群管理员，或机器人尚未同步该群权限", keyboards.MainMenuKeyboard(bot.Self.UserName))
		return false
	}
	return true
}

// render 渲染消息，优先尝试编辑原消息（保持聊天界面干净）
func (h *Handler) render(bot *tgbotapi.BotAPI, target renderTarget, text string, markup tgbotapi.InlineKeyboardMarkup) {
	if target.Edit && target.MessageID > 0 {
		edit := tgbotapi.NewEditMessageTextAndMarkup(target.ChatID, target.MessageID, text, markup)
		if _, err := bot.Send(edit); err == nil {
			return
		}
	}
	msg := tgbotapi.NewMessage(target.ChatID, text)
	msg.ReplyMarkup = markup
	_, _ = bot.Send(msg)
}

// setPending 设置用户的待处理输入状态（用于多步骤交互流程）
func (h *Handler) setPending(userID int64, input pendingInput) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending[userID] = input
}

// getPending 获取用户的待处理输入状态
func (h *Handler) getPending(userID int64) (pendingInput, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	in, ok := h.pending[userID]
	return in, ok
}

// clearPending 清除用户的待处理输入状态
func (h *Handler) clearPending(userID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.pending, userID)
}

// answerCallback 回复回调查询（隐藏按钮上的加载状态）
func (h *Handler) answerCallback(bot *tgbotapi.BotAPI, callbackID, text string) {
	_, _ = bot.Request(tgbotapi.NewCallback(callbackID, text))
}

// answerCallbackAlert 回复回调查询并显示弹窗提示
func (h *Handler) answerCallbackAlert(bot *tgbotapi.BotAPI, callbackID, text string) {
	cfg := tgbotapi.NewCallback(callbackID, text)
	cfg.ShowAlert = true
	_, _ = bot.Request(cfg)
}
