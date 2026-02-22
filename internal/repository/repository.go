package repository

import (
	"errors"
	"fmt"
	"strings"

	"supervisor/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func (r *Repository) UpsertUserFromTG(u *tgbotapi.User) (*model.User, error) {
	if u == nil {
		return nil, errors.New("nil user")
	}
	user := &model.User{TGUserID: u.ID}
	if err := r.db.Where(&model.User{TGUserID: u.ID}).FirstOrCreate(user).Error; err != nil {
		return nil, err
	}
	user.Username = u.UserName
	user.FirstName = u.FirstName
	user.LastName = u.LastName
	if err := r.db.Save(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) UpsertGroup(chat *tgbotapi.Chat) (*model.Group, error) {
	if chat == nil {
		return nil, errors.New("nil chat")
	}
	g := &model.Group{TGGroupID: chat.ID}
	if err := r.db.Where(&model.Group{TGGroupID: chat.ID}).FirstOrCreate(g).Error; err != nil {
		return nil, err
	}
	g.Title = chat.Title
	g.BotAdded = true
	if err := r.db.Save(g).Error; err != nil {
		return nil, err
	}
	return g, nil
}

func (r *Repository) ReplaceGroupAdmins(groupID uint, admins []model.GroupAdmin) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&model.GroupAdmin{}).Error; err != nil {
			return err
		}
		if len(admins) == 0 {
			return nil
		}
		return tx.Create(&admins).Error
	})
}

func (r *Repository) ListGroupsByAdminTGUserID(tgUserID int64) ([]model.Group, error) {
	var groups []model.Group
	err := r.db.
		Table("groups").
		Select("groups.*").
		Joins("join group_admins on group_admins.group_id = groups.id").
		Joins("join users on users.id = group_admins.user_id").
		Where("users.tg_user_id = ?", tgUserID).
		Scan(&groups).Error
	return groups, err
}

func (r *Repository) FindGroupByTGID(tgGroupID int64) (*model.Group, error) {
	var g model.Group
	if err := r.db.Where("tg_group_id = ?", tgGroupID).First(&g).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repository) FindGroupByID(groupID uint) (*model.Group, error) {
	var g model.Group
	if err := r.db.First(&g, groupID).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repository) CheckAdmin(groupID uint, tgUserID int64) (bool, error) {
	var count int64
	err := r.db.
		Table("group_admins").
		Joins("join users on users.id = group_admins.user_id").
		Where("group_admins.group_id = ? and users.tg_user_id = ?", groupID, tgUserID).
		Count(&count).Error
	return count > 0, err
}

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
		default:
			if message == r.Keyword {
				return &r, nil
			}
		}
	}
	return nil, nil
}

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

func (r *Repository) AddPoints(groupID, userID uint, delta int) error {
	up := &model.UserPoint{GroupID: groupID, UserID: userID}
	if err := r.db.Where("group_id = ? and user_id = ?", groupID, userID).FirstOrCreate(up).Error; err != nil {
		return err
	}
	up.Points += delta
	return r.db.Save(up).Error
}

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

func (r *Repository) FindUserByID(id uint) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) ListEnabledScheduledMessages() ([]model.ScheduledMessage, error) {
	var out []model.ScheduledMessage
	err := r.db.Where("enabled = ?", true).Find(&out).Error
	return out, err
}

