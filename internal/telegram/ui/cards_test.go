package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func baseTicket() domain.Ticket {
	return domain.Ticket{
		ID:           7,
		UserID:       1,
		UserName:     "Иван",
		UserUsername: "ivan",
		CategoryID:   1,
		CategoryName: "Принтер",
		Status:       domain.TicketNew,
		CreatedAt:    time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC),
	}
}

func TestTicketCardEscapingAndClipping(t *testing.T) {
	ticket := baseTicket()
	ticket.UserName = "<b>Иван</b>"
	ticket.CategoryName = "<script>alert(1)</script>"
	ticket.Comment = "<img src=x> " + strings.Repeat("я", 3100)

	card := TicketCard(ticket, time.UTC)
	if !strings.Contains(card, "&lt;b&gt;Иван&lt;/b&gt;") {
		t.Fatal("user name is not escaped")
	}
	if !strings.Contains(card, "&lt;script&gt;") || strings.Contains(card, "<script>") {
		t.Fatal("category is not escaped")
	}
	if strings.Contains(card, "<img") || !strings.Contains(card, "&lt;img") {
		t.Fatal("comment is not escaped")
	}
	if !strings.Contains(card, "…") {
		t.Fatal("long comment is not clipped")
	}
}

func TestTicketCardStatuses(t *testing.T) {
	ticket := baseTicket()
	card := TicketCard(ticket, time.UTC)
	if !strings.Contains(card, "Новая") {
		t.Fatalf("new card = %s", card)
	}

	assignee := int64(2)
	taken := time.Date(2026, 11, 24, 10, 5, 0, 0, time.UTC)
	ticket.Status = domain.TicketInProgress
	ticket.AssigneeID = &assignee
	ticket.AssigneeName = "Админ"
	ticket.TakenAt = &taken
	card = TicketCard(ticket, time.UTC)
	if !strings.Contains(card, "В работе") || !strings.Contains(card, "Админ") {
		t.Fatalf("in-progress card = %s", card)
	}

	closed := time.Date(2026, 11, 24, 10, 15, 0, 0, time.UTC)
	ticket.Status = domain.TicketDone
	ticket.ClosedAt = &closed
	card = TicketCard(ticket, time.UTC)
	if !strings.Contains(card, "Решена") || !strings.Contains(card, "Исполнитель") {
		t.Fatalf("done card = %s", card)
	}
}

func TestFormattingHelpers(t *testing.T) {
	msk := time.FixedZone("MSK", 3*3600)
	at := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	if got := FormatTime(at, msk); got != "24.11.2026 13:00" {
		t.Fatalf("FormatTime = %q", got)
	}

	if got := Person("<b>", "user"); got != "&lt;b&gt; (@user)" {
		t.Fatalf("Person = %q", got)
	}
	if got := UserLine("Иван", "", 5); got != "Иван (id 5)" {
		t.Fatalf("UserLine = %q", got)
	}
	if got := clipText("абвгд", 3); got != "абв…" {
		t.Fatalf("clipText = %q", got)
	}
}
