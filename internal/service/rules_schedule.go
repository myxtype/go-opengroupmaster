package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
)

const maxScheduledMessagesPerGroup = 3

func (s *Service) AddAutoReplyByTGGroupID(tgGroupID int64, keyword, reply, matchType string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	matchType = normalizeAutoReplyMatchType(matchType)
	return s.repo.CreateAutoReply(group.ID, keyword, reply, matchType)
}

func (s *Service) AddBannedWordByTGGroupID(tgGroupID int64, word string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.CreateBannedWord(group.ID, word)
}

func (s *Service) ListAutoRepliesByTGGroupID(tgGroupID int64, page, pageSize int) (*AutoReplyPage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListAutoRepliesPage(group.ID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &AutoReplyPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) DeleteAutoReplyByTGGroupID(tgGroupID int64, id uint) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.DeleteAutoReply(group.ID, id)
}

func (s *Service) UpdateAutoReplyByTGGroupID(tgGroupID int64, id uint, keyword, reply, matchType string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	matchType = normalizeAutoReplyMatchType(matchType)
	return s.repo.UpdateAutoReply(group.ID, id, keyword, reply, matchType)
}

func normalizeAutoReplyMatchType(matchType string) string {
	switch strings.TrimSpace(strings.ToLower(matchType)) {
	case "contains":
		return "contains"
	default:
		return "exact"
	}
}

func (s *Service) ListBannedWordsByTGGroupID(tgGroupID int64, page, pageSize int) (*BannedWordPage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListBannedWordsPage(group.ID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &BannedWordPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) DeleteBannedWordByTGGroupID(tgGroupID int64, id uint) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.DeleteBannedWord(group.ID, id)
}

func (s *Service) UpdateBannedWordByTGGroupID(tgGroupID int64, id uint, word string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.UpdateBannedWord(group.ID, id, word)
}

func (s *Service) CreateScheduledMessageByTGGroupID(tgGroupID int64, content, cronExpr string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	total, err := s.repo.CountScheduledMessages(group.ID)
	if err != nil {
		return err
	}
	if total >= maxScheduledMessagesPerGroup {
		return fmt.Errorf("每个群最多创建 %d 条定时消息", maxScheduledMessagesPerGroup)
	}

	item, err := s.repo.CreateScheduledMessage(group.ID, content, cronExpr)
	if err != nil {
		return err
	}
	if s.scheduleRuntime == nil {
		return nil
	}
	if err := s.scheduleRuntime.AddJob(*item); err != nil {
		_ = s.repo.DeleteScheduledMessage(group.ID, item.ID)
		return fmt.Errorf("cron 表达式无效或调度注册失败: %w", err)
	}
	return nil
}

func (s *Service) ListScheduledMessagesByTGGroupID(tgGroupID int64, page, pageSize int) (*ScheduledMessagePage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListScheduledMessagesPage(group.ID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &ScheduledMessagePage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) DeleteScheduledMessageByTGGroupID(tgGroupID int64, id uint) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	return s.repo.DeleteScheduledMessage(group.ID, id)
}

func (s *Service) ToggleScheduledMessageByTGGroupID(tgGroupID int64, id uint) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	return s.repo.ToggleScheduledMessage(group.ID, id)
}
func (s *Service) ParseScheduledInput(raw string) (cronExpr, content string, err error) {
	parts := strings.SplitN(raw, "=>", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid format")
	}
	cronExpr = strings.TrimSpace(parts[0])
	content = strings.TrimSpace(parts[1])
	if cronExpr == "" || content == "" {
		return "", "", errors.New("empty field")
	}
	return cronExpr, content, nil
}

func (s *Service) ValidateCronExpr(cronExpr string) error {
	_, err := cron.ParseStandard(strings.TrimSpace(cronExpr))
	return err
}
