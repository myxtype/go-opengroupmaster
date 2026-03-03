package service

import (
	"context"
	"errors"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (s *Service) HandleSystemMessageCleanup(bot *tgbot.Bot, msg *models.Message) error {
	if msg == nil {
		return nil
	}
	group, err := s.repo.FindGroupByTGID(msg.Chat.ID)
	if err != nil {
		return nil
	}
	cfg, err := s.getSystemCleanConfig(group.ID)
	if err != nil {
		return err
	}

	shouldDelete := false
	action := ""
	switch {
	case len(msg.NewChatMembers) > 0:
		shouldDelete = cfg.Join
		action = "clean_join_message"
	case msg.LeftChatMember != nil:
		shouldDelete = cfg.Leave
		action = "clean_leave_message"
	case msg.PinnedMessage != nil:
		shouldDelete = cfg.Pin
		action = "clean_pin_message"
	case len(msg.NewChatPhoto) > 0 || msg.DeleteChatPhoto:
		shouldDelete = cfg.Photo
		action = "clean_photo_change_message"
	case msg.NewChatTitle != "":
		shouldDelete = cfg.Title
		action = "clean_title_change_message"
	}
	if !shouldDelete || action == "" {
		return nil
	}
	_, _ = bot.DeleteMessage(context.Background(), &tgbot.DeleteMessageParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
	})
	_ = s.repo.CreateLog(group.ID, action, 0, 0)
	return nil
}

func (s *Service) SystemCleanViewByTGGroupID(tgGroupID int64) (*SystemCleanView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.getSystemCleanConfig(group.ID)
	if err != nil {
		return nil, err
	}
	return &SystemCleanView{
		Join:  cfg.Join,
		Leave: cfg.Leave,
		Pin:   cfg.Pin,
		Photo: cfg.Photo,
		Title: cfg.Title,
	}, nil
}

func (s *Service) ToggleSystemCleanByTGGroupID(tgGroupID int64, key string) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	cfg, err := s.getSystemCleanConfig(group.ID)
	if err != nil {
		return false, err
	}
	var v bool
	switch key {
	case "join":
		cfg.Join = !cfg.Join
		v = cfg.Join
	case "leave":
		cfg.Leave = !cfg.Leave
		v = cfg.Leave
	case "pin":
		cfg.Pin = !cfg.Pin
		v = cfg.Pin
	case "photo":
		cfg.Photo = !cfg.Photo
		v = cfg.Photo
	case "title":
		cfg.Title = !cfg.Title
		v = cfg.Title
	default:
		return false, errors.New("unknown system clean key")
	}
	if err := s.saveSystemCleanConfig(group.ID, cfg); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, "toggle_system_clean_"+key, 0, 0)
	return v, nil
}

func (s *Service) ApplySystemCleanPresetByTGGroupID(tgGroupID int64, preset string) (*SystemCleanView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	var cfg systemCleanConfig
	switch preset {
	case "strict":
		cfg = systemCleanConfig{Join: true, Leave: true, Pin: true, Photo: true, Title: true}
	case "off":
		cfg = systemCleanConfig{Join: false, Leave: false, Pin: false, Photo: false, Title: false}
	default:
		// recommended
		cfg = systemCleanConfig{Join: true, Leave: true, Pin: false, Photo: false, Title: false}
		preset = "recommended"
	}
	if err := s.saveSystemCleanConfig(group.ID, cfg); err != nil {
		return nil, err
	}
	_ = s.repo.CreateLog(group.ID, "apply_system_clean_preset_"+preset, 0, 0)
	return &SystemCleanView{
		Join: cfg.Join, Leave: cfg.Leave, Pin: cfg.Pin, Photo: cfg.Photo, Title: cfg.Title,
	}, nil
}
