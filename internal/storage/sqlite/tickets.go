package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

const ticketSelect = `SELECT t.id, t.user_id, u.full_name, u.username, t.category_id, t.category_name,
		COALESCE(t.comment, ''), t.status, t.assignee_id, COALESCE(a.full_name, ''), COALESCE(a.username, ''),
		t.created_at, t.taken_at, t.closed_at
	FROM tickets t
	JOIN users u ON u.telegram_id = t.user_id
	LEFT JOIN users a ON a.telegram_id = t.assignee_id`

func scanTicket(row interface{ Scan(dest ...any) error }) (domain.Ticket, error) {
	var (
		t          domain.Ticket
		status     string
		createdAt  string
		assigneeID sql.NullInt64
		takenAt    sql.NullString
		closedAt   sql.NullString
	)
	if err := row.Scan(&t.ID, &t.UserID, &t.UserName, &t.UserUsername, &t.CategoryID, &t.CategoryName,
		&t.Comment, &status, &assigneeID, &t.AssigneeName, &t.AssigneeUsername,
		&createdAt, &takenAt, &closedAt); err != nil {
		return domain.Ticket{}, notFound(err)
	}

	t.Status = domain.TicketStatus(status)
	t.CreatedAt = parseTime(createdAt)
	if assigneeID.Valid {
		v := assigneeID.Int64
		t.AssigneeID = &v
	}
	if takenAt.Valid {
		v := parseTime(takenAt.String)
		t.TakenAt = &v
	}
	if closedAt.Valid {
		v := parseTime(closedAt.String)
		t.ClosedAt = &v
	}
	return t, nil
}

func collectTickets(rows *sql.Rows) ([]domain.Ticket, error) {
	defer rows.Close()

	var out []domain.Ticket
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (d *DB) CreateTicket(ctx context.Context, userID, categoryID int64, categoryName, comment string, now time.Time) (domain.Ticket, error) {
	res, err := d.ExecContext(ctx, `INSERT INTO tickets (user_id, category_id, category_name, comment, status, created_at)
		VALUES (?, ?, ?, ?, 'new', ?)`,
		userID, categoryID, categoryName, comment, formatTime(now))
	if err != nil {
		return domain.Ticket{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Ticket{}, err
	}
	return d.GetTicket(ctx, id)
}

func (d *DB) GetTicket(ctx context.Context, id int64) (domain.Ticket, error) {
	return scanTicket(d.QueryRowContext(ctx, ticketSelect+` WHERE t.id = ?`, id))
}

func (d *DB) TakeTicket(ctx context.Context, id, adminID int64, now time.Time) (bool, error) {
	res, err := d.ExecContext(ctx, `UPDATE tickets SET status = 'in_progress', assignee_id = ?, taken_at = ?
		WHERE id = ? AND status = 'new'`,
		adminID, formatTime(now), id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (d *DB) CloseTicket(ctx context.Context, id, actorID int64, isOwner bool, now time.Time) (bool, error) {
	var (
		res sql.Result
		err error
	)
	if isOwner {
		res, err = d.ExecContext(ctx, `UPDATE tickets SET status = 'done', closed_at = ?,
				assignee_id = COALESCE(assignee_id, ?),
				taken_at = CASE WHEN taken_at IS NULL THEN ? ELSE taken_at END
			WHERE id = ? AND status IN ('new', 'in_progress')`,
			formatTime(now), actorID, formatTime(now), id)
	} else {
		res, err = d.ExecContext(ctx, `UPDATE tickets SET status = 'done', closed_at = ?
			WHERE id = ? AND status = 'in_progress' AND assignee_id = ?`,
			formatTime(now), id, actorID)
	}
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (d *DB) ListUserTickets(ctx context.Context, userID int64, limit int) ([]domain.Ticket, error) {
	rows, err := d.QueryContext(ctx, ticketSelect+` WHERE t.user_id = ? ORDER BY t.id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	return collectTickets(rows)
}

func (d *DB) ListOpenTickets(ctx context.Context) ([]domain.Ticket, error) {
	rows, err := d.QueryContext(ctx, ticketSelect+` WHERE t.status IN ('new', 'in_progress')
		ORDER BY CASE t.status WHEN 'new' THEN 0 ELSE 1 END, t.id`)
	if err != nil {
		return nil, err
	}
	return collectTickets(rows)
}

func (d *DB) ReleaseTicketsOfAdmin(ctx context.Context, adminID int64) ([]int64, error) {
	rows, err := d.QueryContext(ctx, `SELECT id FROM tickets WHERE assignee_id = ? AND status = 'in_progress'`, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if _, err := d.ExecContext(ctx, `UPDATE tickets SET status = 'new', assignee_id = NULL, taken_at = NULL
		WHERE assignee_id = ? AND status = 'in_progress'`, adminID); err != nil {
		return ids, err
	}
	return ids, nil
}
