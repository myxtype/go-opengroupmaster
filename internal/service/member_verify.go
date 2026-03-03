package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"time"

	"supervisor/internal/model"

	"github.com/mojocn/base64Captcha"
	"gorm.io/gorm"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var ErrVerifyWrongAnswer = errors.New("wrong answer")
var chineseCaptchaPool = []string{"中", "文", "验", "证", "群", "聊", "机", "器", "人", "安", "全", "风", "火", "山", "海", "云", "星", "龙", "虎", "盾"}

const (
	verifyFailLimit          = 3
	verifyPermanentMuteHours = 24 * 365 * 10
)

type verifyChallengeOptions struct {
	mode          string
	tgGroupID     int64
	tgUserID      int64
	target        *models.User
	timeoutMins   int
	timeoutAction string
	allowFallback bool
}

type verifyChallengePayload struct {
	mode       string
	answer     string
	text       string
	entities   []models.MessageEntity
	markup     models.InlineKeyboardMarkup
	photoName  string
	photoBytes []byte
	audioName  string
	audioBytes []byte
}

func inlineKeyboardButtonData(text, data string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, CallbackData: data}
}

func inlineKeyboardButtonURL(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, URL: url}
}

func inlineKeyboardRow(buttons ...models.InlineKeyboardButton) []models.InlineKeyboardButton {
	return buttons
}

