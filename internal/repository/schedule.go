package repository

import "supervisor/internal/model"

func (r *Repository) ListEnabledScheduledMessages() ([]model.ScheduledMessage, error) {
	var out []model.ScheduledMessage
	err := r.db.Where("enabled = ?", true).Find(&out).Error
	return out, err
}

func (r *Repository) CreateScheduledMessage(groupID uint, content, cronExpr, buttonRows, mediaType, mediaFileID string, pinMessage bool) (*model.ScheduledMessage, error) {
	item := &model.ScheduledMessage{
		GroupID:     groupID,
		Content:     content,
		CronExpr:    cronExpr,
		Enabled:     true,
		ButtonRows:  buttonRows,
		MediaType:   mediaType,
		MediaFileID: mediaFileID,
		PinMessage:  pinMessage,
	}
	if err := r.db.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repository) CountScheduledMessages(groupID uint) (int64, error) {
	var total int64
	err := r.db.Model(&model.ScheduledMessage{}).Where("group_id = ?", groupID).Count(&total).Error
	return total, err
}

func (r *Repository) ListScheduledMessagesPage(groupID uint, page, pageSize int) ([]model.ScheduledMessage, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 5
	}
	var total int64
	if err := r.db.Model(&model.ScheduledMessage{}).Where("group_id = ?", groupID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	out := make([]model.ScheduledMessage, 0, pageSize)
	err := r.db.Where("group_id = ?", groupID).Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&out).Error
	return out, total, err
}

func (r *Repository) DeleteScheduledMessage(groupID, id uint) error {
	return r.db.Where("group_id = ? and id = ?", groupID, id).Delete(&model.ScheduledMessage{}).Error
}

func (r *Repository) ToggleScheduledMessage(groupID, id uint) (bool, error) {
	var item model.ScheduledMessage
	if err := r.db.Where("group_id = ? and id = ?", groupID, id).First(&item).Error; err != nil {
		return false, err
	}
	item.Enabled = !item.Enabled
	return item.Enabled, r.db.Save(&item).Error
}

func (r *Repository) ToggleScheduledPinMessage(groupID, id uint) (bool, error) {
	var item model.ScheduledMessage
	if err := r.db.Where("group_id = ? and id = ?", groupID, id).First(&item).Error; err != nil {
		return false, err
	}
	item.PinMessage = !item.PinMessage
	return item.PinMessage, r.db.Save(&item).Error
}

func (r *Repository) GetScheduledMessage(groupID, id uint) (*model.ScheduledMessage, error) {
	var item model.ScheduledMessage
	if err := r.db.Where("group_id = ? and id = ?", groupID, id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) SaveScheduledMessage(item *model.ScheduledMessage) error {
	if item == nil {
		return nil
	}
	return r.db.Save(item).Error
}
