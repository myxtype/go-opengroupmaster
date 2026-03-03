package repository

import (
	"regexp"
	"strings"

	"supervisor/internal/model"
)

func (r *Repository) GetAutoReplies(groupID uint) ([]model.AutoReply, error) {
	var out []model.AutoReply
	err := r.db.Where("group_id = ?", groupID).Find(&out).Error
	return out, err
}

func (r *Repository) MatchAutoReply(groupID uint, message string) (*model.AutoReply, error) {
	rules, err := r.GetAutoReplies(groupID)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		r := rules[i]
		switch r.MatchType {
		case "contains":
			if strings.Contains(message, r.Keyword) {
				return &r, nil
			}
		case "regex":
			re, compileErr := regexp.Compile(r.Keyword)
			if compileErr != nil {
				continue
			}
			if re.MatchString(message) {
				return &r, nil
			}
		default:
			if message == r.Keyword {
				return &r, nil
			}
		}
	}
	return nil, nil
}

func (r *Repository) CountAutoReplies(groupID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.AutoReply{}).Where("group_id = ?", groupID).Count(&count).Error
	return count, err
}

func (r *Repository) CreateAutoReply(groupID uint, keyword, reply, matchType, buttonRows string) error {
	item := &model.AutoReply{
		GroupID:    groupID,
		Keyword:    keyword,
		Reply:      reply,
		MatchType:  matchType,
		ButtonRows: buttonRows,
	}
	return r.db.Create(item).Error
}

func (r *Repository) ListAutoRepliesPage(groupID uint, page, pageSize int) ([]model.AutoReply, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 5
	}
	var total int64
	if err := r.db.Model(&model.AutoReply{}).Where("group_id = ?", groupID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	out := make([]model.AutoReply, 0, pageSize)
	err := r.db.Where("group_id = ?", groupID).Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&out).Error
	return out, total, err
}

func (r *Repository) DeleteAutoReply(groupID, id uint) error {
	return r.db.Where("group_id = ? and id = ?", groupID, id).Delete(&model.AutoReply{}).Error
}

func (r *Repository) UpdateAutoReply(groupID, id uint, keyword, reply, matchType string) error {
	updates := map[string]any{"keyword": keyword, "reply": reply, "match_type": matchType}
	return r.db.Model(&model.AutoReply{}).Where("group_id = ? and id = ?", groupID, id).Updates(updates).Error
}

func (r *Repository) UpdateAutoReplyWithButtons(groupID, id uint, keyword, reply, matchType, buttonRows string) error {
	updates := map[string]any{
		"keyword":     keyword,
		"reply":       reply,
		"match_type":  matchType,
		"button_rows": buttonRows,
	}
	return r.db.Model(&model.AutoReply{}).Where("group_id = ? and id = ?", groupID, id).Updates(updates).Error
}
