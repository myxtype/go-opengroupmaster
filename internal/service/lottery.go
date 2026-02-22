package service

import (
	"errors"
	"math/rand"
	"sort"
	"time"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *Service) CreateLotteryByTGGroupID(tgGroupID int64, title string, winners int) (*model.Lottery, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	if winners <= 0 {
		winners = 1
	}
	return s.repo.CreateLottery(group.ID, title, winners)
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
	return s.repo.JoinLottery(lottery.ID, u.ID)
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
