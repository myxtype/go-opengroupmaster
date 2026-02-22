package service

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *Service) OnNewMembers(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) error {
	if len(msg.NewChatMembers) == 0 {
		return nil
	}
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		return nil
	}
	for _, m := range msg.NewChatMembers {
		s.markJoin(group.TGGroupID, m.ID)
	}

	verifyEnabled, err := s.IsFeatureEnabled(group.ID, featureJoinVerify, false)
	if err == nil && verifyEnabled {
		cfg, _ := s.getJoinVerifyConfig(group.ID)
		timeout := cfg.TimeoutSec
		if timeout <= 0 {
			timeout = 120
		}
		for _, m := range msg.NewChatMembers {
			if m.IsBot {
				continue
			}
			restrict := tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: m.ID},
				UntilDate:        time.Now().Add(time.Duration(timeout) * time.Second).Unix(),
				Permissions:      &tgbotapi.ChatPermissions{},
			}
			_, _ = bot.Request(restrict)

			pending := verifyPending{Deadline: time.Now().Add(time.Duration(timeout) * time.Second), Mode: cfg.Type}
			verifyText := fmt.Sprintf("新成员 @%s 请在 %d 秒内完成验证，否则将被移出。", m.UserName, timeout)
			keyboard := tgbotapi.NewInlineKeyboardMarkup()
			if cfg.Type == "math" {
				a := rand.Intn(9) + 1
				b := rand.Intn(9) + 1
				answer := a + b
				pending.Answer = answer
				verifyText = fmt.Sprintf("新成员 @%s 请完成算术验证：%d + %d = ?（%d 秒内）", m.UserName, a, b, timeout)
				options := buildMathOptions(answer)
				row := make([]tgbotapi.InlineKeyboardButton, 0, len(options))
				for _, opt := range options {
					row = append(row, tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(opt), fmt.Sprintf("verify:math:%d:%d:%d", group.TGGroupID, m.ID, opt)))
				}
				keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
			} else {
				keyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("我已验证", fmt.Sprintf("verify:pass:%d:%d", group.TGGroupID, m.ID)),
					),
				)
			}
			s.addVerifyPending(group.TGGroupID, m.ID, pending)
			verifyMsg := tgbotapi.NewMessage(msg.Chat.ID, verifyText)
			verifyMsg.ReplyMarkup = keyboard
			_, _ = bot.Send(verifyMsg)
			_ = s.repo.CreateLog(group.ID, "join_verify_pending", 0, 0)
			go s.verifyTimeoutKick(bot, group.TGGroupID, m.ID, time.Duration(timeout)*time.Second)
		}
	}

	welcomeEnabled, err := s.IsFeatureEnabled(group.ID, featureWelcome, true)
	if err != nil || !welcomeEnabled {
		return err
	}
	text, err := s.getWelcomeText(group.ID)
	if err != nil {
		return err
	}
	mentions := formatWelcomeMentions(msg.NewChatMembers)
	if strings.Contains(text, "{user}") {
		text = strings.ReplaceAll(text, "{user}", mentions)
	} else if mentions != "" {
		text = fmt.Sprintf("%s\n%s", text, mentions)
	}
	_, err = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
	return err
}
func (s *Service) PassVerification(bot *tgbotapi.BotAPI, tgGroupID, tgUserID, actorID int64, answer *int) error {
	if actorID != tgUserID {
		return errors.New("only target user can verify")
	}
	pending, ok := s.getVerifyPending(tgGroupID, tgUserID)
	if !ok || time.Now().After(pending.Deadline) {
		s.popVerifyPending(tgGroupID, tgUserID)
		return errors.New("verification expired")
	}
	if pending.Mode == "math" {
		if answer == nil || *answer != pending.Answer {
			return errors.New("wrong answer")
		}
	}
	if !s.popVerifyPending(tgGroupID, tgUserID) {
		return errors.New("verification expired")
	}

	perms := &tgbotapi.ChatPermissions{
		CanSendMessages:       true,
		CanSendMediaMessages:  true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
	}
	_, err := bot.Request(tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
		Permissions:      perms,
	})
	if err != nil {
		return err
	}
	if group, gErr := s.repo.FindGroupByTGID(tgGroupID); gErr == nil {
		_ = s.repo.CreateLog(group.ID, "join_verify_pass", 0, 0)
	}
	return nil
}
func (s *Service) markJoin(tgGroupID, tgUserID int64) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.joinAt[key] = time.Now()
}

func (s *Service) getJoinAt(tgGroupID, tgUserID int64) (time.Time, bool) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.joinAt[key]
	return t, ok
}

func (s *Service) clearJoinAt(tgGroupID, tgUserID int64) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.joinAt, key)
}

func (s *Service) addVerifyPending(tgGroupID, tgUserID int64, p verifyPending) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verify[key] = p
}

func (s *Service) getVerifyPending(tgGroupID, tgUserID int64) (verifyPending, bool) {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.verify[key]
	return p, ok
}

func (s *Service) popVerifyPending(tgGroupID, tgUserID int64) bool {
	key := fmt.Sprintf("%d:%d", tgGroupID, tgUserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.verify[key]
	if ok {
		delete(s.verify, key)
	}
	return ok
}

func (s *Service) verifyTimeoutKick(bot *tgbotapi.BotAPI, tgGroupID, tgUserID int64, after time.Duration) {
	time.Sleep(after)
	if !s.popVerifyPending(tgGroupID, tgUserID) {
		return
	}
	_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
		UntilDate:        time.Now().Add(24 * time.Hour).Unix(),
	})
	_, _ = bot.Send(tgbotapi.NewMessage(tgGroupID, fmt.Sprintf("用户 %d 验证超时，已移出群组", tgUserID)))
	if group, err := s.repo.FindGroupByTGID(tgGroupID); err == nil {
		_ = s.repo.CreateLog(group.ID, "join_verify_timeout_kick", 0, 0)
	}
}
func buildMathOptions(answer int) []int {
	opts := map[int]struct{}{answer: {}}
	for len(opts) < 4 {
		delta := rand.Intn(7) - 3
		if delta == 0 {
			continue
		}
		v := answer + delta
		if v > 0 {
			opts[v] = struct{}{}
		}
	}
	out := make([]int, 0, len(opts))
	for k := range opts {
		out = append(out, k)
	}
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func formatWelcomeMentions(users []tgbotapi.User) string {
	mentions := make([]string, 0, len(users))
	for _, u := range users {
		if u.UserName != "" {
			mentions = append(mentions, "@"+u.UserName)
		} else {
			name := strings.TrimSpace(u.FirstName + " " + u.LastName)
			if name != "" {
				mentions = append(mentions, name)
			}
		}
	}
	return strings.Join(mentions, " ")
}
