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
	top, err := s.repo.TopUsersByPoints(group.ID, limit)
	if err != nil {
		return nil, err
	}
	out := &GroupStats{GroupTitle: group.Title, GroupID: group.TGGroupID}
	for _, row := range top {
		user, err := s.repo.FindUserByID(row.UserID)
		if err != nil {
			continue
		}
		name := user.Username
		if name == "" {
			name = strings.TrimSpace(user.FirstName + " " + user.LastName)
		}
		if name == "" {
			name = fmt.Sprintf("uid:%d", user.TGUserID)
		}
		out.TopUsers = append(out.TopUsers, UserScore{DisplayName: name, Points: row.Points})
	}
	return out, nil
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
