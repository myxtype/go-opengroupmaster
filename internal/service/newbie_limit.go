package service

import (
	"context"
	"fmt"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (s *Service) NewbieLimitViewByTGGroupID(tgGroupID int64) (*NewbieLimitView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureNewbieLimit, false)
	if err != nil {
		return nil, err
	}
	minutes, err := s.getNewbieLimitMinutes(group.ID)
	if err != nil {
		return nil, err
	}
	return &NewbieLimitView{
		Enabled: enabled,
		Minutes: minutes,
	}, nil
}

func (s *Service) SetNewbieLimitEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	if err := s.repo.UpsertFeatureEnabled(group.ID, featureNewbieLimit, enabled); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_newbie_limit_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) newbieLimitRestrictionDeadline(groupID uint, joinedAt time.Time) (time.Time, bool, error) {
	enabled, err := s.IsFeatureEnabled(groupID, featureNewbieLimit, false)
	if err != nil {
		return time.Time{}, false, err
	}
	if !enabled {
		return time.Time{}, false, nil
	}
	minutes, err := s.getNewbieLimitMinutes(groupID)
	if err != nil {
		return time.Time{}, false, err
	}
	if minutes <= 0 {
		return time.Time{}, false, nil
	}
	if joinedAt.IsZero() {
		joinedAt = time.Now()
	}
	deadline := joinedAt.Add(time.Duration(minutes) * time.Minute)
	if !deadline.After(time.Now()) {
		return deadline, false, nil
	}
	return deadline, true, nil
}

func (s *Service) restrictMemberNoSpeak(bot *tgbot.Bot, tgGroupID, tgUserID int64, until time.Time) error {
	if bot == nil || until.IsZero() {
		return nil
	}
	_, err := bot.RestrictChatMember(context.Background(), &tgbot.RestrictChatMemberParams{
		ChatID:      tgGroupID,
		UserID:      tgUserID,
		UntilDate:   int(until.Unix()),
		Permissions: &models.ChatPermissions{},
	})
	return err
}

func (s *Service) restoreMemberSpeak(bot *tgbot.Bot, tgGroupID, tgUserID int64) error {
	if bot == nil {
		return nil
	}
	perms := &models.ChatPermissions{
		CanSendMessages:       true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
	}
	_, err := bot.RestrictChatMember(context.Background(), &tgbot.RestrictChatMemberParams{
		ChatID:      tgGroupID,
		UserID:      tgUserID,
		Permissions: perms,
	})
	return err
}
