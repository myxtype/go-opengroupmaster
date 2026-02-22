package bot

import (
	"log"

	"supervisor/internal/handler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Run(bot *tgbotapi.BotAPI, h *handler.Handler, logger *log.Logger) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		h.HandleUpdate(bot, update)
		if update.Message != nil {
			logger.Printf("update from chat=%d", update.Message.Chat.ID)
		}
	}
}
