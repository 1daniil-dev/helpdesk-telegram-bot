package telegram

import (
	"context"
	"log/slog"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/config"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/handlers"
)

type Bot struct {
	api *telego.Bot
	cfg *config.Config
	log *slog.Logger
	app *handlers.App
}

func New(api *telego.Bot, cfg *config.Config, log *slog.Logger, app *handlers.App) *Bot {
	return &Bot{api: api, cfg: cfg, log: log, app: app}
}

func (b *Bot) Run(ctx context.Context) error {
	if err := b.api.DeleteWebhook(ctx, &telego.DeleteWebhookParams{DropPendingUpdates: true}); err != nil {
		b.log.Warn("не удалось удалить webhook / сбросить старые обновления", "err", err)
	}

	updates, err := b.api.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{Timeout: 30})
	if err != nil {
		return err
	}

	bh, err := th.NewBotHandler(b.api, updates, th.WithErrorHandler(func(ctx *th.Context, update telego.Update, err error) {
		b.log.Error("обработка обновления", "update", update.UpdateID, "err", err)
	}))
	if err != nil {
		return err
	}

	b.app.Register(bh)

	if err := b.api.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: []telego.BotCommand{
			{Command: "start", Description: "Главное меню"},
			{Command: "help", Description: "Помощь"},
			{Command: "admin", Description: "Админка"},
		},
	}); err != nil {
		b.log.Warn("не удалось установить команды бота", "err", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		err := bh.Start()
		if ctx.Err() == nil {
			if err != nil {
				b.log.Error("обработчик обновлений остановлен", "err", err)
			} else {
				b.log.Warn("обработчик обновлений завершился")
			}
			cancel()
		}
	}()

	b.log.Info("бот запущен")
	<-ctx.Done()
	_ = bh.Stop()
	return nil
}
