package repository

import (
	"strings"
	"time"

	"supervisor/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type wordCloudWordStatRow struct {
	Word  string
	Total int64
}

type wordCloudUserContributionRow struct {
	UserID       uint
	TokenTotal   int64
	MessageTotal int64
}

type wordCloudDailySummaryRow struct {
	UsersTotal   int64
	MessageTotal int64
}

func (r *Repository) AddWordCloudMessageAndTokens(groupID, userID uint, dayKey string, tokenCounts map[string]int, tokenTotal int) error {
	if groupID == 0 || userID == 0 || strings.TrimSpace(dayKey) == "" {
		return nil
	}
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "group_id"}, {Name: "day_key"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"message_count": gorm.Expr("word_cloud_daily_user_stats.message_count + ?", 1),
				"token_count":   gorm.Expr("word_cloud_daily_user_stats.token_count + ?", tokenTotal),
				"updated_at":    now,
			}),
		}).Create(&model.WordCloudDailyUserStat{
			GroupID:      groupID,
			DayKey:       dayKey,
			UserID:       userID,
			MessageCount: 1,
			TokenCount:   tokenTotal,
		}).Error; err != nil {
			return err
		}

		for word, count := range tokenCounts {
			if count <= 0 {
				continue
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "group_id"}, {Name: "day_key"}, {Name: "user_id"}, {Name: "word"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"count":      gorm.Expr("word_cloud_tokens.count + ?", count),
					"updated_at": now,
				}),
			}).Create(&model.WordCloudToken{
				GroupID: groupID,
				DayKey:  dayKey,
				UserID:  userID,
				Word:    word,
				Count:   count,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) ListWordCloudWordStats(groupID uint, dayKey string, limit int) ([]wordCloudWordStatRow, error) {
	if limit <= 0 {
		limit = 200
	}
	rows := make([]wordCloudWordStatRow, 0, limit)
	err := r.db.Model(&model.WordCloudToken{}).
		Select("word, coalesce(sum(count), 0) as total").
		Where("group_id = ? and day_key = ?", groupID, dayKey).
		Group("word").
		Order("total desc, word asc").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) WordCloudDailySummary(groupID uint, dayKey string) (wordCloudDailySummaryRow, error) {
	out := wordCloudDailySummaryRow{}
	err := r.db.Model(&model.WordCloudDailyUserStat{}).
		Select("count(*) as users_total, coalesce(sum(message_count), 0) as message_total").
		Where("group_id = ? and day_key = ?", groupID, dayKey).
		Scan(&out).Error
	return out, err
}

func (r *Repository) TopWordCloudContributors(groupID uint, dayKey string, limit int) ([]wordCloudUserContributionRow, error) {
	if limit <= 0 {
		limit = 10
	}
	out := make([]wordCloudUserContributionRow, 0, limit)
	err := r.db.Model(&model.WordCloudDailyUserStat{}).
		Select("user_id, coalesce(sum(token_count), 0) as token_total, coalesce(sum(message_count), 0) as message_total").
		Where("group_id = ? and day_key = ?", groupID, dayKey).
		Group("user_id").
		Order("token_total desc, message_total desc, user_id asc").
		Limit(limit).
		Scan(&out).Error
	return out, err
}

func (r *Repository) AddWordCloudBlacklistWord(groupID uint, word string) error {
	word = strings.TrimSpace(strings.ToLower(word))
	if word == "" {
		return nil
	}
	item := &model.WordCloudBlacklistWord{GroupID: groupID, Word: word}
	return r.db.Where("group_id = ? and word = ?", groupID, word).FirstOrCreate(item).Error
}

func (r *Repository) RemoveWordCloudBlacklistWord(groupID uint, word string) error {
	word = strings.TrimSpace(strings.ToLower(word))
	if word == "" {
		return nil
	}
	return r.db.Where("group_id = ? and word = ?", groupID, word).Delete(&model.WordCloudBlacklistWord{}).Error
}

func (r *Repository) ListWordCloudBlacklistWordsPage(groupID uint, page, pageSize int) ([]model.WordCloudBlacklistWord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var total int64
	if err := r.db.Model(&model.WordCloudBlacklistWord{}).Where("group_id = ?", groupID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	out := make([]model.WordCloudBlacklistWord, 0, pageSize)
	err := r.db.Where("group_id = ?", groupID).
		Order("id asc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&out).Error
	return out, total, err
}

func (r *Repository) ListWordCloudBlacklistWords(groupID uint) ([]model.WordCloudBlacklistWord, error) {
	out := make([]model.WordCloudBlacklistWord, 0, 64)
	err := r.db.Where("group_id = ?", groupID).Order("id asc").Find(&out).Error
	return out, err
}

func (r *Repository) CountWordCloudBlacklistWords(groupID uint) (int64, error) {
	var total int64
	err := r.db.Model(&model.WordCloudBlacklistWord{}).Where("group_id = ?", groupID).Count(&total).Error
	return total, err
}

func (r *Repository) ListWordCloudEnabledGroups() ([]model.Group, error) {
	out := make([]model.Group, 0, 32)
	err := r.db.Table("groups").
		Select("groups.*").
		Joins("join group_settings on group_settings.group_id = groups.id").
		Where("group_settings.feature_key = ? and group_settings.enabled = ?", "word_cloud", true).
		Order("groups.id asc").
		Find(&out).Error
	return out, err
}

// DeleteWordCloudDataOlderThan 删除指定日期前的词云统计数据
// 涉及表：
//   - WordCloudToken: 按群/日期/用户/词语的分词聚合记录
//   - WordCloudDailyUserStat: 按群/日期/用户的每日统计（发言数、贡献词数）
//
// 返回删除的总记录数
func (r *Repository) DeleteWordCloudDataOlderThan(cutoffDate time.Time) (int64, error) {
	dayKey := cutoffDate.In(time.Local).Format("2006-01-02")
	var totalDeleted int64

	// 先删除子表 WordCloudToken
	result := r.db.Where("day_key < ?", dayKey).Delete(&model.WordCloudToken{})
	if result.Error != nil {
		return 0, result.Error
	}
	totalDeleted += result.RowsAffected

	// 再删除父表 WordCloudDailyUserStat
	result = r.db.Where("day_key < ?", dayKey).Delete(&model.WordCloudDailyUserStat{})
	if result.Error != nil {
		return totalDeleted, result.Error
	}
	totalDeleted += result.RowsAffected

	return totalDeleted, nil
}
