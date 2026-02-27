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

func (r *Repository) IsLotteryParticipant(lotteryID, userID uint) (bool, error) {
	var total int64
	err := r.db.Model(&model.LotteryParticipant{}).
		Where("lottery_id = ? and user_id = ?", lotteryID, userID).
		Count(&total).Error
	return total > 0, err
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

func (r *Repository) CountLotteryParticipantsByLotteryIDs(lotteryIDs []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(lotteryIDs))
	if len(lotteryIDs) == 0 {
		return out, nil
	}
	for _, id := range lotteryIDs {
		out[id] = 0
	}
	type row struct {
		LotteryID uint
		Total     int64
	}
	var rows []row
	if err := r.db.Model(&model.LotteryParticipant{}).
		Select("lottery_id, count(*) as total").
		Where("lottery_id IN ?", lotteryIDs).
		Group("lottery_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		out[item.LotteryID] = item.Total
	}
	return out, nil
}

func (r *Repository) CountLotteries(groupID uint) (int64, error) {
	var total int64
	err := r.db.Model(&model.Lottery{}).Where("group_id = ?", groupID).Count(&total).Error
	return total, err
}

func (r *Repository) CountLotteriesByStatus(groupID uint, status string) (int64, error) {
	var total int64
	err := r.db.Model(&model.Lottery{}).Where("group_id = ? and status = ?", groupID, status).Count(&total).Error
	return total, err
}

func (r *Repository) ListLotteriesPage(groupID uint, page, pageSize int) ([]model.Lottery, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 5
	}
	var total int64
	if err := r.db.Model(&model.Lottery{}).Where("group_id = ?", groupID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	out := make([]model.Lottery, 0, pageSize)
	err := r.db.Where("group_id = ?", groupID).
		Order("id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&out).Error
	return out, total, err
}

func (r *Repository) CancelLottery(groupID, lotteryID uint) (bool, error) {
	tx := r.db.Model(&model.Lottery{}).
		Where("group_id = ? and id = ? and status = 'active'", groupID, lotteryID).
		Update("status", "canceled")
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (r *Repository) GetLatestLottery(groupID uint) (*model.Lottery, error) {
	var l model.Lottery
	if err := r.db.Where("group_id = ?", groupID).Order("id desc").First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}
