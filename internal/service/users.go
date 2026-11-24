package service

import (
	"context"
	"strconv"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

type UserService struct {
	users  UserStore
	owners map[int64]struct{}
}

func NewUserService(users UserStore, owners map[int64]struct{}) *UserService {
	return &UserService{users: users, owners: owners}
}

func (s *UserService) Ensure(ctx context.Context, p domain.Profile) (domain.User, error) {
	if p.FullName() == "" {
		p.FirstName = strconv.FormatInt(p.ID, 10)
		p.LastName = ""
	}

	if err := s.users.UpsertUser(ctx, p, time.Now()); err != nil {
		return domain.User{}, err
	}

	u, err := s.users.GetUser(ctx, p.ID)
	if err != nil {
		return domain.User{}, err
	}

	if _, isOwner := s.owners[u.ID]; isOwner && u.Role != domain.RoleAdmin {
		if err := s.users.SetRole(ctx, u.ID, domain.RoleAdmin); err != nil {
			return domain.User{}, err
		}
		u.Role = domain.RoleAdmin
	}

	return u, nil
}

func (s *UserService) Get(ctx context.Context, id int64) (domain.User, error) {
	return s.users.GetUser(ctx, id)
}

func (s *UserService) ListAdmins(ctx context.Context) ([]domain.User, error) {
	return s.users.ListAdmins(ctx)
}
