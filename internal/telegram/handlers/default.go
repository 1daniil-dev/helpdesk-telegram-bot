package handlers

import (
	"context"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/session"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/ui"
)

func (a *App) Default(ctx context.Context, update telego.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil || msg.Chat.Type != telego.ChatTypePrivate {
		return
	}

	u, err := a.ensureUser(ctx, msg.From)
	if err != nil {
		a.Log.Error("регистрация пользователя", "err", err)
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		a.sendText(ctx, msg.Chat.ID, ui.TextOnlyText(), ui.ReplyMenu(a.isAdmin(u)))
		return
	}

	if strings.HasPrefix(text, "/") {
		a.sendText(ctx, msg.Chat.ID, ui.UnknownText(), ui.ReplyMenu(a.isAdmin(u)))
		return
	}

	if a.tryRedeemInvite(ctx, msg.Chat.ID, u, text) {
		return
	}

	switch text {
	case ui.BtnNewTicket:
		a.Sessions.Clear(u.ID)
		a.openTicketFlow(ctx, msg.Chat.ID, u)
		return
	case ui.BtnMyTickets:
		a.Sessions.Clear(u.ID)
		a.openMyTickets(ctx, msg.Chat.ID, u)
		return
	case ui.BtnOpenTickets:
		a.Sessions.Clear(u.ID)
		a.openOpenTickets(ctx, msg.Chat.ID, u)
		return
	case ui.BtnAdmin:
		a.Sessions.Clear(u.ID)
		a.openAdminMenu(ctx, msg.Chat.ID, u)
		return
	}

	if s, ok := a.Sessions.Get(u.ID); ok {
		switch s.State {
		case session.StateTicketComment, session.StateTicketConfirm:
			s.Comment = text
			s.State = session.StateTicketConfirm
			a.Sessions.Set(u.ID, s)
			a.sendText(ctx, msg.Chat.ID, ui.TicketPreview(s.CategoryName, s.Comment), ui.ConfirmTicket())
			return

		case session.StateCategoryAdd:
			a.handleAddCategoryName(ctx, msg.Chat.ID, u, text)
			return

		case session.StateCategoryRename:
			a.handleRenameCategoryName(ctx, msg.Chat.ID, u, s, text)
			return
		}
	}

	a.sendText(ctx, msg.Chat.ID, ui.StartText(u, a.isAdmin(u)), ui.ReplyMenu(a.isAdmin(u)))
}
