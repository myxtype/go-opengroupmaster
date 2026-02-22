package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"supervisor/internal/model"
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

func joinWinnerNames(winners []model.User) string {
	if len(winners) == 0 {
		return "无"
	}
	names := make([]string, 0, len(winners))
	for _, w := range winners {
		if w.Username != "" {
			names = append(names, "@"+w.Username)
			continue
		}
		n := strings.TrimSpace(w.FirstName + " " + w.LastName)
		if n == "" {
			n = fmt.Sprintf("uid:%d", w.TGUserID)
		}
		names = append(names, n)
	}
	return strings.Join(names, ", ")
}

func onOffWithEmoji(v bool) string {
	if v {
		return "启用✅"
	}
	return "关闭❌"
}
