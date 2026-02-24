package handler

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func (h *Handler) handleChatMemberUpdate(update *tgbotapi.ChatMemberUpdated) {
	if update == nil {
		return
	}
	if err := h.service.TrackInviteByChatMemberUpdate(update); err != nil {
		h.logger.Printf("track invite by chat member update failed: %v", err)
	}
}
