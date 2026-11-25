package service

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

const (
	InviteTTL = 24 * time.Hour

	codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLength   = 8
)

type AdminDeps struct {
	Users    UserStore
	Invites  InviteStore
	Tickets  TicketStore
	Cards    CardStore
	Notifier Notifier
	Owners   map[int64]struct{}
	Now      func() time.Time
}

type AdminService struct {
	users    UserStore
	invites  InviteStore
	tickets  TicketStore
	cards    CardStore
	notifier Notifier
	owners   map[int64]struct{}
	now      func() time.Time
}

func NewAdminService(d AdminDeps) *AdminService {
	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &AdminService{
		users:    d.Users,
		invites:  d.Invites,
		tickets:  d.Tickets,
		cards:    d.Cards,
		notifier: d.Notifier,
		owners:   d.Owners,
		now:      now,
	}
}

func (s *AdminService) ListAdmins(ctx context.Context) ([]domain.User, error) {
	return s.users.ListAdmins(ctx)
}

func (s *AdminService) ListActiveInvites(ctx context.Context) ([]domain.Invite, error) {
	return s.invites.ListActiveInvites(ctx, s.now())
}

func (s *AdminService) CreateInvite(ctx context.Context, actorID int64, isOwner bool) (domain.Invite, error) {
	if !isOwner {
		return domain.Invite{}, domain.ErrForbidden
	}

	now := s.now()
	expires := now.Add(InviteTTL)

	var code string
	for attempt := 0; attempt < 5; attempt++ {
		candidate, err := generateCode()
		if err != nil {
			return domain.Invite{}, err
		}
		if _, err := s.invites.GetInviteByCode(ctx, candidate); err == domain.ErrNotFound {
			code = candidate
			break
		}
	}
	if code == "" {
		return domain.Invite{}, domain.ErrConflict
	}

	return s.invites.CreateInvite(ctx, code, actorID, now, expires)
}

func (s *AdminService) RedeemInvite(ctx context.Context, rawCode string, userID int64) (domain.User, error) {
	code := NormalizeCode(rawCode)
	if code == "" {
		return domain.User{}, domain.ErrInviteNotFound
	}

	invite, err := s.invites.GetInviteByCode(ctx, code)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, domain.ErrInviteNotFound
		}
		return domain.User{}, err
	}

	now := s.now()
	switch {
	case invite.Revoked:
		return domain.User{}, domain.ErrInviteRevoked
	case invite.UsedBy != nil:
		return domain.User{}, domain.ErrInviteUsed
	case !now.Before(invite.ExpiresAt):
		return domain.User{}, domain.ErrInviteExpired
	}

	user, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	if user.IsAdmin() {
		return user, domain.ErrAlreadyAdmin
	}

	ok, err := s.invites.RedeemInvite(ctx, invite.ID, userID, now)
	if err != nil {
		return domain.User{}, err
	}
	if !ok {
		return domain.User{}, domain.ErrInviteUsed
	}

	if err := s.users.SetRole(ctx, userID, domain.RoleAdmin); err != nil {
		return domain.User{}, err
	}
	user, err = s.users.GetUser(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}

	s.notifier.AdminJoined(ctx, user)

	if tickets, err := s.tickets.ListOpenTickets(ctx); err == nil {
		for _, ticket := range tickets {
			s.notifier.TicketChanged(ctx, ticket.ID)
		}
	}

	return user, nil
}

func (s *AdminService) RevokeInvite(ctx context.Context, actorID int64, isOwner bool, inviteID int64) error {
	if !isOwner {
		return domain.ErrForbidden
	}
	return s.invites.RevokeInvite(ctx, inviteID)
}

func (s *AdminService) RemoveAdmin(ctx context.Context, actorID int64, isOwner bool, targetID int64) ([]int64, error) {
	if !isOwner {
		return nil, domain.ErrForbidden
	}
	if _, ok := s.owners[targetID]; ok {
		return nil, domain.ErrForbidden
	}

	if err := s.users.SetRole(ctx, targetID, domain.RoleUser); err != nil {
		return nil, err
	}
	if err := s.cards.DeleteCardsForAdmin(ctx, targetID); err != nil {
		return nil, err
	}

	released, err := s.tickets.ReleaseTicketsOfAdmin(ctx, targetID)
	if err != nil {
		return nil, err
	}
	for _, ticketID := range released {
		s.notifier.TicketChanged(ctx, ticketID)
	}
	s.notifier.AdminRemoved(ctx, targetID)

	return released, nil
}

func NormalizeCode(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "ADM_")
	s = strings.TrimPrefix(s, "ADMIN_")
	if len(s) != codeLength {
		return ""
	}
	for _, r := range s {
		if !strings.ContainsRune(codeAlphabet, r) {
			return ""
		}
	}
	return s
}

func generateCode() (string, error) {
	buf := make([]byte, codeLength)
	max := big.NewInt(int64(len(codeAlphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		buf[i] = codeAlphabet[n.Int64()]
	}
	return string(buf), nil
}
