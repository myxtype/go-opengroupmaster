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

var ErrVerifyWrongAnswer = errors.New("wrong answer")
var chineseCaptchaPool = []string{"中", "文", "验", "证", "群", "聊", "机", "器", "人", "安", "全", "风", "火", "山", "海", "云", "星", "龙", "虎", "盾"}

type verifyChallengeOptions struct {
	mode          string
	tgGroupID     int64
	tgUserID      int64
	target        *tgbotapi.User
	timeoutMins   int
	timeoutAction string
	retry         bool
	allowFallback bool
}

type verifyChallengePayload struct {
	mode       string
	answer     string
	text       string
	entities   []tgbotapi.MessageEntity
	markup     tgbotapi.InlineKeyboardMarkup
	photoName  string
	photoBytes []byte
	audioName  string
	audioBytes []byte
}

func buildVerifyChallenge(opts verifyChallengeOptions) (verifyChallengePayload, error) {
	if opts.timeoutMins <= 0 {
		opts.timeoutMins = 1
	}
	target := opts.target
	if target == nil {
		target = &tgbotapi.User{ID: opts.tgUserID, FirstName: "该用户"}
	}
	buildButton := func(suffix string) verifyChallengePayload {
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		return verifyChallengePayload{
			mode:     "button",
			text:     text,
			entities: entities,
			markup: tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("我已验证", fmt.Sprintf("verify:button:%d:%d", opts.tgGroupID, opts.tgUserID)),
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
		if opts.retry {
			suffix = fmt.Sprintf(" 回答错误，请重新完成算术验证：%d + %d = ?（剩余 %d 分钟）", a, b, opts.timeoutMins)
		}
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		options := buildMathOptions(answer)
		row := make([]tgbotapi.InlineKeyboardButton, 0, len(options))
		for _, opt := range options {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(opt), fmt.Sprintf("verify:math:%d:%d:%d", opts.tgGroupID, opts.tgUserID, opt)))
		}
		return verifyChallengePayload{
			mode:     "math",
			answer:   strconv.Itoa(answer),
			text:     text,
			entities: entities,
			markup:   tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(row...)),
		}, nil
	case "captcha":
		code, imgBytes, err := buildCaptchaImage()
		if err != nil || strings.TrimSpace(code) == "" || len(imgBytes) == 0 {
			if opts.allowFallback {
				suffix := fmt.Sprintf(" 请点击按钮完成验证（%d 分钟内）", opts.timeoutMins)
				if opts.retry {
					suffix = fmt.Sprintf(" 回答错误，请点击按钮完成验证（剩余 %d 分钟）", opts.timeoutMins)
				}
				return buildButton(suffix), nil
			}
			return verifyChallengePayload{}, errors.New("build captcha failed")
		}
		suffix := fmt.Sprintf(" 请点击与图片验证码一致的数字（%d 分钟内）", opts.timeoutMins)
		if opts.retry {
			suffix = fmt.Sprintf(" 回答错误，请重新点击与图片验证码一致的数字（剩余 %d 分钟）", opts.timeoutMins)
		}
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		options := buildCaptchaOptions(code)
		return verifyChallengePayload{
			mode:     "captcha",
			answer:   code,
			text:     text,
			entities: entities,
			markup: tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(options[0], fmt.Sprintf("verify:captcha:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[0])),
					tgbotapi.NewInlineKeyboardButtonData(options[1], fmt.Sprintf("verify:captcha:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[1])),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(options[2], fmt.Sprintf("verify:captcha:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[2])),
					tgbotapi.NewInlineKeyboardButtonData(options[3], fmt.Sprintf("verify:captcha:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[3])),
				),
			),
			photoName:  "verify_captcha.png",
			photoBytes: imgBytes,
		}, nil
	case "zhchar":
		ch, imgBytes, err := buildChineseCaptchaImage()
		if err != nil || strings.TrimSpace(ch) == "" || len(imgBytes) == 0 {
			if opts.allowFallback {
				suffix := fmt.Sprintf(" 请点击按钮完成验证（%d 分钟内）", opts.timeoutMins)
				if opts.retry {
					suffix = fmt.Sprintf(" 回答错误，请点击按钮完成验证（剩余 %d 分钟）", opts.timeoutMins)
				}
				return buildButton(suffix), nil
			}
			return verifyChallengePayload{}, errors.New("build zhchar captcha failed")
		}
		suffix := fmt.Sprintf(" 请点击与图片验证码一致的中文字符（%d 分钟内）", opts.timeoutMins)
		if opts.retry {
			suffix = fmt.Sprintf(" 回答错误，请重新点击与图片验证码一致的中文字符（剩余 %d 分钟）", opts.timeoutMins)
		}
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		options := buildChineseCaptchaOptions(ch)
		return verifyChallengePayload{
			mode:     "zhchar",
			answer:   ch,
			text:     text,
			entities: entities,
			markup: tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(options[0], fmt.Sprintf("verify:zhchar:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[0])),
					tgbotapi.NewInlineKeyboardButtonData(options[1], fmt.Sprintf("verify:zhchar:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[1])),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(options[2], fmt.Sprintf("verify:zhchar:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[2])),
					tgbotapi.NewInlineKeyboardButtonData(options[3], fmt.Sprintf("verify:zhchar:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[3])),
				),
			),
			photoName:  "verify_zhchar.png",
			photoBytes: imgBytes,
		}, nil
	case "zhvoice":
		code, audioBytes, err := buildAudioCaptcha("zh")
		if err != nil || strings.TrimSpace(code) == "" || len(audioBytes) == 0 {
			if opts.allowFallback {
				suffix := fmt.Sprintf(" 请点击按钮完成验证（%d 分钟内）", opts.timeoutMins)
				if opts.retry {
					suffix = fmt.Sprintf(" 回答错误，请点击按钮完成验证（剩余 %d 分钟）", opts.timeoutMins)
				}
				return buildButton(suffix), nil
			}
			return verifyChallengePayload{}, errors.New("build zhvoice captcha failed")
		}
		suffix := fmt.Sprintf(" 请收听语音验证码并点击对应数字（%d 分钟内）", opts.timeoutMins)
		if opts.retry {
			suffix = fmt.Sprintf(" 回答错误，请重新收听语音验证码并点击对应数字（剩余 %d 分钟）", opts.timeoutMins)
		}
		text, entities := composeTextWithUserMention("新成员 ", target, suffix)
		options := buildCaptchaOptions(code)
		return verifyChallengePayload{
			mode:     "zhvoice",
			answer:   code,
			text:     text,
			entities: entities,
			markup: tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(options[0], fmt.Sprintf("verify:zhvoice:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[0])),
					tgbotapi.NewInlineKeyboardButtonData(options[1], fmt.Sprintf("verify:zhvoice:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[1])),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(options[2], fmt.Sprintf("verify:zhvoice:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[2])),
					tgbotapi.NewInlineKeyboardButtonData(options[3], fmt.Sprintf("verify:zhvoice:%d:%d:%s", opts.tgGroupID, opts.tgUserID, options[3])),
				),
			),
			audioName:  "verify_zhvoice.wav",
			audioBytes: audioBytes,
		}, nil
	default:
		if opts.retry {
			return buildButton(fmt.Sprintf(" 回答错误，请点击按钮完成验证（剩余 %d 分钟）", opts.timeoutMins)), nil
		}
		return buildButton(fmt.Sprintf(" 请在 %d 分钟内完成验证，否则将%s。", opts.timeoutMins, verifyTimeoutActionText(opts.timeoutAction))), nil
	}
}

func sendVerifyChallenge(bot *tgbotapi.BotAPI, chatID int64, payload verifyChallengePayload) (int, error) {
	if bot == nil {
		return 0, errors.New("bot is nil")
	}
	if len(payload.photoBytes) > 0 {
		name := strings.TrimSpace(payload.photoName)
		if name == "" {
			name = "verify_captcha.png"
		}
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: name, Bytes: payload.photoBytes})
		photo.Caption = payload.text
		photo.CaptionEntities = payload.entities
		photo.ReplyMarkup = payload.markup
		sent, err := bot.Send(photo)
		if err != nil {
			return 0, err
		}
		return sent.MessageID, nil
	}
	if len(payload.audioBytes) > 0 {
		name := strings.TrimSpace(payload.audioName)
		if name == "" {
			name = "verify_audio.wav"
		}
		doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: name, Bytes: payload.audioBytes})
		doc.Caption = payload.text
		doc.CaptionEntities = payload.entities
		doc.ReplyMarkup = payload.markup
		sent, err := bot.Send(doc)
		if err != nil {
			return 0, err
		}
		return sent.MessageID, nil
	}
	msg := tgbotapi.NewMessage(chatID, payload.text)
	msg.Entities = payload.entities
	msg.ReplyMarkup = payload.markup
	sent, err := bot.Send(msg)
	if err != nil {
		return 0, err
	}
	return sent.MessageID, nil
}

