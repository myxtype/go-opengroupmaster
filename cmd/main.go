package main

import (
	"log"
	"time"

	"supervisor/internal/bot"
	"supervisor/internal/config"
	"supervisor/internal/handler"
	"supervisor/internal/repository"
	"supervisor/internal/scheduler"
	"supervisor/internal/service"
	"supervisor/pkg/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	l := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	repo, err := repository.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}

	botAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("init bot: %v", err)
	}
	botAPI.Debug = cfg.BotDebug

	svc := service.New(repo, l, cfg)
	svc.SetAdminSyncInterval(time.Duration(cfg.AdminSyncIntervalSecs) * time.Second)
	svc.StartAutoDeleteWorker(botAPI)
	defer svc.StopAutoDeleteWorker()
	svc.StartJoinVerifyWorker(botAPI)
	defer svc.StopJoinVerifyWorker()
	h := handler.New(svc, l)

	sch := scheduler.New(svc, botAPI, l)
	svc.SetScheduleRuntime(sch)
	if err := sch.Start(); err != nil {
		l.Printf("scheduler start failed: %v", err)
	}
	defer sch.Stop()

	l.Printf("bot authorized on account %s", botAPI.Self.UserName)
	bot.Run(botAPI, h, l, cfg.UpdateWorkers)
}
