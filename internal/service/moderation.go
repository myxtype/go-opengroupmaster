package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"supervisor/internal/model"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gorm.io/gorm"
)

// 正则表达式模式定义
var urlPattern = regexp.MustCompile(`(?i)(?:\b(?:https?|ftp)://[^\s]+|\btg://[^\s]+|\bwww\.[^\s]+|\b(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}(?::\d{1,5})?(?:/[^\s]*)?)`)
var ethAddressPattern = regexp.MustCompile(`(?i)\b0x[a-f0-9]{40}\b`)
var mentionPattern = regexp.MustCompile(`@[A-Za-z0-9_]{2,}`)

// CheckMessageAndRespond 处理群组消息的主要入口
// 顺序：黑名单检查 -> 违规处理 -> 积分签到/命令 -> 关键词监控 -> 抽奖 -> 自动回复 -> 积分奖励 -> 词云统计
func (s *Service) CheckMessageAndRespond(bot *tgbot.Bot, msg *models.Message) error {
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	// 黑名单用户直接踢出群组
	blacklistedHandled, err := s.handleGroupBlacklistModeration(bot, msg, group)
	if err != nil {
		return err
	}
	if blacklistedHandled {
		return nil
	}

	// 违规检查和处理（违禁词、反垃圾、反刷屏、夜间模式等）
	handled, err := s.applyModeration(bot, msg, group)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	if msg.Text != "" {
		// 积分相关命令处理（签到、查询积分、积分排行）
		handled, err := s.handlePointsTextCommand(bot, group, msg)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}

		// 关键词监控（命中则私聊通知管理员）
		_ = s.notifyKeywordMonitor(bot, group, msg)

		// 抽奖关键词匹配（消耗积分参与）
		if msg.From != nil {
			matched, joined, err := s.TryJoinLotteryByKeyword(group, msg.From, msg.Text)
			if err != nil {
				if errors.Is(err, ErrInsufficientPoints) {
					cfg, cfgErr := s.getPointsConfig(group.ID)
					if cfgErr == nil {
						_, _ = bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
							ChatID: msg.Chat.ID,
							Text:   fmt.Sprintf("当前积分不足，参与抽奖需要 %d 积分。发送“%s”可查询积分。", cfg.LotteryCost, cfg.BalanceAlias),
						})
					} else {
						_, _ = bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
							ChatID: msg.Chat.ID,
							Text:   "当前积分不足，无法参与抽奖。",
						})
					}
					return nil
				}
				return err
			}
			if matched {
				// 配置了自动删除关键词消息则安排删除
				deleteAfter := time.Duration(0)
				if mins, cfgErr := s.LotteryDeleteKeywordMinutesByGroupID(group.ID); cfgErr == nil && mins > 0 {
					deleteAfter = time.Duration(mins) * time.Minute
					s.ScheduleMessageDelete(msg.Chat.ID, msg.ID, deleteAfter)
				}
				if joined {
					replyMsg, sendErr := bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
						ChatID: msg.Chat.ID,
						Text:   "参与抽奖成功",
						ReplyParameters: &models.ReplyParameters{
							MessageID: msg.ID,
						},
					})
					if sendErr == nil && deleteAfter > 0 {
						s.ScheduleMessageDelete(msg.Chat.ID, replyMsg.ID, deleteAfter)
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
			reply := &tgbot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   rule.Reply,
				ReplyParameters: &models.ReplyParameters{
					MessageID: msg.ID,
				},
			}
			if markup, ok := InlineKeyboardFromButtonRowsJSON(rule.ButtonRows); ok {
				reply.ReplyMarkup = markup
			}
			_, _ = bot.SendMessage(context.Background(), reply)
		}
	}

	// 发言增加积分（排除系统消息和命令）
	if msg.From != nil {
		_ = s.rewardMessagePoints(group, msg)
	}
	// 词云分词统计（仅在功能开启时生效）
	s.collectWordCloudMessage(msg, group)

	return nil
}

