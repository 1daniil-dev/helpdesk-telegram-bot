package handlers

import (
	"context"
	"errors"

	"github.com/mymmrac/telego"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/session"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/ui"
)

func (a *App) startTicket(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	cats, err := a.Categories.List(ctx, true)
	if err != nil {
		a.Log.Error("список категорий", "err", err)
		a.editOrSend(ctx, cb, ui.ErrorText(), nil)
		return
	}
	if len(cats) == 0 {
		a.editOrSend(ctx, cb, ui.NoCategoriesText(), ui.BackToMain())
		return
	}
	a.editOrSend(ctx, cb, ui.ChooseCategoryText(), ui.CategoryPicker(cats))
}

func (a *App) openTicketFlow(ctx context.Context, chatID int64, u domain.User) {
	cats, err := a.Categories.List(ctx, true)
	if err != nil {
		a.Log.Error("список категорий", "err", err)
		a.sendText(ctx, chatID, ui.ErrorText(), nil)
		return
	}
	if len(cats) == 0 {
		a.sendText(ctx, chatID, ui.NoCategoriesText(), ui.BackToMain())
		return
	}
	a.sendText(ctx, chatID, ui.ChooseCategoryText(), ui.CategoryPicker(cats))
}

func (a *App) chooseCategory(ctx context.Context, cb *telego.CallbackQuery, u domain.User, data string) {
	id := callbackID(data, "tcat:")
	if id == 0 {
		return
	}
	cat, err := a.Categories.Get(ctx, id)
	if err != nil || !cat.Active {
		a.editOrSend(ctx, cb, "⚠️ Эта категория больше недоступна. Выбери другую.", nil)
		return
	}

	a.Sessions.Set(u.ID, session.Data{
		State:        session.StateTicketComment,
		CategoryID:   cat.ID,
		CategoryName: cat.Name,
	})
	a.editOrSend(ctx, cb, ui.EnterCommentText(cat.Name), ui.CommentPrompt())
}

func (a *App) skipComment(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	s, ok := a.Sessions.Get(u.ID)
	if !ok || (s.State != session.StateTicketComment && s.State != session.StateTicketConfirm) {
		a.editOrSend(ctx, cb, ui.SessionExpiredText(), nil)
		return
	}

	s.Comment = ""
	s.State = session.StateTicketConfirm
	a.Sessions.Set(u.ID, s)

	a.editOrSend(ctx, cb, ui.TicketPreview(s.CategoryName, s.Comment), ui.ConfirmTicket())
}

func (a *App) submitTicket(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	s, ok := a.Sessions.Get(u.ID)
	if !ok || s.State != session.StateTicketConfirm {
		a.editOrSend(ctx, cb, ui.SessionExpiredText(), nil)
		return
	}

	ticket, err := a.Tickets.Create(ctx, u.ID, s.CategoryID, s.Comment)
	switch {
	case err == nil:
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, domain.ErrValidation):
		a.Sessions.Clear(u.ID)
		a.editOrSend(ctx, cb, "⚠️ Категория больше недоступна. Начни заново.", nil)
		return
	default:
		a.Log.Error("создать заявку", "err", err)
		a.editOrSend(ctx, cb, ui.ErrorText(), nil)
		return
	}

	a.Sessions.Clear(u.ID)
	a.editOrSend(ctx, cb, ui.TicketCreatedText(ticket.ID), nil)
}

func (a *App) showMyTickets(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	tickets, err := a.Tickets.ListMine(ctx, u.ID, 10)
	if err != nil {
		a.Log.Error("мои заявки", "err", err)
		a.editOrSend(ctx, cb, ui.ErrorText(), nil)
		return
	}
	if len(tickets) == 0 {
		a.editOrSend(ctx, cb, ui.MyTicketsEmptyText(), ui.MyTicketsMenu())
		return
	}
	a.editOrSend(ctx, cb, ui.MyTicketsText(tickets, a.Cfg.Location), ui.MyTicketsMenu())
}

func (a *App) openMyTickets(ctx context.Context, chatID int64, u domain.User) {
	tickets, err := a.Tickets.ListMine(ctx, u.ID, 10)
	if err != nil {
		a.Log.Error("мои заявки", "err", err)
		a.sendText(ctx, chatID, ui.ErrorText(), nil)
		return
	}
	if len(tickets) == 0 {
		a.sendText(ctx, chatID, ui.MyTicketsEmptyText(), ui.MyTicketsMenu())
		return
	}
	a.sendText(ctx, chatID, ui.MyTicketsText(tickets, a.Cfg.Location), ui.MyTicketsMenu())
}

func (a *App) showOpenTickets(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	if !a.isAdmin(u) {
		a.editOrSend(ctx, cb, ui.AdminDeniedText(), nil)
		return
	}
	a.showOpenTicketsMessage(ctx, func(text string, kb *telego.InlineKeyboardMarkup) {
		a.editOrSend(ctx, cb, text, kb)
	})
}

func (a *App) openOpenTickets(ctx context.Context, chatID int64, u domain.User) {
	if !a.isAdmin(u) {
		a.sendText(ctx, chatID, ui.AdminDeniedText(), nil)
		return
	}
	a.showOpenTicketsMessage(ctx, func(text string, kb *telego.InlineKeyboardMarkup) {
		if kb != nil {
			a.sendText(ctx, chatID, text, kb)
		} else {
			a.sendText(ctx, chatID, text, nil)
		}
	})
}

func (a *App) showOpenTicketsMessage(ctx context.Context, render func(text string, kb *telego.InlineKeyboardMarkup)) {
	tickets, err := a.Tickets.ListOpen(ctx)
	if err != nil {
		a.Log.Error("открытые заявки", "err", err)
		render(ui.ErrorText(), nil)
		return
	}
	if len(tickets) == 0 {
		render(ui.OpenTicketsEmptyText(), ui.OpenTicketsMenu())
		return
	}
	render(ui.OpenTicketsText(tickets, a.Cfg.Location), ui.OpenTicketsMenu())
}
