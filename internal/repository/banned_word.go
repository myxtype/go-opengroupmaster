package repository

import (
	"strings"

	"supervisor/internal/model"
)

func (r *Repository) GetBannedWords(groupID uint) ([]model.BannedWord, error) {
	var out []model.BannedWord
	err := r.db.Where("group_id = ?", groupID).Find(&out).Error
	return out, err
}

func (r *Repository) ContainsBannedWord(groupID uint, msg string) (bool, error) {
	words, err := r.GetBannedWords(groupID)
	if err != nil {
		return false, err
	}
	for _, w := range words {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(w.Word)) {
			return true, nil
		}
	}
	return false, nil
}

func (r *Repository) CountBannedWords(groupID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.BannedWord{}).Where("group_id = ?", groupID).Count(&count).Error
	return count, err
}

func (r *Repository) CreateBannedWord(groupID uint, word string) error {
	item := &model.BannedWord{GroupID: groupID, Word: word}
	return r.db.Create(item).Error
}

func (r *Repository) ListBannedWordsPage(groupID uint, page, pageSize int) ([]model.BannedWord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 5
	}
	var total int64
	if err := r.db.Model(&model.BannedWord{}).Where("group_id = ?", groupID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	out := make([]model.BannedWord, 0, pageSize)
	err := r.db.Where("group_id = ?", groupID).Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&out).Error
	return out, total, err
}

func (r *Repository) DeleteBannedWord(groupID, id uint) error {
	return r.db.Where("group_id = ? and id = ?", groupID, id).Delete(&model.BannedWord{}).Error
}

func (r *Repository) DeleteBannedWordsByWord(groupID uint, word string) (int64, error) {
	tx := r.db.Where("group_id = ? and lower(word) = lower(?)", groupID, word).Delete(&model.BannedWord{})
	return tx.RowsAffected, tx.Error
}

func (r *Repository) UpdateBannedWord(groupID, id uint, word string) error {
	return r.db.Model(&model.BannedWord{}).Where("group_id = ? and id = ?", groupID, id).Update("word", word).Error
}
