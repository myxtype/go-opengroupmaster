package handler

import (
	"context"
	"supervisor/internal/handler/keyboards"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ensureAdmin 确保用户是目标群的管理员
func (h *Handler) ensureAdmin(bot *tgbot.Bot, target renderTarget, tgUserID, tgGroupID int64) bool {
	ok, err := h.service.IsAdminByTGGroupID(tgGroupID, tgUserID)
	if err != nil || !ok {
		h.render(bot, target, "你不是该群管理员，或机器人尚未同步该群权限", keyboards.MainMenuKeyboard(h.botUsername))
		return false
	}
	return true
}

// render 渲染消息，优先尝试编辑原消息（保持聊天界面干净）
func (h *Handler) render(bot *tgbot.Bot, target renderTarget, text string, markup models.InlineKeyboardMarkup) {
	if target.Edit && target.MessageID > 0 {
		if _, err := bot.EditMessageText(context.Background(), &tgbot.EditMessageTextParams{
			ChatID:      target.ChatID,
			MessageID:   target.MessageID,
			Text:        text,
			ReplyMarkup: markup,
		}); err == nil {
			return
		}
	}
	_, _ = bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
		ChatID:      target.ChatID,
		Text:        text,
		ReplyMarkup: markup,
	})
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
func (h *Handler) answerCallback(bot *tgbot.Bot, callbackID, text string) {
	_, _ = bot.AnswerCallbackQuery(context.Background(), &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
}

// answerCallbackAlert 回复回调查询并显示弹窗提示
func (h *Handler) answerCallbackAlert(bot *tgbot.Bot, callbackID, text string) {
	_, _ = bot.AnswerCallbackQuery(context.Background(), &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       true,
	})
}
