package sqlite

import (
	"context"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func (d *DB) ListCategories(ctx context.Context, activeOnly bool) ([]domain.Category, error) {
	q := `SELECT id, name, sort_order, is_active FROM categories`
	if activeOnly {
		q += ` WHERE is_active = 1`
	}
	q += ` ORDER BY sort_order, id`

	rows, err := d.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Category
	for rows.Next() {
		var (
			c      domain.Category
			active int
		)
		if err := rows.Scan(&c.ID, &c.Name, &c.SortOrder, &active); err != nil {
			return nil, err
		}
		c.Active = active != 0
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) GetCategory(ctx context.Context, id int64) (domain.Category, error) {
	var (
		c      domain.Category
		active int
	)
	err := d.QueryRowContext(ctx, `SELECT id, name, sort_order, is_active FROM categories WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.SortOrder, &active)
	if err != nil {
		return domain.Category{}, notFound(err)
	}
	c.Active = active != 0
	return c, nil
}

func (d *DB) AddCategory(ctx context.Context, name string) (domain.Category, error) {
	res, err := d.ExecContext(ctx, `INSERT INTO categories (name, sort_order)
		VALUES (?, COALESCE((SELECT MAX(sort_order) + 1 FROM categories), 0))`, name)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Category{}, domain.ErrConflict
		}
		return domain.Category{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Category{}, err
	}
	return d.GetCategory(ctx, id)
}

func (d *DB) RenameCategory(ctx context.Context, id int64, name string) error {
	_, err := d.ExecContext(ctx, `UPDATE categories SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

func (d *DB) SetCategoryActive(ctx context.Context, id int64, active bool) error {
	_, err := d.ExecContext(ctx, `UPDATE categories SET is_active = ? WHERE id = ?`, active, id)
	return err
}
