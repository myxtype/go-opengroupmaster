package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

	if msg.Text != "" {
		// 关键词监控
		_ = s.notifyKeywordMonitor(bot, group, msg)

		// 抽奖关键词
		if msg.From != nil {
			matched, joined, err := s.TryJoinLotteryByKeyword(group, msg.From, msg.Text)
			if err != nil {
				return err
			}
			if matched {
				deleteAfter := time.Duration(0)
				if mins, cfgErr := s.LotteryDeleteKeywordMinutesByGroupID(group.ID); cfgErr == nil && mins > 0 {
					deleteAfter = time.Duration(mins) * time.Minute
					s.ScheduleMessageDelete(msg.Chat.ID, msg.MessageID, deleteAfter)
				}
				if joined {
					reply := tgbotapi.NewMessage(msg.Chat.ID, "参与抽奖成功")
					reply.ReplyToMessageID = msg.MessageID
					replyMsg, sendErr := bot.Send(reply)
					if sendErr == nil && deleteAfter > 0 {
						s.ScheduleMessageDelete(msg.Chat.ID, replyMsg.MessageID, deleteAfter)
					}
				}
				return nil
			}
		}

		// 自动回复匹配
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

	// 增加积分
	if msg.From != nil {
		u, err := s.repo.UpsertUserFromTG(msg.From)
		if err == nil {
			_ = s.repo.AddPoints(group.ID, u.ID, 1)
		}
	}

	return nil
}

