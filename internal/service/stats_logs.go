package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *Service) GroupStatsByTGGroupID(tgGroupID int64, limit int) (*GroupStats, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	nowUTC := time.Now().UTC()
	dayKey := pointsDayKey(nowUTC)
	dayKey7 := pointsDayKey(nowUTC.AddDate(0, 0, -6))
	dayKey30 := pointsDayKey(nowUTC.AddDate(0, 0, -29))
	since7 := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -6)
	since30 := since7.AddDate(0, 0, -23)

	pointsSummary, err := s.repo.SummarizeUserPoints(group.ID)
	if err != nil {
		return nil, err
	}
	inviteAll, err := s.repo.SummarizeInviteEvents(group.ID, time.Time{})
	if err != nil {
		return nil, err
	}
	invite7, err := s.repo.SummarizeInviteEvents(group.ID, since7)
	if err != nil {
		return nil, err
	}
	invite30, err := s.repo.SummarizeInviteEvents(group.ID, since30)
	if err != nil {
		return nil, err
	}
	messageAll, err := s.repo.SummarizePointEvents(group.ID, pointsEventMessage, "")
	if err != nil {
		return nil, err
	}
	messageToday, err := s.repo.SummarizePointEvents(group.ID, pointsEventMessage, dayKey)
	if err != nil {
		return nil, err
	}
	checkinToday, err := s.repo.SummarizePointEvents(group.ID, pointsEventCheckin, dayKey)
	if err != nil {
		return nil, err
	}
	message7, err := s.repo.SummarizePointEventsSinceDay(group.ID, pointsEventMessage, dayKey7)
	if err != nil {
		return nil, err
	}
	message30, err := s.repo.SummarizePointEventsSinceDay(group.ID, pointsEventMessage, dayKey30)
	if err != nil {
		return nil, err
	}
	checkin7, err := s.repo.SummarizePointEventsSinceDay(group.ID, pointsEventCheckin, dayKey7)
	if err != nil {
		return nil, err
	}
	checkin30, err := s.repo.SummarizePointEventsSinceDay(group.ID, pointsEventCheckin, dayKey30)
	if err != nil {
		return nil, err
	}
	top, err := s.repo.TopUsersByPointEventType(group.ID, pointsEventMessage, limit)
	if err != nil {
		return nil, err
	}
	out := &GroupStats{
		GroupTitle:            group.Title,
		GroupID:               group.TGGroupID,
		DayKey:                dayKey,
		PointsUsersTotal:      pointsSummary.UsersTotal,
		PointsTotal:           pointsSummary.PointsTotal,
		InviteTotal:           inviteAll.EventsTotal,
		MessageEventsTotal:    messageAll.EventsTotal,
		MessagePointsTotal:    messageAll.DeltaTotal,
		MessageUsersTotal:     messageAll.UsersTotal,
		TodayMessagePoints:    messageToday.DeltaTotal,
		TodayMessageUsers:     messageToday.UsersTotal,
		TodayCheckins:         checkinToday.EventsTotal,
		Recent7MessagePoints:  message7.DeltaTotal,
		Recent7MessageUsers:   message7.UsersTotal,
		Recent7MessageEvents:  message7.EventsTotal,
		Recent7Checkins:       checkin7.EventsTotal,
		Recent7Invites:        invite7.EventsTotal,
		Recent30MessagePoints: message30.DeltaTotal,
		Recent30MessageUsers:  message30.UsersTotal,
		Recent30MessageEvents: message30.EventsTotal,
		Recent30Checkins:      checkin30.EventsTotal,
		Recent30Invites:       invite30.EventsTotal,
	}
	for _, row := range top {
		out.TopUsers = append(out.TopUsers, UserScore{
			DisplayName: statsDisplayName(row.Username, row.FirstName, row.LastName, row.TGUserID, row.UserID),
			Points:      int(row.Points),
		})
	}
	return out, nil
}

func statsDisplayName(username, firstName, lastName string, tgUserID int64, userID uint) string {
	if username != "" {
		return "@" + username
	}
	name := strings.TrimSpace(strings.TrimSpace(firstName) + " " + strings.TrimSpace(lastName))
	if name != "" {
		return name
	}
	if tgUserID > 0 {
		return fmt.Sprintf("uid:%d", tgUserID)
	}
	return fmt.Sprintf("user:%d", userID)
}

func (s *Service) ListLogsByTGGroupID(tgGroupID int64, page, pageSize int, action string) (*LogPage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListLogsPage(group.ID, page, pageSize, action)
	if err != nil {
		return nil, err
	}
	return &LogPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) ExportLogsCSVByTGGroupID(tgGroupID int64, action string) (string, []byte, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", nil, err
	}
	items, err := s.repo.ListLogsForExport(group.ID, action, 2000)
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "action", "operator_id", "target_id", "created_at"})
	for _, item := range items {
		_ = w.Write([]string{
			strconv.FormatUint(uint64(item.ID), 10),
			item.Action,
			strconv.FormatUint(uint64(item.OperatorID), 10),
			strconv.FormatUint(uint64(item.TargetID), 10),
			item.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", nil, err
	}
	file := fmt.Sprintf("logs_%d_%s.csv", tgGroupID, time.Now().Format("20060102150405"))
	return file, buf.Bytes(), nil
}
