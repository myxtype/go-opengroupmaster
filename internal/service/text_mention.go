package service

import (
	"fmt"
	"strings"

	"supervisor/pkg/tgmention"

	"github.com/go-telegram/bot/models"
)

func utf16TextLen(s string) int {
	return tgmention.UTF16Len(s)
}

func userMentionLabel(u *models.User) string {
	if u == nil {
		return "该用户"
	}
	return tgmention.UserLabel(tgmention.UserRef{
		ID:        u.ID,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Fallback:  "该用户",
	})
}

func composeTextWithUserMention(prefix string, user *models.User, suffix string) (string, []models.MessageEntity) {
	if user == nil {
		return tgmention.ComposeTextWithMention(prefix, tgmention.UserRef{Fallback: "该用户"}, suffix)
	}
	return tgmention.ComposeTextWithMention(prefix, tgmention.UserRef{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Fallback:  "该用户",
	}, suffix)
}

func composeAntiSpamAlertWithMention(user *models.User, reasonLabel string, actionLabel string) (string, []models.MessageEntity) {
	reason := strings.TrimSpace(reasonLabel)
	if reason == "" {
		reason = "规则判定"
	}
	action := strings.TrimSpace(actionLabel)
	if action == "" {
		action = "已撤回（不处罚）"
	}
	return composeTextWithUserMention("", user, fmt.Sprintf(" 正在发送垃圾消息。\n原因：%s\n处理：%s", reason, action))
}

func formatWelcomeMentions(users []models.User) (string, []models.MessageEntity) {
	refs := make([]tgmention.UserRef, 0, len(users))
	for _, u := range users {
		refs = append(refs, tgmention.UserRef{
			ID:        u.ID,
			Username:  u.Username,
			FirstName: u.FirstName,
			LastName:  u.LastName,
		})
	}
	return tgmention.JoinMentions(refs, " ")
}

func shiftMentionEntities(entities []models.MessageEntity, offset int) []models.MessageEntity {
	return tgmention.ShiftEntities(entities, offset)
}

func buildWelcomeTextWithMentions(template string, users []models.User) (string, []models.MessageEntity) {
	mentionsText, mentionsEntities := formatWelcomeMentions(users)
	if strings.TrimSpace(mentionsText) == "" {
		return template, nil
	}
	if strings.Contains(template, "{user}") {
		parts := strings.Split(template, "{user}")
		var textBuilder strings.Builder
		entities := make([]models.MessageEntity, 0, len(mentionsEntities)*(len(parts)-1))
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
