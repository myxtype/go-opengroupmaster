package service

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const wordCloudIdleWait = time.Minute

func (s *Service) StartWordCloudWorker(bot *tgbotapi.BotAPI) {
	if bot == nil {
		return
	}
	s.wordCloudMu.Lock()
	if s.wordCloudStop != nil {
		s.wordCloudMu.Unlock()
		return
	}
	wake := make(chan struct{}, 1)
	stop := make(chan struct{})
	done := make(chan struct{})
	s.wordCloudWake = wake
	s.wordCloudStop = stop
	s.wordCloudDone = done
	s.wordCloudMu.Unlock()

	go s.runWordCloudWorker(bot, wake, stop, done)
	s.wakeWordCloudWorker()
}

func (s *Service) StopWordCloudWorker() {
	s.wordCloudMu.Lock()
	stop := s.wordCloudStop
	done := s.wordCloudDone
	if stop == nil {
		s.wordCloudMu.Unlock()
		return
	}
	s.wordCloudWake = nil
	s.wordCloudStop = nil
	s.wordCloudDone = nil
	s.wordCloudMu.Unlock()

	close(stop)
	<-done
}

func (s *Service) wakeWordCloudWorker() {
	s.wordCloudMu.Lock()
	wake := s.wordCloudWake
	s.wordCloudMu.Unlock()
	if wake == nil {
		return
	}
	select {
	case wake <- struct{}{}:
	default:
	}
}

func (s *Service) runWordCloudWorker(bot *tgbotapi.BotAPI, wake <-chan struct{}, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	for {
		s.processWordCloudPush(bot)
		timer := time.NewTimer(wordCloudIdleWait)
		select {
		case <-stop:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-wake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
		}
	}
}

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
