package repository

import (
	"errors"
	"strings"
	"supervisor/internal/model"

	"gorm.io/gorm"
)

func (r *Repository) CreateLottery(groupID uint, title, joinKeyword string, winners int) (*model.Lottery, error) {
	if strings.TrimSpace(joinKeyword) == "" {
		joinKeyword = "参加"
	}
	l := &model.Lottery{GroupID: groupID, Title: title, JoinKeyword: strings.TrimSpace(joinKeyword), WinnersCount: winners, Status: "active"}
	return l, r.db.Create(l).Error
}

func (r *Repository) GetActiveLottery(groupID uint) (*model.Lottery, error) {
	var l model.Lottery
	if err := r.db.Where("group_id = ? and status = 'active'", groupID).Last(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repository) JoinLottery(lotteryID, userID uint) (bool, error) {
	var existed model.LotteryParticipant
	err := r.db.Where("lottery_id = ? and user_id = ?", lotteryID, userID).First(&existed).Error
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	lp := &model.LotteryParticipant{LotteryID: lotteryID, UserID: userID}
	if err := r.db.Create(lp).Error; err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) ListLotteryParticipantUserIDs(lotteryID uint) ([]uint, error) {
	var parts []model.LotteryParticipant
	if err := r.db.Where("lottery_id = ?", lotteryID).Find(&parts).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(parts))
	for _, p := range parts {
		ids = append(ids, p.UserID)
	}
	return ids, nil
}

func (r *Repository) CloseLottery(lotteryID uint) error {
	return r.db.Model(&model.Lottery{}).Where("id = ?", lotteryID).Update("status", "closed").Error
}

func (r *Repository) CountLotteryParticipants(lotteryID uint) (int64, error) {
	var total int64
	err := r.db.Model(&model.LotteryParticipant{}).Where("lottery_id = ?", lotteryID).Count(&total).Error
	return total, err
}

func (r *Repository) GetLatestLottery(groupID uint) (*model.Lottery, error) {
	var l model.Lottery
	if err := r.db.Where("group_id = ?", groupID).Order("id desc").First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}
