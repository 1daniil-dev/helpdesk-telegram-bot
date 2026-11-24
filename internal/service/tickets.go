package service

import (
	"context"
	"strings"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

type TicketService struct {
	tickets    TicketStore
	categories CategoryStore
	notifier   Notifier
	now        func() time.Time
}

func NewTicketService(tickets TicketStore, categories CategoryStore, notifier Notifier, now func() time.Time) *TicketService {
	if now == nil {
		now = time.Now
	}
	return &TicketService{tickets: tickets, categories: categories, notifier: notifier, now: now}
}

func (s *TicketService) Create(ctx context.Context, userID, categoryID int64, comment string) (domain.Ticket, error) {
	category, err := s.categories.GetCategory(ctx, categoryID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !category.Active {
		return domain.Ticket{}, domain.ErrValidation
	}

	ticket, err := s.tickets.CreateTicket(ctx, userID, category.ID, category.Name,
		strings.TrimSpace(comment), s.now())
	if err != nil {
		return domain.Ticket{}, err
	}

	s.notifier.TicketChanged(ctx, ticket.ID)
	return ticket, nil
}

func (s *TicketService) Get(ctx context.Context, id int64) (domain.Ticket, error) {
	return s.tickets.GetTicket(ctx, id)
}

func (s *TicketService) ListMine(ctx context.Context, userID int64, limit int) ([]domain.Ticket, error) {
	return s.tickets.ListUserTickets(ctx, userID, limit)
}

func (s *TicketService) ListOpen(ctx context.Context) ([]domain.Ticket, error) {
	return s.tickets.ListOpenTickets(ctx)
}

func (s *TicketService) Take(ctx context.Context, ticketID int64, admin domain.User) (domain.Ticket, error) {
	ticket, err := s.tickets.GetTicket(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if ticket.IsDone() {
		return ticket, domain.ErrTicketDone
	}

	ok, err := s.tickets.TakeTicket(ctx, ticketID, admin.ID, s.now())
	if err != nil {
		return domain.Ticket{}, err
	}
	if !ok {
		return s.conflict(ctx, ticketID)
	}

	final, err := s.tickets.GetTicket(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	s.notifier.TicketChanged(ctx, final.ID)
	s.notifier.TicketTaken(ctx, final, admin)
	return final, nil
}

func (s *TicketService) Close(ctx context.Context, ticketID int64, actor domain.User, isOwner bool) (domain.Ticket, error) {
	ticket, err := s.tickets.GetTicket(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if ticket.IsDone() {
		return ticket, domain.ErrTicketDone
	}

	if ticket.Status == domain.TicketInProgress && !isOwner &&
		(ticket.AssigneeID == nil || *ticket.AssigneeID != actor.ID) {
		return ticket, domain.ErrCloseDenied
	}

	if ticket.Status == domain.TicketNew {
		ok, err := s.tickets.TakeTicket(ctx, ticketID, actor.ID, s.now())
		if err != nil {
			return domain.Ticket{}, err
		}
		if !ok {
			return s.conflict(ctx, ticketID)
		}
	}

	ok, err := s.tickets.CloseTicket(ctx, ticketID, actor.ID, isOwner, s.now())
	if err != nil {
		return domain.Ticket{}, err
	}
	if !ok {
		return s.conflict(ctx, ticketID)
	}

	final, err := s.tickets.GetTicket(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	s.notifier.TicketChanged(ctx, final.ID)
	s.notifier.TicketClosed(ctx, final)
	return final, nil
}

func (s *TicketService) conflict(ctx context.Context, ticketID int64) (domain.Ticket, error) {
	fresh, err := s.tickets.GetTicket(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if fresh.IsDone() {
		return fresh, domain.ErrTicketDone
	}
	return fresh, domain.ErrAlreadyTaken
}
