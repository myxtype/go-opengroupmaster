package handler

import "strings"

func isWordCloudAutoPushDisableInput(raw string) bool {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "关闭", "不自动推送", "off":
		return true
	default:
		return false
	}
}
