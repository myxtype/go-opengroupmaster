package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"supervisor/internal/model"
	"supervisor/internal/tgmention"

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

func boolIcon(v bool) string {
	if v {
		return "✅"
	}
	return "❌"
}

func lotteryDeleteDesc(minutes int) string {
	if minutes <= 0 {
		return "关闭自动删除口令消息"
	}
	return fmt.Sprintf("%d分钟后自动删除群成员参加抽奖发送的口令消息", minutes)
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
