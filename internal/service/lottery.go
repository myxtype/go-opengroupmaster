package service

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func (s *Service) CreateLotteryByTGGroupID(tgGroupID int64, title string, winners int) (*model.Lottery, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	if winners <= 0 {
		winners = 1
	}
	return s.repo.CreateLottery(group.ID, title, "参加", winners)
}

func (s *Service) JoinActiveLotteryByTGGroupID(tgGroupID int64, tgUser *tgbotapi.User) error {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	lottery, err := s.repo.GetActiveLottery(group.ID)
	if err != nil {
		return err
	}
	u, err := s.repo.UpsertUserFromTG(tgUser)
	if err != nil {
		return err
	}
	_, err = s.repo.JoinLottery(lottery.ID, u.ID)
	return err
}

func (s *Service) DrawActiveLotteryByTGGroupID(tgGroupID int64) ([]model.User, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	lottery, err := s.repo.GetActiveLottery(group.ID)
	if err != nil {
		return nil, err
	}
	ids, err := s.repo.ListLotteryParticipantUserIDs(lottery.ID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errors.New("no participants")
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	count := lottery.WinnersCount
	if count > len(ids) {
		count = len(ids)
	}
	ids = ids[:count]
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	winners := make([]model.User, 0, count)
	for _, id := range ids {
		u, err := s.repo.FindUserByID(id)
		if err != nil {
			continue
		}
		winners = append(winners, *u)
	}
	_ = s.repo.CloseLottery(lottery.ID)
	return winners, nil
}

func (s *Service) LotteryPanelViewByTGGroupID(tgGroupID int64) (*LotteryPanelView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.getLotteryConfig(group.ID)
	if err != nil {
		return nil, err
	}
	out := &LotteryPanelView{
		PublishPin:        cfg.PublishPin,
		ResultPin:         cfg.ResultPin,
		DeleteKeywordMins: cfg.DeleteKeywordMinutes,
	}

	active, err := s.repo.GetActiveLottery(group.ID)
	if err == nil && active != nil {
		count, _ := s.repo.CountLotteryParticipants(active.ID)
		kw := strings.TrimSpace(active.JoinKeyword)
		if kw == "" {
			kw = "参加"
		}
		out.ActiveID = active.ID
		out.ActiveTitle = active.Title
		out.ActiveJoinKeyword = kw
		out.ActiveWinnersCount = active.WinnersCount
		out.ActiveParticipants = count
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	latest, err := s.repo.GetLatestLottery(group.ID)
	if err == nil && latest != nil {
		kw := strings.TrimSpace(latest.JoinKeyword)
		if kw == "" {
			kw = "参加"
		}
		out.LatestID = latest.ID
		out.LatestTitle = latest.Title
		out.LatestJoinKeyword = kw
		out.LatestStatus = latest.Status
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return out, nil
}

func (s *Service) ToggleLotteryConfigByTGGroupID(tgGroupID int64, key string) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	cfg, err := s.getLotteryConfig(group.ID)
	if err != nil {
		return false, err
	}
	var v bool
	switch key {
	case "publish_pin":
		cfg.PublishPin = !cfg.PublishPin
		v = cfg.PublishPin
	case "result_pin":
		cfg.ResultPin = !cfg.ResultPin
		v = cfg.ResultPin
	default:
		return false, errors.New("unknown lottery config key")
	}
	if err := s.saveLotteryConfig(group.ID, cfg); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, "toggle_lottery_"+key, 0, 0)
	return v, nil
}

func (s *Service) CycleLotteryDeleteKeywordMinutesByTGGroupID(tgGroupID int64) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getLotteryConfig(group.ID)
	if err != nil {
		return 0, err
	}
	next := 1
	switch cfg.DeleteKeywordMinutes {
	case 0:
		next = 1
	case 1:
		next = 3
	case 3:
		next = 5
	case 5:
		next = 10
	case 10:
		next = 30
	default:
		next = 0
	}
	cfg.DeleteKeywordMinutes = next
	if err := s.saveLotteryConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_lottery_delete_keyword_%d", next), 0, 0)
	return next, nil
}

func (s *Service) SetLotteryDeleteKeywordMinutesByTGGroupID(tgGroupID int64, minutes int) (int, error) {
	if !isAllowedLotteryDeleteKeywordMinutes(minutes) {
		return 0, errors.New("invalid lottery delete keyword minutes")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	cfg, err := s.getLotteryConfig(group.ID)
	if err != nil {
		return 0, err
	}
	cfg.DeleteKeywordMinutes = minutes
	if err := s.saveLotteryConfig(group.ID, cfg); err != nil {
		return 0, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_lottery_delete_keyword_%d", minutes), 0, 0)
	return minutes, nil
}

func (s *Service) LotteryDeleteKeywordMinutesByGroupID(groupID uint) (int, error) {
	cfg, err := s.getLotteryConfig(groupID)
	if err != nil {
		return 0, err
	}
	return cfg.DeleteKeywordMinutes, nil
}

func isAllowedLotteryDeleteKeywordMinutes(minutes int) bool {
	switch minutes {
	case 0, 1, 3, 5, 10, 30:
		return true
	default:
		return false
	}
}

func (s *Service) PinLotteryMessageByTGGroupID(bot *tgbotapi.BotAPI, tgGroupID int64, messageID int, kind string) error {
	if bot == nil || messageID <= 0 {
		return nil
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return err
	}
	cfg, err := s.getLotteryConfig(group.ID)
	if err != nil {
		return err
	}
	shouldPin := false
	switch kind {
	case "publish":
		shouldPin = cfg.PublishPin
	case "result":
		shouldPin = cfg.ResultPin
	default:
		return errors.New("unknown lottery pin kind")
	}
	if !shouldPin {
		return nil
	}
	_, err = bot.Request(tgbotapi.PinChatMessageConfig{
		ChatID:              tgGroupID,
		MessageID:           messageID,
		DisableNotification: true,
	})
	return err
}

func (s *Service) CreateLotteryByTGGroupIDWithKeyword(tgGroupID int64, title string, winners int, keyword string) (*model.Lottery, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	if winners <= 0 {
		winners = 1
	}
	if strings.TrimSpace(keyword) == "" {
		keyword = "参加"
	}
	return s.repo.CreateLottery(group.ID, title, strings.TrimSpace(keyword), winners)
}

func (s *Service) TryJoinLotteryByKeyword(group *model.Group, tgUser *tgbotapi.User, text string) (bool, bool, error) {
	lottery, err := s.repo.GetActiveLottery(group.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false, nil
		}
		return false, false, err
	}
	keyword := strings.TrimSpace(lottery.JoinKeyword)
	if keyword == "" {
		keyword = "参加"
	}
	if !strings.EqualFold(strings.TrimSpace(text), keyword) {
		return false, false, nil
	}
	u, err := s.repo.UpsertUserFromTG(tgUser)
	if err != nil {
		return true, false, err
	}
	created, err := s.repo.JoinLottery(lottery.ID, u.ID)
	if err != nil {
		return true, false, err
	}
	return true, created, nil
}
