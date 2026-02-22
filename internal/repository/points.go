package repository

import "supervisor/internal/model"

func (r *Repository) AddPoints(groupID, userID uint, delta int) error {
	up := &model.UserPoint{GroupID: groupID, UserID: userID}
	if err := r.db.Where("group_id = ? and user_id = ?", groupID, userID).FirstOrCreate(up).Error; err != nil {
		return err
	}
	up.Points += delta
	return r.db.Save(up).Error
}

func (r *Repository) TopUsersByPoints(groupID uint, limit int) ([]model.UserPoint, error) {
	if limit <= 0 {
		limit = 10
	}
	out := make([]model.UserPoint, 0, limit)
	err := r.db.Where("group_id = ?", groupID).Order("points desc").Limit(limit).Find(&out).Error
	return out, err
}
