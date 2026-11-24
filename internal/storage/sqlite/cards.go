package sqlite

import (
	"context"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func (d *DB) SaveCard(ctx context.Context, ticketID, adminID int64, messageID int) error {
	_, err := d.ExecContext(ctx, `INSERT INTO ticket_cards (ticket_id, admin_id, message_id)
		VALUES (?, ?, ?)
		ON CONFLICT(ticket_id, admin_id) DO UPDATE SET message_id = excluded.message_id`,
		ticketID, adminID, messageID)
	return err
}

func (d *DB) ListCards(ctx context.Context, ticketID int64) ([]domain.TicketCard, error) {
	rows, err := d.QueryContext(ctx, `SELECT admin_id, message_id FROM ticket_cards WHERE ticket_id = ?`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.TicketCard
	for rows.Next() {
		var c domain.TicketCard
		if err := rows.Scan(&c.AdminID, &c.MessageID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) DeleteCard(ctx context.Context, ticketID, adminID int64) error {
	_, err := d.ExecContext(ctx, `DELETE FROM ticket_cards WHERE ticket_id = ? AND admin_id = ?`, ticketID, adminID)
	return err
}

func (d *DB) DeleteCardsForAdmin(ctx context.Context, adminID int64) error {
	_, err := d.ExecContext(ctx, `DELETE FROM ticket_cards WHERE admin_id = ?`, adminID)
	return err
}
