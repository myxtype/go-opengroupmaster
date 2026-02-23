package service

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

var urlPattern = regexp.MustCompile(`(?i)\b(?:https?://|www\.|t\.me/|telegram\.me/)[^\s]+`)

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
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 命中本群黑名单，已移出群组", msg.From.UserName)))
			_ = s.repo.CreateLog(group.ID, "group_blacklist_kick", 0, 0)
			return nil
		}
	}

	if msg.Text != "" {
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

		_ = s.notifyKeywordMonitor(bot, group, msg)

		banned, err := s.repo.ContainsBannedWord(group.ID, msg.Text)
		if err != nil {
			return err
		}
		if banned {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			warn := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 消息触发违禁词，已删除", msg.From.UserName))
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
	if msg.From == nil {
		return false, nil
	}
	isAdmin, err := s.repo.CheckAdmin(group.ID, msg.From.ID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return false, nil
	}

	antiSpam, err := s.IsFeatureEnabled(group.ID, featureAntiSpam, false)
	if err != nil {
		return false, err
	}
	if antiSpam {
		cfg, err := s.getAntiSpamConfig(group.ID)
		if err != nil {
			return false, err
		}
		blocked, _ := containsBlockedLink(msg.Text, cfg)
		if blocked {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 检测到可疑链接，消息已删除", msg.From.UserName)))
			_ = s.repo.CreateLog(group.ID, "anti_spam_delete", 0, 0)
			return true, nil
		}
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
	_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 新成员限制中，暂不可发链接或媒体", msg.From.UserName)))
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
	if u == nil {
		return "该用户"
	}
	if strings.TrimSpace(u.UserName) != "" {
		return "@" + strings.TrimSpace(u.UserName)
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		return fmt.Sprintf("uid:%d", u.ID)
	}
	return name
}

func containsLink(text string) bool {
	return urlPattern.MatchString(strings.ToLower(text))
}

func containsBlockedLink(text string, cfg antiSpamConfig) (bool, string) {
	matches := urlPattern.FindAllString(text, -1)
	for _, raw := range matches {
		domain := extractDomain(raw)
		if domain == "" {
			return true, raw
		}
		if !domainAllowed(cfg.WhitelistDomains, domain) {
			return true, raw
		}
	}
	return false, ""
}

func extractDomain(rawURL string) string {
	s := rawURL
	if strings.HasPrefix(strings.ToLower(s), "www.") || strings.HasPrefix(strings.ToLower(s), "t.me/") || strings.HasPrefix(strings.ToLower(s), "telegram.me/") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	return host
}

func normalizeSpamText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

func domainAllowed(whitelist []string, domain string) bool {
	for _, item := range whitelist {
		if strings.EqualFold(strings.TrimSpace(item), domain) {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
