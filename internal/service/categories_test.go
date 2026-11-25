package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func TestCategoryAddValidation(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	svc := NewCategoryService(store)

	created, err := svc.Add(ctx, "  Почта  ")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if created.Name != "Почта" || !created.Active {
		t.Fatalf("created = %+v", created)
	}

	if _, err := svc.Add(ctx, "Почта"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate: err = %v, want ErrConflict", err)
	}
	if _, err := svc.Add(ctx, " a "); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("short: err = %v, want ErrValidation", err)
	}
	if _, err := svc.Add(ctx, strings.Repeat("x", 41)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("long: err = %v, want ErrValidation", err)
	}
}

func TestCategoryRenameAndToggle(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	svc := NewCategoryService(store)

	first, err := svc.Add(ctx, "Принтер")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Add(ctx, "Почта"); err != nil {
		t.Fatal(err)
	}

	if err := svc.Rename(ctx, first.ID, "  Печать  "); err != nil {
		t.Fatalf("rename: %v", err)
	}
	renamed, err := svc.Get(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Печать" {
		t.Fatalf("name = %q, want Печать", renamed.Name)
	}

	if err := svc.Rename(ctx, first.ID, "Почта"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("rename duplicate: err = %v, want ErrConflict", err)
	}
	if err := svc.Rename(ctx, 999, "Новое"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("rename missing: err = %v, want ErrNotFound", err)
	}
	if err := svc.Rename(ctx, first.ID, "x"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("rename short: err = %v, want ErrValidation", err)
	}

	hidden, err := svc.Toggle(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if hidden.Active {
		t.Fatal("category is still active")
	}
	active, err := svc.List(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range active {
		if c.ID == first.ID {
			t.Fatal("hidden category is present in active list")
		}
	}

	shown, err := svc.Toggle(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !shown.Active {
		t.Fatal("category did not become active")
	}
	if _, err := svc.Get(ctx, 999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("get missing: err = %v, want ErrNotFound", err)
	}
}
