package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mojocn/base64Captcha"
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
		member := m
		_, _ = s.repo.UpsertUserFromTG(&member)
	}

	verifyEnabled, err := s.IsFeatureEnabled(group.ID, featureJoinVerify, false)
	if err == nil && verifyEnabled {
		cfg, _ := s.getJoinVerifyConfig(group.ID)
		timeoutMins := cfg.TimeoutMinutes
		if timeoutMins <= 0 {
			timeoutMins = 5
		}
		timeout := time.Duration(timeoutMins) * time.Minute
		for _, m := range msg.NewChatMembers {
			if m.IsBot {
				continue
			}
			restrict := tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: m.ID},
				UntilDate:        time.Now().Add(timeout).Unix(),
				Permissions:      &tgbotapi.ChatPermissions{},
			}
			_, _ = bot.Request(restrict)

			pending := verifyPending{
				Deadline:      time.Now().Add(timeout),
				Mode:          cfg.Type,
				TimeoutAction: cfg.TimeoutAction,
			}
			verifyText := fmt.Sprintf("新成员 %s 请在 %d 分钟内完成验证，否则将%s。", verifyUserDisplayName(&m), timeoutMins, verifyTimeoutActionText(cfg.TimeoutAction))
			keyboard := tgbotapi.NewInlineKeyboardMarkup()
			if cfg.Type == "math" {
				a := rand.Intn(9) + 1
				b := rand.Intn(9) + 1
				answer := a + b
				pending.Answer = strconv.Itoa(answer)
				verifyText = fmt.Sprintf("新成员 %s 请完成算术验证：%d + %d = ?（%d 分钟内）", verifyUserDisplayName(&m), a, b, timeoutMins)
				options := buildMathOptions(answer)
				row := make([]tgbotapi.InlineKeyboardButton, 0, len(options))
				for _, opt := range options {
					row = append(row, tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(opt), fmt.Sprintf("verify:math:%d:%d:%d", group.TGGroupID, m.ID, opt)))
				}
				keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
			} else if cfg.Type == "captcha" {
				captchaCode, imgBytes, imgErr := buildCaptchaImage()
				if imgErr == nil && strings.TrimSpace(captchaCode) != "" && len(imgBytes) > 0 {
					pending.Answer = captchaCode
					verifyText = fmt.Sprintf("新成员 %s 请点击与图片验证码一致的数字（%d 分钟内）", verifyUserDisplayName(&m), timeoutMins)
					options := buildCaptchaOptions(captchaCode)
					keyboard = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData(options[0], fmt.Sprintf("verify:captcha:%d:%d:%s", group.TGGroupID, m.ID, options[0])),
							tgbotapi.NewInlineKeyboardButtonData(options[1], fmt.Sprintf("verify:captcha:%d:%d:%s", group.TGGroupID, m.ID, options[1])),
						),
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData(options[2], fmt.Sprintf("verify:captcha:%d:%d:%s", group.TGGroupID, m.ID, options[2])),
							tgbotapi.NewInlineKeyboardButtonData(options[3], fmt.Sprintf("verify:captcha:%d:%d:%s", group.TGGroupID, m.ID, options[3])),
						),
					)
					photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FileBytes{Name: "verify_captcha.png", Bytes: imgBytes})
					photo.Caption = verifyText
					photo.ReplyMarkup = keyboard
					if sent, sendErr := bot.Send(photo); sendErr == nil {
						pending.MessageID = sent.MessageID
					}
				} else {
					// Fallback: captcha generation failed, degrade to button verification.
					pending.Mode = "button"
					keyboard = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("我已验证", fmt.Sprintf("verify:button:%d:%d", group.TGGroupID, m.ID)),
						),
					)
					verifyText = fmt.Sprintf("新成员 %s 请点击按钮完成验证（%d 分钟内）", verifyUserDisplayName(&m), timeoutMins)
				}
			} else {
				keyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("我已验证", fmt.Sprintf("verify:button:%d:%d", group.TGGroupID, m.ID)),
					),
				)
			}
			if pending.MessageID == 0 {
				verifyMsg := tgbotapi.NewMessage(msg.Chat.ID, verifyText)
				verifyMsg.ReplyMarkup = keyboard
				if sent, sendErr := bot.Send(verifyMsg); sendErr == nil {
					pending.MessageID = sent.MessageID
				}
			}
			s.addVerifyPending(group.TGGroupID, m.ID, pending)
			_ = s.repo.CreateLog(group.ID, "join_verify_pending", 0, 0)
			go s.verifyTimeoutHandle(bot, group.TGGroupID, m.ID, timeout)
		}
	}

	welcomeEnabled, err := s.IsFeatureEnabled(group.ID, featureWelcome, true)
	if err != nil || !welcomeEnabled {
		return err
	}
	cfg, err := s.getWelcomeConfig(group.ID)
	if err != nil {
		return err
	}
	if cfg.Mode == "join" {
		users := make([]tgbotapi.User, 0, len(msg.NewChatMembers))
		for _, m := range msg.NewChatMembers {
			if m.IsBot {
				continue
			}
			users = append(users, m)
		}
		if len(users) > 0 {
			return s.sendWelcome(bot, msg.Chat.ID, group.ID, users, cfg)
		}
	}
	return nil
}
func (s *Service) PassVerification(bot *tgbotapi.BotAPI, tgGroupID, tgUserID, actorID int64, mode string, answer string) error {
	if actorID != tgUserID {
		return errors.New("only target user can verify")
	}
	pending, ok := s.getVerifyPending(tgGroupID, tgUserID)
	if !ok || time.Now().After(pending.Deadline) {
		s.popVerifyPending(tgGroupID, tgUserID)
		return errors.New("verification expired")
	}
	if pending.Mode != mode {
		return errors.New("wrong verify mode")
	}
	if pending.Mode == "math" || pending.Mode == "captcha" {
		if strings.TrimSpace(answer) == "" || strings.TrimSpace(answer) != pending.Answer {
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
	if pending.MessageID > 0 {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(tgGroupID, pending.MessageID))
	}

	if group, gErr := s.repo.FindGroupByTGID(tgGroupID); gErr == nil {
		_ = s.repo.CreateLog(group.ID, "join_verify_pass", 0, 0)
		welcomeEnabled, wErr := s.IsFeatureEnabled(group.ID, featureWelcome, true)
		if wErr == nil && welcomeEnabled {
			cfg, cErr := s.getWelcomeConfig(group.ID)
			if cErr == nil && cfg.Mode == "verify" {
				member, mErr := bot.GetChatMember(tgbotapi.GetChatMemberConfig{
					ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: tgGroupID, UserID: tgUserID},
				})
				if mErr == nil && member.User != nil {
					_ = s.sendWelcome(bot, tgGroupID, group.ID, []tgbotapi.User{*member.User}, cfg)
				}
			}
		}
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

func (s *Service) verifyTimeoutHandle(bot *tgbotapi.BotAPI, tgGroupID, tgUserID int64, after time.Duration) {
	time.Sleep(after)
	pending, ok := s.getVerifyPending(tgGroupID, tgUserID)
	if !ok {
		return
	}
	if !s.popVerifyPending(tgGroupID, tgUserID) {
		return
	}
	if pending.TimeoutAction == "kick" {
		_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
			UntilDate:        time.Now().Add(1 * time.Minute).Unix(),
		})
		_, _ = bot.Request(tgbotapi.UnbanChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
			OnlyIfBanned:     true,
		})
		_, _ = bot.Send(tgbotapi.NewMessage(tgGroupID, fmt.Sprintf("用户 %d 验证超时，已移出群组", tgUserID)))
	} else {
		_, _ = bot.Request(tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
			UntilDate:        time.Now().Add(24 * time.Hour).Unix(),
			Permissions:      &tgbotapi.ChatPermissions{},
		})
		_, _ = bot.Send(tgbotapi.NewMessage(tgGroupID, fmt.Sprintf("用户 %d 验证超时，已禁言 24 小时", tgUserID)))
	}
	if pending.MessageID > 0 {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(tgGroupID, pending.MessageID))
	}
	if group, err := s.repo.FindGroupByTGID(tgGroupID); err == nil {
		action := "mute"
		if pending.TimeoutAction == "kick" {
			action = "kick"
		}
		_ = s.repo.CreateLog(group.ID, "join_verify_timeout_"+action, 0, 0)
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

func buildCaptchaOptions(answer string) []string {
	opts := map[string]struct{}{answer: {}}
	for len(opts) < 4 {
		v := randomDigits(4)
		opts[v] = struct{}{}
	}
	out := make([]string, 0, len(opts))
	for k := range opts {
		out = append(out, k)
	}
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func randomDigits(n int) string {
	if n <= 0 {
		n = 4
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(byte('0' + rand.Intn(10)))
	}
	return b.String()
}

func verifyUserDisplayName(u *tgbotapi.User) string {
	if u == nil {
		return "新成员"
	}
	if strings.TrimSpace(u.UserName) != "" {
		return "@" + strings.TrimSpace(u.UserName)
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name != "" {
		return name
	}
	return fmt.Sprintf("uid:%d", u.ID)
}

func verifyTimeoutActionText(action string) string {
	if action == "kick" {
		return "踢出"
	}
	return "禁言"
}

func buildCaptchaImage() (string, []byte, error) {
	driver := base64Captcha.NewDriverDigit(80, 240, 4, 0.7, 80)
	captcha := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore)
	_, b64s, answer, err := captcha.Generate()
	if err != nil {
		return "", nil, err
	}
	encoded := b64s
	if i := strings.Index(encoded, ","); i >= 0 && i+1 < len(encoded) {
		encoded = encoded[i+1:]
	}
	imgBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(answer), imgBytes, nil
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

func (s *Service) sendWelcome(bot *tgbotapi.BotAPI, chatID int64, groupID uint, users []tgbotapi.User, cfg welcomeConfig) error {
	mentions := formatWelcomeMentions(users)
	text := cfg.Text
	if strings.Contains(text, "{user}") {
		text = strings.ReplaceAll(text, "{user}", mentions)
	} else if mentions != "" {
		text = fmt.Sprintf("%s\n%s", text, mentions)
	}

	var markup any
	if len(cfg.ButtonRows) > 0 {
		rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(cfg.ButtonRows))
		for _, rowCfg := range cfg.ButtonRows {
			row := make([]tgbotapi.InlineKeyboardButton, 0, len(rowCfg))
			for _, btn := range rowCfg {
				if strings.TrimSpace(btn.Text) == "" || strings.TrimSpace(btn.URL) == "" {
					continue
				}
				row = append(row, tgbotapi.NewInlineKeyboardButtonURL(btn.Text, btn.URL))
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
		if len(rows) > 0 {
			markup = tgbotapi.NewInlineKeyboardMarkup(rows...)
		}
	}

	sentMessageID := 0
	if strings.TrimSpace(cfg.MediaFileID) != "" {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(cfg.MediaFileID))
		photo.Caption = text
		if m, ok := markup.(tgbotapi.InlineKeyboardMarkup); ok {
			photo.ReplyMarkup = m
		}
		msg, err := bot.Send(photo)
		if err != nil {
			return err
		}
		sentMessageID = msg.MessageID
	} else {
		message := tgbotapi.NewMessage(chatID, text)
		if m, ok := markup.(tgbotapi.InlineKeyboardMarkup); ok {
			message.ReplyMarkup = m
		}
		msg, err := bot.Send(message)
		if err != nil {
			return err
		}
		sentMessageID = msg.MessageID
	}

	_ = s.repo.CreateLog(groupID, "welcome_sent_"+cfg.Mode, 0, 0)
	if cfg.DeleteMinutes > 0 && sentMessageID > 0 {
		go func(chatID int64, messageID int, minutes int) {
			time.Sleep(time.Duration(minutes) * time.Minute)
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
		}(chatID, sentMessageID, cfg.DeleteMinutes)
	}
	return nil
}