func inlineKeyboardMarkup(rows ...[]models.InlineKeyboardButton) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func buildVerifyChallenge(opts verifyChallengeOptions) (verifyChallengePayload, error) {
	if opts.timeoutMins <= 0 {
		opts.timeoutMins = 1
	}
	target := opts.target
	if target == nil {
		target = &models.User{ID: opts.tgUserID, FirstName: "该用户"}
	}
	buildButton := func(suffix string) verifyChallengePayload {
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		return verifyChallengePayload{
			mode:     "button",
			text:     text,
			entities: entities,
			markup: inlineKeyboardMarkup(
				inlineKeyboardRow(
					inlineKeyboardButtonData("我已验证", fmt.Sprintf("verify:button:%d:%d", opts.tgGroupID, opts.tgUserID)),
				),
			),
		}
	}

	switch opts.mode {
	case "math":
		a := rand.Intn(9) + 1
		b := rand.Intn(9) + 1
		answer := a + b
		suffix := fmt.Sprintf(" 请完成算术验证：%d + %d = ?（%d 分钟内）", a, b, opts.timeoutMins)
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		options := buildMathOptions(answer)
		row := make([]models.InlineKeyboardButton, 0, len(options))
		for _, opt := range options {
			row = append(row, inlineKeyboardButtonData(strconv.Itoa(opt), fmt.Sprintf("verify:math:%d:%d:%d", opts.tgGroupID, opts.tgUserID, opt)))
		}
		return verifyChallengePayload{
			mode:     "math",
			answer:   strconv.Itoa(answer),
			text:     text,
			entities: entities,
			markup:   inlineKeyboardMarkup(inlineKeyboardRow(row...)),
		}, nil
	case "captcha":
		code, imgBytes, err := buildCaptchaImage()
		if err != nil || strings.TrimSpace(code) == "" || len(imgBytes) == 0 {
			if opts.allowFallback {
				return buildButton(fmt.Sprintf(" 请点击按钮完成验证（%d 分钟内）", opts.timeoutMins)), nil
			}
			return verifyChallengePayload{}, errors.New("build captcha failed")
		}
		suffix := fmt.Sprintf(" 请点击与图片验证码一致的数字（%d 分钟内）", opts.timeoutMins)
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		options := buildCaptchaOptions(code)
		return verifyChallengePayload{
			mode:     "captcha",
			answer:   code,
			text:     text,
			entities: entities,
			markup: inlineKeyboardMarkup(
				inlineKeyboardRow(
					inlineKeyboardButtonData(options[0], fmt.Sprintf("verify:captcha:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[0])),
					inlineKeyboardButtonData(options[1], fmt.Sprintf("verify:captcha:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[1])),
				),
				inlineKeyboardRow(
					inlineKeyboardButtonData(options[2], fmt.Sprintf("verify:captcha:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[2])),
					inlineKeyboardButtonData(options[3], fmt.Sprintf("verify:captcha:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[3])),
				),
			),
			photoName:  "verify_captcha.png",
			photoBytes: imgBytes,
		}, nil
	case "zhchar":
		ch, imgBytes, err := buildChineseCaptchaImage()
		if err != nil || strings.TrimSpace(ch) == "" || len(imgBytes) == 0 {
			if opts.allowFallback {
				return buildButton(fmt.Sprintf(" 请点击按钮完成验证（%d 分钟内）", opts.timeoutMins)), nil
			}
			return verifyChallengePayload{}, errors.New("build zhchar captcha failed")
		}
		suffix := fmt.Sprintf(" 请点击与图片验证码一致的中文字符（%d 分钟内）", opts.timeoutMins)
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		options := buildChineseCaptchaOptions(ch)
		return verifyChallengePayload{
			mode:     "zhchar",
			answer:   ch,
			text:     text,
			entities: entities,
			markup: inlineKeyboardMarkup(
				inlineKeyboardRow(
					inlineKeyboardButtonData(options[0], fmt.Sprintf("verify:zhchar:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[0])),
					inlineKeyboardButtonData(options[1], fmt.Sprintf("verify:zhchar:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[1])),
				),
				inlineKeyboardRow(
					inlineKeyboardButtonData(options[2], fmt.Sprintf("verify:zhchar:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[2])),
					inlineKeyboardButtonData(options[3], fmt.Sprintf("verify:zhchar:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[3])),
				),
			),
			photoName:  "verify_zhchar.png",
			photoBytes: imgBytes,
		}, nil
	case "zhvoice":
		code, audioBytes, err := buildAudioCaptcha("zh")
		if err != nil || strings.TrimSpace(code) == "" || len(audioBytes) == 0 {
			if opts.allowFallback {
				return buildButton(fmt.Sprintf(" 请点击按钮完成验证（%d 分钟内）", opts.timeoutMins)), nil
			}
			return verifyChallengePayload{}, errors.New("build zhvoice captcha failed")
		}
		suffix := fmt.Sprintf(" 请收听语音验证码并点击对应数字（%d 分钟内）", opts.timeoutMins)
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		options := buildCaptchaOptions(code)
		return verifyChallengePayload{
			mode:     "zhvoice",
			answer:   code,
			text:     text,
			entities: entities,
			markup: inlineKeyboardMarkup(
				inlineKeyboardRow(
					inlineKeyboardButtonData(options[0], fmt.Sprintf("verify:zhvoice:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[0])),
					inlineKeyboardButtonData(options[1], fmt.Sprintf("verify:zhvoice:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[1])),
				),
				inlineKeyboardRow(
					inlineKeyboardButtonData(options[2], fmt.Sprintf("verify:zhvoice:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[2])),
					inlineKeyboardButtonData(options[3], fmt.Sprintf("verify:zhvoice:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[3])),
				),
			),
			audioName:  "verify_zhvoice.wav",
			audioBytes: audioBytes,
		}, nil
	default:
		return buildButton(fmt.Sprintf(" 请在 %d 分钟内完成验证，否则将%s。", opts.timeoutMins, verifyTimeoutActionText(opts.timeoutAction))), nil
	}
}

func sendVerifyChallenge(bot *tgbot.Bot, chatID int64, payload verifyChallengePayload) (int, error) {
	if bot == nil {
		return 0, errors.New("bot is nil")
	}
	if len(payload.photoBytes) > 0 {
		name := strings.TrimSpace(payload.photoName)
		if name == "" {
			name = "verify_captcha.png"
		}
		photo := &tgbot.SendPhotoParams{
			ChatID:          chatID,
			Photo:           &models.InputFileUpload{Filename: name, Data: bytes.NewReader(payload.photoBytes)},
			Caption:         payload.text,
			CaptionEntities: payload.entities,
			ReplyMarkup:     payload.markup,
		}
		sent, err := bot.SendPhoto(context.Background(), photo)
		if err != nil {
			return 0, err
		}
		return sent.ID, nil
	}
	if len(payload.audioBytes) > 0 {
		name := strings.TrimSpace(payload.audioName)
		if name == "" {
			name = "verify_audio.wav"
		}
		doc := &tgbot.SendDocumentParams{
			ChatID:          chatID,
			Document:        &models.InputFileUpload{Filename: name, Data: bytes.NewReader(payload.audioBytes)},
			Caption:         payload.text,
			CaptionEntities: payload.entities,
			ReplyMarkup:     payload.markup,
		}
		sent, err := bot.SendDocument(context.Background(), doc)
		if err != nil {
			return 0, err
		}
		return sent.ID, nil
	}
	sent, err := bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        payload.text,
		Entities:    payload.entities,
		ReplyMarkup: payload.markup,
	})
	if err != nil {
		return 0, err
	}
	return sent.ID, nil
}

func (s *Service) OnNewMembers(bot *tgbot.Bot, msg *models.Message) error {
	if len(msg.NewChatMembers) == 0 {
		return nil
	}
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		return nil
	}
	joinAt := time.Unix(int64(msg.Date), 0)
	if msg.Date <= 0 {
		joinAt = time.Now()
	}
	for _, m := range msg.NewChatMembers {
		_, _ = s.repo.UpsertUserFromTG(&m)
	}

	// 新成员限制
	newbieDeadline, newbieRestrict, newbieErr := s.newbieLimitRestrictionDeadline(group.ID, joinAt)
	if newbieErr != nil {
		s.logger.Printf("compute newbie limit deadline failed group=%d: %v", group.ID, newbieErr)
	}

	// 进群验证
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
			_, _ = bot.RestrictChatMember(context.Background(), &tgbot.RestrictChatMemberParams{
				ChatID:      msg.Chat.ID,
				UserID:      m.ID,
				UntilDate:   int(deadline.Unix()),
				Permissions: &models.ChatPermissions{},
			})

			pending := verifyPending{
				TGGroupID:     group.TGGroupID,
				TGUserID:      m.ID,
				Deadline:      deadline,
				Mode:          cfg.Type,
				TimeoutAction: cfg.TimeoutAction,
			}
			challenge, cErr := buildVerifyChallenge(verifyChallengeOptions{
				mode:          cfg.Type,
				tgGroupID:     group.TGGroupID,
				tgUserID:      m.ID,
				target:        &target,
				timeoutMins:   timeoutMins,
				timeoutAction: cfg.TimeoutAction,
				allowFallback: true,
			})
			if cErr != nil {
				s.logger.Printf("build verify challenge failed group=%d user=%d mode=%s: %v", group.TGGroupID, m.ID, cfg.Type, cErr)
				continue
			}
			pending.Mode = challenge.mode
			pending.Answer = challenge.answer
			msgID, sendErr := sendVerifyChallenge(bot, msg.Chat.ID, challenge)
			if sendErr != nil {
				s.logger.Printf("send verify challenge failed group=%d user=%d mode=%s: %v", group.TGGroupID, m.ID, pending.Mode, sendErr)
				continue
			}
			pending.MessageID = msgID
			if err := s.addVerifyPending(pending); err != nil {
				s.logger.Printf("upsert join verify pending failed group=%d user=%d: %v", group.TGGroupID, m.ID, err)
				continue
			}
			_ = s.repo.CreateLog(group.ID, "join_verify_pending", 0, 0)
		}
	}

	// 新成员限制
	if (err != nil || !verifyEnabled) && newbieRestrict {
		for _, m := range msg.NewChatMembers {
			if m.IsBot {
				continue
			}
			if err := s.restrictMemberNoSpeak(bot, msg.Chat.ID, m.ID, newbieDeadline); err != nil {
				s.logger.Printf("restrict newbie member failed group=%d user=%d: %v", msg.Chat.ID, m.ID, err)
				continue
			}
			_ = s.repo.CreateLog(group.ID, "newbie_limit_restrict", 0, 0)
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
	// 在入群模式下，欢迎消息发送给新成员
	if cfg.Mode == "join" {
		users := make([]models.User, 0, len(msg.NewChatMembers))
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

// PassVerification 验证用户
func (s *Service) PassVerification(bot *tgbot.Bot, cb *models.CallbackQuery, tgGroupID, tgUserID int64, mode string, answer string) error {
	if cb.From.ID != tgUserID {
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
	if slices.Contains([]string{"math", "captcha", "zhchar", "zhvoice"}, pending.Mode) {
		if strings.TrimSpace(answer) == "" || strings.TrimSpace(answer) != pending.Answer {
			pending.FailCount++
			if pending.FailCount >= verifyFailLimit {
				popped, popErr := s.popVerifyPendingByID(pending.ID)
				if popErr != nil {
					return popErr
				}
				if !popped {
					return errors.New("verification expired")
				}
				s.applyVerifyTimeout(bot, pending)
				return errors.New("verification expired")
			}
			if err := s.refreshVerifyChallenge(bot, cb, pending); err != nil {
				s.logger.Printf("refresh verify challenge failed group=%d user=%d mode=%s: %v", tgGroupID, tgUserID, pending.Mode, err)
				if saveErr := s.addVerifyPending(pending); saveErr != nil {
					s.logger.Printf("persist verify fail count failed group=%d user=%d mode=%s: %v", tgGroupID, tgUserID, pending.Mode, saveErr)
				}
			}
			return ErrVerifyWrongAnswer
		}
	}
	popped, err := s.popVerifyPendingByID(pending.ID)
	if err != nil {
		return err
	}
	if !popped {
		return errors.New("verification expired")
	}

	now := time.Now()
	group, gErr := s.repo.FindGroupByTGID(tgGroupID)
	keepRestricted := false
	restrictUntil := now
	if gErr == nil {
		if newbieUntil, ok, nErr := s.newbieRestrictionUntil(group.ID, now, now); nErr == nil && ok {
			keepRestricted = true
			restrictUntil = newbieUntil
		}
	}
	if keepRestricted {
		if err := s.restrictMemberNoSpeak(bot, tgGroupID, tgUserID, restrictUntil); err != nil {
			return err
		}
	} else if err := s.restoreMemberSpeak(bot, tgGroupID, tgUserID); err != nil {
		return err
	}
	if pending.MessageID > 0 {
		_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{ChatID: tgGroupID, MessageID: pending.MessageID})
	}

	if gErr == nil {
		_ = s.repo.CreateLog(group.ID, "join_verify_pass", 0, 0)
		welcomeEnabled, wErr := s.IsFeatureEnabled(group.ID, featureWelcome, true)
		if wErr == nil && welcomeEnabled {
			cfg, cErr := s.getWelcomeConfig(group.ID)
			if cErr == nil && cfg.Mode == "verify" {
				member, mErr := bot.GetChatMember(context.Background(), &tgbot.GetChatMemberParams{
					ChatID: tgGroupID,
					UserID: tgUserID,
				})
				if mErr == nil {
					if u := chatMemberUser(*member); u != nil {
						_ = s.sendWelcome(bot, tgGroupID, group.ID, []models.User{*u}, cfg)
					}
				}
			}
		}
	}
	return nil
}

func (s *Service) refreshVerifyChallenge(bot *tgbot.Bot, cb *models.CallbackQuery, pending verifyPending) error {
	if bot == nil {
		return errors.New("bot is nil")
	}
	remainMins := int(time.Until(pending.Deadline).Minutes())
	if remainMins <= 0 {
		remainMins = 1
	}
	challenge, err := buildVerifyChallenge(verifyChallengeOptions{
		mode:          pending.Mode,
		tgGroupID:     pending.TGGroupID,
		tgUserID:      pending.TGUserID,
		target:        &cb.From,
		timeoutMins:   remainMins,
		timeoutAction: pending.TimeoutAction,
		allowFallback: true,
	})
	if err != nil {
		return err
	}
	newPending := pending
	newPending.Mode = challenge.mode
	newPending.Answer = challenge.answer
	msgID, sendErr := sendVerifyChallenge(bot, pending.TGGroupID, challenge)
	if sendErr != nil {
		return sendErr
	}
	newPending.MessageID = msgID

	if pending.MessageID > 0 {
		_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{ChatID: pending.TGGroupID, MessageID: pending.MessageID})
	}
	return s.addVerifyPending(newPending)
}

func (s *Service) addVerifyPending(p verifyPending) error {
	return s.repo.UpsertJoinVerifyPending(&model.JoinVerifyPending{
		TGGroupID:     p.TGGroupID,
		TGUserID:      p.TGUserID,
		Mode:          p.Mode,
		Answer:        p.Answer,
		FailCount:     p.FailCount,
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
		FailCount:     row.FailCount,
		MessageID:     row.MessageID,
		TimeoutAction: row.TimeoutAction,
	}, true, nil
}

func (s *Service) newbieRestrictionUntil(groupID uint, joinedAt, now time.Time) (time.Time, bool, error) {
	enabled, err := s.IsFeatureEnabled(groupID, featureNewbieLimit, false)
	if err != nil {
		return time.Time{}, false, err
	}
	if !enabled {
		return time.Time{}, false, nil
	}
	minutes, err := s.getNewbieLimitMinutes(groupID)
	if err != nil {
		return time.Time{}, false, err
	}
	if minutes <= 0 {
		return time.Time{}, false, nil
	}
	if joinedAt.IsZero() {
		joinedAt = now
	}
	until := joinedAt.Add(time.Duration(minutes) * time.Minute)
	if !until.After(now) {
		return until, false, nil
	}
	return until, true, nil
}

func (s *Service) popVerifyPendingByID(id uint) (bool, error) {
	return s.repo.DeleteJoinVerifyPendingByID(id)
}

func (s *Service) applyVerifyTimeout(bot *tgbot.Bot, pending verifyPending) {
	tgGroupID := pending.TGGroupID
	tgUserID := pending.TGUserID
	if pending.TimeoutAction == "kick" {
		_, _ = bot.BanChatMember(context.Background(), &tgbot.BanChatMemberParams{
			ChatID:    tgGroupID,
			UserID:    tgUserID,
			UntilDate: int(time.Now().Add(1 * time.Minute).Unix()),
		})
		_, _ = bot.UnbanChatMember(context.Background(), &tgbot.UnbanChatMemberParams{
			ChatID:       tgGroupID,
			UserID:       tgUserID,
			OnlyIfBanned: true,
		})
	} else {
		_, _ = bot.RestrictChatMember(context.Background(), &tgbot.RestrictChatMemberParams{
			ChatID:      tgGroupID,
			UserID:      tgUserID,
			UntilDate:   int(time.Now().Add(verifyPermanentMuteHours * time.Hour).Unix()),
			Permissions: &models.ChatPermissions{},
		})
	}
	if pending.MessageID > 0 {
		_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{ChatID: tgGroupID, MessageID: pending.MessageID})
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
	opts := map[string]struct{}{answer: {}}
	for len(opts) < 4 {
		opts[chineseCaptchaPool[rand.Intn(len(chineseCaptchaPool))]] = struct{}{}
	}
	out := make([]string, 0, len(opts))
	for k := range opts {
		out = append(out, k)
	}
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func buildChineseCaptchaImage() (string, []byte, error) {
	source := strings.Join(chineseCaptchaPool, ",")
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
	return "永久禁言"
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

func buildAudioCaptcha(language string) (string, []byte, error) {
	driver := base64Captcha.NewDriverAudio(4, language)
	captcha := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore)
	_, b64s, answer, err := captcha.Generate()
	if err != nil {
		return "", nil, err
	}
	encoded := b64s
	if i := strings.Index(encoded, ","); i >= 0 && i+1 < len(encoded) {
		encoded = encoded[i+1:]
	}
	audioBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(answer), audioBytes, nil
}

func (s *Service) sendWelcome(bot *tgbot.Bot, chatID int64, groupID uint, users []models.User, cfg welcomeConfig) error {
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

func (s *Service) sendWelcomePreview(bot *tgbot.Bot, chatID int64, users []models.User, cfg welcomeConfig) error {
	_, err := s.sendWelcomeMessage(bot, chatID, users, cfg)
	return err
}

func (s *Service) sendWelcomeMessage(bot *tgbot.Bot, chatID int64, users []models.User, cfg welcomeConfig) (int, error) {
	text, entities := buildWelcomeTextWithMentions(cfg.Text, users)

	var markup any
	if len(cfg.ButtonRows) > 0 {
		rows := make([][]models.InlineKeyboardButton, 0, len(cfg.ButtonRows))
		for _, rowCfg := range cfg.ButtonRows {
			row := make([]models.InlineKeyboardButton, 0, len(rowCfg))
			for _, btn := range rowCfg {
				if strings.TrimSpace(btn.Text) == "" || strings.TrimSpace(btn.URL) == "" {
					continue
				}
				row = append(row, inlineKeyboardButtonURL(btn.Text, btn.URL))
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
		if len(rows) > 0 {
			markup = inlineKeyboardMarkup(rows...)
		}
	}

	sentMessageID := 0
	if strings.TrimSpace(cfg.MediaFileID) != "" {
		photo := &tgbot.SendPhotoParams{
			ChatID:          chatID,
			Photo:           &models.InputFileString{Data: cfg.MediaFileID},
			Caption:         text,
			CaptionEntities: entities,
		}
		if m, ok := markup.(models.InlineKeyboardMarkup); ok {
			photo.ReplyMarkup = m
		}
		msg, err := bot.SendPhoto(context.Background(), photo)
		if err != nil {
			return 0, err
		}
		sentMessageID = msg.ID
	} else {
		message := &tgbot.SendMessageParams{
			ChatID:   chatID,
			Text:     text,
			Entities: entities,
		}
		if m, ok := markup.(models.InlineKeyboardMarkup); ok {
			message.ReplyMarkup = m
		}
		msg, err := bot.SendMessage(context.Background(), message)
		if err != nil {
			return 0, err
		}
		sentMessageID = msg.ID
	}
	return sentMessageID, nil
}
