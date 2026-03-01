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

	// 初始化数据库仓库
	repo, err := repository.New(cfg.DBPath, cfg.GormLogSilent)
	if err != nil {
		log.Fatalf("init repository: %v", err)
	}

	// 初始化 Telegram Bot
	botAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("init bot: %v", err)
	}
	botAPI.Debug = cfg.BotDebug

	// 初始化服务层并配置管理员同步间隔
	svc := service.New(repo, l, cfg)
	svc.SetAdminSyncInterval(time.Duration(cfg.AdminSyncIntervalSecs) * time.Second)
	h := handler.New(svc, l)

	// 初始化调度器（用于定时任务）
	sch := scheduler.New(svc, botAPI, l)
	svc.SetScheduleRuntime(sch)
	if err := sch.Start(); err != nil {
		l.Printf("scheduler start failed: %v", err)
	}
	defer sch.Stop()

	l.Printf("bot authorized on account %s", botAPI.Self.UserName)
	// 启动 Bot（包含并发 update worker 处理）
	bot.Run(botAPI, h, l, cfg.UpdateWorkers)
}
