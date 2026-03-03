package handler

import (
	"bytes"
	"context"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func sendText(bot *tgbot.Bot, chatID int64, text string) (*models.Message, error) {
	return bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
}

func sendTextWithEntities(bot *tgbot.Bot, chatID int64, text string, entities []models.MessageEntity) (*models.Message, error) {
	return bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
		ChatID:   chatID,
		Text:     text,
		Entities: entities,
	})
}

func sendTextReply(bot *tgbot.Bot, chatID int64, replyToMessageID int, text string) (*models.Message, error) {
	return bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
		ReplyParameters: &models.ReplyParameters{
			MessageID: replyToMessageID,
		},
	})
}

func sendDocumentBytes(bot *tgbot.Bot, chatID int64, name string, content []byte, caption string) (*models.Message, error) {
	return bot.SendDocument(context.Background(), &tgbot.SendDocumentParams{
		ChatID:   chatID,
		Document: &models.InputFileUpload{Filename: name, Data: bytes.NewReader(content)},
		Caption:  caption,
	})
}

func isPrivateChat(chat models.Chat) bool {
	return chat.Type == "private"
}

func isGroupChat(chat models.Chat) bool {
	return chat.Type == "group"
}

func isSuperGroupChat(chat models.Chat) bool {
	return chat.Type == "supergroup"
}

func isCommandMessage(msg *models.Message) bool {
	if msg == nil {
		return false
	}
	if len(msg.Entities) == 0 {
		return false
	}
	e := msg.Entities[0]
	return e.Type == models.MessageEntityTypeBotCommand && e.Offset == 0 && e.Length > 1
}

func messageCommand(msg *models.Message) string {
	if !isCommandMessage(msg) || msg == nil {
		return ""
	}
	runes := []rune(msg.Text)
	if len(runes) < 2 {
		return ""
	}
	entity := msg.Entities[0]
	if entity.Length > len(runes) {
		return ""
	}
	raw := string(runes[1:entity.Length])
	if at := strings.Index(raw, "@"); at >= 0 {
		raw = raw[:at]
	}
	return strings.TrimSpace(raw)
}

func messageCommandArguments(msg *models.Message) string {
	if !isCommandMessage(msg) || msg == nil {
		return ""
	}
	runes := []rune(msg.Text)
	entity := msg.Entities[0]
	if entity.Length >= len(runes) {
		return ""
	}
	return strings.TrimSpace(string(runes[entity.Length:]))
}

func forwardFromUser(msg *models.Message) *models.User {
	if msg == nil || msg.ForwardOrigin == nil || msg.ForwardOrigin.MessageOriginUser == nil {
		return nil
	}
	u := msg.ForwardOrigin.MessageOriginUser.SenderUser
	return &u
}

func callbackMessage(cb *models.CallbackQuery) *models.Message {
	if cb == nil {
		return nil
	}
	return cb.Message.Message
}