func (s *Service) OnNewMembers(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) error {
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
			restrictUntil := deadline
			if newbieRestrict && newbieDeadline.After(restrictUntil) {
				restrictUntil = newbieDeadline
			}
			target := m
			restrict := tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: msg.Chat.ID, UserID: m.ID},
				UntilDate:        restrictUntil.Unix(),
				Permissions:      &tgbotapi.ChatPermissions{},
			}
			_, _ = bot.Request(restrict)

			pending := verifyPending{
				TGGroupID:     group.TGGroupID,
				TGUserID:      m.ID,
				Deadline:      deadline,
				RestrictUntil: restrictUntil,
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
			s.wakeJoinVerifyWorker()
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

// PassVerification 验证用户
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
	if pending.Mode == "math" || pending.Mode == "captcha" || pending.Mode == "zhchar" || pending.Mode == "zhvoice" {
		if strings.TrimSpace(answer) == "" || strings.TrimSpace(answer) != pending.Answer {
			if err := s.refreshVerifyChallenge(bot, pending); err != nil {
				s.logger.Printf("refresh verify challenge failed group=%d user=%d mode=%s: %v", tgGroupID, tgUserID, pending.Mode, err)
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

	if pending.RestrictUntil.After(time.Now()) {
		if err := s.restrictMemberNoSpeak(bot, tgGroupID, tgUserID, pending.RestrictUntil); err != nil {
			return err
		}
	} else {
		if err := s.restoreMemberSpeak(bot, tgGroupID, tgUserID); err != nil {
			return err
		}
	}
	if pending.MessageID > 0 {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(tgGroupID, pending.MessageID))
	}

	group, gErr := s.repo.FindGroupByTGID(tgGroupID)
	if gErr == nil {
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

func (s *Service) refreshVerifyChallenge(bot *tgbotapi.BotAPI, pending verifyPending) error {
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
		target:        &tgbotapi.User{ID: pending.TGUserID, FirstName: "该用户"},
		timeoutMins:   remainMins,
		timeoutAction: pending.TimeoutAction,
		retry:         true,
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
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(pending.TGGroupID, pending.MessageID))
	}
	return s.addVerifyPending(newPending)
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
		RestrictUntil: p.RestrictUntil,
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
		RestrictUntil: row.RestrictUntil,
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
