package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func (s *Service) CheckMessageAndRespond(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) error {
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if msg.From != nil {
		blacklisted, err := s.repo.IsGlobalBlacklisted(msg.From.ID)
		if err == nil && blacklisted {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
				UntilDate:        time.Now().Add(24 * time.Hour).Unix(),
			})
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 命中全局黑名单，已移出群组", msg.From.UserName)))
			_ = s.repo.CreateLog(group.ID, "global_blacklist_kick", 0, 0)
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
	if antiSpam && containsLink(msg.Text) {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 检测到链接，消息已删除", msg.From.UserName)))
		_ = s.repo.CreateLog(group.ID, "anti_spam_delete", 0, 0)
		return true, nil
	}

	antiFlood, err := s.IsFeatureEnabled(group.ID, featureAntiFlood, false)
	if err != nil {
		return false, err
	}
	if antiFlood && s.isFlooding(group.TGGroupID, msg.From.ID) {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		restrict := tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: msg.From.ID},
			UntilDate:        time.Now().Add(60 * time.Second).Unix(),
			Permissions:      &tgbotapi.ChatPermissions{},
		}
		_, _ = bot.Request(restrict)
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("@%s 触发反刷屏，已禁言 60 秒", msg.From.UserName)))
		_ = s.repo.CreateLog(group.ID, "anti_flood_restrict", 0, 0)
		return true, nil
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

func (s *Service) isFlooding(tgGroupID, tgUserID int64) bool {
	now := time.Now().Unix()
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.flood[key]
	valid := make([]int64, 0, len(items)+1)
	for _, ts := range items {
		if now-ts <= 10 {
			valid = append(valid, ts)
		}
	}
	valid = append(valid, now)
	s.flood[key] = valid
	return len(valid) > 5
}

func containsLink(text string) bool {
	l := strings.ToLower(text)
	return strings.Contains(l, "http://") ||
		strings.Contains(l, "https://") ||
		strings.Contains(l, "t.me/") ||
		strings.Contains(l, "www.")
}
