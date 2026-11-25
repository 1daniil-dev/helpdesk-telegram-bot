package service

import (
	"context"
	"errors"
	"testing"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func ticketFixture() (*fakeStore, *fakeNotifier, *TicketService, domain.Category) {
	store := newFakeStore()
	store.addUser(1, "Автор", domain.RoleUser)
	store.addUser(2, "Админ", domain.RoleAdmin)
	store.addUser(3, "Второй админ", domain.RoleAdmin)
	store.addUser(9, "Владелец", domain.RoleAdmin)
	category := store.addCategory("Принтер / сканер", true)
	notifier := &fakeNotifier{}
	svc := NewTicketService(store, store, notifier, fixedClock)
	return store, notifier, svc, category
}

func TestTicketCreate(t *testing.T) {
	ctx := context.Background()
	store, notifier, svc, category := ticketFixture()

	ticket, err := svc.Create(ctx, 1, category.ID, "  не включается  ")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ticket.Status != domain.TicketNew {
		t.Fatalf("status = %q, want new", ticket.Status)
	}
	if ticket.Comment != "не включается" {
		t.Fatalf("comment = %q, want trimmed", ticket.Comment)
	}
	if ticket.CategoryName != category.Name {
		t.Fatalf("category = %q", ticket.CategoryName)
	}
	if len(notifier.changed) != 1 {
		t.Fatalf("notifier changed = %d, want 1", len(notifier.changed))
	}

	hidden := store.addCategory("Скрытая", false)
	if _, err := svc.Create(ctx, 1, hidden.ID, ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("hidden category: err = %v, want ErrValidation", err)
	}
	if _, err := svc.Create(ctx, 1, 999, ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing category: err = %v, want ErrNotFound", err)
	}
}

func TestTicketTakeAndConflict(t *testing.T) {
	ctx := context.Background()
	_, notifier, svc, category := ticketFixture()

	ticket, err := svc.Create(ctx, 1, category.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Take(ctx, 999, domain.User{ID: 2, Role: domain.RoleAdmin})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("take missing: err = %v, want ErrNotFound", err)
	}

	taken, err := svc.Take(ctx, ticket.ID, domain.User{ID: 2, FullName: "Админ", Role: domain.RoleAdmin})
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if taken.Status != domain.TicketInProgress {
		t.Fatalf("status = %q, want in_progress", taken.Status)
	}
	if taken.AssigneeID == nil || *taken.AssigneeID != 2 {
		t.Fatalf("assignee = %v, want 2", taken.AssigneeID)
	}
	if len(notifier.taken) != 1 {
		t.Fatalf("notifier taken = %d, want 1", len(notifier.taken))
	}

	conflict, err := svc.Take(ctx, ticket.ID, domain.User{ID: 3, FullName: "Второй админ", Role: domain.RoleAdmin})
	if !errors.Is(err, domain.ErrAlreadyTaken) {
		t.Fatalf("second take: err = %v, want ErrAlreadyTaken", err)
	}
	if conflict.AssigneeID == nil || *conflict.AssigneeID != 2 {
		t.Fatalf("conflict assignee = %v, want 2", conflict.AssigneeID)
	}
	if len(notifier.taken) != 1 {
		t.Fatalf("notifier taken after conflict = %d, want 1", len(notifier.taken))
	}
}

func TestTicketClosePermissions(t *testing.T) {
	ctx := context.Background()
	_, notifier, svc, category := ticketFixture()

	ticket, err := svc.Create(ctx, 1, category.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Take(ctx, ticket.ID, domain.User{ID: 2, Role: domain.RoleAdmin}); err != nil {
		t.Fatal(err)
	}

	_, err = svc.Close(ctx, ticket.ID, domain.User{ID: 3, Role: domain.RoleAdmin}, false)
	if !errors.Is(err, domain.ErrCloseDenied) {
		t.Fatalf("close by other: err = %v, want ErrCloseDenied", err)
	}
	if len(notifier.closed) != 0 {
		t.Fatalf("notifier closed = %d, want 0", len(notifier.closed))
	}

	closed, err := svc.Close(ctx, ticket.ID, domain.User{ID: 2, Role: domain.RoleAdmin}, false)
	if err != nil {
		t.Fatalf("close by assignee: %v", err)
	}
	if !closed.IsDone() {
		t.Fatalf("status = %q, want done", closed.Status)
	}
	if len(notifier.closed) != 1 {
		t.Fatalf("notifier closed = %d, want 1", len(notifier.closed))
	}

	again, err := svc.Close(ctx, ticket.ID, domain.User{ID: 2, Role: domain.RoleAdmin}, false)
	if !errors.Is(err, domain.ErrTicketDone) {
		t.Fatalf("close again: err = %v, want ErrTicketDone", err)
	}
	if !again.IsDone() {
		t.Fatalf("again status = %q, want done", again.Status)
	}
	if _, err := svc.Take(ctx, ticket.ID, domain.User{ID: 3, Role: domain.RoleAdmin}); !errors.Is(err, domain.ErrTicketDone) {
		t.Fatalf("take done: err = %v, want ErrTicketDone", err)
	}
}

func TestOwnerClosesNewTicket(t *testing.T) {
	ctx := context.Background()
	_, notifier, svc, category := ticketFixture()

	ticket, err := svc.Create(ctx, 1, category.ID, "")
	if err != nil {
		t.Fatal(err)
	}

	closed, err := svc.Close(ctx, ticket.ID, domain.User{ID: 9, FullName: "Владелец", Role: domain.RoleAdmin}, true)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if !closed.IsDone() {
		t.Fatalf("status = %q, want done", closed.Status)
	}
	if closed.AssigneeID == nil || *closed.AssigneeID != 9 {
		t.Fatalf("assignee = %v, want owner", closed.AssigneeID)
	}
	if len(notifier.closed) != 1 {
		t.Fatalf("notifier closed = %d, want 1", len(notifier.closed))
	}
	if len(notifier.taken) != 0 {
		t.Fatalf("notifier taken = %d, want 0", len(notifier.taken))
	}
}

func TestAdminClosesNewTicketDirectly(t *testing.T) {
	ctx := context.Background()
	_, notifier, svc, category := ticketFixture()

	ticket, err := svc.Create(ctx, 1, category.ID, "")
	if err != nil {
		t.Fatal(err)
	}

	closed, err := svc.Close(ctx, ticket.ID, domain.User{ID: 2, FullName: "Админ", Role: domain.RoleAdmin}, false)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if !closed.IsDone() {
		t.Fatalf("status = %q, want done", closed.Status)
	}
	if closed.AssigneeID == nil || *closed.AssigneeID != 2 {
		t.Fatalf("assignee = %v, want 2", closed.AssigneeID)
	}
	if len(notifier.closed) != 1 {
		t.Fatalf("notifier closed = %d, want 1", len(notifier.closed))
	}
	if len(notifier.taken) != 0 {
		t.Fatalf("notifier taken = %d, want 0", len(notifier.taken))
	}
}

func TestTicketLists(t *testing.T) {
	ctx := context.Background()
	_, _, svc, category := ticketFixture()

	first, err := svc.Create(ctx, 1, category.ID, "первая")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Create(ctx, 1, category.ID, "вторая")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Take(ctx, first.ID, domain.User{ID: 2, Role: domain.RoleAdmin}); err != nil {
		t.Fatal(err)
	}

	mine, err := svc.ListMine(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 2 || mine[0].ID != second.ID || mine[1].ID != first.ID {
		t.Fatalf("mine = %+v, want newest first", mine)
	}

	open, err := svc.ListOpen(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 2 || open[0].ID != second.ID {
		t.Fatalf("open = %+v, want new ticket first", open)
	}
}
