package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"supervisor/internal/model"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	pointsEventCheckin  = "checkin"
	pointsEventMessage  = "message"
	pointsEventInvite   = "invite"
	pointsEventLottery  = "lottery"
	pointsEventAdminAdd = "admin_add"
	pointsEventAdminSub = "admin_sub"
)

var ErrInsufficientPoints = errors.New("insufficient points")

func pointsDayKey(now time.Time) string {
	return now.UTC().Format("2006-01-02")
}

func pointsDisplayName(u *model.User) string {
	if u == nil {
		return "unknown"
	}
	if strings.TrimSpace(u.Username) != "" {
		return "@" + strings.TrimSpace(u.Username)
	}
	name := strings.TrimSpace(strings.TrimSpace(u.FirstName) + " " + strings.TrimSpace(u.LastName))
	if name != "" {
		return name
	}
	return fmt.Sprintf("uid:%d", u.TGUserID)
}

func (s *Service) pointsEnabled(groupID uint) (bool, error) {
	return s.IsFeatureEnabled(groupID, featurePoints, false)
}

func (s *Service) recordPointEvent(groupID, userID uint, eventType string, delta int, now time.Time) error {
	if eventType == "" || delta == 0 {
		return nil
	}
	return s.repo.CreatePointEvent(&model.PointEvent{
		GroupID: groupID,
		UserID:  userID,
		DayKey:  pointsDayKey(now),
		Type:    eventType,
		Delta:   delta,
	})
}

func (s *Service) awardPointsWithDailyLimit(groupID, userID uint, eventType string, reward int, dailyLimit int, now time.Time) (int, int, error) {
	if reward <= 0 {
		current, err := s.repo.UserPoints(groupID, userID)
		return 0, current, err
	}
	gain := reward
	if dailyLimit > 0 {
		earned, err := s.repo.SumPointEventDeltaByDayAndType(groupID, userID, pointsDayKey(now), eventType)
		if err != nil {
			return 0, 0, err
		}
		if earned >= dailyLimit {
			current, cErr := s.repo.UserPoints(groupID, userID)
			return 0, current, cErr
		}
		remain := dailyLimit - earned
		if gain > remain {
			gain = remain
		}
	}
	if gain <= 0 {
		current, err := s.repo.UserPoints(groupID, userID)
		return 0, current, err
	}
	applied, current, err := s.repo.AdjustPoints(groupID, userID, gain)
	if err != nil {
		return 0, 0, err
	}
	if applied > 0 {
		_ = s.recordPointEvent(groupID, userID, eventType, applied, now)
	}
	return applied, current, nil
}

func (s *Service) spendPoints(groupID, userID uint, eventType string, cost int, now time.Time) (bool, int, error) {
	if cost <= 0 {
		current, err := s.repo.UserPoints(groupID, userID)
		return true, current, err
	}
	applied, current, err := s.repo.AdjustPoints(groupID, userID, -cost)
	if err != nil {
		return false, 0, err
	}
	if applied == 0 {
		return false, current, nil
	}
	if -applied < cost {
		if applied < 0 {
			refund, refundCurrent, refundErr := s.repo.AdjustPoints(groupID, userID, -applied)
			if refundErr == nil && refund > 0 {
				current = refundCurrent
			}
		}
		return false, current, nil
	}
	if applied < 0 {
		_ = s.recordPointEvent(groupID, userID, eventType, applied, now)
	}
	return true, current, nil
}

func (s *Service) PointsPanelViewByTGGroupID(tgGroupID int64) (*PointsPanelView, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return nil, err
	}
	enabled, err := s.pointsEnabled(group.ID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.getPointsConfig(group.ID)
	if err != nil {
		return nil, err
	}
	return &PointsPanelView{Enabled: enabled, Config: cfg}, nil
}

