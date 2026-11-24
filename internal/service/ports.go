package service

import (
	"context"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

type UserStore interface {
	UpsertUser(ctx context.Context, p domain.Profile, now time.Time) error
	GetUser(ctx context.Context, id int64) (domain.User, error)
	SetRole(ctx context.Context, id int64, role domain.Role) error
	ListAdmins(ctx context.Context) ([]domain.User, error)
}

type InviteStore interface {
	CreateInvite(ctx context.Context, code string, createdBy int64, now, expiresAt time.Time) (domain.Invite, error)
	GetInviteByCode(ctx context.Context, code string) (domain.Invite, error)
	RedeemInvite(ctx context.Context, id, userID int64, now time.Time) (bool, error)
	RevokeInvite(ctx context.Context, id int64) error
	ListActiveInvites(ctx context.Context, now time.Time) ([]domain.Invite, error)
}

type CategoryStore interface {
	ListCategories(ctx context.Context, activeOnly bool) ([]domain.Category, error)
	GetCategory(ctx context.Context, id int64) (domain.Category, error)
	AddCategory(ctx context.Context, name string) (domain.Category, error)
	RenameCategory(ctx context.Context, id int64, name string) error
	SetCategoryActive(ctx context.Context, id int64, active bool) error
}

type TicketStore interface {
	CreateTicket(ctx context.Context, userID, categoryID int64, categoryName, comment string, now time.Time) (domain.Ticket, error)
	GetTicket(ctx context.Context, id int64) (domain.Ticket, error)
	TakeTicket(ctx context.Context, id, adminID int64, now time.Time) (bool, error)
	CloseTicket(ctx context.Context, id, actorID int64, isOwner bool, now time.Time) (bool, error)
	ListUserTickets(ctx context.Context, userID int64, limit int) ([]domain.Ticket, error)
	ListOpenTickets(ctx context.Context) ([]domain.Ticket, error)
	ReleaseTicketsOfAdmin(ctx context.Context, adminID int64) ([]int64, error)
}

type CardStore interface {
	SaveCard(ctx context.Context, ticketID, adminID int64, messageID int) error
	ListCards(ctx context.Context, ticketID int64) ([]domain.TicketCard, error)
	DeleteCard(ctx context.Context, ticketID, adminID int64) error
	DeleteCardsForAdmin(ctx context.Context, adminID int64) error
}

type Notifier interface {
	TicketChanged(ctx context.Context, ticketID int64)
	TicketTaken(ctx context.Context, ticket domain.Ticket, admin domain.User)
	TicketClosed(ctx context.Context, ticket domain.Ticket)
	AdminJoined(ctx context.Context, admin domain.User)
	AdminRemoved(ctx context.Context, userID int64)
}