// CheckEditedMessageAndModerate 处理编辑消息的违规检查
// 注意：编辑消息仅做风控检测，避免重复触发积分/自动回复等流程
func (s *Service) CheckEditedMessageAndModerate(bot *tgbot.Bot, msg *models.Message) error {
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	blacklistedHandled, err := s.handleGroupBlacklistModeration(bot, msg, group)
	if err != nil {
		return err
	}
	if blacklistedHandled {
		return nil
	}
	_, err = s.applyModeration(bot, msg, group)
	return err
}

// handleGroupBlacklistModeration 处理黑名单用户的消息
// 黑名单用户消息会被直接删除并踢出群组（24小时封禁）
func (s *Service) handleGroupBlacklistModeration(bot *tgbot.Bot, msg *models.Message, group *model.Group) (bool, error) {
	if msg == nil || group == nil || msg.From == nil {
		return false, nil
	}
	blacklisted, err := s.repo.IsGroupBlacklisted(group.ID, msg.From.ID)
	if err != nil {
		return false, nil
	}
	if !blacklisted {
		return false, nil
	}
	// 删除消息并封禁用户
	_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID})
	_, _ = bot.BanChatMember(context.Background(), &tgbot.BanChatMemberParams{
		ChatID:    msg.Chat.ID,
		UserID:    msg.From.ID,
		UntilDate: int(time.Now().Add(24 * time.Hour).Unix()),
	})
	alertText, entities := composeTextWithUserMention("", msg.From, " 命中本群黑名单，已移出群组")
	_, _ = bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
		ChatID:   msg.Chat.ID,
		Text:     alertText,
		Entities: entities,
	})
	_ = s.repo.CreateLog(group.ID, "group_blacklist_kick", 0, 0)
	return true, nil
}