// applyModeration 统一处理消息的违规检查和处理
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

	// 违禁词
	handled, err := s.applyBannedWordModeration(bot, msg, group)
	if err != nil {
		return false, err
	}
	if handled {
		return true, nil
	}

	// 反垃圾逻辑
	spamState, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return false, err
	}
	if spamState.Enabled {
		cfg := normalizeAntiSpamConfig(spamState.Config)
		blocked, reasonCode, reasonLabel := antiSpamViolation(msg, cfg)
		decisionSource := "rule"

		// 规则先判定；未命中规则且开启 AI 时，使用 AI 二分类补充判断。
		if !blocked && cfg.AIEnabled {
			aiResult, fromCache, aiErr := s.classifyAntiSpamWithAI(msg)
			if aiErr != nil {
				decisionSource = "rule_fallback"
				if s.logger != nil {
					s.logger.Printf("anti spam ai fallback chat=%d msg=%d err=%v", msg.Chat.ID, msg.MessageID, aiErr)
				}
			} else if aiResult.IsSpamBy(cfg.AISpamScore) {
				blocked = true
				reasonCode = "ai"
				decisionSource = "ai"
				if fromCache {
					decisionSource = "ai_cache"
				}
				reasonLabel = fmt.Sprintf("AI:%d分 %s", aiResult.Score, strings.TrimSpace(aiResult.Reason))
			}
		}

		if blocked {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			appliedPenalty := cfg.Penalty
			actionLabel := moderationPenaltyActionLabel(appliedPenalty, cfg.MuteMinutes, cfg.BanMinutes)
			targetID := uint(0)
			if msg.From != nil {
				if u, upsertErr := s.repo.UpsertUserFromTG(msg.From); upsertErr == nil {
					targetID = u.ID
				}
			}
			if msg.From == nil && (cfg.Penalty == antiFloodPenaltyWarn || cfg.Penalty == antiFloodPenaltyMute || cfg.Penalty == antiFloodPenaltyKick || cfg.Penalty == antiFloodPenaltyKickBan) {
				appliedPenalty = antiFloodPenaltyDeleteOnly
				actionLabel = moderationPenaltyActionLabel(appliedPenalty, cfg.MuteMinutes, cfg.BanMinutes)
			} else if msg.From != nil {
				muteMinutes := cfg.MuteMinutes
				banMinutes := cfg.BanMinutes
				if targetID > 0 {
					appliedPenalty, actionLabel, muteMinutes, banMinutes = s.resolveWarnablePenalty(
						group.ID,
						targetID,
						moderationPenaltyConfig{
							Penalty:               cfg.Penalty,
							WarnThreshold:         cfg.WarnThreshold,
							WarnAction:            cfg.WarnAction,
							WarnActionMuteMinutes: cfg.WarnActionMuteMinutes,
							WarnActionBanMinutes:  cfg.WarnActionBanMinutes,
							MuteMinutes:           cfg.MuteMinutes,
							BanMinutes:            cfg.BanMinutes,
						},
						s.repo.CountAntiSpamWarnsSinceLastAction,
						"anti_spam_warn",
						"anti_spam_warn_action_applied",
					)
				}
				if appliedPenalty != antiFloodPenaltyWarn {
					applyPenaltyToMember(bot, msg.Chat.ID, msg.From.ID, appliedPenalty, muteMinutes, banMinutes)
				}
			}
			if cfg.WarnDeleteSec != -1 {
				reasonText := strings.TrimSpace(reasonLabel)
				if reasonText == "" {
					reasonText = "规则判定"
				}
				alertText := fmt.Sprintf("%s 正在发送垃圾消息。\n原因：%s\n处理：%s", antiSpamActorDisplayName(msg), reasonText, actionLabel)
				var alertEntities []tgbotapi.MessageEntity
				if msg.From != nil {
					alertText, alertEntities = composeAntiSpamAlertWithMention(msg.From, reasonLabel, actionLabel)
				}
				alert := tgbotapi.NewMessage(msg.Chat.ID, alertText)
				alert.Entities = alertEntities
				if msg.From != nil {
					alert.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("管理员解禁", fmt.Sprintf("feat:mod:spamunlock:%d:%d", msg.Chat.ID, msg.From.ID)),
						),
					)
				}
				alertMsg, sendErr := bot.Send(alert)
				if sendErr == nil && cfg.WarnDeleteSec > 0 {
					s.ScheduleMessageDelete(msg.Chat.ID, alertMsg.MessageID, time.Duration(cfg.WarnDeleteSec)*time.Second)
				}
			}
			logReason := safeActionToken(reasonCode)
			if logReason == "" {
				logReason = "unknown"
			}
			logSource := safeActionToken(decisionSource)
			if logSource == "" {
				logSource = "rule"
			}
			_ = s.repo.CreateLog(group.ID, fmt.Sprintf("anti_spam_%s_%s_%s", appliedPenalty, logSource, logReason), 0, targetID)
			return true, nil
		}
	}

	if msg.From == nil {
		return false, nil
	}

	// 反刷屏逻辑
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return false, err
	}
	if state.Enabled {
		cfg := normalizeAntiFloodConfig(state.Config)
		flooding, reason := s.isFlooding(group.TGGroupID, msg.From.ID, msg.Text, cfg)
		if flooding {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
			appliedPenalty := cfg.Penalty
			actionLabel := moderationPenaltyActionLabel(appliedPenalty, cfg.MuteMinutes, cfg.BanMinutes)
			targetID := uint(0)
			if u, upsertErr := s.repo.UpsertUserFromTG(msg.From); upsertErr == nil {
				targetID = u.ID
			}
			muteMinutes := cfg.MuteMinutes
			banMinutes := cfg.BanMinutes
			if targetID > 0 {
				appliedPenalty, actionLabel, muteMinutes, banMinutes = s.resolveWarnablePenalty(
					group.ID,
					targetID,
					moderationPenaltyConfig{
						Penalty:               cfg.Penalty,
						WarnThreshold:         cfg.WarnThreshold,
						WarnAction:            cfg.WarnAction,
						WarnActionMuteMinutes: cfg.WarnActionMuteMinutes,
						WarnActionBanMinutes:  cfg.WarnActionBanMinutes,
						MuteMinutes:           cfg.MuteMinutes,
						BanMinutes:            cfg.BanMinutes,
					},
					s.repo.CountAntiFloodWarnsSinceLastAction,
					"anti_flood_warn",
					"anti_flood_warn_action_applied",
				)
			}
			if appliedPenalty != antiFloodPenaltyWarn {
				applyPenaltyToMember(bot, msg.Chat.ID, msg.From.ID, appliedPenalty, muteMinutes, banMinutes)
			}
			alertText := fmt.Sprintf("%s 触发反刷屏，已%s", floodUserDisplayName(msg.From), actionLabel)
			if reason == "high_freq" {
				alertText = fmt.Sprintf("%s（%d秒内%d条）", alertText, cfg.WindowSec, cfg.MaxMessages)
			}
			alert, sendErr := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, alertText))
			if sendErr == nil && cfg.WarnDeleteSec > 0 {
				s.ScheduleMessageDelete(msg.Chat.ID, alert.MessageID, time.Duration(cfg.WarnDeleteSec)*time.Second)
			}
			_ = s.repo.CreateLog(group.ID, "anti_flood_"+appliedPenalty+"_"+reason, 0, targetID)
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) applyBannedWordModeration(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, group *model.Group) (bool, error) {
	if msg.Text == "" {
		return false, nil
	}
	bwEnabled, bwCfg, err := s.bannedWordStateByGroupID(group.ID)
	if err != nil {
		return false, err
	}
	if !bwEnabled {
		return false, nil
	}
	banned, err := s.repo.ContainsBannedWord(group.ID, msg.Text)
	if err != nil {
		return false, err
	}
	if !banned {
		return false, nil
	}

	_, _ = bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	alertText := "消息触发违禁词，已撤回"
	var alertEntities []tgbotapi.MessageEntity
	logTargetID := uint(0)
	appliedPenalty := antiFloodPenaltyDeleteOnly
	actionLabel := moderationPenaltyActionLabel(appliedPenalty, bwCfg.MuteMinutes, bwCfg.BanMinutes)
	if msg.From != nil {
		appliedPenalty = bwCfg.Penalty
		u, upsertErr := s.repo.UpsertUserFromTG(msg.From)
		if upsertErr != nil {
			muteMinutes := bwCfg.MuteMinutes
			banMinutes := bwCfg.BanMinutes
			actionLabel = moderationPenaltyActionLabel(appliedPenalty, muteMinutes, banMinutes)
			if appliedPenalty != antiFloodPenaltyWarn {
				applyPenaltyToMember(bot, msg.Chat.ID, msg.From.ID, appliedPenalty, muteMinutes, banMinutes)
			}
		} else {
			logTargetID = u.ID
			var muteMinutes int
			var banMinutes int
			appliedPenalty, actionLabel, muteMinutes, banMinutes = s.resolveWarnablePenalty(
				group.ID,
				u.ID,
				moderationPenaltyConfig{
					Penalty:               bwCfg.Penalty,
					WarnThreshold:         bwCfg.WarnThreshold,
					WarnAction:            bwCfg.WarnAction,
					WarnActionMuteMinutes: bwCfg.WarnActionMuteMinutes,
					WarnActionBanMinutes:  bwCfg.WarnActionBanMinutes,
					MuteMinutes:           bwCfg.MuteMinutes,
					BanMinutes:            bwCfg.BanMinutes,
				},
				s.repo.CountBannedWordWarnsSinceLastAction,
				"banned_word_warn",
				"banned_word_warn_action_applied",
			)
			if appliedPenalty != antiFloodPenaltyWarn {
				applyPenaltyToMember(bot, msg.Chat.ID, msg.From.ID, appliedPenalty, muteMinutes, banMinutes)
			}
		}
		alertText, alertEntities = composeTextWithUserMention("", msg.From, fmt.Sprintf(" 消息触发违禁词，已%s", actionLabel))
	}
	_ = s.repo.CreateLog(group.ID, "banned_word_penalty_"+appliedPenalty, 0, logTargetID)
	alert := tgbotapi.NewMessage(msg.Chat.ID, alertText)
	alert.Entities = alertEntities
	alertMsg, sendErr := bot.Send(alert)
	if sendErr == nil && bwCfg.WarnDeleteMinutes > 0 {
		s.ScheduleMessageDelete(msg.Chat.ID, alertMsg.MessageID, time.Duration(bwCfg.WarnDeleteMinutes)*time.Minute)
	}
	_ = s.repo.CreateLog(group.ID, "banned_word_delete", 0, logTargetID)
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

func applyPenaltyToMember(bot *tgbotapi.BotAPI, tgGroupID, tgUserID int64, penalty string, muteMinutes, banMinutes int) {
	switch penalty {
	case antiFloodPenaltyMute:
		restrict := tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
			UntilDate:        time.Now().Add(time.Duration(muteMinutes) * time.Minute).Unix(),
			Permissions:      &tgbotapi.ChatPermissions{},
		}
		_, _ = bot.Request(restrict)
	case antiFloodPenaltyKick:
		_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
			UntilDate:        time.Now().Add(1 * time.Minute).Unix(),
		})
		_, _ = bot.Request(tgbotapi.UnbanChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
			OnlyIfBanned:     true,
		})
	case antiFloodPenaltyKickBan:
		_, _ = bot.Request(tgbotapi.BanChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: tgGroupID, UserID: tgUserID},
			RevokeMessages:   true,
			UntilDate:        time.Now().Add(time.Duration(banMinutes) * time.Minute).Unix(),
		})
	}
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

