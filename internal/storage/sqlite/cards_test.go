package sqlite

import (
	"context"
	"testing"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func TestCardStorage(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	seedUser(t, db, 1, "Автор", domain.RoleUser)
	category := firstCategory(t, db)
	ticket, err := db.CreateTicket(ctx, 1, category.ID, category.Name, "", testNow)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveCard(ctx, ticket.ID, 2, 100); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCard(ctx, ticket.ID, 2, 101); err != nil {
		t.Fatal(err)
	}
	cards, err := db.ListCards(ctx, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards[0].MessageID != 101 {
		t.Fatalf("cards = %+v, want updated message 101", cards)
	}

	if err := db.SaveCard(ctx, ticket.ID, 3, 200); err != nil {
		t.Fatal(err)
	}
	cards, err = db.ListCards(ctx, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 2 {
		t.Fatalf("cards = %d, want 2", len(cards))
	}

	if err := db.DeleteCard(ctx, ticket.ID, 2); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteCardsForAdmin(ctx, 3); err != nil {
		t.Fatal(err)
	}
	cards, err = db.ListCards(ctx, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 0 {
		t.Fatalf("cards = %+v, want empty", cards)
	}
}
