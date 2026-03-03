package handler

import (
	tgbot "github.com/go-telegram/bot"
)

func (h *Handler) sendPendingParentPanel(bot *tgbot.Bot, target renderTarget, userID int64, pending pendingInput) {
	switch pending.Kind {
	case "auto_add", "auto_add_mode", "auto_add_keyword", "auto_add_reply", "auto_add_buttons", "auto_edit", "auto_edit_mode", "auto_edit_keyword", "auto_edit_reply", "auto_edit_buttons":
		page := pending.Page
		if page < 1 {
			page = 1
		}
		h.sendAutoReplyList(bot, target, userID, pending.TGGroupID, page)
	case "bw_add", "bw_edit":
		page := pending.Page
		if page < 1 {
			page = 1
		}
		h.sendBannedWordList(bot, target, userID, pending.TGGroupID, page)
	case "lottery_create", "lottery_create_title", "lottery_create_winners", "lottery_create_keyword":
		h.sendLotteryPanel(bot, target, userID, pending.TGGroupID)
	case "sched_add_cron", "sched_add_content", "sched_add_buttons", "sched_add_pin":
		page := pending.Page
		if page < 1 {
			page = 1
		}
		h.sendScheduledList(bot, target, userID, pending.TGGroupID, page)
	case "sched_edit_text", "sched_edit_media", "sched_edit_buttons", "sched_edit_cron":
		page := pending.Page
		if page < 1 {
			page = 1
		}
		h.sendScheduledEditPanel(bot, target, userID, pending.TGGroupID, pending.RuleID, page)
	case "chain_create_mode", "chain_create_count", "chain_create_duration", "chain_create_intro":
		h.sendChainPanel(bot, target, userID, pending.TGGroupID)
	case "poll_create", "poll_create_question", "poll_create_option":
		h.sendPollPanel(bot, target, userID, pending.TGGroupID)
	case "monitor_add", "monitor_remove":
		h.sendMonitorPanel(bot, target, userID, pending.TGGroupID)
	case "wc_set_push_time", "wc_black_add", "wc_black_remove":
		h.sendWordCloudPanel(bot, target, userID, pending.TGGroupID)
	case "rbac_set_role", "rbac_set_acl":
		h.sendRBACPanel(bot, target, userID, pending.TGGroupID)
	case "black_add", "black_add_reason", "black_remove":
		h.sendBlacklistPanel(bot, target, userID, pending.TGGroupID)
	case "welcome_edit", "welcome_edit_media", "welcome_edit_button":
		h.sendWelcomePanel(bot, target, userID, pending.TGGroupID)
	case "spam_msg_len", "spam_name_len", "spam_exception_add", "spam_exception_remove":
		h.sendAntiSpamPanel(bot, target, userID, pending.TGGroupID)
	case "spam_warn_threshold", "spam_warn_action_mute_minutes", "spam_warn_action_ban_minutes", "spam_mute_minutes", "spam_ban_minutes":
		h.sendAntiSpamPenaltyPanel(bot, target, userID, pending.TGGroupID)
	case "flood_warn_threshold", "flood_warn_action_mute_minutes", "flood_warn_action_ban_minutes", "flood_mute_minutes", "flood_ban_minutes":
		h.sendAntiFloodPenaltyPanel(bot, target, userID, pending.TGGroupID)
	case "spam_ai_spam_score":
		h.sendAntiSpamAIPanel(bot, target, userID, pending.TGGroupID)
	case "night_tz", "night_start_hour", "night_end_hour":
		h.sendNightModePanel(bot, target, userID, pending.TGGroupID)
	case "points_checkin_keyword", "points_checkin_reward":
		h.sendPointsCheckinPanel(bot, target, userID, pending.TGGroupID)
	case "points_message_reward", "points_message_daily", "points_message_min_len":
		h.sendPointsMessagePanel(bot, target, userID, pending.TGGroupID)
	case "points_invite_reward", "points_invite_daily":
		h.sendPointsInvitePanel(bot, target, userID, pending.TGGroupID)
	case "points_balance_alias", "points_rank_alias", "points_admin_add", "points_admin_sub", "points_admin_add_value", "points_admin_sub_value":
		h.sendPointsPanel(bot, target, userID, pending.TGGroupID)
	case "invite_set_expire", "invite_set_member_limit", "invite_set_generate_limit":
		h.sendInvitePanel(bot, target, userID, pending.TGGroupID)
	default:
		h.sendGroupPanel(bot, target, userID, pending.TGGroupID)
	}
}
