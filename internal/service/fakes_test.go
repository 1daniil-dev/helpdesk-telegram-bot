package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

type fakeStore struct {
	users          map[int64]domain.User
	invites        map[int64]domain.Invite
	categories     map[int64]domain.Category
	tickets        map[int64]domain.Ticket
	cards          map[int64]map[int64]int
	nextInviteID   int64
	nextCategoryID int64
	nextTicketID   int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users:      make(map[int64]domain.User),
		invites:    make(map[int64]domain.Invite),
		categories: make(map[int64]domain.Category),
		tickets:    make(map[int64]domain.Ticket),
		cards:      make(map[int64]map[int64]int),
	}
}

func (f *fakeStore) addUser(id int64, name string, role domain.Role) {
	f.users[id] = domain.User{ID: id, FullName: name, Role: role}
}

func (f *fakeStore) addCategory(name string, active bool) domain.Category {
	f.nextCategoryID++
	c := domain.Category{
		ID:        f.nextCategoryID,
		Name:      name,
		SortOrder: int(f.nextCategoryID),
		Active:    active,
	}
	f.categories[c.ID] = c
	return c
}

// ---------------------------------------------------------------- users

func (f *fakeStore) UpsertUser(_ context.Context, p domain.Profile, _ time.Time) error {
	u, ok := f.users[p.ID]
	if !ok {
		u = domain.User{ID: p.ID, Role: domain.RoleUser}
	}
	u.FullName = p.FullName()
	u.Username = p.Username
	f.users[p.ID] = u
	return nil
}

func (f *fakeStore) GetUser(_ context.Context, id int64) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeStore) SetRole(_ context.Context, id int64, role domain.Role) error {
	u, ok := f.users[id]
	if !ok {
		return nil
	}
	u.Role = role
	f.users[id] = u
	return nil
}

