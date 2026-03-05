package service

import (
	"time"

	tgbot "github.com/go-telegram/bot"
)

func (s *Service) processWordCloudPush(bot *tgbot.Bot) {
	groups, err := s.repo.ListWordCloudEnabledGroups()
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("word cloud list groups failed: %v", err)
		}
		return
	}
	now := time.Now()
	for _, group := range groups {
		currentGroup := group
		ready, readyErr := s.wordCloudReadyToPush(&currentGroup, now)
		if readyErr != nil || !ready {
			continue
		}
		if err := s.SendWordCloudReportByTGGroupID(bot, group.TGGroupID, false); err != nil && s.logger != nil {
			s.logger.Printf("word cloud auto push failed group=%d err=%v", group.TGGroupID, err)
		}
	}
}

// RunWordCloudTick executes one word cloud push maintenance cycle.
func (s *Service) RunWordCloudTick(bot *tgbot.Bot) {
	if bot == nil {
		return
	}
	s.processWordCloudPush(bot)
}