// applyModeration 统一处理消息的违规检查和处理
// 检查顺序：夜间模式 -> 违禁词 -> AI/规则反垃圾 -> 反刷屏
// 管理员消息和 SenderChat 信息不受限制
func (s *Service) applyModeration(bot *tgbot.Bot, msg *models.Message, group *model.Group) (bool, error) {
	if msg.From == nil && msg.SenderChat == nil {
		return false, nil
	}
	if msg.From != nil {
		// 管理员消息免检
		isAdmin, err := s.repo.CheckAdmin(group.ID, msg.From.ID)
		if err != nil {
			return false, err
		}
		if isAdmin {
			return false, nil
		}
	}

	// 夜间模式检查（00:00-08:00）
	nightState, err := s.getNightModeState(group.ID)
	if err != nil {
		return false, err
	}
	if nightState.Enabled {
		cfg := normalizeNightModeConfig(nightState.Config)
		if isNightWindowNow(cfg.TimezoneOffsetMinutes, cfg.StartHour, cfg.EndHour, time.Now()) {
			switch cfg.Mode {
			case nightModeGlobalMute:
				// 全局禁言：删除所有消息
				_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID})
				_ = s.repo.CreateLog(group.ID, "night_mode_global_mute_delete", 0, 0)
				return true, nil
			default:
				if isNightMediaMessage(msg) {
					// 媒体消息禁发
					_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID})
					_ = s.repo.CreateLog(group.ID, "night_mode_delete_media", 0, 0)
					return true, nil
				}
			}
		}
	}

	// 违禁词检测
	handled, err := s.applyBannedWordModeration(bot, msg, group)
	if err != nil {
		return false, err
	}
	if handled {
		return true, nil
	}

	// 反垃圾逻辑：规则引擎 + AI 二分类（可选）
	spamState, err := s.getAntiSpamState(group.ID)
	if err != nil {
		return false, err
	}
	if spamState.Enabled {
		cfg := normalizeAntiSpamConfig(spamState.Config)
		// 命中例外关键词直接跳过
		if !antiSpamExceptionMatched(msg, cfg.ExceptionKeywords) {
			blocked, reasonCode, reasonLabel := antiSpamViolation(msg, cfg)
			decisionSource := "rule"

			// 规则未命中且开启 AI 时，使用 AI 补充判断
			if !blocked && cfg.AIEnabled && s.antiSpamAIAvailable() {
				aiResult, fromCache, aiErr := s.classifyAntiSpamWithAI(msg, cfg.AIStrictness)
				if aiErr != nil {
					decisionSource = "rule_fallback"
					if s.logger != nil {
						s.logger.Printf("anti spam ai fallback chat=%d msg=%d err=%v", msg.Chat.ID, msg.ID, aiErr)
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
				// 删除消息
				_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID})
				appliedPenalty := cfg.Penalty
				actionLabel := moderationPenaltyActionLabel(appliedPenalty, cfg.MuteMinutes, cfg.BanMinutes)
				targetID := uint(0)
				if msg.From != nil {
					if u, upsertErr := s.repo.UpsertUserFromTG(msg.From); upsertErr == nil {
						targetID = u.ID
					}
				}
				// SenderChat 无法被惩罚，降级为仅删除
				if msg.From == nil && (cfg.Penalty == antiFloodPenaltyWarn || cfg.Penalty == antiFloodPenaltyMute || cfg.Penalty == antiFloodPenaltyKick || cfg.Penalty == antiFloodPenaltyKickBan) {
					appliedPenalty = antiFloodPenaltyDeleteOnly
					actionLabel = moderationPenaltyActionLabel(appliedPenalty, cfg.MuteMinutes, cfg.BanMinutes)
				} else if msg.From != nil {
					// 应用警告阈值和阶梯惩罚
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
					// 除警告外应用其他惩罚
					if appliedPenalty != antiFloodPenaltyWarn {
						applyPenaltyToMember(bot, msg.Chat.ID, msg.From.ID, appliedPenalty, muteMinutes, banMinutes)
					}
				}
				// 发送违规提醒消息（支持自动删除）
				if cfg.WarnDeleteSec != -1 {
					reasonText := strings.TrimSpace(reasonLabel)
					if reasonText == "" {
						reasonText = "规则判定"
					}
					alertText := fmt.Sprintf("%s 正在发送垃圾消息。\n原因：%s\n处理：%s", antiSpamActorDisplayName(msg), reasonText, actionLabel)
					var alertEntities []models.MessageEntity
					if msg.From != nil {
						alertText, alertEntities = composeAntiSpamAlertWithMention(msg.From, reasonLabel, actionLabel)
					}
					alert := &tgbot.SendMessageParams{ChatID: msg.Chat.ID, Text: alertText, Entities: alertEntities}
					if msg.From != nil {
						buttonLabel := "管理员解禁"
						buttonAction := fmt.Sprintf("feat:mod:spamunlock:%d:%d", msg.Chat.ID, msg.From.ID)
						if appliedPenalty == antiFloodPenaltyWarn {
							buttonLabel = "撤销警告"
							buttonAction = fmt.Sprintf("feat:mod:spamwarnrevoke:%d:%d", msg.Chat.ID, msg.From.ID)
						}
						alert.ReplyMarkup = models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{Text: buttonLabel, CallbackData: buttonAction}}}}
					}
					alertMsg, sendErr := bot.SendMessage(context.Background(), alert)
					if sendErr == nil && cfg.WarnDeleteSec > 0 {
						s.ScheduleMessageDelete(msg.Chat.ID, alertMsg.ID, time.Duration(cfg.WarnDeleteSec)*time.Second)
					}
				}
				// 记录审计日志
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
	}

	if msg.From == nil {
		return false, nil
	}

	// 反刷屏检测（基于时间窗口内的消息数量）
	state, err := s.getAntiFloodState(group.ID)
	if err != nil {
		return false, err
	}
	if state.Enabled {
		cfg := normalizeAntiFloodConfig(state.Config)
		flooding, reason := s.isFlooding(group.TGGroupID, msg.From.ID, msg.Text, cfg)
		if flooding {
			// 删除刷屏消息
			_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID})
			appliedPenalty := cfg.Penalty
			actionLabel := moderationPenaltyActionLabel(appliedPenalty, cfg.MuteMinutes, cfg.BanMinutes)
			targetID := uint(0)
			if u, upsertErr := s.repo.UpsertUserFromTG(msg.From); upsertErr == nil {
				targetID = u.ID
			}
			// 应用警告阈值和阶梯惩罚
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
			// 构建提醒消息
			alertText := fmt.Sprintf("%s 触发反刷屏，已%s", floodUserDisplayName(msg.From), actionLabel)
			if reason == "high_freq" {
				alertText = fmt.Sprintf("%s（%d秒内%d条）", alertText, cfg.WindowSec, cfg.MaxMessages)
			}
			alert := &tgbot.SendMessageParams{ChatID: msg.Chat.ID, Text: alertText}
			if msg.From != nil {
				buttonLabel := "管理员解禁"
				buttonAction := fmt.Sprintf("feat:mod:spamunlock:%d:%d", msg.Chat.ID, msg.From.ID)
				if appliedPenalty == antiFloodPenaltyWarn {
					buttonLabel = "撤销警告"
					buttonAction = fmt.Sprintf("feat:mod:floodwarnrevoke:%d:%d", msg.Chat.ID, msg.From.ID)
				}
				alert.ReplyMarkup = models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{Text: buttonLabel, CallbackData: buttonAction}}}}
			}
			alertMsg, sendErr := bot.SendMessage(context.Background(), alert)
			if sendErr == nil && cfg.WarnDeleteSec > 0 {
				s.ScheduleMessageDelete(msg.Chat.ID, alertMsg.ID, time.Duration(cfg.WarnDeleteSec)*time.Second)
			}
			_ = s.repo.CreateLog(group.ID, "anti_flood_"+appliedPenalty+"_"+reason, 0, targetID)
			return true, nil
		}
	}
	return false, nil
}

