package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

var urlPattern = regexp.MustCompile(`(?i)\b(?:https?://|www\.|t\.me/|telegram\.me/)[^\s]+`)
var ethAddressPattern = regexp.MustCompile(`(?i)\b0x[a-f0-9]{40}\b`)
var mentionPattern = regexp.MustCompile(`@[A-Za-z0-9_]{2,}`)

func (s *Service) CheckMessageAndRespond(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) error {
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if msg.From != nil {
		blacklisted, err := s.repo.IsGroupBlacklisted(group.ID, msg.From.ID)
		if err == nil && blacklisted {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
				UntilDate:        time.Now().Add(24 * time.Hour).Unix(),
			})
			alertText, entities := composeTextWithUserMention("", msg.From, " 命中本群黑名单，已移出群组")
			alert := tgbotapi.NewMessage(msg.Chat.ID, alertText)
			alert.Entities = entities
			_, _ = bot.Send(alert)
			_ = s.repo.CreateLog(group.ID, "group_blacklist_kick", 0, 0)
			return nil
		}
	}

	handled, err := s.applyModeration(bot, msg, group)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	limited, err := s.applyNewbieLimit(bot, msg, group)
	if err != nil {
		return err
	}
	if limited {
		return nil
	}

	if msg.Text != "" {
		_ = s.notifyKeywordMonitor(bot, group, msg)

		banned, err := s.repo.ContainsBannedWord(group.ID, msg.Text)
		if err != nil {
			return err
		}
		if banned {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			warnText, entities := composeTextWithUserMention("", msg.From, " 消息触发违禁词，已删除")
			warn := tgbotapi.NewMessage(msg.Chat.ID, warnText)
			warn.Entities = entities
			_, _ = bot.Send(warn)
			_ = s.repo.CreateLog(group.ID, "banned_word_delete", 0, 0)
			return nil
		}

		if msg.From != nil {
			matched, joined, err := s.TryJoinLotteryByKeyword(group, msg.From, msg.Text)
			if err != nil {
				return err
			}
			if matched {
				if mins, cfgErr := s.LotteryDeleteKeywordMinutesByGroupID(group.ID); cfgErr == nil && mins > 0 {
					go func(chatID int64, messageID int, delayMins int) {
						time.Sleep(time.Duration(delayMins) * time.Minute)
						_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
					}(msg.Chat.ID, msg.MessageID, mins)
				}
				if joined {
					reply := tgbotapi.NewMessage(msg.Chat.ID, "参与抽奖成功")
					reply.ReplyToMessageID = msg.MessageID
					_, _ = bot.Send(reply)
				}
				return nil
			}
		}

		rule, err := s.repo.MatchAutoReply(group.ID, msg.Text)
		if err != nil {
			return err
		}
		if rule != nil {
			reply := tgbotapi.NewMessage(msg.Chat.ID, rule.Reply)
			reply.ReplyToMessageID = msg.MessageID
			if markup, ok := InlineKeyboardFromButtonRowsJSON(rule.ButtonRows); ok {
				reply.ReplyMarkup = markup
			}
			_, _ = bot.Send(reply)
		}
	}

	if msg.From != nil {
		u, err := s.repo.UpsertUserFromTG(msg.From)
		if err == nil {
			_ = s.repo.AddPoints(group.ID, u.ID, 1)
		}
	}

	return nil
}

