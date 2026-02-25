package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"supervisor/internal/model"
	"supervisor/pkg/tgmention"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func permissionFeatureKey(feature, action string) string {
	switch feature {
	case "rbac", "black":
		return "security"
	case "invite":
		return "invite"
	case "poll":
		return "poll"
	case "chain":
		return "chain"
	case "monitor":
		return "monitor"
	case "sched":
		return "schedule"
	case "auto":
		return "auto_reply"
	case "bw":
		return "banned_words"
	case "logs":
		return "logs"
	case "stats":
		return "stats"
	case "mod":
		return "moderation"
	case "sys":
		return "system_clean"
	case "lottery":
		return "lottery"
	case "welcome":
		return "welcome"
	}
	_ = action
	return ""
}

func parseInt64Suffix(data, prefix string) (int64, error) {
	if !strings.HasPrefix(data, prefix) {
		return 0, errors.New("invalid prefix")
	}
	return strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
}

func parseIntSuffix(data, prefix string) (int, error) {
	if !strings.HasPrefix(data, prefix) {
		return 0, errors.New("invalid prefix")
	}
	return strconv.Atoi(strings.TrimPrefix(data, prefix))
}

func maxPages(total int64, pageSize int) int {
	if total <= 0 {
		return 1
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if pages < 1 {
		return 1
	}
	return pages
}

func joinWinnerNames(winners []model.User) (string, []tgbotapi.MessageEntity) {
	if len(winners) == 0 {
		return "无", nil
	}
	refs := make([]tgmention.UserRef, 0, len(winners))
	for _, w := range winners {
		refs = append(refs, tgmention.UserRef{
			ID:        w.TGUserID,
			Username:  w.Username,
			FirstName: w.FirstName,
			LastName:  w.LastName,
		})
	}
	return tgmention.JoinMentions(refs, ", ")
}

func lotteryResultText(winners []model.User) (string, []tgbotapi.MessageEntity) {
	namesText, entities := joinWinnerNames(winners)
	prefix := "开奖结果："
	return prefix + namesText, tgmention.ShiftEntities(entities, tgmention.UTF16Len(prefix))
}

func onOffWithEmoji(v bool) string {
	if v {
		return "启用✅"
	}
	return "关闭❌"
}

func inviteExpireText(unixTs int64) string {
	if unixTs <= 0 {
		return "无限制"
	}
	return time.Unix(unixTs, 0).In(time.Local).Format("2006-01-02 15:04")
}

func inviteLimitText(v int) string {
	if v <= 0 {
		return "无限制"
	}
	return strconv.Itoa(v)
}

func chainLimitText(v int) string {
	if v <= 0 {
		return "不限人数"
	}
	return fmt.Sprintf("%d 人", v)
}

func chainDeadlineText(deadlineUnix int64) string {
	if deadlineUnix <= 0 {
		return "不限时"
	}
	t := time.Unix(deadlineUnix, 0).In(time.Local)
	return fmt.Sprintf("%s %s", t.Format("2006-01-02 15:04:05"), utcOffsetLabel(t))
}

func utcOffsetLabel(t time.Time) string {
	_, offset := t.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hour := offset / 3600
	minute := (offset % 3600) / 60
	if minute == 0 {
		return fmt.Sprintf("UTC%s%d", sign, hour)
	}
	return fmt.Sprintf("UTC%s%d:%02d", sign, hour, minute)
}

func boolIcon(v bool) string {
	if v {
		return "✅"
	}
	return "❌"
}

func lotteryDeleteDesc(minutes int) string {
	if minutes <= 0 {
		return "关闭自动删除口令和参与成功提示消息"
	}
	return fmt.Sprintf("%d分钟后自动删除群成员参加抽奖发送的口令及参与成功提示消息", minutes)
}

func bannedWordDeleteText(minutes int) string {
	if minutes <= 0 {
		return "关闭"
	}
	return fmt.Sprintf("%d分钟", minutes)
}

func bannedWordWarnActionLabel(action string, muteMinutes int, banMinutes int) string {
	switch strings.TrimSpace(strings.ToLower(action)) {
	case "mute":
		return fmt.Sprintf("禁言 %d 分钟", muteMinutes)
	case "kick":
		return "踢出"
	case "kick_ban":
		return fmt.Sprintf("踢出+封禁 %d 分钟", banMinutes)
	default:
		return "禁言 60 分钟"
	}
}

func bannedWordPenaltyText(penalty string, warnThreshold int, warnAction string, warnActionMuteMinutes int, warnActionBanMinutes int, muteMinutes int, banMinutes int) string {
	switch strings.TrimSpace(strings.ToLower(penalty)) {
	case "warn":
		return fmt.Sprintf("警告 %d 次后%s", warnThreshold, bannedWordWarnActionLabel(warnAction, warnActionMuteMinutes, warnActionBanMinutes))
	case "mute":
		return fmt.Sprintf("禁言 %d 分钟", muteMinutes)
	case "kick":
		return "踢出"
	case "kick_ban":
		return fmt.Sprintf("踢出+封禁 %d 分钟", banMinutes)
	default:
		return "仅撤回消息+不惩罚"
	}
}

func lotteryStatusLabel(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "active":
		return "未开奖"
	case "closed":
		return "已开奖"
	case "canceled":
		return "已取消"
	default:
		if strings.TrimSpace(status) == "" {
			return "未知"
		}
		return status
	}
}

func antiFloodPenaltyText(penalty string, muteSec int) string {
	switch penalty {
	case "warn":
		return "警告"
	case "mute":
		return fmt.Sprintf("禁言（%d秒）", muteSec)
	case "kick":
		return "踢出"
	case "kick_ban":
		return "踢出+封禁"
	default:
		return "撤回消息+不处罚"
	}
}

func antiFloodAlertDeleteText(seconds int) string {
	if seconds <= 0 {
		return "不自动删除"
	}
	return fmt.Sprintf("%d秒", seconds)
}

func verifyTypeLabel(v string) string {
	switch v {
	case "math":
		return "数学题"
	case "captcha":
		return "验证码"
	case "zhchar":
		return "中文字符验证码"
	default:
		return "按钮"
	}
}

func verifyTimeoutActionLabel(v string) string {
	if v == "kick" {
		return "踢出"
	}
	return "禁言"
}

func nightModeActionLabel(mode string) string {
	if mode == "global_mute" {
		return "全局禁言（删除所有消息）"
	}
	return "删除媒体（视频/图片等）"
}

func autoReplyMatchTypeLabel(v string) string {
	if strings.TrimSpace(strings.ToLower(v)) == "contains" {
		return "包含触发"
	}
	return "精准触发"
}

func scheduledMediaTypeLabel(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "photo":
		return "图片"
	case "video":
		return "视频"
	case "document":
		return "文件"
	case "animation":
		return "动图"
	default:
		return "文本"
	}
}

func scheduledContentPreview(content string, maxLen int) string {
	txt := strings.TrimSpace(strings.ReplaceAll(content, "\n", " "))
	if txt == "" {
		return "（无文字）"
	}
	if maxLen <= 0 {
		maxLen = 30
	}
	runes := []rune(txt)
	if len(runes) <= maxLen {
		return txt
	}
	return string(runes[:maxLen]) + "..."
}

func buttonRowsCount(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	var rows [][]struct {
		Text string `json:"text"`
		URL  string `json:"url"`
	}
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return 0
	}
	total := 0
	for _, row := range rows {
		total += len(row)
	}
	return total
}
