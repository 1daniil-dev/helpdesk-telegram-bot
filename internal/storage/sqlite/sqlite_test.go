package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

var testNow = time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedUser(t *testing.T, db *DB, id int64, name string, role domain.Role) {
	t.Helper()
	ctx := context.Background()
	if err := db.UpsertUser(ctx, domain.Profile{ID: id, FirstName: name}, testNow); err != nil {
		t.Fatalf("upsert user %d: %v", id, err)
	}
	if role == domain.RoleAdmin {
		if err := db.SetRole(ctx, id, domain.RoleAdmin); err != nil {
			t.Fatalf("set role %d: %v", id, err)
		}
	}
}

func firstCategory(t *testing.T, db *DB) domain.Category {
	t.Helper()
	cats, err := db.ListCategories(context.Background(), true)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(cats) == 0 {
		t.Fatal("no active categories")
	}
	return cats[0]
}

func TestMigrationsAndSeed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("user_version = %d, want 1", version)
	}
	cats, err := db.ListCategories(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 7 {
		t.Fatalf("categories = %d, want 7", len(cats))
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer reopened.Close()

	if err := reopened.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("user_version after reopen = %d, want 1", version)
	}
	cats, err = reopened.ListCategories(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 7 {
		t.Fatalf("categories after reopen = %d, want 7", len(cats))
	}
}

func TestUsersAndCategoriesStorage(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	profile := domain.Profile{ID: 1, FirstName: "Иван", LastName: "Петров", Username: "ivan"}
	if err := db.UpsertUser(ctx, profile, testNow); err != nil {
		t.Fatal(err)
	}
	user, err := db.GetUser(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if user.FullName != "Иван Петров" || user.Username != "ivan" || user.IsAdmin() {
		t.Fatalf("user = %+v", user)
	}

	if err := db.UpsertUser(ctx, domain.Profile{ID: 1, FirstName: "Пётр", Username: "petr"}, testNow); err != nil {
		t.Fatal(err)
	}
	user, err = db.GetUser(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if user.FullName != "Пётр" || user.Username != "petr" {
		t.Fatalf("updated user = %+v", user)
	}

	if err := db.SetRole(ctx, 1, domain.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	admins, err := db.ListAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(admins) != 1 || !admins[0].IsAdmin() {
		t.Fatalf("admins = %+v", admins)
	}
	if _, err := db.GetUser(ctx, 404); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing user: err = %v, want ErrNotFound", err)
	}

	category, err := db.AddCategory(ctx, "  Принтер  ")
	if err != nil {
		t.Fatal(err)
	}
	if category.Name != "  Принтер  " {
		t.Fatalf("storage must keep raw name, got %q", category.Name)
	}
	if _, err := db.AddCategory(ctx, "  Принтер  "); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate category: err = %v, want ErrConflict", err)
	}
	if err := db.RenameCategory(ctx, category.ID, "Печать"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetCategoryActive(ctx, category.ID, false); err != nil {
		t.Fatal(err)
	}
	active, err := db.ListCategories(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range active {
		if c.ID == category.ID {
			t.Fatal("hidden category returned as active")
		}
	}
	all, err := db.ListCategories(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range all {
		if c.ID == category.ID {
			found = true
			if c.Active {
				t.Fatal("hidden category marked active")
			}
		}
	}
	if !found {
		t.Fatal("hidden category missing from full list")
	}
	if _, err := db.GetCategory(ctx, 999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing category: err = %v, want ErrNotFound", err)
	}
}
