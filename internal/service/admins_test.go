package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func adminFixture(now func() time.Time) (*fakeStore, *fakeNotifier, *AdminService, *TicketService, domain.Category) {
	store := newFakeStore()
	store.addUser(1, "Автор", domain.RoleUser)
	store.addUser(2, "Кандидат", domain.RoleUser)
	store.addUser(3, "Второй", domain.RoleUser)
	store.addUser(9, "Владелец", domain.RoleAdmin)
	category := store.addCategory("Принтер / сканер", true)

	notifier := &fakeNotifier{}
	admins := NewAdminService(AdminDeps{
		Users:    store,
		Invites:  store,
		Tickets:  store,
		Cards:    store,
		Notifier: notifier,
		Owners:   map[int64]struct{}{9: {}},
		Now:      now,
	})
	tickets := NewTicketService(store, store, notifier, now)
	return store, notifier, admins, tickets, category
}

func TestCreateInviteOwnerOnly(t *testing.T) {
	ctx := context.Background()
	_, _, admins, _, _ := adminFixture(fixedClock)

	invite, err := admins.CreateInvite(ctx, 9, true)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if len(invite.Code) != 8 {
		t.Fatalf("code length = %d, want 8", len(invite.Code))
	}
	if invite.CreatedBy != 9 {
		t.Fatalf("created by = %d, want 9", invite.CreatedBy)
	}
	if !invite.ExpiresAt.Equal(fixedTime.Add(InviteTTL)) {
		t.Fatalf("expires = %v, want %v", invite.ExpiresAt, fixedTime.Add(InviteTTL))
	}

	if _, err := admins.CreateInvite(ctx, 9, false); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-owner invite: err = %v, want ErrForbidden", err)
	}
}

