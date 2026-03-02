package tgmention

import (
	"fmt"
	"strings"
	"unicode/utf16"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserRef struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	Fallback  string
}

const (
	maxDisplayNameRunes = 16
	nameMaskRune        = '░'
)

func UTF16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func maskLongName(name string) string {
	runes := []rune(strings.TrimSpace(name))
	if len(runes) == 0 {
		return ""
	}

	if len(runes) > maxDisplayNameRunes {
		runes = runes[:maxDisplayNameRunes]
	}

	for i := 1; i < len(runes); i += 2 {
		runes[i] = nameMaskRune
	}
	return string(runes)
}

func UserLabel(u UserRef) string {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name != "" {
		return maskLongName(name)
	}
	if username := strings.TrimSpace(u.Username); username != "" {
		return "@" + maskLongName(username)
	}
	if fallback := strings.TrimSpace(u.Fallback); fallback != "" {
		return fallback
	}
	if u.ID != 0 {
		return fmt.Sprintf("uid:%d", u.ID)
	}
	return "该用户"
}

func ComposeTextWithMention(prefix string, user UserRef, suffix string) (string, []tgbotapi.MessageEntity) {
	mentionText := UserLabel(user)
	text := prefix + mentionText + suffix
	if user.ID == 0 {
		return text, nil
	}
	entityUser := tgbotapi.User{
		ID:        user.ID,
		UserName:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}
	return text, []tgbotapi.MessageEntity{
		{
			Type:   "text_mention",
			Offset: UTF16Len(prefix),
			Length: UTF16Len(mentionText),
			User:   &entityUser,
		},
	}
}

func JoinMentions(users []UserRef, sep string) (string, []tgbotapi.MessageEntity) {
	if len(users) == 0 {
		return "", nil
	}
	var textBuilder strings.Builder
	entities := make([]tgbotapi.MessageEntity, 0, len(users))
	offset := 0
	sepLen := UTF16Len(sep)
	for _, user := range users {
		label := UserLabel(user)
		if strings.TrimSpace(label) == "" {
			continue
		}
		if textBuilder.Len() > 0 {
			textBuilder.WriteString(sep)
			offset += sepLen
		}
		start := offset
		textBuilder.WriteString(label)
		labelLen := UTF16Len(label)
		offset += labelLen
		if user.ID != 0 {
			entityUser := tgbotapi.User{
				ID:        user.ID,
				UserName:  user.Username,
				FirstName: user.FirstName,
				LastName:  user.LastName,
			}
			entities = append(entities, tgbotapi.MessageEntity{
				Type:   "text_mention",
				Offset: start,
				Length: labelLen,
				User:   &entityUser,
			})
		}
	}
	return textBuilder.String(), entities
}

func ShiftEntities(entities []tgbotapi.MessageEntity, offset int) []tgbotapi.MessageEntity {
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