// applyBannedWordModeration 检查并处理违禁词
// 逻辑：检测违禁词 -> 删除消息 -> 应用惩罚（禁言/踢出/封禁）-> 发送提醒
func (s *Service) applyBannedWordModeration(bot *tgbot.Bot, msg *models.Message, group *model.Group) (bool, error) {
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
	// 检测违禁词
	banned, err := s.repo.ContainsBannedWord(group.ID, msg.Text)
	if err != nil {
		return false, err
	}
	if !banned {
		return false, nil
	}

	// 删除消息
	_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID})
	alertText := "消息触发违禁词，已撤回"
	var alertEntities []models.MessageEntity
	logTargetID := uint(0)
	appliedPenalty := antiFloodPenaltyDeleteOnly
	actionLabel := moderationPenaltyActionLabel(appliedPenalty, bwCfg.MuteMinutes, bwCfg.BanMinutes)
	if msg.From != nil {
		// 应用惩罚（根据违禁词配置或警告阈值阶梯惩罚）
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
	alert := &tgbot.SendMessageParams{ChatID: msg.Chat.ID, Text: alertText, Entities: alertEntities}
	if msg.From != nil {
		buttonLabel := "管理员解禁"
		buttonAction := fmt.Sprintf("feat:mod:spamunlock:%d:%d", msg.Chat.ID, msg.From.ID)
		if appliedPenalty == antiFloodPenaltyWarn {
			buttonLabel = "撤销警告"
			buttonAction = fmt.Sprintf("feat:mod:bwwarnrevoke:%d:%d", msg.Chat.ID, msg.From.ID)
		}
		alert.ReplyMarkup = models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{Text: buttonLabel, CallbackData: buttonAction}}}}
	}
	alertMsg, sendErr := bot.SendMessage(context.Background(), alert)
	if sendErr == nil && bwCfg.WarnDeleteMinutes > 0 {
		s.ScheduleMessageDelete(msg.Chat.ID, alertMsg.ID, time.Duration(bwCfg.WarnDeleteMinutes)*time.Minute)
	}
	_ = s.repo.CreateLog(group.ID, "banned_word_delete", 0, logTargetID)
	return true, nil
}

// isFlooding 检测用户是否触发反刷屏
// 使用滑动时间窗口检测，维护每个用户的最近消息时间戳
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

// floodUserDisplayName 获取刷屏用户的显示名称（用于提醒消息）
func floodUserDisplayName(u *models.User) string {
	return userMentionLabel(u)
}