func TestRedeemInviteFlow(t *testing.T) {
	ctx := context.Background()
	store, notifier, admins, _, _ := adminFixture(fixedClock)

	invite, err := admins.CreateInvite(ctx, 9, true)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := admins.RedeemInvite(ctx, "adm_"+strings.ToLower(invite.Code), 2)
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if !admin.IsAdmin() {
		t.Fatalf("role = %q, want admin", admin.Role)
	}
	if len(notifier.joined) != 1 {
		t.Fatalf("notifier joined = %d, want 1", len(notifier.joined))
	}

	if _, err := admins.RedeemInvite(ctx, invite.Code, 3); !errors.Is(err, domain.ErrInviteUsed) {
		t.Fatalf("reuse: err = %v, want ErrInviteUsed", err)
	}

	second, err := admins.CreateInvite(ctx, 9, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admins.RedeemInvite(ctx, second.Code, 2); !errors.Is(err, domain.ErrAlreadyAdmin) {
		t.Fatalf("already admin: err = %v, want ErrAlreadyAdmin", err)
	}
	stored, err := store.GetInviteByCode(ctx, second.Code)
	if err != nil {
		t.Fatal(err)
	}
	if stored.UsedBy != nil {
		t.Fatal("invite was consumed for an existing admin")
	}

	if _, err := admins.RedeemInvite(ctx, "ZZZZZZZZ", 3); !errors.Is(err, domain.ErrInviteNotFound) {
		t.Fatalf("unknown code: err = %v, want ErrInviteNotFound", err)
	}
	if _, err := admins.RedeemInvite(ctx, "short", 3); !errors.Is(err, domain.ErrInviteNotFound) {
		t.Fatalf("short code: err = %v, want ErrInviteNotFound", err)
	}
}

func TestInviteExpiry(t *testing.T) {
	ctx := context.Background()
	store, notifier, admins, _, _ := adminFixture(fixedClock)

	invite, err := admins.CreateInvite(ctx, 9, true)
	if err != nil {
		t.Fatal(err)
	}

	later := NewAdminService(AdminDeps{
		Users:    store,
		Invites:  store,
		Tickets:  store,
		Cards:    store,
		Notifier: notifier,
		Owners:   map[int64]struct{}{9: {}},
		Now:      func() time.Time { return fixedTime.Add(25 * time.Hour) },
	})
	if _, err := later.RedeemInvite(ctx, invite.Code, 2); !errors.Is(err, domain.ErrInviteExpired) {
		t.Fatalf("expired invite: err = %v, want ErrInviteExpired", err)
	}
}

func TestInviteRevoked(t *testing.T) {
	ctx := context.Background()
	_, _, admins, _, _ := adminFixture(fixedClock)

	invite, err := admins.CreateInvite(ctx, 9, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := admins.RevokeInvite(ctx, 9, true, invite.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := admins.RedeemInvite(ctx, invite.Code, 2); !errors.Is(err, domain.ErrInviteRevoked) {
		t.Fatalf("revoked invite: err = %v, want ErrInviteRevoked", err)
	}
	if err := admins.RevokeInvite(ctx, 9, false, invite.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-owner revoke: err = %v, want ErrForbidden", err)
	}
}

func TestRemoveAdminReleasesTickets(t *testing.T) {
	ctx := context.Background()
	store, notifier, admins, tickets, category := adminFixture(fixedClock)

	invite, err := admins.CreateInvite(ctx, 9, true)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := admins.RedeemInvite(ctx, invite.Code, 2)
	if err != nil {
		t.Fatal(err)
	}

	ticket, err := tickets.Create(ctx, 1, category.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tickets.Take(ctx, ticket.ID, admin); err != nil {
		t.Fatal(err)
	}

	changedBefore := len(notifier.changed)
	released, err := admins.RemoveAdmin(ctx, 9, true, admin.ID)
	if err != nil {
		t.Fatalf("remove admin: %v", err)
	}
	if len(released) != 1 || released[0] != ticket.ID {
		t.Fatalf("released = %v, want [%d]", released, ticket.ID)
	}
	if len(notifier.changed) <= changedBefore {
		t.Fatal("released ticket was not refreshed for admins")
	}
	if len(notifier.removed) != 1 || notifier.removed[0] != admin.ID {
		t.Fatalf("notifier removed = %v, want [%d]", notifier.removed, admin.ID)
	}

	fresh, err := tickets.Get(ctx, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Status != domain.TicketNew || fresh.AssigneeID != nil {
		t.Fatalf("released ticket: status=%q assignee=%v", fresh.Status, fresh.AssigneeID)
	}
	user, err := store.GetUser(ctx, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if user.IsAdmin() {
		t.Fatal("removed admin still has admin role")
	}

	if _, err := admins.RemoveAdmin(ctx, 9, false, 3); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-owner remove: err = %v, want ErrForbidden", err)
	}
	if _, err := admins.RemoveAdmin(ctx, 9, true, 9); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("remove owner: err = %v, want ErrForbidden", err)
	}
}

func TestAdminLists(t *testing.T) {
	ctx := context.Background()
	_, _, admins, _, _ := adminFixture(fixedClock)

	first, err := admins.CreateInvite(ctx, 9, true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := admins.CreateInvite(ctx, 9, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := admins.RevokeInvite(ctx, 9, true, second.ID); err != nil {
		t.Fatal(err)
	}

	active, err := admins.ListActiveInvites(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].ID != first.ID {
		t.Fatalf("active invites = %+v, want only first", active)
	}

	if _, err := admins.RedeemInvite(ctx, first.Code, 2); err != nil {
		t.Fatal(err)
	}
	list, err := admins.ListAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("admins = %d, want 2", len(list))
	}
}

func TestNormalizeCode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HD7K2M9Q", "HD7K2M9Q"},
		{" hd7k2m9q ", "HD7K2M9Q"},
		{"adm_HD7K2M9Q", "HD7K2M9Q"},
		{"ADMIN_HD7K2M9Q", "HD7K2M9Q"},
		{"short", ""},
		{"ZZZZZZZZ", "ZZZZZZZZ"},
		{"HD7K2M9Q1", ""},
	}

	for _, tt := range tests {
		if got := NormalizeCode(tt.input); got != tt.want {
			t.Errorf("NormalizeCode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
