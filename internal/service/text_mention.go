package service

import (
	"fmt"
	"strings"
	"unicode/utf16"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func utf16TextLen(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func userMentionLabel(u *tgbotapi.User) string {
	if u == nil {
		return "该用户"
	}
	if username := strings.TrimSpace(u.UserName); username != "" {
		return "@" + username
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name != "" {
		return name
	}
	return fmt.Sprintf("uid:%d", u.ID)
}

func composeTextWithUserMention(prefix string, user *tgbotapi.User, suffix string) (string, []tgbotapi.MessageEntity) {
	mentionText := userMentionLabel(user)
	text := prefix + mentionText + suffix
	if user == nil || user.ID == 0 {
		return text, nil
	}
	return text, []tgbotapi.MessageEntity{
		{
			Type:   "text_mention",
			Offset: utf16TextLen(prefix),
			Length: utf16TextLen(mentionText),
			User:   user,
		},
	}
}

func formatWelcomeMentions(users []tgbotapi.User) (string, []tgbotapi.MessageEntity) {
	var textBuilder strings.Builder
	entities := make([]tgbotapi.MessageEntity, 0, len(users))
	offset := 0
	for _, u := range users {
		label := userMentionLabel(&u)
		if strings.TrimSpace(label) == "" {
			continue
		}
		if textBuilder.Len() > 0 {
			textBuilder.WriteString(" ")
			offset += utf16TextLen(" ")
		}
		start := offset
		textBuilder.WriteString(label)
		labelLen := utf16TextLen(label)
		offset += labelLen
		if u.ID != 0 {
			userCopy := u
			entities = append(entities, tgbotapi.MessageEntity{
				Type:   "text_mention",
				Offset: start,
				Length: labelLen,
				User:   &userCopy,
			})
		}
	}
	return textBuilder.String(), entities
}

func shiftMentionEntities(entities []tgbotapi.MessageEntity, offset int) []tgbotapi.MessageEntity {
	if len(entities) == 0 {
		return nil
	}
	shifted := make([]tgbotapi.MessageEntity, 0, len(entities))
	for _, entity := range entities {
		item := entity
		item.Offset += offset
		shifted = append(shifted, item)
	}
	return shifted
}

func buildWelcomeTextWithMentions(template string, users []tgbotapi.User) (string, []tgbotapi.MessageEntity) {
	mentionsText, mentionsEntities := formatWelcomeMentions(users)
	if strings.TrimSpace(mentionsText) == "" {
		return template, nil
	}
	if strings.Contains(template, "{user}") {
		parts := strings.Split(template, "{user}")
		var textBuilder strings.Builder
		entities := make([]tgbotapi.MessageEntity, 0, len(mentionsEntities)*(len(parts)-1))
		offset := 0
		for i, part := range parts {
			textBuilder.WriteString(part)
			offset += utf16TextLen(part)
			if i < len(parts)-1 {
				textBuilder.WriteString(mentionsText)
				entities = append(entities, shiftMentionEntities(mentionsEntities, offset)...)
				offset += utf16TextLen(mentionsText)
			}
		}
		return textBuilder.String(), entities
	}
	text := template
	if strings.TrimSpace(text) == "" {
		return mentionsText, mentionsEntities
	}
	text += "\n" + mentionsText
	return text, shiftMentionEntities(mentionsEntities, utf16TextLen(template+"\n"))
}