func (s *Service) classifyAntiSpamWithAI(msg *tgbotapi.Message) (spamAIResult, bool, error) {
	if s.spamAI == nil {
		return spamAIResult{}, false, errors.New("nil ai classifier")
	}
	cacheTTL := s.spamAICacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 7 * 24 * time.Hour
	}
	now := time.Now()
	hash := antiSpamContentHash(msg)
	if hash == "" {
		return spamAIResult{}, false, errors.New("empty content hash")
	}
	if cached, err := s.repo.FindAISpamCache(msg.Chat.ID, hash, now.Add(-cacheTTL)); err == nil && cached != nil {
		var out spamAIResult
		if uErr := json.Unmarshal([]byte(cached.ResultJSON), &out); uErr == nil {
			if normalized, nErr := out.Normalized(); nErr == nil {
				return normalized, true, nil
			}
		}
	}

	result, _, err := s.spamAI.Classify(context.Background(), spamAIInput{Content: antiSpamMessageContent(msg)})
	if err != nil {
		return spamAIResult{}, false, err
	}
	normalized, err := result.Normalized()
	if err != nil {
		return spamAIResult{}, false, err
	}
	if payload, mErr := json.Marshal(normalized); mErr == nil {
		_ = s.repo.UpsertAISpamCache(msg.Chat.ID, hash, string(payload), now)
	}
	if msg != nil && msg.MessageID > 0 && msg.MessageID%200 == 0 {
		_ = s.repo.DeleteAISpamCacheBefore(now.Add(-cacheTTL))
	}
	return normalized, false, nil
}

