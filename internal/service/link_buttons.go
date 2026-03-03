package service

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/go-telegram/bot/models"
)

func parseAndEncodeButtonRows(raw string) (string, error) {
	rows, err := parseOptionalButtonRows(raw)
	if err != nil {
		return "", err
	}
	return encodeButtonRows(rows), nil
}

func parseOptionalButtonRows(raw string) ([][]welcomeButton, error) {
	txt := strings.TrimSpace(raw)
	if slices.Contains([]string{"", "跳过", "关闭"}, txt) {
		return [][]welcomeButton{}, nil
	}
	rows, err := parseWelcomeButtonsInput(raw)
	if err != nil {
		return nil, err
	}
	return normalizeWelcomeButtonRows(rows), nil
}

func encodeButtonRows(rows [][]welcomeButton) string {
	rows = normalizeWelcomeButtonRows(rows)
	if len(rows) == 0 {
		return ""
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeButtonRows(raw string) [][]welcomeButton {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return [][]welcomeButton{}
	}
	var rows [][]welcomeButton
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return [][]welcomeButton{}
	}
	return normalizeWelcomeButtonRows(rows)
}

func inlineKeyboardFromButtonRows(rows [][]welcomeButton) (models.InlineKeyboardMarkup, bool) {
	rows = normalizeWelcomeButtonRows(rows)
	if len(rows) == 0 {
		return models.InlineKeyboardMarkup{}, false
	}
	keyboardRows := make([][]models.InlineKeyboardButton, 0, len(rows))
	for _, rowCfg := range rows {
		row := make([]models.InlineKeyboardButton, 0, len(rowCfg))
		for _, btn := range rowCfg {
			if strings.TrimSpace(btn.Text) == "" || strings.TrimSpace(btn.URL) == "" {
				continue
			}
			row = append(row, models.InlineKeyboardButton{Text: btn.Text, URL: btn.URL})
		}
		if len(row) > 0 {
			keyboardRows = append(keyboardRows, row)
		}
	}
	if len(keyboardRows) == 0 {
		return models.InlineKeyboardMarkup{}, false
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}, true
}

func InlineKeyboardFromButtonRowsJSON(raw string) (models.InlineKeyboardMarkup, bool) {
	return inlineKeyboardFromButtonRows(decodeButtonRows(raw))
}
