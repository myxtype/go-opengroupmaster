package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

var ErrInviteFeatureDisabled = errors.New("invite feature disabled")
var ErrInviteGenerateLimitReached = errors.New("invite generate limit reached")

func (s *Service) InvitePanelViewByTGGroupID(tgGroupID int64) (*InvitePanelView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureInvite, false)
	if err != nil {
		return nil, err
	}
	cfg, err := s.getInviteConfig(group.ID)
	if err != nil {
		return nil, err
	}
	totalInvited, err := s.repo.CountInviteEvents(group.ID)
	if err != nil {
		return nil, err
	}
	generatedCount, err := s.repo.CountInviteLinks(group.ID)
	if err != nil {
		return nil, err
	}
	return &InvitePanelView{
		Enabled:        enabled,
		TotalInvited:   totalInvited,
		ExpireDate:     cfg.ExpireDate,
		MemberLimit:    cfg.MemberLimit,
		GenerateLimit:  cfg.GenerateLimit,
		GeneratedCount: generatedCount,
	}, nil
}

func (s *Service) SetInviteEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	if err := s.repo.UpsertFeatureEnabled(group.ID, featureInvite, enabled); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_invite_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) SetInviteExpireDateByTGGroupID(tgGroupID int64, expireDate int64) (int64, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	if expireDate < 0 {
		return 0, errors.New("invalid invite expire date")
	}
	if expireDate > 0 && expireDate <= time.Now().Unix() {
		return 0, errors.New("invite expire date must be in future")
	}
	cfg, err := s.getInviteConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.ExpireDate = expireDate
	if err := s.saveInviteConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, "set_invite_expire_date", 0, 0)
	return cfg.ExpireDate, nil
}

func (s *Service) SetInviteMemberLimitByTGGroupID(tgGroupID int64, limit int) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	if limit < 0 || limit > 99999 {
		return 0, errors.New("invalid invite member limit")
	}
	cfg, err := s.getInviteConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.MemberLimit = limit
	if err := s.saveInviteConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, "set_invite_member_limit", 0, 0)
	return cfg.MemberLimit, nil
}

func (s *Service) SetInviteGenerateLimitByTGGroupID(tgGroupID int64, limit int) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	if limit < 0 {
		return 0, errors.New("invalid invite generate limit")
	}
	cfg, err := s.getInviteConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.GenerateLimit = limit
	if err := s.saveInviteConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, "set_invite_generate_limit", 0, 0)
	return cfg.GenerateLimit, nil
}

func (s *Service) InviteUserStatsByTGGroupID(tgGroupID, tgUserID int64) (*InviteUserStats, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	return s.inviteUserStatsByGroupID(group.ID, tgUserID)
}

func (s *Service) inviteUserStatsByGroupID(groupID uint, tgUserID int64) (*InviteUserStats, error) {
	invitedCount, err := s.repo.CountInviteEventsByInviter(groupID, tgUserID)
	if err != nil {
		return nil, err
	}
	generatedCount, err := s.repo.CountInviteLinksByCreator(groupID, tgUserID)
	if err != nil {
		return nil, err
	}
	return &InviteUserStats{
		InvitedCount:   invitedCount,
		GeneratedCount: generatedCount,
	}, nil
}

