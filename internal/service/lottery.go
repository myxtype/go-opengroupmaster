package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"supervisor/internal/model"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gorm.io/gorm"
)

const (
	lotteryStatusActive   = "active"
	lotteryStatusClosed   = "closed"
	lotteryStatusCanceled = "canceled"
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

func (s *Service) JoinActiveLotteryByTGGroupID(tgGroupID int64, tgUser *models.User) error {
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

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

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
	if out.CreatedTotal, err = s.repo.CountLotteries(group.ID); err != nil {
		return nil, err
	}
	if out.DrawnTotal, err = s.repo.CountLotteriesByStatus(group.ID, lotteryStatusClosed); err != nil {
		return nil, err
	}
	if out.PendingTotal, err = s.repo.CountLotteriesByStatus(group.ID, lotteryStatusActive); err != nil {
		return nil, err
	}
	if out.CanceledTotal, err = s.repo.CountLotteriesByStatus(group.ID, lotteryStatusCanceled); err != nil {
		return nil, err
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

func (s *Service) ListLotteryRecordsByTGGroupID(tgGroupID int64, page, pageSize int) (*LotteryRecordPage, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListLotteriesPage(group.ID, page, pageSize)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	participantMap, err := s.repo.CountLotteryParticipantsByLotteryIDs(ids)
	if err != nil {
		return nil, err
	}
	out := make([]LotteryRecordItem, 0, len(items))
	for _, item := range items {
		out = append(out, LotteryRecordItem{
			Lottery:      item,
			Participants: participantMap[item.ID],
		})
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 5
	}
	return &LotteryRecordPage{
		Items:    out,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func (s *Service) CancelLotteryByTGGroupID(tgGroupID int64, lotteryID uint) (bool, error) {
	if lotteryID == 0 {
		return false, errors.New("invalid lottery id")
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	ok, err := s.repo.CancelLottery(group.ID, lotteryID)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("cancel_lottery_%d", lotteryID), 0, 0)
	return true, nil
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

func (s *Service) PinLotteryMessageByTGGroupID(bot *tgbot.Bot, tgGroupID int64, messageID int, kind string) error {
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
	_, err = bot.PinChatMessage(context.Background(), &tgbot.PinChatMessageParams{
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

func (s *Service) TryJoinLotteryByKeyword(group *model.Group, tgUser *models.User, text string) (bool, bool, error) {
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
	existed, err := s.repo.IsLotteryParticipant(lottery.ID, u.ID)
	if err != nil {
		return true, false, err
	}
	if existed {
		return true, false, nil
	}
	enabled, err := s.pointsEnabled(group.ID)
	if err != nil {
		return true, false, err
	}
	cost := 0
	if enabled {
		cfg, cfgErr := s.getPointsConfig(group.ID)
		if cfgErr != nil {
			return true, false, cfgErr
		}
		cost = cfg.LotteryCost
	}
	if cost > 0 {
		ok, _, spendErr := s.spendPoints(group.ID, u.ID, pointsEventLottery, cost, time.Now())
		if spendErr != nil {
			return true, false, spendErr
		}
		if !ok {
			return true, false, ErrInsufficientPoints
		}
	}
	created, err := s.repo.JoinLottery(lottery.ID, u.ID)
	if err != nil {
		if cost > 0 {
			refund, _, _ := s.repo.AdjustPoints(group.ID, u.ID, cost)
			if refund > 0 {
				_ = s.recordPointEvent(group.ID, u.ID, pointsEventLottery, refund, time.Now())
			}
		}
		return true, false, err
	}
	if !created && cost > 0 {
		refund, _, _ := s.repo.AdjustPoints(group.ID, u.ID, cost)
		if refund > 0 {
			_ = s.recordPointEvent(group.ID, u.ID, pointsEventLottery, refund, time.Now())
		}
	}
	return true, created, nil
}
