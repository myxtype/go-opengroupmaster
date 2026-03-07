package handler

import (
	"strings"

	"supervisor/internal/handler/keyboards"

	tgbot "github.com/go-telegram/bot"
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

func (h *Handler) handleMyChatMemberUpdate(bot *tgbot.Bot, update *models.ChatMemberUpdated) {
	if update == nil || bot == nil {
		return
	}
	if !isGroupChat(update.Chat) && !isSuperGroupChat(update.Chat) {
		return
	}
	if !isBotActivatedInChat(update) {
		return
	}

	group, err := h.service.RegisterGroup(&update.Chat)
	if err != nil {
		h.logger.Printf("register group from my_chat_member failed chat=%d: %v", update.Chat.ID, err)
	} else {
		_ = h.service.SyncGroupAdmins(bot, group)
	}

	text := botAddedToGroupText(h.botName)
	markup := keyboards.GroupOnboardingKeyboard(h.botUsername)
	if _, err := sendTextWithMarkup(bot, update.Chat.ID, text, markup); err != nil {
		h.logger.Printf("send group onboarding panel failed chat=%d: %v", update.Chat.ID, err)
	}
}

func isBotActivatedInChat(update *models.ChatMemberUpdated) bool {
	if update == nil {
		return false
	}
	return !isActiveChatMember(update.OldChatMember) && isActiveChatMember(update.NewChatMember)
}

func isActiveChatMember(member models.ChatMember) bool {
	switch member.Type {
	case models.ChatMemberTypeOwner, models.ChatMemberTypeAdministrator, models.ChatMemberTypeMember:
		return true
	case models.ChatMemberTypeRestricted:
		return member.Restricted != nil && member.Restricted.IsMember
	default:
		return false
	}
}

func botAddedToGroupText(botName string) string {
	name := strings.TrimSpace(botName)
	if name == "" {
		name = "机器人"
	}
	return name + " 已成功加入本群。\n点击下方按钮前往私聊，打开 /start 后即可开始配置群功能。"
}