func (s *Service) CreateInviteLinkForUserByTGGroupID(bot *tgbotapi.BotAPI, tgGroupID, tgUserID int64) (*InviteGenerateResult, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	enabled, err := s.IsFeatureEnabled(group.ID, featureInvite, false)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrInviteFeatureDisabled
	}
	cfg, err := s.getInviteConfig(group.ID)
	if err != nil {
		return nil, err
	}

	// Prefer returning user's latest still-usable link to avoid generating a new one on every /link query.
	latestLink, err := s.repo.FindLatestInviteLinkByCreator(group.ID, tgUserID)
	if err == nil {
		reusable, reErr := s.isInviteLinkReusable(group.ID, latestLink)
		if reErr != nil {
			return nil, reErr
		}
		if reusable {
			groupGenerated, cErr := s.repo.CountInviteLinks(group.ID)
			if cErr != nil {
				return nil, cErr
			}
			userStats, sErr := s.inviteUserStatsByGroupID(group.ID, tgUserID)
			if sErr != nil {
				return nil, sErr
			}
			return &InviteGenerateResult{
				Link:           latestLink.Link,
				UserStats:      *userStats,
				GroupGenerated: groupGenerated,
				GenerateLimit:  cfg.GenerateLimit,
			}, nil
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	generatedCount, err := s.repo.CountInviteLinks(group.ID)
	if err != nil {
		return nil, err
	}
	if cfg.GenerateLimit > 0 && generatedCount >= int64(cfg.GenerateLimit) {
		return nil, ErrInviteGenerateLimitReached
	}
	if cfg.ExpireDate > 0 && cfg.ExpireDate <= time.Now().Unix() {
		return nil, errors.New("invite expire date must be in future")
	}

	req := tgbotapi.CreateChatInviteLinkConfig{
		ChatConfig: tgbotapi.ChatConfig{ChatID: tgGroupID},
	}
	if cfg.ExpireDate > 0 {
		req.ExpireDate = int(cfg.ExpireDate)
	}
	if cfg.MemberLimit > 0 {
		req.MemberLimit = cfg.MemberLimit
	}
	resp, err := bot.Request(req)
	if err != nil {
		return nil, err
	}
	var chatInvite tgbotapi.ChatInviteLink
	if err := json.Unmarshal(resp.Result, &chatInvite); err != nil {
		return nil, err
	}
	if strings.TrimSpace(chatInvite.InviteLink) == "" {
		return nil, errors.New("telegram returned empty invite link")
	}

	item := &model.InviteLink{
		GroupID:         group.ID,
		CreatorTGUserID: tgUserID,
		Link:            chatInvite.InviteLink,
		ExpireDate:      int64(chatInvite.ExpireDate),
		MemberLimit:     chatInvite.MemberLimit,
	}
	if item.ExpireDate == 0 {
		item.ExpireDate = cfg.ExpireDate
	}
	if item.MemberLimit == 0 {
		item.MemberLimit = cfg.MemberLimit
	}
	if err := s.repo.CreateInviteLink(item); err != nil {
		return nil, err
	}
	_ = s.repo.CreateLog(group.ID, "invite_link_generated", 0, 0)

	groupGenerated, err := s.repo.CountInviteLinks(group.ID)
	if err != nil {
		return nil, err
	}
	userStats, err := s.inviteUserStatsByGroupID(group.ID, tgUserID)
	if err != nil {
		return nil, err
	}
	return &InviteGenerateResult{
		Link:           item.Link,
		UserStats:      *userStats,
		GroupGenerated: groupGenerated,
		GenerateLimit:  cfg.GenerateLimit,
	}, nil
}

func (s *Service) isInviteLinkReusable(groupID uint, link *model.InviteLink) (bool, error) {
	if link == nil || strings.TrimSpace(link.Link) == "" {
		return false, nil
	}
	if link.ExpireDate > 0 && link.ExpireDate <= time.Now().Unix() {
		return false, nil
	}
	if link.MemberLimit > 0 {
		used, err := s.repo.CountInviteEventsByLink(groupID, link.Link)
		if err != nil {
			return false, err
		}
		if used >= int64(link.MemberLimit) {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) TrackInviteByChatMemberUpdate(update *tgbotapi.ChatMemberUpdated) error {
	if update == nil || update.NewChatMember.User == nil {
		return nil
	}
	if !isJoinEvent(update.OldChatMember, update.NewChatMember) {
		return nil
	}
	user := update.NewChatMember.User
	if user.IsBot {
		return nil
	}
	group, err := s.repo.FindGroupByTGID(update.Chat.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	_, _ = s.repo.UpsertUserFromTG(user)

	joinAt := time.Unix(int64(update.Date), 0)
	if update.Date <= 0 {
		joinAt = time.Now()
	}
	firstJoin, err := s.repo.MarkGroupMemberFirstJoin(group.ID, user.ID, joinAt)
	if err != nil {
		return err
	}
	if update.InviteLink == nil || strings.TrimSpace(update.InviteLink.InviteLink) == "" {
		return nil
	}
	if !firstJoin {
		return nil
	}
	linkRow, err := s.repo.FindInviteLinkByLink(group.ID, strings.TrimSpace(update.InviteLink.InviteLink))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if linkRow.CreatorTGUserID == 0 || linkRow.CreatorTGUserID == user.ID {
		return nil
	}
	created, err := s.repo.CreateInviteEvent(&model.InviteEvent{
		GroupID:         group.ID,
		InviterTGUserID: linkRow.CreatorTGUserID,
		InviteeTGUserID: user.ID,
		Link:            linkRow.Link,
		JoinedAt:        joinAt,
	})
	if err != nil {
		return err
	}
	if created {
		_ = s.rewardInvitePoints(group.ID, linkRow.CreatorTGUserID)
		_ = s.repo.CreateLog(group.ID, "invite_valid_join", 0, 0)
	}
	return nil
}

func isJoinEvent(oldChatMember, newChatMember tgbotapi.ChatMember) bool {
	return !isActiveChatMember(oldChatMember) && isActiveChatMember(newChatMember)
}

func isActiveChatMember(chatMember tgbotapi.ChatMember) bool {
	switch chatMember.Status {
	case "member", "administrator", "creator":
		return true
	case "restricted":
		return chatMember.IsMember
	default:
		return false
	}
}

func (s *Service) ExportInviteCSVByTGGroupID(tgGroupID int64) (string, []byte, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", nil, err
	}
	items, err := s.repo.ListInviteEventsForExport(group.ID, 5000)
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "inviter_tg_user_id", "invitee_tg_user_id", "invite_link", "joined_at"})
	for _, item := range items {
		_ = w.Write([]string{
			strconv.FormatUint(uint64(item.ID), 10),
			strconv.FormatInt(item.InviterTGUserID, 10),
			strconv.FormatInt(item.InviteeTGUserID, 10),
			item.Link,
			item.JoinedAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", nil, err
	}
	file := fmt.Sprintf("invites_%d_%s.csv", tgGroupID, time.Now().Format("20060102150405"))
	return file, buf.Bytes(), nil
}

func (s *Service) ClearInviteDataByTGGroupID(tgGroupID int64) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	if err := s.repo.ClearInviteData(group.ID); err != nil {
		return err
	}
	_ = s.repo.CreateLog(group.ID, "invite_data_cleared", 0, 0)
	return nil
}
