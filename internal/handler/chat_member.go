package handler

import (
	"github.com/go-telegram/bot/models"
)

func (h *Handler) handleChatMemberUpdate(update *models.ChatMemberUpdated) {
	if update == nil {
		return
	}
	if err := h.service.TrackInviteByChatMemberUpdate(update); err != nil {
		h.logger.Printf("track invite by chat member update failed: %v", err)
	}
}
