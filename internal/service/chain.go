package service

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"supervisor/internal/model"

	"gorm.io/gorm"
)

var ErrChainNotActive = errors.New("chain not active")
var ErrChainDeadlineReached = errors.New("chain deadline reached")
var ErrChainParticipantLimitReached = errors.New("chain participant limit reached")

func (s *Service) StartChainByTGGroupID(tgGroupID int64, intro string, maxParticipants int, deadlineUnix int64) (uint, error) {
	intro = strings.TrimSpace(intro)
	if intro == "" {
		return 0, errors.New("chain intro is required")
	}
	if maxParticipants < 0 {
		return 0, errors.New("invalid max participants")
	}
	if deadlineUnix > 0 && deadlineUnix <= time.Now().Unix() {
		return 0, errors.New("invalid chain deadline")
	}

	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	item, err := s.repo.CreateChain(group.ID, intro, maxParticipants, deadlineUnix)
	if err != nil {
		return 0, err
	}
	if err := s.repo.CreateLog(group.ID, "chain_start", 0, 0); err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (s *Service) SubmitChainEntryByChainID(chainID uint, tgUserID int64, displayName, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("empty chain content")
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = fmt.Sprintf("uid:%d", tgUserID)
	}

	chain, group, err := s.getChainAndGroupByID(chainID)
	if err != nil {
		return err
	}
	active, deadlineReached, err := s.ensureChainActive(chain)
	if err != nil {
		return err
	}
	if deadlineReached {
		_ = s.repo.CreateLog(group.ID, "chain_auto_close_timeout", 0, 0)
		return ErrChainDeadlineReached
	}
	if !active {
		return ErrChainNotActive
	}

	existed, err := s.repo.GetChainEntry(chain.ID, tgUserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	isNew := errors.Is(err, gorm.ErrRecordNotFound) || existed == nil
	if isNew && chain.MaxParticipants > 0 {
		total, countErr := s.repo.CountChainEntries(chain.ID)
		if countErr != nil {
			return countErr
		}
		if int(total) >= chain.MaxParticipants {
			return ErrChainParticipantLimitReached
		}
	}
	if _, err := s.repo.UpsertChainEntry(chain.ID, tgUserID, displayName, content); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "chain_submit_entry", 0, 0)
}

func (s *Service) UserChainEntryByChainID(chainID uint, tgUserID int64) (string, bool, error) {
	chain, _, err := s.getChainAndGroupByID(chainID)
	if err != nil {
		return "", false, err
	}
	active, _, err := s.ensureChainActive(chain)
	if err != nil {
		return "", false, err
	}
	if !active {
		return "", false, nil
	}
	item, err := s.repo.GetChainEntry(chain.ID, tgUserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return item.Content, true, nil
}

func (s *Service) SetChainAnnouncementMessageID(chainID uint, messageID int) error {
	_, _, err := s.getChainAndGroupByID(chainID)
	if err != nil {
		return err
	}
	return s.repo.SetChainAnnouncementMessageID(chainID, messageID)
}

func (s *Service) CloseChainByTGGroupIDAndChainID(tgGroupID int64, chainID uint) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	chain, err := s.repo.GetChainByID(chainID)
	if err != nil {
		return err
	}
	if chain.GroupID != group.ID {
		return errors.New("chain not in group")
	}
	if chain.Status != "active" {
		return nil
	}
	if err := s.repo.CloseChain(chain.ID); err != nil {
		return err
	}
	return s.repo.CreateLog(group.ID, "chain_close", 0, 0)
}

func (s *Service) ChainViewByChainID(chainID uint) (*ChainView, error) {
	chain, group, err := s.getChainAndGroupByID(chainID)
	if err != nil {
		return nil, err
	}
	active, deadlineReached, err := s.ensureChainActive(chain)
	if err != nil {
		return nil, err
	}
	if deadlineReached {
		_ = s.repo.CreateLog(group.ID, "chain_auto_close_timeout", 0, 0)
	}
	entries, err := s.repo.ListChainEntries(chain.ID)
	if err != nil {
		return nil, err
	}
	view := buildChainView(chain, group.TGGroupID, entries)
	view.Active = active
	return view, nil
}

