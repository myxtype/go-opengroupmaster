package repository

import (
	"errors"

	"supervisor/internal/model"

	"github.com/go-telegram/bot/models"
	"gorm.io/gorm"
)

func (r *Repository) UpsertUserFromTG(u *models.User) (*model.User, error) {
	if u == nil {
		return nil, errors.New("nil user")
	}
	user := &model.User{TGUserID: u.ID}
	if err := r.db.Where(&model.User{TGUserID: u.ID}).FirstOrCreate(user).Error; err != nil {
		return nil, err
	}
	user.Username = u.Username
	user.FirstName = u.FirstName
	user.LastName = u.LastName
	if err := r.db.Save(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) FindUserByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) FindUsersByIDs(ids []uint) ([]model.User, error) {
	if len(ids) == 0 {
		return []model.User{}, nil
	}
	out := make([]model.User, 0, len(ids))
	if err := r.db.Where("id IN ?", ids).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) FindUserByUsername(username string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("lower(username) = lower(?)", username).Order("id desc").First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) FindUserByTGUserID(tgUserID int64) (*model.User, error) {
	var u model.User
	if err := r.db.Where("tg_user_id = ?", tgUserID).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) EnsureUserByTGUserID(tgUserID int64) (*model.User, error) {
	user := &model.User{TGUserID: tgUserID}
	if err := r.db.Where("tg_user_id = ?", tgUserID).FirstOrCreate(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) SetUserLanguage(tgUserID int64, lang string) error {
	user := &model.User{TGUserID: tgUserID}
	if err := r.db.Where("tg_user_id = ?", tgUserID).FirstOrCreate(user).Error; err != nil {
		return err
	}
	user.Language = lang
	return r.db.Save(user).Error
}

func (r *Repository) GetUserLanguage(tgUserID int64) (string, error) {
	var user model.User
	if err := r.db.Where("tg_user_id = ?", tgUserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "zh", nil
		}
		return "", err
	}
	if user.Language == "" {
		return "zh", nil
	}
	return user.Language, nil
}
