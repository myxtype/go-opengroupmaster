package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"
)

const (
	nightModeDeleteMedia = "delete_media"
	nightModeGlobalMute  = "global_mute"

	nightDefaultStartHour = 0
	nightDefaultEndHour   = 8
)

func (s *Service) NightModeViewByTGGroupID(tgGroupID int64) (*NightModeView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	state, err := s.getNightModeState(group.ID)
	if err != nil {
		return nil, err
	}
	cfg := normalizeNightModeConfig(state.Config)
	return &NightModeView{
		Enabled:      state.Enabled,
		TimezoneText: formatUTCOffset(cfg.TimezoneOffsetMinutes),
		Mode:         cfg.Mode,
		StartHour:    cfg.StartHour,
		EndHour:      cfg.EndHour,
		NightWindow:  formatNightWindow(cfg.StartHour, cfg.EndHour),
	}, nil
}

func (s *Service) SetNightModeEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	state, err := s.getNightModeState(group.ID)
	if err != nil {
		return false, err
	}
	state.Enabled = enabled
	if err := s.saveNightModeState(group.ID, state); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_night_mode_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) SetNightModeModeByTGGroupID(tgGroupID int64, mode string) (string, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	state, err := s.getNightModeState(group.ID)
	if err != nil {
		return "", err
	}
	cfg := normalizeNightModeConfig(state.Config)
	switch mode {
	case nightModeDeleteMedia, nightModeGlobalMute:
		cfg.Mode = mode
	default:
		return "", errors.New("invalid night mode")
	}
	state.Config = cfg
	if err := s.saveNightModeState(group.ID, state); err != nil {
		return "", err
	}
	_ = s.repo.CreateLog(group.ID, "set_night_mode_mode_"+cfg.Mode, 0, 0)
	return cfg.Mode, nil
}

func (s *Service) SetNightModeTimezoneByTGGroupID(tgGroupID int64, raw string) (string, error) {
	offsetMinutes, err := parseUTCOffset(raw)
	if err != nil {
		return "", err
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	state, err := s.getNightModeState(group.ID)
	if err != nil {
		return "", err
	}
	cfg := normalizeNightModeConfig(state.Config)
	cfg.TimezoneOffsetMinutes = offsetMinutes
	state.Config = cfg
	if err := s.saveNightModeState(group.ID, state); err != nil {
		return "", err
	}
	tz := formatUTCOffset(offsetMinutes)
	_ = s.repo.CreateLog(group.ID, "set_night_mode_timezone_"+tz, 0, 0)
	return tz, nil
}

func (s *Service) SetNightModeStartHourByTGGroupID(tgGroupID int64, raw string) (int, error) {
	hour, err := parseNightHour(raw)
	if err != nil {
		return 0, err
	}
	if err := s.setNightModeHourByTGGroupID(tgGroupID, true, hour); err != nil {
		return 0, err
	}
	return hour, nil
}

func (s *Service) SetNightModeEndHourByTGGroupID(tgGroupID int64, raw string) (int, error) {
	hour, err := parseNightHour(raw)
	if err != nil {
		return 0, err
	}
	if err := s.setNightModeHourByTGGroupID(tgGroupID, false, hour); err != nil {
		return 0, err
	}
	return hour, nil
}

func (s *Service) setNightModeHourByTGGroupID(tgGroupID int64, isStart bool, hour int) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	state, err := s.getNightModeState(group.ID)
	if err != nil {
		return err
	}
	cfg := normalizeNightModeConfig(state.Config)
	if isStart {
		cfg.StartHour = hour
	} else {
		cfg.EndHour = hour
	}
	state.Config = cfg
	if err := s.saveNightModeState(group.ID, state); err != nil {
		return err
	}
	if isStart {
		_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_night_mode_start_hour_%d", hour), 0, 0)
		return nil
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_night_mode_end_hour_%d", hour), 0, 0)
	return nil
}

func parseUTCOffset(raw string) (int, error) {
	txt := strings.TrimSpace(strings.ToUpper(raw))
	txt = strings.TrimPrefix(txt, "UTC")
	txt = strings.TrimSpace(txt)
	if txt == "" {
		return 0, errors.New("timezone is empty")
	}
	sign := 1
	switch txt[0] {
	case '+':
		txt = strings.TrimSpace(txt[1:])
	case '-':
		sign = -1
		txt = strings.TrimSpace(txt[1:])
	}
	if txt == "" {
		return 0, errors.New("invalid timezone")
	}
	hours := 0
	minutes := 0
	if strings.Contains(txt, ":") {
		parts := strings.SplitN(txt, ":", 2)
		h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, errors.New("invalid timezone")
		}
		m, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, errors.New("invalid timezone")
		}
		hours = h
		minutes = m
	} else {
		h, err := strconv.Atoi(txt)
		if err != nil {
			return 0, errors.New("invalid timezone")
		}
		hours = h
	}
	if hours < 0 || minutes < 0 || minutes >= 60 {
		return 0, errors.New("invalid timezone")
	}
	total := sign * (hours*60 + minutes)
	if total < -12*60 || total > 14*60 {
		return 0, errors.New("timezone out of range")
	}
	return total, nil
}

func parseNightHour(raw string) (int, error) {
	txt := strings.TrimSpace(raw)
	if txt == "" {
		return 0, errors.New("hour is empty")
	}
	hour, err := strconv.Atoi(txt)
	if err != nil {
		return 0, errors.New("invalid hour")
	}
	if hour < 0 || hour > 23 {
		return 0, errors.New("hour out of range")
	}
	return hour, nil
}

func formatUTCOffset(offsetMinutes int) string {
	sign := "+"
	if offsetMinutes < 0 {
		sign = "-"
		offsetMinutes = -offsetMinutes
	}
	h := offsetMinutes / 60
	m := offsetMinutes % 60
	if m == 0 {
		return fmt.Sprintf("UTC%s%d", sign, h)
	}
	return fmt.Sprintf("UTC%s%d:%02d", sign, h, m)
}

func formatNightWindow(startHour, endHour int) string {
	return fmt.Sprintf("%02d:00-%02d:00", startHour, endHour)
}

func isNightWindowNow(offsetMinutes, startHour, endHour int, now time.Time) bool {
	local := now.UTC().Add(time.Duration(offsetMinutes) * time.Minute)
	minuteOfDay := local.Hour()*60 + local.Minute()
	startMinutes := startHour * 60
	endMinutes := endHour * 60
	if startMinutes == endMinutes {
		return true
	}
	if startMinutes < endMinutes {
		return minuteOfDay >= startMinutes && minuteOfDay < endMinutes
	}
	return minuteOfDay >= startMinutes || minuteOfDay < endMinutes
}

func isNightMediaMessage(msg *models.Message) bool {
	if msg == nil {
		return false
	}
	return len(msg.Photo) > 0 ||
		msg.Video != nil ||
		msg.Animation != nil ||
		msg.Document != nil ||
		msg.Audio != nil ||
		msg.Voice != nil ||
		msg.VideoNote != nil ||
		msg.Sticker != nil
}
