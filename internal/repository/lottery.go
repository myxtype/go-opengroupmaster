package repository

import "supervisor/internal/model"

func (r *Repository) CreateLottery(groupID uint, title string, winners int) (*model.Lottery, error) {
	l := &model.Lottery{GroupID: groupID, Title: title, WinnersCount: winners, Status: "active"}
	return l, r.db.Create(l).Error
}

func (r *Repository) GetActiveLottery(groupID uint) (*model.Lottery, error) {
	var l model.Lottery
	if err := r.db.Where("group_id = ? and status = 'active'", groupID).Last(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repository) JoinLottery(lotteryID, userID uint) error {
	lp := &model.LotteryParticipant{LotteryID: lotteryID, UserID: userID}
	return r.db.Where("lottery_id = ? and user_id = ?", lotteryID, userID).FirstOrCreate(lp).Error
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
