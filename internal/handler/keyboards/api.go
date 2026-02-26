package keyboards

import (
	"supervisor/internal/model"
	"supervisor/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func MainMenuKeyboard(botUsername string) tgbotapi.InlineKeyboardMarkup {
	return mainMenuKeyboard(botUsername)
}

func GroupsKeyboard(groups []model.Group, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	return groupsKeyboard(groups, page, totalPages)
}

func GroupPanelKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return groupPanelKeyboard(tgGroupID)
}

func InviteKeyboard(tgGroupID int64, enabled bool) tgbotapi.InlineKeyboardMarkup {
	return inviteKeyboard(tgGroupID, enabled)
}

func InviteExpireInputKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return inviteExpireInputKeyboard(tgGroupID)
}

func InviteMemberInputKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return inviteMemberInputKeyboard(tgGroupID)
}

func InviteGenerateInputKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return inviteGenerateInputKeyboard(tgGroupID)
}

func PendingCancelKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return pendingCancelKeyboard(tgGroupID)
}

func AutoReplyListKeyboard(tgGroupID int64, items []model.AutoReply, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	return autoReplyListKeyboard(tgGroupID, items, page, totalPages)
}

func AutoReplyMatchTypeKeyboard(tgGroupID int64, modeSelectPrefix string) tgbotapi.InlineKeyboardMarkup {
	return autoReplyMatchTypeKeyboard(tgGroupID, modeSelectPrefix)
}

func BannedWordListKeyboard(tgGroupID int64, view *service.BannedWordView, items []model.BannedWord, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	return bannedWordListKeyboard(tgGroupID, view, items, page, totalPages)
}

func BannedWordPenaltyKeyboard(tgGroupID int64, view *service.BannedWordView) tgbotapi.InlineKeyboardMarkup {
	return bannedWordPenaltyKeyboard(tgGroupID, view)
}

func ScheduledListKeyboard(tgGroupID int64, items []model.ScheduledMessage, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	return scheduledListKeyboard(tgGroupID, items, page, totalPages)
}

func ScheduledPinSelectKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return scheduledPinSelectKeyboard(tgGroupID)
}

func ScheduledEditKeyboard(tgGroupID int64, id uint, page int, enabled bool, pin bool) tgbotapi.InlineKeyboardMarkup {
	return scheduledEditKeyboard(tgGroupID, id, page, enabled, pin)
}

func LogListKeyboard(tgGroupID int64, page, totalPages int, filter string) tgbotapi.InlineKeyboardMarkup {
	return logListKeyboard(tgGroupID, page, totalPages, filter)
}

func SystemCleanKeyboard(tgGroupID int64, cfg *service.SystemCleanView) tgbotapi.InlineKeyboardMarkup {
	return systemCleanKeyboard(tgGroupID, cfg)
}

func AntiFloodKeyboard(tgGroupID int64, view *service.AntiFloodView) tgbotapi.InlineKeyboardMarkup {
	return antiFloodKeyboard(tgGroupID, view)
}

func AntiFloodAlertDeleteKeyboard(tgGroupID int64, currentSec int) tgbotapi.InlineKeyboardMarkup {
	return antiFloodAlertDeleteKeyboard(tgGroupID, currentSec)
}

func AntiFloodCountKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	return antiFloodCountKeyboard(tgGroupID, current)
}

func AntiFloodWindowKeyboard(tgGroupID int64, currentSec int) tgbotapi.InlineKeyboardMarkup {
	return antiFloodWindowKeyboard(tgGroupID, currentSec)
}

func AntiSpamKeyboard(tgGroupID int64, view *service.AntiSpamView) tgbotapi.InlineKeyboardMarkup {
	return antiSpamKeyboard(tgGroupID, view)
}

func AntiSpamPenaltyKeyboard(tgGroupID int64, view *service.AntiSpamView) tgbotapi.InlineKeyboardMarkup {
	return antiSpamPenaltyKeyboard(tgGroupID, view)
}

func AntiFloodPenaltyKeyboard(tgGroupID int64, view *service.AntiFloodView) tgbotapi.InlineKeyboardMarkup {
	return antiFloodPenaltyKeyboard(tgGroupID, view)
}

func AntiSpamAlertDeleteKeyboard(tgGroupID int64, currentSec int) tgbotapi.InlineKeyboardMarkup {
	return antiSpamAlertDeleteKeyboard(tgGroupID, currentSec)
}

func AntiSpamAIKeyboard(tgGroupID int64, view *service.AntiSpamView) tgbotapi.InlineKeyboardMarkup {
	return antiSpamAIKeyboard(tgGroupID, view)
}

func VerifyKeyboard(tgGroupID int64, view *service.JoinVerifyView) tgbotapi.InlineKeyboardMarkup {
	return verifyKeyboard(tgGroupID, view)
}

func NewbieLimitKeyboard(tgGroupID int64, view *service.NewbieLimitView) tgbotapi.InlineKeyboardMarkup {
	return newbieLimitKeyboard(tgGroupID, view)
}

func VerifyTimeoutMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	return verifyTimeoutMinutesKeyboard(tgGroupID, current)
}

func NewbieLimitMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	return newbieLimitMinutesKeyboard(tgGroupID, current)
}

func NightModeKeyboard(tgGroupID int64, view *service.NightModeView) tgbotapi.InlineKeyboardMarkup {
	return nightModeKeyboard(tgGroupID, view)
}

func ChainKeyboard(tgGroupID int64, items []service.ChainSummary) tgbotapi.InlineKeyboardMarkup {
	return chainKeyboard(tgGroupID, items)
}

func ChainLimitModeKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return chainLimitModeKeyboard(tgGroupID)
}

func ChainDurationKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return chainDurationKeyboard(tgGroupID)
}

func ChainPublicJoinKeyboard(joinURL string, active bool) tgbotapi.InlineKeyboardMarkup {
	return chainPublicJoinKeyboard(joinURL, active)
}

func MonitorKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return monitorKeyboard(tgGroupID)
}

func PollKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return pollKeyboard(tgGroupID)
}

func LotteryKeyboard(tgGroupID int64, publishPin bool, resultPin bool, deleteKeywordMins int) tgbotapi.InlineKeyboardMarkup {
	return lotteryKeyboard(tgGroupID, publishPin, resultPin, deleteKeywordMins)
}

func LotteryRecordsKeyboard(tgGroupID int64, items []service.LotteryRecordItem, page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	return lotteryRecordsKeyboard(tgGroupID, items, page, totalPages)
}

func LotteryDeleteMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	return lotteryDeleteMinutesKeyboard(tgGroupID, current)
}

func WelcomeKeyboard(tgGroupID int64, enabled bool, mode string, deleteMinutes int) tgbotapi.InlineKeyboardMarkup {
	return welcomeKeyboard(tgGroupID, enabled, mode, deleteMinutes)
}

func WelcomeDeleteMinutesKeyboard(tgGroupID int64, current int) tgbotapi.InlineKeyboardMarkup {
	return welcomeDeleteMinutesKeyboard(tgGroupID, current)
}

func RBACKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return rbacKeyboard(tgGroupID)
}

func BlacklistKeyboard(tgGroupID int64) tgbotapi.InlineKeyboardMarkup {
	return blacklistKeyboard(tgGroupID)
}

func SettingsKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	return settingsKeyboard(lang)
}
