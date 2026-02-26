package handler

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"supervisor/internal/handler/keyboards"
	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) syncChainAnnouncementByID(bot *tgbotapi.BotAPI, chainID uint) {
	if chainID == 0 {
		return
	}
	view, err := h.service.ChainViewByChainID(chainID)
	if err != nil {
		return
	}
	if view.AnnouncementMessageID == 0 && !view.Active {
		return
	}
	text := renderChainAnnouncementText(view)
	joinURL := chainJoinURL(bot.Self.UserName, chainID)
	markup := keyboards.ChainPublicJoinKeyboard(joinURL, view.Active)

	if view.AnnouncementMessageID > 0 {
		edit := tgbotapi.NewEditMessageTextAndMarkup(view.TGGroupID, view.AnnouncementMessageID, text, markup)
		if _, err := bot.Send(edit); err == nil {
			return
		} else if strings.Contains(err.Error(), "message is not modified") {
			return
		}
	}
	msg := tgbotapi.NewMessage(view.TGGroupID, text)
	msg.ReplyMarkup = markup
	sent, err := bot.Send(msg)
	if err != nil {
		return
	}
	_ = h.service.SetChainAnnouncementMessageID(chainID, sent.MessageID)
}

func renderChainAnnouncementText(view *service.ChainView) string {
	if view == nil {
		return "🐉 接龙"
	}
	intro := strings.TrimSpace(view.Intro)
	if intro == "" {
		intro = "暂无接龙说明"
	}
	lines := []string{
		"🐉 接龙",
		"",
		fmt.Sprintf("编号:#%d", view.ID),
		intro,
		"截止时间:" + chainDeadlineText(view.DeadlineUnix),
	}
	if view.MaxParticipants > 0 {
		lines = append(lines, fmt.Sprintf("人数限制:%d（已参与 %d/%d）", view.MaxParticipants, len(view.Entries), view.MaxParticipants))
	} else {
		lines = append(lines, fmt.Sprintf("人数限制:不限（已参与 %d）", len(view.Entries)))
	}
	if len(view.Entries) > 0 {
		lines = append(lines, "")
		for i, item := range view.Entries {
			label := strings.TrimSpace(item.DisplayName)
			if label == "" {
				label = fmt.Sprintf("uid:%d", item.TGUserID)
			}
			userTag := strconv.FormatInt(item.TGUserID, 10)
			lines = append(lines, fmt.Sprintf("%d.%s （%s）", i+1, label, userTag))
			lines = append(lines, item.Content)
			lines = append(lines, "")
		}
	}
	if !view.Active {
		lines = append(lines, "状态:已截止")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func chainJoinURL(botUserName string, chainID uint) string {
	name := strings.TrimSpace(botUserName)
	if name == "" {
		return ""
	}
	payload := url.QueryEscape("chain_" + strconv.FormatUint(uint64(chainID), 10))
	return fmt.Sprintf("https://t.me/%s?start=%s", name, payload)
}

func displayNameFromUser(u *tgbotapi.User) string {
	if u == nil {
		return "匿名"
	}
	name := strings.TrimSpace(strings.TrimSpace(u.FirstName) + " " + strings.TrimSpace(u.LastName))
	if name != "" {
		return name
	}
	if strings.TrimSpace(u.UserName) != "" {
		return "@" + strings.TrimSpace(u.UserName)
	}
	return fmt.Sprintf("uid:%d", u.ID)
}
