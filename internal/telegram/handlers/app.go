package handlers

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/config"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/service"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/session"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/ui"
)

const sessionTTL = 15 * time.Minute

type App struct {
	API        *telego.Bot
	Cfg        *config.Config
	Users      *service.UserService
	Tickets    *service.TicketService
	Admins     *service.AdminService
	Categories *service.CategoryService
	Sessions   *session.Manager
	Log        *slog.Logger
	BotName    string
}

func New(
	api *telego.Bot,
	cfg *config.Config,
	log *slog.Logger,
	botName string,
	users *service.UserService,
	tickets *service.TicketService,
	admins *service.AdminService,
	categories *service.CategoryService,
) *App {
	return &App{
		API:        api,
		Cfg:        cfg,
		Users:      users,
		Tickets:    tickets,
		Admins:     admins,
		Categories: categories,
		Sessions:   session.New(sessionTTL),
		Log:        log,
		BotName:    botName,
	}
}

func (a *App) Register(bh *th.BotHandler) {
	bh.Handle(adapt(a.Start), th.CommandEqual("start"))
	bh.Handle(adapt(a.Help), th.CommandEqual("help"))
	bh.Handle(adapt(a.AdminCommand), th.CommandEqual("admin"))
	bh.Handle(adapt(a.Callback), th.AnyCallbackQuery())
	bh.Handle(adapt(a.Default), th.AnyMessage())
}

func adapt(fn func(context.Context, telego.Update)) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		fn(ctx.Context(), update)
		return nil
	}
}

func (a *App) ensureUser(ctx context.Context, tg *telego.User) (domain.User, error) {
	return a.Users.Ensure(ctx, domain.Profile{
		ID:        tg.ID,
		FirstName: tg.FirstName,
		LastName:  tg.LastName,
		Username:  tg.Username,
	})
}

func (a *App) isAdmin(u domain.User) bool {
	return a.Cfg.IsOwner(u.ID) || u.IsAdmin()
}

func (a *App) isOwner(u domain.User) bool {
	return a.Cfg.IsOwner(u.ID)
}

func (a *App) sendText(ctx context.Context, chatID int64, text string, markup telego.ReplyMarkup) {
	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: ui.ParseModeHTML,
	}
	if markup != nil {
		params.ReplyMarkup = markup
	}
	if _, err := a.API.SendMessage(ctx, params); err != nil {
		a.Log.Warn("не удалось отправить сообщение", "chat", chatID, "err", err)
	}
}

func (a *App) editOrSend(ctx context.Context, cb *telego.CallbackQuery, text string, kb *telego.InlineKeyboardMarkup) {
	if cb.Message != nil {
		params := &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: cb.Message.GetChat().ID},
			MessageID: cb.Message.GetMessageID(),
			Text:      text,
			ParseMode: ui.ParseModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		} else {
			params.ReplyMarkup = ui.EmptyInline()
		}
		if _, err := a.API.EditMessageText(ctx, params); err == nil || isNotModified(err) {
			return
		}
	}
	if kb != nil {
		a.sendText(ctx, cb.From.ID, text, kb)
	} else {
		a.sendText(ctx, cb.From.ID, text, nil)
	}
}

func (a *App) showMainMenu(ctx context.Context, cb *telego.CallbackQuery) {
	a.editOrSend(ctx, cb, ui.MainMenuText(), nil)
}

func isNotModified(err error) bool {
	return err != nil && strings.Contains(err.Error(), "message is not modified")
}

func callbackID(data, prefix string) int64 {
	v, err := strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
