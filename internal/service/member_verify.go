package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mojocn/base64Captcha"
	"gorm.io/gorm"
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
			now := time.Now()
			deadline := now.Add(timeout)
			target := m
			restrict := tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: m.ID},
				UntilDate:        deadline.Unix(),
				Permissions:      &tgbotapi.ChatPermissions{},
			}
			_, _ = bot.Request(restrict)

			pending := verifyPending{
				TGGroupID:     group.TGGroupID,
				TGUserID:      m.ID,
				Deadline:      deadline,
				Mode:          cfg.Type,
				TimeoutAction: cfg.TimeoutAction,
			}
			verifyText, verifyEntities := composeTextWithUserMention("新成员 ", &target, fmt.Sprintf(" 请在 %d 分钟内完成验证，否则将%s。", timeoutMins, verifyTimeoutActionText(cfg.TimeoutAction)))
			keyboard := tgbotapi.NewInlineKeyboardMarkup()
			switch cfg.Type {
			case "math":
				a := rand.Intn(9) + 1
				b := rand.Intn(9) + 1
				answer := a + b
				pending.Answer = strconv.Itoa(answer)
				verifyText, verifyEntities = composeTextWithUserMention("新成员 ", &target, fmt.Sprintf(" 请完成算术验证：%d + %d = ?（%d 分钟内）", a, b, timeoutMins))
				options := buildMathOptions(answer)
				row := make([]tgbotapi.InlineKeyboardButton, 0, len(options))
				for _, opt := range options {
					row = append(row, tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(opt), fmt.Sprintf("verify:math:%d:%d:%d", group.TGGroupID, m.ID, opt)))
				}
				keyboard = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...))
			case "captcha":
				captchaCode, imgBytes, imgErr := buildCaptchaImage()
				if imgErr == nil && strings.TrimSpace(captchaCode) != "" && len(imgBytes) > 0 {
					pending.Answer = captchaCode
					verifyText, verifyEntities = composeTextWithUserMention("新成员 ", &target, fmt.Sprintf(" 请点击与图片验证码一致的数字（%d 分钟内）", timeoutMins))
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
					photo.CaptionEntities = verifyEntities
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
					verifyText, verifyEntities = composeTextWithUserMention("新成员 ", &target, fmt.Sprintf(" 请点击按钮完成验证（%d 分钟内）", timeoutMins))
				}
			case "zhchar":
				captchaChar, imgBytes, imgErr := buildChineseCaptchaImage()
				if imgErr == nil && strings.TrimSpace(captchaChar) != "" && len(imgBytes) > 0 {
					pending.Answer = captchaChar
					verifyText, verifyEntities = composeTextWithUserMention("新成员 ", &target, fmt.Sprintf(" 请点击与图片验证码一致的中文字符（%d 分钟内）", timeoutMins))
					options := buildChineseCaptchaOptions(captchaChar)
					keyboard = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData(options[0], fmt.Sprintf("verify:zhchar:%d:%d:%s", group.TGGroupID, m.ID, options[0])),
							tgbotapi.NewInlineKeyboardButtonData(options[1], fmt.Sprintf("verify:zhchar:%d:%d:%s", group.TGGroupID, m.ID, options[1])),
						),
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData(options[2], fmt.Sprintf("verify:zhchar:%d:%d:%s", group.TGGroupID, m.ID, options[2])),
							tgbotapi.NewInlineKeyboardButtonData(options[3], fmt.Sprintf("verify:zhchar:%d:%d:%s", group.TGGroupID, m.ID, options[3])),
						),
					)
					photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FileBytes{Name: "verify_chinese_captcha.png", Bytes: imgBytes})
					photo.Caption = verifyText
					photo.CaptionEntities = verifyEntities
					photo.ReplyMarkup = keyboard
					if sent, sendErr := bot.Send(photo); sendErr == nil {
						pending.MessageID = sent.MessageID
					}
				} else {
					// Fallback: Chinese captcha generation failed, degrade to button verification.
					pending.Mode = "button"
					keyboard = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("我已验证", fmt.Sprintf("verify:button:%d:%d", group.TGGroupID, m.ID)),
						),
					)
					verifyText, verifyEntities = composeTextWithUserMention("新成员 ", &target, fmt.Sprintf(" 请点击按钮完成验证（%d 分钟内）", timeoutMins))
				}
			default:
				keyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("我已验证", fmt.Sprintf("verify:button:%d:%d", group.TGGroupID, m.ID)),
					),
				)
			}
			if pending.MessageID == 0 {
				verifyMsg := tgbotapi.NewMessage(msg.Chat.ID, verifyText)
				verifyMsg.Entities = verifyEntities
				verifyMsg.ReplyMarkup = keyboard
				if sent, sendErr := bot.Send(verifyMsg); sendErr == nil {
					pending.MessageID = sent.MessageID
				}
			}
			if err := s.addVerifyPending(pending); err != nil {
				s.logger.Printf("upsert join verify pending failed group=%d user=%d: %v", group.TGGroupID, m.ID, err)
				continue
			}
			_ = s.repo.CreateLog(group.ID, "join_verify_pending", 0, 0)
			s.wakeJoinVerifyWorker()
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
	pending, ok, err := s.getVerifyPending(tgGroupID, tgUserID)
	if err != nil {
		return err
	}
	if !ok || time.Now().After(pending.Deadline) {
		return errors.New("verification expired")
	}
	if pending.Mode != mode {
		return errors.New("wrong verify mode")
	}
	if pending.Mode == "math" || pending.Mode == "captcha" || pending.Mode == "zhchar" {
		if strings.TrimSpace(answer) == "" || strings.TrimSpace(answer) != pending.Answer {
			return errors.New("wrong answer")
		}
	}
	popped, err := s.popVerifyPendingByID(pending.ID)
	if err != nil {
		return err
	}
	if !popped {
		return errors.New("verification expired")
	}

	perms := &tgbotapi.ChatPermissions{
		CanSendMessages:       true,
		CanSendMediaMessages:  true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
	}
	_, err = bot.Request(tgbotapi.RestrictChatMemberConfig{
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

func (s *Service) addVerifyPending(p verifyPending) error {
	return s.repo.UpsertJoinVerifyPending(&model.JoinVerifyPending{
		TGGroupID:     p.TGGroupID,
		TGUserID:      p.TGUserID,
		Mode:          p.Mode,
		Answer:        p.Answer,
		MessageID:     p.MessageID,
		TimeoutAction: p.TimeoutAction,
		Deadline:      p.Deadline,
	})
}

func (s *Service) getVerifyPending(tgGroupID, tgUserID int64) (verifyPending, bool, error) {
	row, err := s.repo.GetJoinVerifyPending(tgGroupID, tgUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return verifyPending{}, false, nil
		}
		return verifyPending{}, false, err
	}
	return verifyPending{
		ID:            row.ID,
		TGGroupID:     row.TGGroupID,
		TGUserID:      row.TGUserID,
		Deadline:      row.Deadline,
		Mode:          row.Mode,
		Answer:        row.Answer,
		MessageID:     row.MessageID,
		TimeoutAction: row.TimeoutAction,
	}, true, nil
}

func (s *Service) popVerifyPendingByID(id uint) (bool, error) {
	return s.repo.DeleteJoinVerifyPendingByID(id)
}

func (s *Service) applyVerifyTimeout(bot *tgbotapi.BotAPI, pending verifyPending) {
	tgGroupID := pending.TGGroupID
	tgUserID := pending.TGUserID
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

func buildChineseCaptchaOptions(answer string) []string {
	pool := chineseCaptchaPool()
	opts := map[string]struct{}{answer: {}}
	for len(opts) < 4 {
		opts[pool[rand.Intn(len(pool))]] = struct{}{}
	}
	out := make([]string, 0, len(opts))
	for k := range opts {
		out = append(out, k)
	}
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func buildChineseCaptchaImage() (string, []byte, error) {
	source := strings.Join(chineseCaptchaPool(), ",")
	driver := base64Captcha.NewDriverChinese(
		80,
		240,
		16,
		base64Captcha.OptionShowHollowLine|base64Captcha.OptionShowSineLine,
		1,
		source,
		nil,
		nil,
		[]string{"wqy-microhei.ttc"},
	)
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

func chineseCaptchaPool() []string {
	return []string{"中", "文", "验", "证", "群", "聊", "机", "器", "人", "安", "全", "风", "火", "山", "海", "云", "星", "龙", "虎", "盾"}
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

func (s *Service) sendWelcome(bot *tgbotapi.BotAPI, chatID int64, groupID uint, users []tgbotapi.User, cfg welcomeConfig) error {
	sentMessageID, err := s.sendWelcomeMessage(bot, chatID, users, cfg)
	if err != nil {
		return err
	}
	_ = s.repo.CreateLog(groupID, "welcome_sent_"+cfg.Mode, 0, 0)
	if cfg.DeleteMinutes > 0 && sentMessageID > 0 {
		s.ScheduleMessageDelete(chatID, sentMessageID, time.Duration(cfg.DeleteMinutes)*time.Minute)
	}
	return nil
}

func (s *Service) sendWelcomePreview(bot *tgbotapi.BotAPI, chatID int64, users []tgbotapi.User, cfg welcomeConfig) error {
	_, err := s.sendWelcomeMessage(bot, chatID, users, cfg)
	return err
}

func (s *Service) sendWelcomeMessage(bot *tgbotapi.BotAPI, chatID int64, users []tgbotapi.User, cfg welcomeConfig) (int, error) {
	text, entities := buildWelcomeTextWithMentions(cfg.Text, users)

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
		photo.CaptionEntities = entities
		if m, ok := markup.(tgbotapi.InlineKeyboardMarkup); ok {
			photo.ReplyMarkup = m
		}
		msg, err := bot.Send(photo)
		if err != nil {
			return 0, err
		}
		sentMessageID = msg.MessageID
	} else {
		message := tgbotapi.NewMessage(chatID, text)
		message.Entities = entities
		if m, ok := markup.(tgbotapi.InlineKeyboardMarkup); ok {
			message.ReplyMarkup = m
		}
		msg, err := bot.Send(message)
		if err != nil {
			return 0, err
		}
		sentMessageID = msg.MessageID
	}
	return sentMessageID, nil
}