func (s *Service) applyModeration(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, group *model.Group) (bool, error) {
	if msg.From == nil && msg.SenderChat == nil {
		return false, nil
	}
	if msg.From != nil {
		isAdmin, err := s.repo.CheckAdmin(group.ID, msg.From.ID)
		if err != nil {
			return false, err
		}
		if isAdmin {
			return false, nil
		}
	}

	// 夜间模式逻辑
	nightState, err := s.getNightModeState(group.ID)
	if err != nil {
		return false, err
	}
	if nightState.Enabled {
		cfg := normalizeNightModeConfig(nightState.Config)
		if isNightWindowNow(cfg.TimezoneOffsetMinutes, time.Now()) {
			switch cfg.Mode {
			case nightModeGlobalMute:
				_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
				_ = s.repo.CreateLog(group.ID, "night_mode_global_mute_delete", 0, 0)
				return true, nil
			default:
				if isNightMediaMessage(msg) {
					_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
					_ = s.repo.CreateLog(group.ID, "night_mode_delete_media", 0, 0)
					return true, nil
				}
			}
		}
	}

	// 反垃圾逻辑
	spamState, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return false, err
	}
	if spamState.Enabled {
		cfg := normalizeAntiSpamConfig(spamState.Config)
		blocked, reasonCode, reasonLabel := antiSpamViolation(msg, cfg)
		if blocked {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			appliedPenalty := cfg.Penalty
			if msg.From == nil && (cfg.Penalty == antiFloodPenaltyMute || cfg.Penalty == antiFloodPenaltyKick || cfg.Penalty == antiFloodPenaltyKickBan) {
				appliedPenalty = antiFloodPenaltyDeleteOnly
			}
			switch appliedPenalty {
			case antiFloodPenaltyMute:
				restrict := tgbotapi.RestrictChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
					UntilDate:        time.Now().Add(time.Duration(cfg.MuteSec) * time.Second).Unix(),
					Permissions:      &tgbotapi.ChatPermissions{},
				}
				_, _ = bot.Request(restrict)
			case antiFloodPenaltyKick:
				_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
					UntilDate:        time.Now().Add(1 * time.Minute).Unix(),
				})
				_, _ = bot.Request(tgbotapi.UnbanChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
					OnlyIfBanned:     true,
				})
			case antiFloodPenaltyKickBan:
				_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
					RevokeMessages:   true,
				})
			}
			alertText := fmt.Sprintf("%s 触发反垃圾（%s），已%s", antiSpamActorDisplayName(msg), reasonLabel, antiFloodActionLabel(appliedPenalty, cfg.MuteSec))
			alert, sendErr := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, alertText))
			if sendErr == nil && cfg.WarnDeleteSec > 0 {
				go func(chatID int64, messageID int, seconds int) {
					time.Sleep(time.Duration(seconds) * time.Second)
					_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
				}(msg.Chat.ID, alert.MessageID, cfg.WarnDeleteSec)
			}
			_ = s.repo.CreateLog(group.ID, "anti_spam_"+appliedPenalty+"_"+reasonCode, 0, 0)
			return true, nil
		}
	}

	if msg.From == nil {
		return false, nil
	}

	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return false, err
	}
	if state.Enabled {
		cfg := normalizeAntiFloodConfig(state.Config)
		flooding, reason := s.isFlooding(group.TGGroupID, msg.From.ID, msg.Text, cfg)
		if flooding {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			switch cfg.Penalty {
			case antiFloodPenaltyMute:
				restrict := tgbotapi.RestrictChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
					UntilDate:        time.Now().Add(time.Duration(cfg.MuteSec) * time.Second).Unix(),
					Permissions:      &tgbotapi.ChatPermissions{},
				}
				_, _ = bot.Request(restrict)
			case antiFloodPenaltyKick:
				_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
					UntilDate:        time.Now().Add(1 * time.Minute).Unix(),
				})
				_, _ = bot.Request(tgbotapi.UnbanChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
					OnlyIfBanned:     true,
				})
			case antiFloodPenaltyKickBan:
				_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
					RevokeMessages:   true,
				})
			}
			alertText := fmt.Sprintf("%s 触发反刷屏，已%s", floodUserDisplayName(msg.From), antiFloodActionLabel(cfg.Penalty, cfg.MuteSec))
			if reason == "high_freq" {
				alertText = fmt.Sprintf("%s（%d秒内%d条）", alertText, cfg.WindowSec, cfg.MaxMessages)
			}
			alert, sendErr := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, alertText))
			if sendErr == nil && cfg.WarnDeleteSec > 0 {
				go func(chatID int64, messageID int, seconds int) {
					time.Sleep(time.Duration(seconds) * time.Second)
					_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
				}(msg.Chat.ID, alert.MessageID, cfg.WarnDeleteSec)
			}
			_ = s.repo.CreateLog(group.ID, "anti_flood_"+cfg.Penalty+"_"+reason, 0, 0)
			return true, nil
		}
	}
	return false, nil
}
func (s *Service) applyNewbieLimit(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, group *model.Group) (bool, error) {
	enabled, err := s.IsFeatureEnabled(group.ID, featureNewbieLimit, false)
	if err != nil || !enabled {
		return false, err
	}
	if msg.From == nil {
		return false, nil
	}
	joinAt, ok := s.getJoinAt(group.TGGroupID, msg.From.ID)
	if !ok {
		return false, nil
	}
	minutes, _ := s.getNewbieLimitMinutes(group.ID)
	if time.Since(joinAt) > time.Duration(minutes)*time.Minute {
		s.clearJoinAt(group.TGGroupID, msg.From.ID)
		return false, nil
	}
	if !containsLink(msg.Text) && msg.Photo == nil && msg.Video == nil && msg.Document == nil {
		return false, nil
	}
	_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	noticeText, entities := composeTextWithUserMention("", msg.From, " 新成员限制中，暂不可发链接或媒体")
	notice := tgbotapi.NewMessage(msg.Chat.ID, noticeText)
	notice.Entities = entities
	_, _ = bot.Send(notice)
	_ = s.repo.CreateLog(group.ID, "newbie_limit_delete", 0, 0)
	return true, nil
}

