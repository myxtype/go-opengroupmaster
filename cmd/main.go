package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"supervisor/internal/config"
	"supervisor/internal/handler"
	"supervisor/internal/repository"
	"supervisor/internal/scheduler"
	"supervisor/internal/service"
	"supervisor/pkg/logger"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	l := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	repo, err := repository.New(cfg.DBPath, cfg.GormLogSilent)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	svc := service.New(repo, l, cfg)
	svc.SetAdminSyncInterval(time.Duration(cfg.AdminSyncIntervalSecs) * time.Second)
	h := handler.New(svc, l)

	options := []tgbot.Option{
		tgbot.WithDefaultHandler(func(_ context.Context, bot *tgbot.Bot, update *models.Update) {
			if update == nil {
				return
			}
			h.HandleUpdate(bot, update)
		}),
		tgbot.WithAllowedUpdates([]string{
			models.AllowedUpdateMessage,
			models.AllowedUpdateEditedMessage,
			models.AllowedUpdateCallbackQuery,
			models.AllowedUpdateChatMember,
			models.AllowedUpdateMyChatMember,
		}),
	}
	if cfg.BotDebug {
		options = append(options, tgbot.WithDebug())
	}
	botAPI, err := tgbot.New(cfg.BotToken, options...)
	if err != nil {
		log.Fatalf("init bot: %v", err)
	}
	me, err := botAPI.GetMe(context.Background())
	if err != nil {
		log.Fatalf("get bot me: %v", err)
	}
	h.SetBotUsername(me.Username)

	sch := scheduler.New(svc, botAPI, l)
	svc.SetScheduleRuntime(sch)
	if err := sch.Start(); err != nil {
		l.Printf("scheduler start failed: %v", err)
	}
	defer sch.Stop()

	l.Printf("bot authorized on account %s", me.Username)
	botAPI.Start(ctx)
}