func (s *Service) SetPointsEnabledByTGGroupID(tgGroupID int64, enabled bool) (bool, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return false, err
	}
	if err := s.repo.UpsertFeatureEnabled(group.ID, featurePoints, enabled); err != nil {
		return false, err
	}
	_ = s.repo.CreateLog(group.ID, fmt.Sprintf("set_points_enabled_%t", enabled), 0, 0)
	return enabled, nil
}

func (s *Service) setPointsConfigByTGGroupID(tgGroupID int64, mutate func(pointsConfig) (pointsConfig, error), action string) (pointsConfig, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return pointsConfig{}, err
	}
	cfg, err := s.getPointsConfig(group.ID)
	if err != nil {
		return pointsConfig{}, err
	}
	next, err := mutate(cfg)
	if err != nil {
		return pointsConfig{}, err
	}
	if err := s.savePointsConfig(group.ID, next); err != nil {
		return pointsConfig{}, err
	}
	if strings.TrimSpace(action) != "" {
		_ = s.repo.CreateLog(group.ID, action, 0, 0)
	}
	return next, nil
}

func (s *Service) SetPointsCheckinKeywordByTGGroupID(tgGroupID int64, keyword string) (string, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return "", errors.New("empty checkin keyword")
	}
	cfg, err := s.setPointsConfigByTGGroupID(tgGroupID, func(in pointsConfig) (pointsConfig, error) {
		in.CheckinKeyword = keyword
		return in, nil
	}, "set_points_checkin_keyword")
	if err != nil {
		return "", err
	}
	return cfg.CheckinKeyword, nil
}

func (s *Service) SetPointsCheckinRewardByTGGroupID(tgGroupID int64, reward int) (int, error) {
	if reward <= 0 {
		return 0, errors.New("invalid checkin reward")
	}
	cfg, err := s.setPointsConfigByTGGroupID(tgGroupID, func(in pointsConfig) (pointsConfig, error) {
		in.CheckinReward = reward
		return in, nil
	}, fmt.Sprintf("set_points_checkin_reward_%d", reward))
	if err != nil {
		return 0, err
	}
	return cfg.CheckinReward, nil
}

func (s *Service) SetPointsMessageRewardByTGGroupID(tgGroupID int64, reward int) (int, error) {
	if reward <= 0 {
		return 0, errors.New("invalid message reward")
	}
	cfg, err := s.setPointsConfigByTGGroupID(tgGroupID, func(in pointsConfig) (pointsConfig, error) {
		in.MessageReward = reward
		return in, nil
	}, fmt.Sprintf("set_points_message_reward_%d", reward))
	if err != nil {
		return 0, err
	}
	return cfg.MessageReward, nil
}

func (s *Service) SetPointsMessageDailyLimitByTGGroupID(tgGroupID int64, limit int) (int, error) {
	if limit < 0 {
		return 0, errors.New("invalid message daily limit")
	}
	cfg, err := s.setPointsConfigByTGGroupID(tgGroupID, func(in pointsConfig) (pointsConfig, error) {
		in.MessageDaily = limit
		return in, nil
	}, fmt.Sprintf("set_points_message_daily_%d", limit))
	if err != nil {
		return 0, err
	}
	return cfg.MessageDaily, nil
}

func (s *Service) SetPointsMessageMinLenByTGGroupID(tgGroupID int64, minLen int) (int, error) {
	if minLen < 0 {
		return 0, errors.New("invalid message min length")
	}
	cfg, err := s.setPointsConfigByTGGroupID(tgGroupID, func(in pointsConfig) (pointsConfig, error) {
		in.MessageMinLen = minLen
		return in, nil
	}, fmt.Sprintf("set_points_message_min_len_%d", minLen))
	if err != nil {
		return 0, err
	}
	return cfg.MessageMinLen, nil
}