func (s *Service) ChainViewByTGGroupID(tgGroupID int64) (*ChainView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	chain, err := s.repo.GetLatestChain(group.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &ChainView{Active: false, TGGroupID: tgGroupID, Entries: []ChainEntryView{}}, nil
	}
	if err != nil {
		return nil, err
	}
	active, deadlineReached, err := s.ensureChainActive(chain)
	if err != nil {
		return nil, err
	}
	if deadlineReached {
		_ = s.repo.CreateLog(group.ID, "chain_auto_close_timeout", 0, 0)
	}
	entries, err := s.repo.ListChainEntries(chain.ID)
	if err != nil {
		return nil, err
	}
	view := buildChainView(chain, tgGroupID, entries)
	view.Active = active
	return view, nil
}

func (s *Service) ListActiveChainSummariesByTGGroupID(tgGroupID int64, limit int) ([]ChainSummary, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListActiveChains(group.ID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ChainSummary, 0, len(items))
	for i := range items {
		active, deadlineReached, err := s.ensureChainActive(&items[i])
		if err != nil {
			return nil, err
		}
		if deadlineReached {
			_ = s.repo.CreateLog(group.ID, "chain_auto_close_timeout", 0, 0)
		}
		if !active {
			continue
		}
		total, err := s.repo.CountChainEntries(items[i].ID)
		if err != nil {
			return nil, err
		}
		out = append(out, ChainSummary{
			ID:              items[i].ID,
			Intro:           items[i].Intro,
			MaxParticipants: items[i].MaxParticipants,
			DeadlineUnix:    items[i].DeadlineUnix,
			Participants:    total,
		})
	}
	return out, nil
}

func (s *Service) ExportChainCSVByTGGroupIDAndChainID(tgGroupID int64, chainID uint) (string, []byte, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return "", nil, err
	}
	chain, err := s.repo.GetChainByID(chainID)
	if err != nil {
		return "", nil, err
	}
	if chain.GroupID != group.ID {
		return "", nil, errors.New("chain not in group")
	}
	items, err := s.repo.ListChainEntriesForExport(chain.ID, 5000)
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"chain_id", "index", "tg_user_id", "display_name", "content", "updated_at"})
	for i, item := range items {
		_ = w.Write([]string{
			strconv.FormatUint(uint64(chain.ID), 10),
			strconv.Itoa(i + 1),
			strconv.FormatInt(item.TGUserID, 10),
			item.DisplayName,
			item.Content,
			item.UpdatedAt.In(time.Local).Format(time.RFC3339),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", nil, err
	}
	file := fmt.Sprintf("chain_%d_%s.csv", chainID, time.Now().Format("20060102150405"))
	return file, buf.Bytes(), nil
}

func (s *Service) getChainAndGroupByID(chainID uint) (*model.Chain, *model.Group, error) {
	chain, err := s.repo.GetChainByID(chainID)
	if err != nil {
		return nil, nil, err
	}
	group, err := s.repo.FindGroupByID(chain.GroupID)
	if err != nil {
		return nil, nil, err
	}
	return chain, group, nil
}

func (s *Service) ensureChainActive(chain *model.Chain) (bool, bool, error) {
	if chain == nil {
		return false, false, nil
	}
	if chain.Status != "active" {
		return false, false, nil
	}
	if chain.DeadlineUnix > 0 && time.Now().Unix() >= chain.DeadlineUnix {
		if err := s.repo.CloseChain(chain.ID); err != nil {
			return false, false, err
		}
		chain.Status = "closed"
		return false, true, nil
	}
	return true, false, nil
}

func buildChainView(chain *model.Chain, tgGroupID int64, entries []model.ChainEntry) *ChainView {
	if chain == nil {
		return &ChainView{Active: false, TGGroupID: tgGroupID, Entries: []ChainEntryView{}}
	}
	out := make([]ChainEntryView, 0, len(entries))
	for _, item := range entries {
		out = append(out, ChainEntryView{
			TGUserID:    item.TGUserID,
			DisplayName: item.DisplayName,
			Content:     item.Content,
			UpdatedAt:   item.UpdatedAt.Unix(),
		})
	}
	return &ChainView{
		ID:                    chain.ID,
		TGGroupID:             tgGroupID,
		Active:                chain.Status == "active",
		Intro:                 chain.Intro,
		MaxParticipants:       chain.MaxParticipants,
		DeadlineUnix:          chain.DeadlineUnix,
		AnnouncementMessageID: chain.AnnouncementMessageID,
		Entries:               out,
	}
}