// antiSpamActorDisplayName 获取垃圾消息发送者的显示名称（支持 SenderChat）
func antiSpamActorDisplayName(msg *models.Message) string {
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

// applyPenaltyToMember 对用户应用惩罚（禁言/踢出/踢出+封禁）
func applyPenaltyToMember(bot *tgbot.Bot, tgGroupID, tgUserID int64, penalty string, muteMinutes, banMinutes int) {
	switch penalty {
	case antiFloodPenaltyMute:
		// 禁言指定分钟数
		_, _ = bot.RestrictChatMember(context.Background(), &tgbot.RestrictChatMemberParams{
			ChatID:      tgGroupID,
			UserID:      tgUserID,
			UntilDate:   int(time.Now().Add(time.Duration(muteMinutes) * time.Minute).Unix()),
			Permissions: &models.ChatPermissions{},
		})
	case antiFloodPenaltyKick:
		// 临时封禁 1 分钟后解封（即踢出）
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
	case antiFloodPenaltyKickBan:
		// 踢出并封禁指定分钟数
		_, _ = bot.BanChatMember(context.Background(), &tgbot.BanChatMemberParams{
			ChatID:         tgGroupID,
			UserID:         tgUserID,
			RevokeMessages: true,
			UntilDate:      int(time.Now().Add(time.Duration(banMinutes) * time.Minute).Unix()),
		})
	}
}

// containsLink 检测消息中是否包含链接
// 支持 url 实体和文本中的 http/https/ftp/tg/www:// 等链接
func containsLink(msg *models.Message, content string) bool {
	if msg != nil {
		if containsLinkEntity(msg.Entities) || containsLinkEntity(msg.CaptionEntities) {
			return true
		}
	}
	return urlPattern.MatchString(strings.ToLower(content))
}

// containsLinkEntity 检测消息实体中是否包含链接类型
func containsLinkEntity(entities []models.MessageEntity) bool {
	for _, entity := range entities {
		entityType := strings.ToLower(strings.TrimSpace(string(entity.Type)))
		if slices.Contains([]string{"url", "text_link"}, entityType) {
			return true
		}
	}
	return false
}

// antiSpamMessageContent 提取消息的文本内容（支持 Caption）
func antiSpamMessageContent(msg *models.Message) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(msg.Text) != "" {
		parts = append(parts, msg.Text)
	}
	if strings.TrimSpace(msg.Caption) != "" {
		parts = append(parts, msg.Caption)
	}
	return strings.Join(parts, "\n")
}

// antiSpamExceptionHit 检测内容是否命中例外关键词
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

// antiSpamExceptionMatched 检测消息是否命中例外关键词（用于跳过反垃圾检测）
func antiSpamExceptionMatched(msg *models.Message, keywords []string) bool {
	return antiSpamExceptionHit(antiSpamMessageContent(msg), keywords)
}

