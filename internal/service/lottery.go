package service

import (
	"errors"
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
	out := &LotteryPanelView{}

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
