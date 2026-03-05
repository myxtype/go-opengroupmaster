package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"supervisor/internal/model"
)

const defaultGroupTimezoneOffsetMinutes = 8 * 60

func normalizeGroupTimezoneOffsetMinutes(offsetMinutes int) int {
	if offsetMinutes < -12*60 || offsetMinutes > 14*60 {
		return defaultGroupTimezoneOffsetMinutes
	}
	return offsetMinutes
}

func (s *Service) groupTimezoneOffsetMinutesByGroup(group *model.Group) (int, error) {
	if group == nil {
		return 0, errors.New("nil group")
	}
	offset := normalizeGroupTimezoneOffsetMinutes(group.TimezoneOffsetMinutes)
	if offset != group.TimezoneOffsetMinutes {
		if err := s.repo.UpdateGroupTimezoneOffsetMinutes(group.ID, offset); err != nil {
			return 0, err
		}
		group.TimezoneOffsetMinutes = offset
	}
	return offset, nil
}

func (s *Service) GroupTimezoneViewByTGGroupID(tgGroupID int64) (*GroupTimezoneView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	offset, err := s.groupTimezoneOffsetMinutesByGroup(group)
	if err != nil {
		return nil, err
	}
	return &GroupTimezoneView{
		OffsetMinutes: offset,
		TimezoneText:  formatUTCOffset(offset),
	}, nil
}

func (s *Service) SetGroupTimezoneByTGGroupID(tgGroupID int64, raw string) (string, error) {
	offsetMinutes, err := parseUTCOffset(raw)
	if err != nil {
		return "", err
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", err
	}
	offsetMinutes = normalizeGroupTimezoneOffsetMinutes(offsetMinutes)
	if err := s.repo.UpdateGroupTimezoneOffsetMinutes(group.ID, offsetMinutes); err != nil {
		return "", err
	}
	tz := formatUTCOffset(offsetMinutes)
	_ = s.repo.CreateLog(group.ID, "set_group_timezone_"+tz, 0, 0)
	return tz, nil
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

func timezoneLocation(offsetMinutes int) *time.Location {
	return time.FixedZone(formatUTCOffset(offsetMinutes), offsetMinutes*60)
}

func dateKeyAtTimezone(now time.Time, offsetMinutes int) string {
	loc := timezoneLocation(offsetMinutes)
	return now.In(loc).Format("2006-01-02")
}

func dayStartUTCAtTimezone(now time.Time, offsetMinutes int) time.Time {
	loc := timezoneLocation(offsetMinutes)
	local := now.In(loc)
	startLocal := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return startLocal.UTC()
}