// antiSpamViolation 使用规则引擎检测垃圾消息
// 检测项：图片、联系人分享、频道马甲、转发、链接、@群组、@用户、ETH 地址、超长消息、超长姓名
func antiSpamViolation(msg *models.Message, cfg antiSpamConfig) (bool, string, string) {
	content := antiSpamMessageContent(msg)
	if cfg.BlockPhoto && len(msg.Photo) > 0 {
		return true, "photo", "图片"
	}
	if cfg.BlockContactShare && msg.Contact != nil {
		return true, "contact_share", "联系人分享"
	}
	if cfg.BlockChannelAlias && msg.SenderChat != nil {
		return true, "channel_alias", "频道马甲发言"
	}
	if cfg.BlockForwardFromChannel && (isForwardFromChannel(msg) || msg.IsAutomaticForward) {
		return true, "forward_channel", "来自频道转发"
	}
	if cfg.BlockForwardFromUser && isForwardFromUser(msg) {
		return true, "forward_user", "来自用户转发"
	}
	if cfg.BlockLink && containsLink(msg, content) {
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

// classifyAntiSpamWithAI 使用 AI 进行垃圾消息二分类
// 支持缓存机制避免重复请求，返回判定结果、是否命中缓存、错误信息
func (s *Service) classifyAntiSpamWithAI(msg *models.Message, strictness string) (spamAIResult, bool, error) {
	if s.spamAI == nil {
		return spamAIResult{}, false, errors.New("nil ai classifier")
	}
	strictness = normalizeAntiSpamAIStrictness(strictness)
	cacheTTL := s.spamAICacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 7 * 24 * time.Hour
	}
	now := time.Now()
	hash := antiSpamContentHash(msg)
	if hash == "" {
		return spamAIResult{}, false, errors.New("empty content hash")
	}
	// 检查缓存
	if cached, err := s.repo.FindAISpamCache(msg.Chat.ID, hash, now.Add(-cacheTTL)); err == nil && cached != nil {
		var out spamAIResult
		if uErr := json.Unmarshal([]byte(cached.ResultJSON), &out); uErr == nil {
			if normalized, nErr := out.Normalized(); nErr == nil {
				return normalized, true, nil
			}
		}
	}

	// 调用 AI 服务
	result, err := s.spamAI.Classify(context.Background(), spamAIInput{
		Content:    antiSpamMessageContent(msg),
		Strictness: strictness,
	})
	if err != nil {
		return spamAIResult{}, false, err
	}
	s.logger.Printf("spam ai result: %+v", result)
	normalized, err := result.Normalized()
	if err != nil {
		return spamAIResult{}, false, err
	}
	// 缓存结果
	if payload, mErr := json.Marshal(normalized); mErr == nil {
		_ = s.repo.UpsertAISpamCache(msg.Chat.ID, hash, string(payload), now)
	}
	// 定期清理过期缓存（每 200 条消息清理一次）
	if msg != nil && msg.ID > 0 && msg.ID%200 == 0 {
		_ = s.repo.DeleteAISpamCacheBefore(now.Add(-cacheTTL))
	}
	return normalized, false, nil
}

// antiSpamContentHash 生成消息内容的哈希值用于 AI 缓存键
// 包含内容、媒体类型、转发信息等用于精确匹配
func antiSpamContentHash(msg *models.Message) string {
	if msg == nil {
		return ""
	}
	content := normalizeSpamText(antiSpamMessageContent(msg))
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// antiSpamChatID 获取发送者 Chat 的 ID（ SenderChat 或 ForwardFromChat）
func antiSpamChatID(chat *models.Chat) int64 {
	if chat == nil {
		return 0
	}
	return chat.ID
}

func isForwardFromChannel(msg *models.Message) bool {
	if msg == nil || msg.ForwardOrigin == nil {
		return false
	}
	return msg.ForwardOrigin.Type == models.MessageOriginTypeChannel || msg.ForwardOrigin.Type == models.MessageOriginTypeChat
}

func isForwardFromUser(msg *models.Message) bool {
	if msg == nil || msg.ForwardOrigin == nil {
		return false
	}
	return msg.ForwardOrigin.Type == models.MessageOriginTypeUser || msg.ForwardOrigin.Type == models.MessageOriginTypeHiddenUser
}

// antiSpamNameLength 计算用户或频道名称长度（用于超长名称检测）
func antiSpamNameLength(msg *models.Message) int {
	if msg == nil {
		return 0
	}
	if msg.From != nil {
		name := strings.TrimSpace(msg.From.FirstName + " " + msg.From.LastName)
		if name == "" {
			name = strings.TrimSpace(msg.From.Username)
		}
		return utf8.RuneCountInString(name)
	}
	if msg.SenderChat != nil {
		return utf8.RuneCountInString(strings.TrimSpace(msg.SenderChat.Title))
	}
	return 0
}

// containsAtGroupID 检测是否包含 @群组ID（数字形式的 @）
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

// containsAtUserID 检测是否包含 @用户ID（纯数字）或 tg://user?id= 链接
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

// containsETHAddress 检测是否包含以太坊地址（0x 开头的 40 位十六进制）
func containsETHAddress(content string) bool {
	return ethAddressPattern.MatchString(content)
}

// normalizeSpamText 规范化垃圾消息文本（转小写、合并空格）
func normalizeSpamText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

// safeActionToken 将操作标签转换为安全的下划线命名日志 token
// 过滤非字母数字字符，限制长度为 32
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
