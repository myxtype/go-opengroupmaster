package service

import (
	"strings"

	"supervisor/internal/tgmention"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func utf16TextLen(s string) int {
	return tgmention.UTF16Len(s)
}

func userMentionLabel(u *tgbotapi.User) string {
	if u == nil {
		return "该用户"
	}
	return tgmention.UserLabel(tgmention.UserRef{
		ID:        u.ID,
		Username:  u.UserName,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Fallback:  "该用户",
	})
}

func composeTextWithUserMention(prefix string, user *tgbotapi.User, suffix string) (string, []tgbotapi.MessageEntity) {
	if user == nil {
		return tgmention.ComposeTextWithMention(prefix, tgmention.UserRef{Fallback: "该用户"}, suffix)
	}
	return tgmention.ComposeTextWithMention(prefix, tgmention.UserRef{
		ID:        user.ID,
		Username:  user.UserName,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Fallback:  "该用户",
	}, suffix)
}

func formatWelcomeMentions(users []tgbotapi.User) (string, []tgbotapi.MessageEntity) {
	refs := make([]tgmention.UserRef, 0, len(users))
	for _, u := range users {
		refs = append(refs, tgmention.UserRef{
			ID:        u.ID,
			Username:  u.UserName,
			FirstName: u.FirstName,
			LastName:  u.LastName,
		})
	}
	return tgmention.JoinMentions(refs, " ")
}

func shiftMentionEntities(entities []tgbotapi.MessageEntity, offset int) []tgbotapi.MessageEntity {
	return tgmention.ShiftEntities(entities, offset)
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
