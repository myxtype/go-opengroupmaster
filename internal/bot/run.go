package bot

import (
	"log"
	"sync"

	"supervisor/internal/handler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const updateWorkerQueueSize = 128

func Run(bot *tgbotapi.BotAPI, h *handler.Handler, logger *log.Logger, workers int) {
	if workers < 1 {
		workers = 1
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	logger.Printf("update workers=%d", workers)
	updates := bot.GetUpdatesChan(u)

	chans := make([]chan tgbotapi.Update, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		ch := make(chan tgbotapi.Update, updateWorkerQueueSize)
		chans[i] = ch
		wg.Add(1)
		go func(workerID int, in <-chan tgbotapi.Update) {
			defer wg.Done()
			for update := range in {
				h.HandleUpdate(bot, update)
				if update.Message != nil {
					logger.Printf("worker=%d update from chat=%d", workerID, update.Message.Chat.ID)
				}
			}
		}(i, ch)
	}

	for update := range updates {
		idx := routeWorker(update, workers)
		chans[idx] <- update
	}
	for _, ch := range chans {
		close(ch)
	}
	wg.Wait()
}

func routeWorker(update tgbotapi.Update, workers int) int {
	key := updateRouteKey(update)
	return int(key % uint64(workers))
}

func updateRouteKey(update tgbotapi.Update) uint64 {
	if update.Message != nil {
		return uint64(update.Message.Chat.ID)
	}
	if update.CallbackQuery != nil {
		if update.CallbackQuery.Message != nil {
			return uint64(update.CallbackQuery.Message.Chat.ID)
		}
		if update.CallbackQuery.From != nil {
			return uint64(update.CallbackQuery.From.ID)
		}
	}
	return uint64(update.UpdateID)
}
