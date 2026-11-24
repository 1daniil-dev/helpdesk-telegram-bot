package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/service"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/ui"
)

func (a *App) Start(ctx context.Context, update telego.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil || msg.Chat.Type != telego.ChatTypePrivate {
		return
	}

	u, err := a.ensureUser(ctx, msg.From)
	if err != nil {
		a.Log.Error("регистрация пользователя", "err", err)
		return
	}

	if payload := startPayload(msg.Text); payload != "" {
		if a.tryRedeemInvite(ctx, msg.Chat.ID, u, payload) {
			return
		}
	}

	a.Sessions.Clear(u.ID)
	a.sendText(ctx, msg.Chat.ID, ui.StartText(u, a.isAdmin(u)), ui.ReplyMenu(a.isAdmin(u)))
}

func (a *App) Help(ctx context.Context, update telego.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil || msg.Chat.Type != telego.ChatTypePrivate {
		return
	}
	u, err := a.ensureUser(ctx, msg.From)
	if err != nil {
		a.Log.Error("регистрация пользователя", "err", err)
		return
	}
	a.sendText(ctx, msg.Chat.ID, ui.HelpText(a.isAdmin(u)), ui.ReplyMenu(a.isAdmin(u)))
}

func (a *App) AdminCommand(ctx context.Context, update telego.Update) {
	msg := update.Message
	if msg == nil || msg.From == nil || msg.Chat.Type != telego.ChatTypePrivate {
		return
	}
	u, err := a.ensureUser(ctx, msg.From)
	if err != nil {
		a.Log.Error("регистрация пользователя", "err", err)
		return
	}
	if !a.isAdmin(u) {
		a.sendText(ctx, msg.Chat.ID, ui.AdminDeniedText(), ui.ReplyMenu(false))
		return
	}
	a.sendText(ctx, msg.Chat.ID, ui.AdminMenuText(u, a.isOwner(u)), ui.AdminMenu(a.isOwner(u)))
}

func startPayload(text string) string {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return ""
	}
	return fields[1]
}

func (a *App) tryRedeemInvite(ctx context.Context, chatID int64, u domain.User, raw string) bool {
	code := service.NormalizeCode(raw)
	if code == "" {
		return false
	}

	admin, err := a.Admins.RedeemInvite(ctx, code, u.ID)
	switch {
	case err == nil:
		a.sendText(ctx, chatID, ui.AdminWelcomeText(admin), ui.ReplyMenu(true))
		return true
	case errors.Is(err, domain.ErrInviteNotFound):
		return false
	case errors.Is(err, domain.ErrInviteUsed):
		a.sendText(ctx, chatID, "⌛ Это приглашение уже использовано.", nil)
		return true
	case errors.Is(err, domain.ErrInviteExpired):
		a.sendText(ctx, chatID, "⌛ Срок действия приглашения истёк (24 часа).", nil)
		return true
	case errors.Is(err, domain.ErrInviteRevoked):
		a.sendText(ctx, chatID, "🚫 Это приглашение отозвано. Попроси владельца создать новое.", nil)
		return true
	case errors.Is(err, domain.ErrAlreadyAdmin):
		a.sendText(ctx, chatID, "Ты уже администратор 🙂", ui.ReplyMenu(true))
		return true
	default:
		a.Log.Error("активация приглашения", "err", err)
		a.sendText(ctx, chatID, "⚠️ Не удалось активировать приглашение. Попроси новое.", nil)
		return true
	}
}
