package notify

import (
	"context"
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/config"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/service"
	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/telegram/ui"
)

type Store interface {
	GetTicket(ctx context.Context, id int64) (domain.Ticket, error)
	ListAdmins(ctx context.Context) ([]domain.User, error)
	ListCards(ctx context.Context, ticketID int64) ([]domain.TicketCard, error)
	SaveCard(ctx context.Context, ticketID, adminID int64, messageID int) error
	DeleteCard(ctx context.Context, ticketID, adminID int64) error
}

type Notifier struct {
	api   *telego.Bot
	store Store
	cfg   *config.Config
	log   *slog.Logger
}

var _ service.Notifier = (*Notifier)(nil)

func New(api *telego.Bot, store Store, cfg *config.Config, log *slog.Logger) *Notifier {
	return &Notifier{api: api, store: store, cfg: cfg, log: log}
}

func (n *Notifier) TicketChanged(ctx context.Context, ticketID int64) {
	ticket, err := n.store.GetTicket(ctx, ticketID)
	if err != nil {
		n.log.Warn("не удалось загрузить заявку", "ticket", ticketID, "err", err)
		return
	}

	admins, err := n.store.ListAdmins(ctx)
	if err != nil {
		n.log.Warn("не удалось получить админов", "err", err)
		return
	}

	cards, err := n.store.ListCards(ctx, ticketID)
	if err != nil {
		n.log.Warn("не удалось получить карточки", "ticket", ticketID, "err", err)
		return
	}

	byAdmin := make(map[int64]int, len(cards))
	for _, c := range cards {
		byAdmin[c.AdminID] = c.MessageID
	}

	adminSet := make(map[int64]bool, len(admins))
	for _, admin := range admins {
		adminSet[admin.ID] = true
		text := ui.TicketCard(ticket, n.cfg.Location)
		kb := ui.TicketActions(ticket, admin.ID, n.cfg.IsOwner(admin.ID))

		if messageID, ok := byAdmin[admin.ID]; ok {
			markup := kb
			if markup == nil {
				markup = ui.EmptyInline()
			}
			_, err := n.api.EditMessageText(ctx, &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: admin.ID},
				MessageID:   messageID,
				Text:        text,
				ParseMode:   ui.ParseModeHTML,
				ReplyMarkup: markup,
			})
			if err == nil || isNotModified(err) {
				continue
			}
			n.log.Debug("карточка не обновилась, отправляю заново",
				"ticket", ticketID, "admin", admin.ID, "err", err)
			_ = n.store.DeleteCard(ctx, ticketID, admin.ID)
		}

		params := &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: admin.ID},
			Text:      text,
			ParseMode: ui.ParseModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, err := n.api.SendMessage(ctx, params)
		if err != nil {
			n.log.Warn("не удалось отправить заявку админу", "ticket", ticketID, "admin", admin.ID, "err", err)
			continue
		}
		if err := n.store.SaveCard(ctx, ticketID, admin.ID, msg.MessageID); err != nil {
			n.log.Warn("не удалось сохранить карточку", "ticket", ticketID, "admin", admin.ID, "err", err)
		}
	}

	for _, c := range cards {
		if !adminSet[c.AdminID] {
			_ = n.store.DeleteCard(ctx, ticketID, c.AdminID)
		}
	}
}

func (n *Notifier) TicketTaken(ctx context.Context, ticket domain.Ticket, admin domain.User) {
	n.send(ctx, ticket.UserID, ui.TicketTakenByUser(ticket.ID, admin.FullName), nil)
}

func (n *Notifier) TicketClosed(ctx context.Context, ticket domain.Ticket) {
	n.send(ctx, ticket.UserID, ui.TicketDoneByUser(ticket.ID), nil)
}

func (n *Notifier) AdminJoined(ctx context.Context, admin domain.User) {
	text := "🎉 Новый администратор: " + ui.UserLine(admin.FullName, admin.Username, admin.ID)
	for ownerID := range n.cfg.OwnerIDs {
		if ownerID == admin.ID {
			continue
		}
		n.send(ctx, ownerID, text, nil)
	}
}

func (n *Notifier) AdminRemoved(ctx context.Context, userID int64) {
	n.send(ctx, userID, "ℹ️ Твои права администратора отозваны.", ui.ReplyMenu(false))
}

func (n *Notifier) send(ctx context.Context, chatID int64, text string, markup telego.ReplyMarkup) {
	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: ui.ParseModeHTML,
	}
	if markup != nil {
		params.ReplyMarkup = markup
	}
	if _, err := n.api.SendMessage(ctx, params); err != nil {
		n.log.Warn("не удалось отправить уведомление", "chat", chatID, "err", err)
	}
}

func isNotModified(err error) bool {
	return err != nil && strings.Contains(err.Error(), "message is not modified")
}
