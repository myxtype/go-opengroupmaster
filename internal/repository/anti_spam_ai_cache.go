package repository

import (
	"time"

	"supervisor/internal/model"

	"gorm.io/gorm/clause"
)

func (r *Repository) FindAISpamCache(chatID int64, contentHash string, notBefore time.Time) (*model.AISpamCache, error) {
	var item model.AISpamCache
	if err := r.db.
		Where("chat_id = ? AND content_hash = ? AND created_at >= ?", chatID, contentHash, notBefore).
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpsertAISpamCache(chatID int64, contentHash, resultJSON string, createdAt time.Time) error {
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	item := model.AISpamCache{
		ChatID:      chatID,
		ContentHash: contentHash,
		ResultJSON:  resultJSON,
		CreatedAt:   createdAt,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chat_id"},
			{Name: "content_hash"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"result_json": resultJSON,
			"created_at":  createdAt,
		}),
	}).Create(&item).Error
}

func (r *Repository) DeleteAISpamCacheBefore(cutoff time.Time) error {
	return r.db.Where("created_at < ?", cutoff).Delete(&model.AISpamCache{}).Error
}