func (f *fakeStore) ListAdmins(_ context.Context) ([]domain.User, error) {
	var out []domain.User
	for _, u := range f.users {
		if u.IsAdmin() {
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ---------------------------------------------------------------- invites

func (f *fakeStore) CreateInvite(_ context.Context, code string, createdBy int64, now, expiresAt time.Time) (domain.Invite, error) {
	for _, inv := range f.invites {
		if inv.Code == code {
			return domain.Invite{}, fmt.Errorf("duplicate invite code")
		}
	}
	f.nextInviteID++
	inv := domain.Invite{
		ID:        f.nextInviteID,
		Code:      code,
		CreatedBy: createdBy,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	f.invites[inv.ID] = inv
	return inv, nil
}

func (f *fakeStore) GetInviteByCode(_ context.Context, code string) (domain.Invite, error) {
	for _, inv := range f.invites {
		if inv.Code == code {
			return inv, nil
		}
	}
	return domain.Invite{}, domain.ErrNotFound
}

func (f *fakeStore) RedeemInvite(_ context.Context, id, userID int64, now time.Time) (bool, error) {
	inv, ok := f.invites[id]
	if !ok {
		return false, nil
	}
	if inv.Revoked || inv.UsedBy != nil || !now.Before(inv.ExpiresAt) {
		return false, nil
	}
	usedBy := userID
	usedAt := now
	inv.UsedBy = &usedBy
	inv.UsedAt = &usedAt
	f.invites[id] = inv
	return true, nil
}

func (f *fakeStore) RevokeInvite(_ context.Context, id int64) error {
	inv, ok := f.invites[id]
	if !ok {
		return domain.ErrNotFound
	}
	inv.Revoked = true
	f.invites[id] = inv
	return nil
}

func (f *fakeStore) ListActiveInvites(_ context.Context, now time.Time) ([]domain.Invite, error) {
	var out []domain.Invite
	for _, inv := range f.invites {
		if inv.IsActive(now) {
			out = append(out, inv)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

// ---------------------------------------------------------------- categories

func (f *fakeStore) ListCategories(_ context.Context, activeOnly bool) ([]domain.Category, error) {
	var out []domain.Category
	for _, c := range f.categories {
		if activeOnly && !c.Active {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (f *fakeStore) GetCategory(_ context.Context, id int64) (domain.Category, error) {
	c, ok := f.categories[id]
	if !ok {
		return domain.Category{}, domain.ErrNotFound
	}
	return c, nil
}

func (f *fakeStore) AddCategory(_ context.Context, name string) (domain.Category, error) {
	for _, c := range f.categories {
		if c.Name == name {
			return domain.Category{}, domain.ErrConflict
		}
	}
	f.nextCategoryID++
	c := domain.Category{
		ID:        f.nextCategoryID,
		Name:      name,
		SortOrder: int(f.nextCategoryID),
		Active:    true,
	}
	f.categories[c.ID] = c
	return c, nil
}

func (f *fakeStore) RenameCategory(_ context.Context, id int64, name string) error {
	if _, ok := f.categories[id]; !ok {
		return domain.ErrNotFound
	}
	for otherID, c := range f.categories {
		if otherID != id && c.Name == name {
			return domain.ErrConflict
		}
	}
	c := f.categories[id]
	c.Name = name
	f.categories[id] = c
	return nil
}

func (f *fakeStore) SetCategoryActive(_ context.Context, id int64, active bool) error {
	c, ok := f.categories[id]
	if !ok {
		return domain.ErrNotFound
	}
	c.Active = active
	f.categories[id] = c
	return nil
}

// ---------------------------------------------------------------- tickets

func (f *fakeStore) CreateTicket(_ context.Context, userID, categoryID int64, categoryName, comment string, now time.Time) (domain.Ticket, error) {
	f.nextTicketID++
	t := domain.Ticket{
		ID:           f.nextTicketID,
		UserID:       userID,
		CategoryID:   categoryID,
		CategoryName: categoryName,
		Comment:      comment,
		Status:       domain.TicketNew,
		CreatedAt:    now,
	}
	if u, ok := f.users[userID]; ok {
		t.UserName = u.FullName
		t.UserUsername = u.Username
	}
	f.tickets[t.ID] = t
	return t, nil
}

func (f *fakeStore) GetTicket(_ context.Context, id int64) (domain.Ticket, error) {
	t, ok := f.tickets[id]
	if !ok {
		return domain.Ticket{}, domain.ErrNotFound
	}
	return t, nil
}

func (f *fakeStore) TakeTicket(_ context.Context, id, adminID int64, now time.Time) (bool, error) {
	t, ok := f.tickets[id]
	if !ok || t.Status != domain.TicketNew {
		return false, nil
	}
	assignee := adminID
	takenAt := now
	t.Status = domain.TicketInProgress
	t.AssigneeID = &assignee
	t.TakenAt = &takenAt
	if u, ok := f.users[adminID]; ok {
		t.AssigneeName = u.FullName
		t.AssigneeUsername = u.Username
	}
	f.tickets[id] = t
	return true, nil
}

func (f *fakeStore) CloseTicket(_ context.Context, id, actorID int64, isOwner bool, now time.Time) (bool, error) {
	t, ok := f.tickets[id]
	if !ok || t.Status == domain.TicketDone {
		return false, nil
	}
	if !isOwner {
		if t.Status != domain.TicketInProgress || t.AssigneeID == nil || *t.AssigneeID != actorID {
			return false, nil
		}
	}

	closedAt := now
	t.Status = domain.TicketDone
	t.ClosedAt = &closedAt
	if t.AssigneeID == nil {
		assignee := actorID
		t.AssigneeID = &assignee
		if u, ok := f.users[actorID]; ok {
			t.AssigneeName = u.FullName
			t.AssigneeUsername = u.Username
		}
	}
	if t.TakenAt == nil {
		t.TakenAt = &closedAt
	}
	f.tickets[id] = t
	return true, nil
}

func (f *fakeStore) ListUserTickets(_ context.Context, userID int64, limit int) ([]domain.Ticket, error) {
	var out []domain.Ticket
	for _, t := range f.tickets {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeStore) ListOpenTickets(_ context.Context) ([]domain.Ticket, error) {
	var out []domain.Ticket
	for _, t := range f.tickets {
		if t.IsOpen() {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if (out[i].Status == domain.TicketNew) != (out[j].Status == domain.TicketNew) {
			return out[i].Status == domain.TicketNew
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (f *fakeStore) ReleaseTicketsOfAdmin(_ context.Context, adminID int64) ([]int64, error) {
	var ids []int64
	for id, t := range f.tickets {
		if t.Status != domain.TicketInProgress || t.AssigneeID == nil || *t.AssigneeID != adminID {
			continue
		}
		ids = append(ids, id)
		t.Status = domain.TicketNew
		t.AssigneeID = nil
		t.TakenAt = nil
		f.tickets[id] = t
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// ---------------------------------------------------------------- cards

func (f *fakeStore) SaveCard(_ context.Context, ticketID, adminID int64, messageID int) error {
	if f.cards[ticketID] == nil {
		f.cards[ticketID] = make(map[int64]int)
	}
	f.cards[ticketID][adminID] = messageID
	return nil
}

func (f *fakeStore) ListCards(_ context.Context, ticketID int64) ([]domain.TicketCard, error) {
	var out []domain.TicketCard
	for adminID, messageID := range f.cards[ticketID] {
		out = append(out, domain.TicketCard{AdminID: adminID, MessageID: messageID})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AdminID < out[j].AdminID })
	return out, nil
}

func (f *fakeStore) DeleteCard(_ context.Context, ticketID, adminID int64) error {
	delete(f.cards[ticketID], adminID)
	return nil
}

func (f *fakeStore) DeleteCardsForAdmin(_ context.Context, adminID int64) error {
	for ticketID := range f.cards {
		delete(f.cards[ticketID], adminID)
	}
	return nil
}

// ---------------------------------------------------------------- notifier

type fakeNotifier struct {
	changed []int64
	taken   []domain.Ticket
	closed  []domain.Ticket
	joined  []domain.User
	removed []int64
}

func (n *fakeNotifier) TicketChanged(_ context.Context, ticketID int64) {
	n.changed = append(n.changed, ticketID)
}

func (n *fakeNotifier) TicketTaken(_ context.Context, ticket domain.Ticket, admin domain.User) {
	n.taken = append(n.taken, ticket)
}

func (n *fakeNotifier) TicketClosed(_ context.Context, ticket domain.Ticket) {
	n.closed = append(n.closed, ticket)
}

func (n *fakeNotifier) AdminJoined(_ context.Context, admin domain.User) {
	n.joined = append(n.joined, admin)
}

func (n *fakeNotifier) AdminRemoved(_ context.Context, userID int64) {
	n.removed = append(n.removed, userID)
}

var fixedTime = time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)

func fixedClock() time.Time {
	return fixedTime
}
