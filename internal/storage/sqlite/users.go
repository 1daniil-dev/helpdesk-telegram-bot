package sqlite

import (
	"context"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func (d *DB) UpsertUser(ctx context.Context, p domain.Profile, now time.Time) error {
	_, err := d.ExecContext(ctx, `INSERT INTO users (telegram_id, full_name, username, role, created_at)
		VALUES (?, ?, ?, 'user', ?)
		ON CONFLICT(telegram_id) DO UPDATE SET
			full_name = excluded.full_name,
			username  = excluded.username`,
		p.ID, p.FullName(), p.Username, formatTime(now))
	return err
}

func (d *DB) GetUser(ctx context.Context, id int64) (domain.User, error) {
	var (
		u    domain.User
		role string
	)
	err := d.QueryRowContext(ctx, `SELECT telegram_id, full_name, username, role FROM users WHERE telegram_id = ?`, id).
		Scan(&u.ID, &u.FullName, &u.Username, &role)
	if err != nil {
		return domain.User{}, notFound(err)
	}
	u.Role = domain.Role(role)
	return u, nil
}

func (d *DB) SetRole(ctx context.Context, id int64, role domain.Role) error {
	_, err := d.ExecContext(ctx, `UPDATE users SET role = ? WHERE telegram_id = ?`, string(role), id)
	return err
}

func (d *DB) ListAdmins(ctx context.Context) ([]domain.User, error) {
	rows, err := d.QueryContext(ctx, `SELECT telegram_id, full_name, username, role
		FROM users WHERE role = 'admin' ORDER BY full_name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		var (
			u    domain.User
			role string
		)
		if err := rows.Scan(&u.ID, &u.FullName, &u.Username, &role); err != nil {
			return nil, err
		}
		u.Role = domain.Role(role)
		out = append(out, u)
	}
	return out, rows.Err()
}
