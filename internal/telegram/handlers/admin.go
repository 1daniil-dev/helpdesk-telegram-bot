package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mymmrac/telego"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/session"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/ui"
)

func (a *App) showAdminMenu(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	if !a.isAdmin(u) {
		a.editOrSend(ctx, cb, ui.AdminDeniedText(), nil)
		return
	}
	a.editOrSend(ctx, cb, ui.AdminMenuText(u, a.isOwner(u)), ui.AdminMenu(a.isOwner(u)))
}

func (a *App) openAdminMenu(ctx context.Context, chatID int64, u domain.User) {
	if !a.isAdmin(u) {
		a.sendText(ctx, chatID, ui.AdminDeniedText(), nil)
		return
	}
	a.sendText(ctx, chatID, ui.AdminMenuText(u, a.isOwner(u)), ui.AdminMenu(a.isOwner(u)))
}

func (a *App) showAdminsMenu(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	if !a.isOwner(u) {
		a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
		return
	}

	now := time.Now()
	admins, err := a.Admins.ListAdmins(ctx)
	if err != nil {
		a.Log.Error("список админов", "err", err)
		a.editOrSend(ctx, cb, ui.ErrorText(), nil)
		return
	}
	invites, err := a.Admins.ListActiveInvites(ctx)
	if err != nil {
		a.Log.Error("список приглашений", "err", err)
		a.editOrSend(ctx, cb, ui.ErrorText(), nil)
		return
	}

	a.editOrSend(ctx, cb,
		ui.AdminsListText(admins, invites, a.Cfg.OwnerIDs, now),
		ui.AdminsMenu(admins, invites, a.Cfg.OwnerIDs),
	)
}

func (a *App) createInvite(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	if !a.isOwner(u) {
		a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
		return
	}

	invite, err := a.Admins.CreateInvite(ctx, u.ID, true)
	if err != nil {
		a.Log.Error("создать приглашение", "err", err)
		a.editOrSend(ctx, cb, "⚠️ Не удалось создать код, попробуй ещё раз.", ui.BackToAdmins())
		return
	}

	link := ""
	if a.BotName != "" {
		link = "https://t.me/" + a.BotName + "?start=adm_" + invite.Code
	}
	a.sendText(ctx, cb.From.ID, ui.InviteCreatedText(invite.Code, link, invite.ExpiresAt, a.Cfg.Location), ui.BackToAdmins())
	a.showAdminsMenu(ctx, cb, u)
}

func (a *App) showRemoveAdminConfirm(ctx context.Context, cb *telego.CallbackQuery, u domain.User, data string) {
	if !a.isOwner(u) {
		a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
		return
	}
	id := callbackID(data, "adm:del:")
	if id == 0 {
		return
	}
	if a.Cfg.IsOwner(id) {
		a.editOrSend(ctx, cb, "👑 Владельца снять нельзя.", ui.BackToAdmins())
		return
	}
	target, err := a.Users.Get(ctx, id)
	if err != nil {
		a.editOrSend(ctx, cb, "Администратор не найден.", ui.BackToAdmins())
		return
	}
	a.editOrSend(ctx, cb, ui.RemoveAdminConfirmText(target), ui.RemoveAdminConfirm(id))
}

func (a *App) removeAdmin(ctx context.Context, cb *telego.CallbackQuery, u domain.User, data string) {
	if !a.isOwner(u) {
		a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
		return
	}
	id := callbackID(data, "adm:delok:")
	if id == 0 {
		return
	}
	if a.Cfg.IsOwner(id) {
		a.editOrSend(ctx, cb, "👑 Владельца снять нельзя.", ui.BackToAdmins())
		return
	}

	if _, err := a.Admins.RemoveAdmin(ctx, u.ID, true, id); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
			return
		}
		a.Log.Error("снять админа", "err", err)
		a.editOrSend(ctx, cb, ui.ErrorText(), ui.BackToAdmins())
		return
	}
	a.showAdminsMenu(ctx, cb, u)
}

func (a *App) revokeInvite(ctx context.Context, cb *telego.CallbackQuery, u domain.User, data string) {
	if !a.isOwner(u) {
		a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
		return
	}
	id := callbackID(data, "inv:revoke:")
	if id == 0 {
		return
	}
	if err := a.Admins.RevokeInvite(ctx, u.ID, true, id); err != nil {
		a.Log.Error("отозвать приглашение", "err", err)
		a.editOrSend(ctx, cb, ui.ErrorText(), ui.BackToAdmins())
		return
	}
	a.showAdminsMenu(ctx, cb, u)
}