func (s *Service) isFlooding(tgGroupID, tgUserID int64, text string, cfg antiFloodConfig) (bool, string) {
	now := time.Now().Unix()
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.flood[key]
	valid := make([]floodEvent, 0, len(items)+1)
	for _, item := range items {
		if now-item.Timestamp <= int64(cfg.WindowSec) {
			valid = append(valid, item)
		}
	}
	norm := normalizeSpamText(text)
	valid = append(valid, floodEvent{Timestamp: now, Text: norm})
	s.flood[key] = valid
	if len(valid) >= cfg.MaxMessages {
		return true, "high_freq"
	}
	return false, ""
}

func floodUserDisplayName(u *tgbotapi.User) string {
	return userMentionLabel(u)
}

func antiSpamActorDisplayName(msg *tgbotapi.Message) string {
	if msg == nil {
		return "该用户"
	}
	if msg.From != nil {
		return floodUserDisplayName(msg.From)
	}
	if msg.SenderChat != nil {
		title := strings.TrimSpace(msg.SenderChat.Title)
		if title != "" {
			return title
		}
		return fmt.Sprintf("chat:%d", msg.SenderChat.ID)
	}
	return "该用户"
}

func containsLink(text string) bool {
	return urlPattern.MatchString(strings.ToLower(text))
}

func antiSpamMessageContent(msg *tgbotapi.Message) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(msg.Text) != "" {
		parts = append(parts, msg.Text)
	}
	if strings.TrimSpace(msg.Caption) != "" {
		parts = append(parts, msg.Caption)
	}
	return strings.Join(parts, "\n")
}

func antiSpamExceptionHit(content string, keywords []string) bool {
	if strings.TrimSpace(content) == "" || len(keywords) == 0 {
		return false
	}
	lower := strings.ToLower(content)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(strings.TrimSpace(kw))) {
			return true
		}
	}
	return false
}

func antiSpamViolation(msg *tgbotapi.Message, cfg antiSpamConfig) (bool, string, string) {
	content := antiSpamMessageContent(msg)
	if antiSpamExceptionHit(content, cfg.ExceptionKeywords) {
		return false, "", ""
	}
	if cfg.BlockPhoto && len(msg.Photo) > 0 {
		return true, "photo", "图片"
	}
	if cfg.BlockChannelAlias && msg.SenderChat != nil {
		return true, "channel_alias", "频道马甲发言"
	}
	if cfg.BlockForwardFromChannel && (msg.ForwardFromChat != nil || msg.IsAutomaticForward) {
		return true, "forward_channel", "来自频道转发"
	}
	if cfg.BlockForwardFromUser && (msg.ForwardFrom != nil || strings.TrimSpace(msg.ForwardSenderName) != "") {
		return true, "forward_user", "来自用户转发"
	}
	if cfg.BlockLink && containsLink(content) {
		return true, "link", "链接"
	}
	if cfg.BlockAtGroupID && containsAtGroupID(content) {
		return true, "at_group", "@群组ID"
	}
	if cfg.BlockAtUserID && containsAtUserID(content) {
		return true, "at_user", "@用户ID"
	}
	if cfg.BlockEthAddress && containsETHAddress(content) {
		return true, "eth", "以太坊地址"
	}
	if cfg.BlockLongMessage && utf8.RuneCountInString(strings.TrimSpace(content)) > cfg.MaxMessageLength {
		return true, "long_message", "超长消息"
	}
	if cfg.BlockLongName && antiSpamNameLength(msg) > cfg.MaxNameLength {
		return true, "long_name", "超长姓名"
	}
	return false, "", ""
}

func antiSpamNameLength(msg *tgbotapi.Message) int {
	if msg == nil {
		return 0
	}
	if msg.From != nil {
		name := strings.TrimSpace(msg.From.FirstName + " " + msg.From.LastName)
		if name == "" {
			name = strings.TrimSpace(msg.From.UserName)
		}
		return utf8.RuneCountInString(name)
	}
	if msg.SenderChat != nil {
		return utf8.RuneCountInString(strings.TrimSpace(msg.SenderChat.Title))
	}
	return 0
}

func containsAtGroupID(content string) bool {
	for _, token := range mentionPattern.FindAllString(content, -1) {
		id := strings.TrimPrefix(token, "@")
		if id == "" {
			continue
		}
		numeric := true
		for _, r := range id {
			if r < '0' || r > '9' {
				numeric = false
				break
			}
		}
		if !numeric {
			return true
		}
	}
	return false
}

func containsAtUserID(content string) bool {
	for _, token := range mentionPattern.FindAllString(content, -1) {
		id := strings.TrimPrefix(token, "@")
		if id == "" {
			continue
		}
		numeric := true
		for _, r := range id {
			if r < '0' || r > '9' {
				numeric = false
				break
			}
		}
		if numeric {
			return true
		}
	}
	return strings.Contains(strings.ToLower(content), "tg://user?id=")
}

func containsETHAddress(content string) bool {
	return ethAddressPattern.MatchString(content)
}

func normalizeSpamText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}