func (s *Service) SetPointsInviteRewardByTGGroupID(tgGroupID int64, reward int) (int, error) {
	if reward < 0 {
		return 0, errors.New("invalid invite reward")
	}
	cfg, err := s.setPointsConfigByTGGroupID(tgGroupID, func(in pointsConfig) (pointsConfig, error) {
		in.InviteReward = reward
		return in, nil
	}, fmt.Sprintf("set_points_invite_reward_%d", reward))
	if err != nil {
		return 0, err
	}
	return cfg.InviteReward, nil
}

func (s *Service) SetPointsInviteDailyLimitByTGGroupID(tgGroupID int64, limit int) (int, error) {
	if limit < 0 {
		return 0, errors.New("invalid invite daily limit")
	}
	cfg, err := s.setPointsConfigByTGGroupID(tgGroupID, func(in pointsConfig) (pointsConfig, error) {
		in.InviteDaily = limit
		return in, nil
	}, fmt.Sprintf("set_points_invite_daily_%d", limit))
	if err != nil {
		return 0, err
	}
	return cfg.InviteDaily, nil
}

func (s *Service) SetPointsBalanceAliasByTGGroupID(tgGroupID int64, alias string) (string, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", errors.New("empty balance alias")
	}
	cfg, err := s.setPointsConfigByTGGroupID(tgGroupID, func(in pointsConfig) (pointsConfig, error) {
		in.BalanceAlias = alias
		return in, nil
	}, "set_points_balance_alias")
	if err != nil {
		return "", err
	}
	return cfg.BalanceAlias, nil
}

func (s *Service) SetPointsRankAliasByTGGroupID(tgGroupID int64, alias string) (string, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", errors.New("empty rank alias")
	}
	cfg, err := s.setPointsConfigByTGGroupID(tgGroupID, func(in pointsConfig) (pointsConfig, error) {
		in.RankAlias = alias
		return in, nil
	}, "set_points_rank_alias")
	if err != nil {
		return "", err
	}
	return cfg.RankAlias, nil
}

func (s *Service) AdjustPointsByTargetTGUserID(tgGroupID, targetTGUserID int64, delta int) (int, int, error) {
	if delta == 0 {
		return 0, 0, nil
	}
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, 0, err
	}
	user, err := s.repo.EnsureUserByTGUserID(targetTGUserID)
	if err != nil {
		return 0, 0, err
	}
	applied, current, err := s.repo.AdjustPoints(group.ID, user.ID, delta)
	if err != nil {
		return 0, 0, err
	}
	if applied != 0 {
		eventType := pointsEventAdminAdd
		if applied < 0 {
			eventType = pointsEventAdminSub
		}
		_ = s.recordPointEvent(group.ID, user.ID, eventType, applied, time.Now())
	}
	if delta > 0 {
		_ = s.repo.CreateLog(group.ID, fmt.Sprintf("points_admin_add_%d", delta), 0, user.ID)
	} else {
		_ = s.repo.CreateLog(group.ID, fmt.Sprintf("points_admin_sub_%d", -delta), 0, user.ID)
	}
	return applied, current, nil
}

func (s *Service) UserPointsByTGGroupAndUserID(tgGroupID, tgUserID int64) (int, error) {
	group, err := s.repo.FindGroupByTGID(tgGroupID)
	if err != nil {
		return 0, err
	}
	user, err := s.repo.EnsureUserByTGUserID(tgUserID)
	if err != nil {
		return 0, err
	}
	return s.repo.UserPoints(group.ID, user.ID)
}

func (s *Service) rewardMessagePoints(group *model.Group, msg *models.Message) error {
	if group == nil || msg == nil || msg.From == nil {
		return nil
	}
	enabled, err := s.pointsEnabled(group.ID)
	if err != nil || !enabled {
		return err
	}
	cfg, err := s.getPointsConfig(group.ID)
	if err != nil {
		return err
	}
	if cfg.MessageMinLen > 0 {
		content := strings.TrimSpace(msg.Text)
		if content == "" {
			content = strings.TrimSpace(msg.Caption)
		}
		if utf8.RuneCountInString(content) < cfg.MessageMinLen {
			return nil
		}
	}
	u, err := s.repo.UpsertUserFromTG(msg.From)
	if err != nil {
		return nil
	}
	_, _, err = s.awardPointsWithDailyLimit(group.ID, u.ID, pointsEventMessage, cfg.MessageReward, cfg.MessageDaily, time.Now())
	return err
}