func (a *App) showCategoriesMenu(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	if !a.isOwner(u) {
		a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
		return
	}
	cats, err := a.Categories.List(ctx, false)
	if err != nil {
		a.Log.Error("список категорий", "err", err)
		a.editOrSend(ctx, cb, ui.ErrorText(), nil)
		return
	}
	a.editOrSend(ctx, cb, ui.CategoriesText(cats, ""), ui.CategoriesMenu(cats))
}

func (a *App) showCategoriesMenuSend(ctx context.Context, chatID int64, u domain.User, note string) {
	if !a.isOwner(u) {
		a.sendText(ctx, chatID, ui.OwnerDeniedText(), nil)
		return
	}
	cats, err := a.Categories.List(ctx, false)
	if err != nil {
		a.Log.Error("список категорий", "err", err)
		a.sendText(ctx, chatID, ui.ErrorText(), nil)
		return
	}
	a.sendText(ctx, chatID, ui.CategoriesText(cats, note), ui.CategoriesMenu(cats))
}

func (a *App) beginAddCategory(ctx context.Context, cb *telego.CallbackQuery, u domain.User) {
	if !a.isOwner(u) {
		a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
		return
	}
	a.Sessions.Set(u.ID, session.Data{State: session.StateCategoryAdd})
	a.editOrSend(ctx, cb, ui.AddCategoryPrompt(), ui.CancelToCategories())
}

func (a *App) beginRenameCategory(ctx context.Context, cb *telego.CallbackQuery, u domain.User, data string) {
	if !a.isOwner(u) {
		a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
		return
	}
	id := callbackID(data, "cats:ren:")
	if id == 0 {
		return
	}
	cat, err := a.Categories.Get(ctx, id)
	if err != nil {
		a.editOrSend(ctx, cb, "Категория не найдена.", ui.CancelToCategories())
		return
	}
	a.Sessions.Set(u.ID, session.Data{
		State:        session.StateCategoryRename,
		CategoryID:   cat.ID,
		CategoryName: cat.Name,
	})
	a.editOrSend(ctx, cb, ui.RenameCategoryPrompt(cat.Name), ui.CancelToCategories())
}

func (a *App) toggleCategory(ctx context.Context, cb *telego.CallbackQuery, u domain.User, data string) {
	if !a.isOwner(u) {
		a.editOrSend(ctx, cb, ui.OwnerDeniedText(), nil)
		return
	}
	id := callbackID(data, "cats:tgl:")
	if id == 0 {
		return
	}
	if _, err := a.Categories.Toggle(ctx, id); err != nil {
		a.Log.Error("переключить категорию", "err", err)
		a.editOrSend(ctx, cb, ui.ErrorText(), nil)
		return
	}
	a.showCategoriesMenu(ctx, cb, u)
}

func (a *App) handleAddCategoryName(ctx context.Context, chatID int64, u domain.User, name string) {
	if !a.isOwner(u) {
		a.Sessions.Clear(u.ID)
		a.sendText(ctx, chatID, ui.OwnerDeniedText(), nil)
		return
	}

	category, err := a.Categories.Add(ctx, name)
	switch {
	case err == nil:
	case errors.Is(err, domain.ErrValidation):
		a.sendText(ctx, chatID, ui.CategoryNameInvalidText(), ui.CancelToCategories())
		return
	case errors.Is(err, domain.ErrConflict):
		a.sendText(ctx, chatID, ui.CategoryExistsText(), ui.CancelToCategories())
		return
	default:
		a.Log.Error("добавить категорию", "err", err)
		a.Sessions.Clear(u.ID)
		a.sendText(ctx, chatID, ui.ErrorText(), nil)
		return
	}

	a.Sessions.Clear(u.ID)
	a.showCategoriesMenuSend(ctx, chatID, u, ui.CategoryAddedNote(category.Name))
}

func (a *App) handleRenameCategoryName(ctx context.Context, chatID int64, u domain.User, s session.Data, name string) {
	if !a.isOwner(u) {
		a.Sessions.Clear(u.ID)
		a.sendText(ctx, chatID, ui.OwnerDeniedText(), nil)
		return
	}

	err := a.Categories.Rename(ctx, s.CategoryID, name)
	switch {
	case err == nil:
	case errors.Is(err, domain.ErrValidation):
		a.sendText(ctx, chatID, ui.CategoryNameInvalidText(), ui.CancelToCategories())
		return
	case errors.Is(err, domain.ErrConflict):
		a.sendText(ctx, chatID, ui.CategoryExistsText(), ui.CancelToCategories())
		return
	default:
		a.Log.Error("переименовать категорию", "err", err)
		a.Sessions.Clear(u.ID)
		a.sendText(ctx, chatID, ui.ErrorText(), nil)
		return
	}

	a.Sessions.Clear(u.ID)
	a.showCategoriesMenuSend(ctx, chatID, u, ui.CategoryRenamedNote(strings.TrimSpace(name)))
}
