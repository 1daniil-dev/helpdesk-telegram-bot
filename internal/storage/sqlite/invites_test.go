package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func TestInviteStorage(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	invite, err := db.CreateInvite(ctx, "HD7K2M9Q", 9, testNow, testNow.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if invite.Code != "HD7K2M9Q" || invite.UsedBy != nil || invite.Revoked {
		t.Fatalf("invite = %+v", invite)
	}

	got, err := db.GetInviteByCode(ctx, "HD7K2M9Q")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != invite.ID {
		t.Fatalf("id = %d, want %d", got.ID, invite.ID)
	}

	ok, err := db.RedeemInvite(ctx, invite.ID, 2, testNow.Add(time.Hour))
	if err != nil || !ok {
		t.Fatalf("redeem: ok=%v err=%v", ok, err)
	}
	ok, err = db.RedeemInvite(ctx, invite.ID, 3, testNow.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("invite redeemed twice")
	}

	expired, err := db.CreateInvite(ctx, "AB2CD3EF", 9, testNow, testNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	ok, err = db.RedeemInvite(ctx, expired.ID, 4, testNow.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expired invite redeemed")
	}

	revoked, err := db.CreateInvite(ctx, "AB2CD3EG", 9, testNow, testNow.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RevokeInvite(ctx, revoked.ID); err != nil {
		t.Fatal(err)
	}
	ok, err = db.RedeemInvite(ctx, revoked.ID, 5, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("revoked invite redeemed")
	}

	fresh, err := db.CreateInvite(ctx, "AB2CD3EH", 9, testNow, testNow.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	active, err := db.ListActiveInvites(ctx, testNow.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].ID != fresh.ID {
		t.Fatalf("active invites = %+v, want only fresh", active)
	}

	if _, err := db.GetInviteByCode(ctx, "ZZZZZZZZ"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing invite: err = %v, want ErrNotFound", err)
	}
}
