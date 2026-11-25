package service

import (
	"context"
	"errors"
	"testing"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func TestEnsureUser(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	owners := map[int64]struct{}{9: {}}
	svc := NewUserService(store, owners)

	user, err := svc.Ensure(ctx, domain.Profile{ID: 1, FirstName: "Иван", LastName: "Петров", Username: "ivan"})
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != domain.RoleUser {
		t.Fatalf("role = %q, want user", user.Role)
	}
	if user.FullName != "Иван Петров" {
		t.Fatalf("name = %q", user.FullName)
	}

	owner, err := svc.Ensure(ctx, domain.Profile{ID: 9, FirstName: "Владелец"})
	if err != nil {
		t.Fatal(err)
	}
	if !owner.IsAdmin() {
		t.Fatalf("owner role = %q, want admin", owner.Role)
	}

	updated, err := svc.Ensure(ctx, domain.Profile{ID: 9, FirstName: "Другой", LastName: "Владелец"})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.IsAdmin() {
		t.Fatal("owner lost admin role on re-ensure")
	}
	if updated.FullName != "Другой Владелец" {
		t.Fatalf("name = %q", updated.FullName)
	}

	fallback, err := svc.Ensure(ctx, domain.Profile{ID: 5})
	if err != nil {
		t.Fatal(err)
	}
	if fallback.FullName != "5" {
		t.Fatalf("fallback name = %q, want 5", fallback.FullName)
	}

	admins, err := svc.ListAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(admins) != 1 || admins[0].ID != 9 {
		t.Fatalf("admins = %+v, want only owner", admins)
	}

	if _, err := svc.Get(ctx, 404); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("get missing: err = %v, want ErrNotFound", err)
	}
}
