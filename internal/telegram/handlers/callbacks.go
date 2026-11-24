package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/mymmrac/telego"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/ui"
)

func (a *App) Callback(ctx context.Context, update telego.Update) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}

	if err := a.API.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{CallbackQueryID: cb.ID}); err != nil {
		a.Log.Debug("answer callback", "err", err)
	}

	u, err := a.ensureUser(ctx, &cb.From)
	if err != nil {
		a.Log.Error("регистрация пользователя", "err", err)
		return
	}

	data := cb.Data

	switch {
	case data == "menu:main":
		a.Sessions.Clear(u.ID)
		a.showMainMenu(ctx, cb)

	case data == "tnew":
		a.Sessions.Clear(u.ID)
		a.startTicket(ctx, cb, u)

	case len(data) > 5 && data[:5] == "tcat:":
		a.Sessions.Clear(u.ID)
		a.chooseCategory(ctx, cb, u, data)

	case data == "tskip":
		a.skipComment(ctx, cb, u)

	case data == "tsend":
		a.submitTicket(ctx, cb, u)

	case data == "tcancel":
		a.Sessions.Clear(u.ID)
		a.editOrSend(ctx, cb, ui.CancelText(), nil)

	case data == "tmy":
		a.Sessions.Clear(u.ID)
		a.showMyTickets(ctx, cb, u)

	case data == "topen":
		a.Sessions.Clear(u.ID)
		a.showOpenTickets(ctx, cb, u)

	case data == "adm:menu":
		a.Sessions.Clear(u.ID)
		a.showAdminMenu(ctx, cb, u)

	case data == "adm:list":
		a.Sessions.Clear(u.ID)
		a.showAdminsMenu(ctx, cb, u)

	case data == "adm:add":
		a.Sessions.Clear(u.ID)
		a.createInvite(ctx, cb, u)

	case len(data) > 8 && data[:8] == "adm:del:":
		a.Sessions.Clear(u.ID)
		a.showRemoveAdminConfirm(ctx, cb, u, data)

	case len(data) > 10 && data[:10] == "adm:delok:":
		a.removeAdmin(ctx, cb, u, data)

	case len(data) > 11 && data[:11] == "inv:revoke:":
		a.revokeInvite(ctx, cb, u, data)

	case data == "cats:menu":
		a.Sessions.Clear(u.ID)
		a.showCategoriesMenu(ctx, cb, u)

	case data == "cats:add":
		a.Sessions.Clear(u.ID)
		a.beginAddCategory(ctx, cb, u)

	case len(data) > 9 && data[:9] == "cats:ren:":
		a.Sessions.Clear(u.ID)
		a.beginRenameCategory(ctx, cb, u, data)

	case len(data) > 9 && data[:9] == "cats:tgl:":
		a.Sessions.Clear(u.ID)
		a.toggleCategory(ctx, cb, u, data)

	case len(data) > 5 && data[:5] == "take:":
		a.takeTicket(ctx, cb, u, data)

	case len(data) > 5 && data[:5] == "done:":
		a.closeTicket(ctx, cb, u, data)
	}
}

func (a *App) takeTicket(ctx context.Context, cb *telego.CallbackQuery, u domain.User, data string) {
	if !a.isAdmin(u) {
		a.sendText(ctx, cb.From.ID, ui.AdminDeniedText(), nil)
		return
	}

	id := callbackID(data, "take:")
	if id == 0 {
		return
	}

	ticket, err := a.Tickets.Take(ctx, id, u)
	switch {
	case err == nil:
		return
	case errors.Is(err, domain.ErrAlreadyTaken), errors.Is(err, domain.ErrTicketDone):
		a.sendText(ctx, cb.From.ID, ui.TakeConflictText(ticket), nil)
	case errors.Is(err, domain.ErrNotFound):
		a.sendText(ctx, cb.From.ID, fmt.Sprintf("Заявка #%d не найдена.", id), nil)
	default:
		a.Log.Error("взять заявку", "ticket", id, "err", err)
		a.sendText(ctx, cb.From.ID, ui.ErrorText(), nil)
	}
}

func (a *App) closeTicket(ctx context.Context, cb *telego.CallbackQuery, u domain.User, data string) {
	if !a.isAdmin(u) {
		a.sendText(ctx, cb.From.ID, ui.AdminDeniedText(), nil)
		return
	}

	id := callbackID(data, "done:")
	if id == 0 {
		return
	}

	ticket, err := a.Tickets.Close(ctx, id, u, a.isOwner(u))
	switch {
	case err == nil:
		return
	case errors.Is(err, domain.ErrTicketDone):
		a.sendText(ctx, cb.From.ID, ui.AlreadyDoneText(ticket.ID), nil)
	case errors.Is(err, domain.ErrCloseDenied):
		a.sendText(ctx, cb.From.ID, ui.CloseDeniedText(ticket), nil)
	case errors.Is(err, domain.ErrAlreadyTaken):
		a.sendText(ctx, cb.From.ID, ui.TakeConflictText(ticket), nil)
	case errors.Is(err, domain.ErrNotFound):
		a.sendText(ctx, cb.From.ID, fmt.Sprintf("Заявка #%d не найдена.", id), nil)
	default:
		a.Log.Error("закрыть заявку", "ticket", id, "err", err)
		a.sendText(ctx, cb.From.ID, ui.ErrorText(), nil)
	}
}
