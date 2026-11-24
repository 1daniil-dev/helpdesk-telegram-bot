package service

import (
	"context"
	"strings"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

type CategoryService struct {
	categories CategoryStore
}

func NewCategoryService(categories CategoryStore) *CategoryService {
	return &CategoryService{categories: categories}
}

func (s *CategoryService) List(ctx context.Context, activeOnly bool) ([]domain.Category, error) {
	return s.categories.ListCategories(ctx, activeOnly)
}

func (s *CategoryService) Get(ctx context.Context, id int64) (domain.Category, error) {
	return s.categories.GetCategory(ctx, id)
}

func (s *CategoryService) Add(ctx context.Context, name string) (domain.Category, error) {
	name, err := validateCategoryName(name)
	if err != nil {
		return domain.Category{}, err
	}
	return s.categories.AddCategory(ctx, name)
}

func (s *CategoryService) Rename(ctx context.Context, id int64, name string) error {
	name, err := validateCategoryName(name)
	if err != nil {
		return err
	}
	return s.categories.RenameCategory(ctx, id, name)
}

func (s *CategoryService) Toggle(ctx context.Context, id int64) (domain.Category, error) {
	category, err := s.categories.GetCategory(ctx, id)
	if err != nil {
		return domain.Category{}, err
	}
	if err := s.categories.SetCategoryActive(ctx, id, !category.Active); err != nil {
		return domain.Category{}, err
	}
	category.Active = !category.Active
	return category, nil
}

func validateCategoryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if n := len([]rune(name)); n < 2 || n > 40 {
		return "", domain.ErrValidation
	}
	return name, nil
}
