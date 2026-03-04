package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
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
	allowedUpdates := []string{
		models.AllowedUpdateMessage,
		models.AllowedUpdateEditedMessage,
		models.AllowedUpdateCallbackQuery,
		models.AllowedUpdateChatMember,
		models.AllowedUpdateMyChatMember,
	}

	options := []tgbot.Option{
		tgbot.WithDefaultHandler(func(_ context.Context, bot *tgbot.Bot, update *models.Update) {
			if update == nil {
				return
			}
			h.HandleUpdate(bot, update)
		}),
		tgbot.WithAllowedUpdates(allowedUpdates),
	}
	if cfg.WebhookSecretToken != "" {
		options = append(options, tgbot.WithWebhookSecretToken(cfg.WebhookSecretToken))
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

	switch cfg.BotRunMode {
	case "webhook":
		webhookURL, err := url.Parse(cfg.WebhookURL)
		if err != nil {
			log.Fatalf("parse webhook url: %v", err)
		}
		webhookPath := webhookURL.Path
		if webhookPath == "" {
			webhookPath = "/"
		}

		ok, err := botAPI.SetWebhook(ctx, &tgbot.SetWebhookParams{
			URL:                cfg.WebhookURL,
			AllowedUpdates:     allowedUpdates,
			DropPendingUpdates: cfg.WebhookDropPending,
			SecretToken:        cfg.WebhookSecretToken,
		})
		if err != nil {
			log.Fatalf("set webhook: %v", err)
		}
		if !ok {
			log.Fatalf("set webhook: telegram api returned false")
		}

		mux := http.NewServeMux()
		mux.HandleFunc(webhookPath, botAPI.WebhookHandler())
		server := &http.Server{
			Addr:    cfg.WebhookListenAddr,
			Handler: mux,
		}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
				l.Printf("webhook server shutdown failed: %v", err)
			}
		}()
		go botAPI.StartWebhook(ctx)

		l.Printf("running in webhook mode, listen=%s path=%s", cfg.WebhookListenAddr, webhookPath)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("webhook server: %v", err)
		}
	default:
		_, err := botAPI.DeleteWebhook(ctx, &tgbot.DeleteWebhookParams{
			DropPendingUpdates: false,
		})
		if err != nil {
			log.Fatalf("delete webhook before polling: %v", err)
		}
		l.Printf("running in polling mode")
		botAPI.Start(ctx)
	}
}
