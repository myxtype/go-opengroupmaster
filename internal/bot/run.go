package bot

import (
	"log"
	"sync"

	"supervisor/internal/handler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const updateWorkerQueueSize = 128

// Run 启动 Telegram Bot，使用分片并发处理 Update
// workers: 并发 worker 数量
// 同一 chat/user 的 Update 保持顺序处理，避免私聊输入态错乱
func Run(bot *tgbotapi.BotAPI, h *handler.Handler, logger *log.Logger, workers int) {
	if workers < 1 {
		workers = 1
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "edited_message", "callback_query", "chat_member", "my_chat_member"}

	logger.Printf("update workers=%d", workers)
	updates := bot.GetUpdatesChan(u)

	// 为每个 worker 创建独立的队列
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

	// 将 Update 分发到对应的 worker 队列
	// 路由键基于 ChatID 或 UserID，确保同一会话的 Update 由同一 worker 处理
	for update := range updates {
		idx := routeWorker(update, workers)
		chans[idx] <- update
	}
	for _, ch := range chans {
		close(ch)
	}
	wg.Wait()
}

// routeWorker 根据 Update 内容确定分配的 worker 索引
func routeWorker(update tgbotapi.Update, workers int) int {
	key := updateRouteKey(update)
	return int(key % uint64(workers))
}

// updateRouteKey 从 Update 中提取路由键（ChatID 或 UserID）
// 用于确保同一 chat/user 的 Update 由同一 worker 顺序处理
func updateRouteKey(update tgbotapi.Update) uint64 {
	if update.Message != nil {
		return uint64(update.Message.Chat.ID)
	}
	if update.EditedMessage != nil {
		return uint64(update.EditedMessage.Chat.ID)
	}
	if update.CallbackQuery != nil {
		if update.CallbackQuery.Message != nil {
			return uint64(update.CallbackQuery.Message.Chat.ID)
		}
		if update.CallbackQuery.From != nil {
			return uint64(update.CallbackQuery.From.ID)
		}
	}
	if update.ChatMember != nil {
		return uint64(update.ChatMember.Chat.ID)
	}
	if update.MyChatMember != nil {
		return uint64(update.MyChatMember.Chat.ID)
	}
	return uint64(update.UpdateID)
}
