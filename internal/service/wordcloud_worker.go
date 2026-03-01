package service

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *Service) processWordCloudPush(bot *tgbotapi.BotAPI) {
	groups, err := s.repo.ListWordCloudEnabledGroups()
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("word cloud list groups failed: %v", err)
		}
		return
	}
	now := time.Now()
	for _, group := range groups {
		ready, readyErr := s.wordCloudReadyToPush(group.ID, now)
		if readyErr != nil || !ready {
			continue
		}
		if err := s.SendWordCloudReportByTGGroupID(bot, group.TGGroupID, false); err != nil && s.logger != nil {
			s.logger.Printf("word cloud auto push failed group=%d err=%v", group.TGGroupID, err)
		}
	}
}

// RunWordCloudTick executes one word cloud push maintenance cycle.
func (s *Service) RunWordCloudTick(bot *tgbotapi.BotAPI) {
	if bot == nil {
		return
	}
	s.processWordCloudPush(bot)
}