func (s *Service) rewardInvitePoints(groupID uint, inviterTGUserID int64) error {
	enabled, err := s.pointsEnabled(groupID)
	if err != nil || !enabled {
		return err
	}
	cfg, err := s.getPointsConfig(groupID)
	if err != nil {
		return err
	}
	if cfg.InviteReward <= 0 {
		return nil
	}
	u, err := s.repo.EnsureUserByTGUserID(inviterTGUserID)
	if err != nil {
		return err
	}
	_, _, err = s.awardPointsWithDailyLimit(groupID, u.ID, pointsEventInvite, cfg.InviteReward, cfg.InviteDaily, time.Now())
	return err
}

func (s *Service) handlePointsTextCommand(bot *tgbot.Bot, group *model.Group, msg *models.Message) (bool, error) {
	if bot == nil || group == nil || msg == nil || msg.From == nil || strings.TrimSpace(msg.Text) == "" {
		return false, nil
	}
	enabled, err := s.pointsEnabled(group.ID)
	if err != nil || !enabled {
		return false, err
	}
	cfg, err := s.getPointsConfig(group.ID)
	if err != nil {
		return false, err
	}
	text := strings.TrimSpace(msg.Text)
	if strings.EqualFold(text, cfg.CheckinKeyword) {
		u, err := s.repo.UpsertUserFromTG(msg.From)
		if err != nil {
			return true, err
		}
		day := pointsDayKey(time.Now())
		signed, err := s.repo.ExistsPointEventByDayAndType(group.ID, u.ID, day, pointsEventCheckin)
		if err != nil {
			return true, err
		}
		if signed {
			current, cErr := s.repo.UserPoints(group.ID, u.ID)
			if cErr != nil {
				return true, cErr
			}
			_, _ = bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
				ChatID: msg.Chat.ID,
				Text:   fmt.Sprintf("今天已经签到过了，当前积分：%d", current),
			})
			return true, nil
		}
		got, current, err := s.awardPointsWithDailyLimit(group.ID, u.ID, pointsEventCheckin, cfg.CheckinReward, cfg.CheckinReward, time.Now())
		if err != nil {
			return true, err
		}
		_ = s.repo.CreateLog(group.ID, "points_checkin", u.ID, 0)
		_, _ = bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   fmt.Sprintf("签到成功，获得 %d 积分，当前积分：%d", got, current),
		})
		return true, nil
	}

	if strings.EqualFold(text, cfg.BalanceAlias) {
		u, err := s.repo.UpsertUserFromTG(msg.From)
		if err != nil {
			return true, err
		}
		current, err := s.repo.UserPoints(group.ID, u.ID)
		if err != nil {
			return true, err
		}
		_, _ = bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   fmt.Sprintf("你的当前积分：%d", current),
		})
		return true, nil
	}

	if strings.EqualFold(text, cfg.RankAlias) {
		top, err := s.repo.TopUsersByPoints(group.ID, 10)
		if err != nil {
			return true, err
		}
		lines := []string{"积分排行："}
		if len(top) == 0 {
			lines = append(lines, "暂无积分数据")
		} else {
			for i, row := range top {
				u, uErr := s.repo.FindUserByID(row.UserID)
				if uErr != nil {
					lines = append(lines, fmt.Sprintf("%d. uid:%d - %d", i+1, row.UserID, row.Points))
					continue
				}
				lines = append(lines, fmt.Sprintf("%d. %s - %d", i+1, pointsDisplayName(u), row.Points))
			}
		}
		_, _ = bot.SendMessage(context.Background(), &tgbot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   strings.Join(lines, "\n"),
		})
		return true, nil
	}
	return false, nil
}
