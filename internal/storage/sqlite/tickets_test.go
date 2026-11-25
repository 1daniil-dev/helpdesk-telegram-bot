package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func TestTicketStorageLifecycle(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	seedUser(t, db, 1, "Автор", domain.RoleUser)
	seedUser(t, db, 2, "Админ", domain.RoleAdmin)
	seedUser(t, db, 3, "Второй админ", domain.RoleAdmin)
	category := firstCategory(t, db)

	ticket, err := db.CreateTicket(ctx, 1, category.ID, category.Name, "не работает", testNow)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ticket.Status != domain.TicketNew {
		t.Fatalf("status = %q, want new", ticket.Status)
	}

	if err := db.RenameCategory(ctx, category.ID, "Переименована"); err != nil {
		t.Fatal(err)
	}
	stored, err := db.GetTicket(ctx, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.CategoryName != category.Name {
		t.Fatalf("category snapshot = %q, want %q", stored.CategoryName, category.Name)
	}

	ok, err := db.TakeTicket(ctx, ticket.ID, 2, testNow)
	if err != nil || !ok {
		t.Fatalf("take: ok=%v err=%v", ok, err)
	}
	ok, err = db.TakeTicket(ctx, ticket.ID, 3, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("ticket taken twice")
	}

	stored, err = db.GetTicket(ctx, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.TicketInProgress || stored.AssigneeID == nil || *stored.AssigneeID != 2 {
		t.Fatalf("after take = %+v", stored)
	}

	ok, err = db.CloseTicket(ctx, ticket.ID, 3, false, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("ticket closed by non-assignee")
	}

	ok, err = db.CloseTicket(ctx, ticket.ID, 2, false, testNow)
	if err != nil || !ok {
		t.Fatalf("close: ok=%v err=%v", ok, err)
	}
	stored, err = db.GetTicket(ctx, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.TicketDone || stored.ClosedAt == nil {
		t.Fatalf("after close = %+v", stored)
	}

	ok, err = db.CloseTicket(ctx, ticket.ID, 2, false, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("ticket closed twice")
	}

	second, err := db.CreateTicket(ctx, 1, category.ID, category.Name, "", testNow)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := db.TakeTicket(ctx, second.ID, 2, testNow); err != nil || !ok {
		t.Fatalf("take second: ok=%v err=%v", ok, err)
	}
	released, err := db.ReleaseTicketsOfAdmin(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(released) != 1 || released[0] != second.ID {
		t.Fatalf("released = %v, want [%d]", released, second.ID)
	}
	fresh, err := db.GetTicket(ctx, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Status != domain.TicketNew || fresh.AssigneeID != nil || fresh.TakenAt != nil {
		t.Fatalf("released ticket = %+v", fresh)
	}

	mine, err := db.ListUserTickets(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 2 {
		t.Fatalf("my tickets = %d, want 2", len(mine))
	}
	open, err := db.ListOpenTickets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].ID != second.ID {
		t.Fatalf("open tickets = %+v", open)
	}

	if _, err := db.GetTicket(ctx, 999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing ticket: err = %v, want ErrNotFound", err)
	}
}

func TestTicketAlwaysReturnsTimesInUTC(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	seedUser(t, db, 1, "Автор", domain.RoleUser)
	category := firstCategory(t, db)

	local := time.FixedZone("MSK", 3*3600)
	now := time.Date(2026, 11, 24, 13, 5, 0, 0, local)
	ticket, err := db.CreateTicket(ctx, 1, category.ID, category.Name, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if ticket.CreatedAt.Location() != time.UTC {
		t.Fatalf("location = %v, want UTC", ticket.CreatedAt.Location())
	}
	if !ticket.CreatedAt.Equal(now) {
		t.Fatalf("created at = %v, want %v", ticket.CreatedAt, now)
	}
}
