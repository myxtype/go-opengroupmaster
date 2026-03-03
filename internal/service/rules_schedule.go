package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"supervisor/internal/model"

	"github.com/robfig/cron/v3"
)

const maxScheduledMessagesPerGroup = 3

func (s *Service) AddAutoReplyByTGGroupID(tgGroupID int64, keyword, reply, matchType string) error {
	return s.AddAutoReplyByTGGroupIDWithButtons(tgGroupID, keyword, reply, matchType, "")
}

func (s *Service) AddAutoReplyByTGGroupIDWithButtons(tgGroupID int64, keyword, reply, matchType, rawButtons string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	matchType = normalizeAutoReplyMatchType(matchType)
	if matchType == "regex" {
		if _, err := regexp.Compile(keyword); err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	}
	buttonRows, err := parseAndEncodeButtonRows(rawButtons)
	if err != nil {
		return err
	}
	return s.repo.CreateAutoReply(group.ID, keyword, reply, matchType, buttonRows)
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
	if matchType == "regex" {
		if _, err := regexp.Compile(keyword); err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	}
	return s.repo.UpdateAutoReply(group.ID, id, keyword, reply, matchType)
}

func (s *Service) UpdateAutoReplyByTGGroupIDWithButtons(tgGroupID int64, id uint, keyword, reply, matchType, rawButtons string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	matchType = normalizeAutoReplyMatchType(matchType)
	if matchType == "regex" {
		if _, err := regexp.Compile(keyword); err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	}
	buttonRows, err := parseAndEncodeButtonRows(rawButtons)
	if err != nil {
		return err
	}
	return s.repo.UpdateAutoReplyWithButtons(group.ID, id, keyword, reply, matchType, buttonRows)
}

func normalizeAutoReplyMatchType(matchType string) string {
	switch strings.TrimSpace(strings.ToLower(matchType)) {
	case "contains":
		return "contains"
	case "regex":
		return "regex"
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
	return s.CreateScheduledMessageByTGGroupIDAdvanced(tgGroupID, content, cronExpr, "", "", "", false)
}

func (s *Service) CreateScheduledMessageByTGGroupIDWithButtons(tgGroupID int64, content, cronExpr, rawButtons string) error {
	return s.CreateScheduledMessageByTGGroupIDAdvanced(tgGroupID, content, cronExpr, rawButtons, "", "", false)
}

func (s *Service) CreateScheduledMessageByTGGroupIDAdvanced(tgGroupID int64, content, cronExpr, rawButtons, mediaType, mediaFileID string, pinMessage bool) error {
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

	buttonRows, err := parseAndEncodeButtonRows(rawButtons)
	if err != nil {
		return err
	}
	mediaType, mediaFileID, err = normalizeScheduledMedia(mediaType, mediaFileID)
	if err != nil {
		return err
	}
	content = strings.TrimSpace(content)
	if content == "" && mediaType == "" {
		return errors.New("empty scheduled message")
	}

	item, err := s.repo.CreateScheduledMessage(group.ID, content, cronExpr, buttonRows, mediaType, mediaFileID, pinMessage)
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
	enabled, err := s.repo.ToggleScheduledMessage(group.ID, id)
	if err != nil {
		return false, err
	}
	if err := s.reloadScheduledJob(group.ID, id); err != nil {
		return false, err
	}
	return enabled, nil
}

func (s *Service) ToggleScheduledPinByTGGroupID(tgGroupID int64, id uint) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	pin, err := s.repo.ToggleScheduledPinMessage(group.ID, id)
	if err != nil {
		return false, err
	}
	if err := s.reloadScheduledJob(group.ID, id); err != nil {
		return false, err
	}
	return pin, nil
}

func (s *Service) GetScheduledMessageByTGGroupID(tgGroupID int64, id uint) (*model.ScheduledMessage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetScheduledMessage(group.ID, id)
}

func (s *Service) UpdateScheduledTextByTGGroupID(tgGroupID int64, id uint, text string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	item, err := s.repo.GetScheduledMessage(group.ID, id)
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if text == "" && strings.TrimSpace(item.MediaType) == "" {
		return errors.New("empty scheduled message")
	}
	item.Content = text
	if err := s.repo.SaveScheduledMessage(item); err != nil {
		return err
	}
	return s.reloadScheduledJob(group.ID, id)
}

func (s *Service) UpdateScheduledCronByTGGroupID(tgGroupID int64, id uint, cronExpr string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cronExpr = strings.TrimSpace(cronExpr)
	if err := s.ValidateCronExpr(cronExpr); err != nil {
		return err
	}
	item, err := s.repo.GetScheduledMessage(group.ID, id)
	if err != nil {
		return err
	}
	item.CronExpr = cronExpr
	if err := s.repo.SaveScheduledMessage(item); err != nil {
		return err
	}
	return s.reloadScheduledJob(group.ID, id)
}

func (s *Service) UpdateScheduledButtonsByTGGroupID(tgGroupID int64, id uint, rawButtons string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	buttonRows, err := parseAndEncodeButtonRows(rawButtons)
	if err != nil {
		return err
	}
	item, err := s.repo.GetScheduledMessage(group.ID, id)
	if err != nil {
		return err
	}
	item.ButtonRows = buttonRows
	if err := s.repo.SaveScheduledMessage(item); err != nil {
		return err
	}
	return s.reloadScheduledJob(group.ID, id)
}

func (s *Service) UpdateScheduledMediaByTGGroupID(tgGroupID int64, id uint, mediaType, mediaFileID string) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	item, err := s.repo.GetScheduledMessage(group.ID, id)
	if err != nil {
		return err
	}
	mediaType, mediaFileID, err = normalizeScheduledMedia(mediaType, mediaFileID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(item.Content) == "" && mediaType == "" {
		return errors.New("empty scheduled message")
	}
	item.MediaType = mediaType
	item.MediaFileID = mediaFileID
	if err := s.repo.SaveScheduledMessage(item); err != nil {
		return err
	}
	return s.reloadScheduledJob(group.ID, id)
}

func (s *Service) reloadScheduledJob(groupID, id uint) error {
	if s.scheduleRuntime == nil {
		return nil
	}
	item, err := s.repo.GetScheduledMessage(groupID, id)
	if err != nil {
		return err
	}
	return s.scheduleRuntime.AddJob(*item)
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

func normalizeScheduledMedia(mediaType, mediaFileID string) (string, string, error) {
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))
	mediaFileID = strings.TrimSpace(mediaFileID)
	if mediaType == "" && mediaFileID == "" {
		return "", "", nil
	}
	switch mediaType {
	case "photo", "video", "document", "animation":
	default:
		return "", "", errors.New("unsupported media type")
	}
	if mediaFileID == "" {
		return "", "", errors.New("media file id is required")
	}
	return mediaType, mediaFileID, nil
}