func antiSpamContentHash(msg *tgbotapi.Message) string {
	if msg == nil {
		return ""
	}
	content := normalizeSpamText(antiSpamMessageContent(msg))
	parts := []string{
		content,
		fmt.Sprintf("photo:%d", len(msg.Photo)),
		fmt.Sprintf("video:%t", msg.Video != nil),
		fmt.Sprintf("animation:%t", msg.Animation != nil),
		fmt.Sprintf("document:%t", msg.Document != nil),
		fmt.Sprintf("sticker:%t", msg.Sticker != nil),
		fmt.Sprintf("voice:%t", msg.Voice != nil),
		fmt.Sprintf("audio:%t", msg.Audio != nil),
		fmt.Sprintf("sender_chat:%d", antiSpamChatID(msg.SenderChat)),
		fmt.Sprintf("forward_chat:%d", antiSpamChatID(msg.ForwardFromChat)),
		fmt.Sprintf("auto_forward:%t", msg.IsAutomaticForward),
	}
	raw := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func antiSpamChatID(chat *tgbotapi.Chat) int64 {
	if chat == nil {
		return 0
	}
	return chat.ID
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

func safeActionToken(v string) string {
	value := strings.ToLower(strings.TrimSpace(v))
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if len(out) > 32 {
		out = out[:32]
	}
	return out
}
