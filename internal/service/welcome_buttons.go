package service

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

func parseWelcomeButtonsInput(raw string) ([][]welcomeButton, error) {
	lines := strings.Split(raw, "\n")
	rows := make([][]welcomeButton, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "&&")
		if len(parts) > 2 {
			return nil, errors.New("每行最多配置两个按钮，请用 && 分隔")
		}
		row := make([]welcomeButton, 0, len(parts))
		for _, part := range parts {
			btn, err := parseWelcomeButtonSpec(part)
			if err != nil {
				return nil, err
			}
			row = append(row, btn)
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return nil, errors.New("至少需要一个按钮")
	}
	return rows, nil
}

func parseWelcomeButtonSpec(spec string) (welcomeButton, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return welcomeButton{}, errors.New("按钮配置不能为空")
	}
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return welcomeButton{}, errors.New("按钮格式错误，请按：按钮文字 - 链接")
	}
	text := strings.TrimSpace(parts[0])
	rawURL := strings.TrimSpace(parts[1])
	if text == "" || rawURL == "" {
		return welcomeButton{}, errors.New("按钮文字和链接都不能为空")
	}
	normURL, err := normalizeWelcomeButtonURL(rawURL)
	if err != nil {
		return welcomeButton{}, err
	}
	return welcomeButton{Text: text, URL: normURL}, nil
}

func normalizeWelcomeButtonURL(raw string) (string, error) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", errors.New("链接不能为空")
	}
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return "", errors.New("链接格式无效")
	}
	if !slices.Contains([]string{"http", "https"}, parsed.Scheme) {
		return "", errors.New("仅支持 http/https 链接")
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return "", errors.New("链接缺少域名")
	}
	return parsed.String(), nil
}

func normalizeWelcomeButtonRows(rows [][]welcomeButton) [][]welcomeButton {
	out := make([][]welcomeButton, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		clean := make([]welcomeButton, 0, 2)
		for _, btn := range row {
			text := strings.TrimSpace(btn.Text)
			link := strings.TrimSpace(btn.URL)
			if text == "" || link == "" {
				continue
			}
			normURL, err := normalizeWelcomeButtonURL(link)
			if err != nil {
				continue
			}
			clean = append(clean, welcomeButton{Text: text, URL: normURL})
			if len(clean) >= 2 {
				break
			}
		}
		if len(clean) > 0 {
			out = append(out, clean)
		}
	}
	return out
}

func welcomeButtonsSummaryText(rows [][]welcomeButton) string {
	if len(rows) == 0 {
		return "未设置"
	}
	count := 0
	for _, row := range rows {
		count += len(row)
	}
	return fmt.Sprintf("%d 个按钮（%d 行）", count, len(rows))
}
