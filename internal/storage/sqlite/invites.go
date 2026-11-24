package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

const inviteSelect = `SELECT id, code, created_by, created_at, expires_at, used_by, used_at, revoked
	FROM admin_invites`

func scanInvite(row interface{ Scan(dest ...any) error }) (domain.Invite, error) {
	var (
		inv       domain.Invite
		createdAt string
		expiresAt string
		usedBy    sql.NullInt64
		usedAt    sql.NullString
		revoked   int
	)
	if err := row.Scan(&inv.ID, &inv.Code, &inv.CreatedBy, &createdAt, &expiresAt, &usedBy, &usedAt, &revoked); err != nil {
		return domain.Invite{}, notFound(err)
	}

	inv.CreatedAt = parseTime(createdAt)
	inv.ExpiresAt = parseTime(expiresAt)
	if usedBy.Valid {
		v := usedBy.Int64
		inv.UsedBy = &v
	}
	if usedAt.Valid {
		v := parseTime(usedAt.String)
		inv.UsedAt = &v
	}
	inv.Revoked = revoked != 0

	return inv, nil
}

func (d *DB) CreateInvite(ctx context.Context, code string, createdBy int64, now, expiresAt time.Time) (domain.Invite, error) {
	if _, err := d.ExecContext(ctx, `INSERT INTO admin_invites (code, created_by, created_at, expires_at)
		VALUES (?, ?, ?, ?)`,
		code, createdBy, formatTime(now), formatTime(expiresAt)); err != nil {
		return domain.Invite{}, err
	}
	return d.GetInviteByCode(ctx, code)
}

func (d *DB) GetInviteByCode(ctx context.Context, code string) (domain.Invite, error) {
	return scanInvite(d.QueryRowContext(ctx, inviteSelect+` WHERE code = ?`, code))
}

func (d *DB) ListActiveInvites(ctx context.Context, now time.Time) ([]domain.Invite, error) {
	rows, err := d.QueryContext(ctx, inviteSelect+` WHERE used_by IS NULL AND revoked = 0 AND expires_at > ?
		ORDER BY id DESC`, formatTime(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Invite
	for rows.Next() {
		inv, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (d *DB) RedeemInvite(ctx context.Context, id, userID int64, now time.Time) (bool, error) {
	res, err := d.ExecContext(ctx, `UPDATE admin_invites SET used_by = ?, used_at = ?
		WHERE id = ? AND used_by IS NULL AND revoked = 0 AND expires_at > ?`,
		userID, formatTime(now), id, formatTime(now))
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (d *DB) RevokeInvite(ctx context.Context, id int64) error {
	_, err := d.ExecContext(ctx, `UPDATE admin_invites SET revoked = 1 WHERE id = ?`, id)
	return err
}
