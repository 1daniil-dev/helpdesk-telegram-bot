package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"
	_ "time/tzdata"

	"github.com/mymmrac/telego"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/config"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/service"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/storage/sqlite"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/handlers"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/notify"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("конфигурация", "err", err)
		os.Exit(1)
	}

	store, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		log.Error("база данных", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	api, err := telego.NewBot(cfg.BotToken, telego.WithDefaultLogger(false, true))
	if err != nil {
		log.Error("создать бота", "err", err)
		os.Exit(1)
	}

	me, err := api.GetMe(ctx)
	if err != nil {
		log.Error("не удалось получить данные бота: проверь BOT_TOKEN", "err", err)
		os.Exit(1)
	}

	notifier := notify.New(api, store, cfg, log)
	users := service.NewUserService(store, cfg.OwnerIDs)
	tickets := service.NewTicketService(store, store, notifier, time.Now)
	admins := service.NewAdminService(service.AdminDeps{
		Users:    store,
		Invites:  store,
		Tickets:  store,
		Cards:    store,
		Notifier: notifier,
		Owners:   cfg.OwnerIDs,
		Now:      time.Now,
	})
	categories := service.NewCategoryService(store)

	app := handlers.New(api, cfg, log, me.Username, users, tickets, admins, categories)
	bot := telegram.New(api, cfg, log, app)

	log.Info("бот запускается", "username", me.Username, "owners", len(cfg.OwnerIDs))
	if err := bot.Run(ctx); err != nil {
		log.Error("работа бота", "err", err)
		os.Exit(1)
	}
	log.Info("бот остановлен")
}