func (r *Repository) CreateDefaultDataIfEmpty(groupID uint) error {
	// Seed minimal defaults so MVP can be seen immediately in a new group.
	var count int64
	if err := r.db.Model(&model.AutoReply{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := r.db.Create(&model.AutoReply{GroupID: groupID, Keyword: "你好", Reply: "你好，我是 GroupMaster Bot", MatchType: "exact"}).Error; err != nil {
			return err
		}
	}
	if err := r.db.Model(&model.BannedWord{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := r.db.Create(&model.BannedWord{GroupID: groupID, Word: "spam"}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) CountAutoReplies(groupID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.AutoReply{}).Where("group_id = ?", groupID).Count(&count).Error
	return count, err
}

func (r *Repository) CountBannedWords(groupID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.BannedWord{}).Where("group_id = ?", groupID).Count(&count).Error
	return count, err
}

func (r *Repository) CreateAutoReply(groupID uint, keyword, reply, matchType string) error {
	item := &model.AutoReply{
		GroupID:   groupID,
		Keyword:   keyword,
		Reply:     reply,
		MatchType: matchType,
	}
	return r.db.Create(item).Error
}

func (r *Repository) CreateBannedWord(groupID uint, word string) error {
	item := &model.BannedWord{GroupID: groupID, Word: word}
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
	err := r.db.Where("group_id = ?", groupID).
		Order("id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&out).Error
	return out, total, err
}

func (r *Repository) DeleteAutoReply(groupID, id uint) error {
	return r.db.Where("group_id = ? and id = ?", groupID, id).Delete(&model.AutoReply{}).Error
}

func (r *Repository) UpdateAutoReply(groupID, id uint, keyword, reply, matchType string) error {
	updates := map[string]any{
		"keyword":    keyword,
		"reply":      reply,
		"match_type": matchType,
	}
	return r.db.Model(&model.AutoReply{}).Where("group_id = ? and id = ?", groupID, id).Updates(updates).Error
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
	err := r.db.Where("group_id = ?", groupID).
		Order("id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&out).Error
	return out, total, err
}

func (r *Repository) DeleteBannedWord(groupID, id uint) error {
	return r.db.Where("group_id = ? and id = ?", groupID, id).Delete(&model.BannedWord{}).Error
}

func (r *Repository) UpdateBannedWord(groupID, id uint, word string) error {
	return r.db.Model(&model.BannedWord{}).Where("group_id = ? and id = ?", groupID, id).Update("word", word).Error
}

func (r *Repository) GetGroupSetting(groupID uint, featureKey string) (*model.GroupSetting, error) {
	var setting model.GroupSetting
	if err := r.db.Where("group_id = ? and feature_key = ?", groupID, featureKey).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *Repository) UpsertFeatureEnabled(groupID uint, featureKey string, enabled bool) error {
	setting := &model.GroupSetting{GroupID: groupID, FeatureKey: featureKey}
	if err := r.db.Where("group_id = ? and feature_key = ?", groupID, featureKey).FirstOrCreate(setting).Error; err != nil {
		return err
	}
	setting.Enabled = enabled
	return r.db.Save(setting).Error
}

func (r *Repository) UpsertFeatureConfig(groupID uint, featureKey string, config string) error {
	setting := &model.GroupSetting{GroupID: groupID, FeatureKey: featureKey}
	if err := r.db.Where("group_id = ? and feature_key = ?", groupID, featureKey).FirstOrCreate(setting).Error; err != nil {
		return err
	}
	setting.Config = config
	return r.db.Save(setting).Error
}

func (r *Repository) CreateScheduledMessage(groupID uint, content, cronExpr string) error {
	item := &model.ScheduledMessage{
		GroupID:  groupID,
		Content:  content,
		CronExpr: cronExpr,
		Enabled:  true,
	}
	return r.db.Create(item).Error
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
	err := r.db.Where("group_id = ?", groupID).
		Order("id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&out).Error
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

func (r *Repository) TopUsersByPoints(groupID uint, limit int) ([]model.UserPoint, error) {
	if limit <= 0 {
		limit = 10
	}
	out := make([]model.UserPoint, 0, limit)
	err := r.db.Where("group_id = ?", groupID).Order("points desc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) CreateLog(groupID uint, action string, operatorID, targetID uint) error {
	if action == "" {
		return fmt.Errorf("action is required")
	}
	l := &model.Log{GroupID: groupID, Action: action, OperatorID: operatorID, TargetID: targetID}
	return r.db.Create(l).Error
}

func (r *Repository) ListLogsPage(groupID uint, page, pageSize int, action string) ([]model.Log, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	q := r.db.Model(&model.Log{}).Where("group_id = ?", groupID)
	if action != "" && action != "all" {
		q = q.Where("action = ?", action)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	out := make([]model.Log, 0, pageSize)
	err := q.
		Order("id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&out).Error
	return out, total, err
}

func (r *Repository) ListLogsForExport(groupID uint, action string, limit int) ([]model.Log, error) {
	if limit <= 0 {
		limit = 1000
	}
	q := r.db.Where("group_id = ?", groupID)
	if action != "" && action != "all" {
		q = q.Where("action = ?", action)
	}
	out := make([]model.Log, 0, limit)
	err := q.Order("id desc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) ListAdminTGUserIDsByGroupID(groupID uint) ([]int64, error) {
	rows := make([]struct {
		TGUserID int64
	}, 0)
	err := r.db.
		Table("group_admins").
		Select("users.tg_user_id as tg_user_id").
		Joins("join users on users.id = group_admins.user_id").
		Where("group_admins.group_id = ?", groupID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.TGUserID)
	}
	return out, nil
}

func (r *Repository) AddGlobalBlacklist(tgUserID int64, reason string) error {
	item := &model.GlobalBlacklist{TGUserID: tgUserID, Reason: reason}
	return r.db.Where("tg_user_id = ?", tgUserID).FirstOrCreate(item).Error
}

func (r *Repository) RemoveGlobalBlacklist(tgUserID int64) error {
	return r.db.Where("tg_user_id = ?", tgUserID).Delete(&model.GlobalBlacklist{}).Error
}

func (r *Repository) IsGlobalBlacklisted(tgUserID int64) (bool, error) {
	var count int64
	err := r.db.Model(&model.GlobalBlacklist{}).Where("tg_user_id = ?", tgUserID).Count(&count).Error
	return count > 0, err
}

func (r *Repository) ListGlobalBlacklist() ([]model.GlobalBlacklist, error) {
	out := make([]model.GlobalBlacklist, 0)
	err := r.db.Order("id desc").Find(&out).Error
	return out, err
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
