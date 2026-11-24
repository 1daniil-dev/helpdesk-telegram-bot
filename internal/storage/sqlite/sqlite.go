package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

const timeLayout = time.RFC3339

type DB struct {
	*sql.DB
}

var migrations = [][]string{
	{
		`CREATE TABLE IF NOT EXISTS users (
			telegram_id INTEGER PRIMARY KEY,
			full_name   TEXT NOT NULL,
			username    TEXT NOT NULL DEFAULT '',
			role        TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
			created_at  TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS admin_invites (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			code       TEXT NOT NULL UNIQUE,
			created_by INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			used_by    INTEGER,
			used_at    TEXT,
			revoked    INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL UNIQUE,
			sort_order INTEGER NOT NULL DEFAULT 0,
			is_active  INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS tickets (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id       INTEGER NOT NULL,
			category_id   INTEGER NOT NULL,
			category_name TEXT NOT NULL,
			comment       TEXT NOT NULL DEFAULT '',
			status        TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'in_progress', 'done')),
			assignee_id   INTEGER,
			created_at    TEXT NOT NULL,
			taken_at      TEXT,
			closed_at     TEXT,
			FOREIGN KEY (user_id) REFERENCES users (telegram_id),
			FOREIGN KEY (category_id) REFERENCES categories (id),
			FOREIGN KEY (assignee_id) REFERENCES users (telegram_id)
		)`,
		`CREATE TABLE IF NOT EXISTS ticket_cards (
			ticket_id  INTEGER NOT NULL,
			admin_id   INTEGER NOT NULL,
			message_id INTEGER NOT NULL,
			PRIMARY KEY (ticket_id, admin_id),
			FOREIGN KEY (ticket_id) REFERENCES tickets (id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_user ON tickets (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets (status)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_assignee ON tickets (assignee_id)`,
		`CREATE INDEX IF NOT EXISTS idx_invites_code ON admin_invites (code)`,
	},
}

var defaultCategories = []string{
	"Компьютер / ноутбук",
	"Принтер / сканер",
	"Интернет / сеть",
	"Почта",
	"Программы / ПО",
	"Доступы и пароли",
	"Другое",
}

func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("создать папку для базы: %w", err)
		}
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
		filepath.ToSlash(path),
	)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("открыть базу: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("подключиться к базе: %w", err)
	}

	d := &DB{sqlDB}
	if err := d.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	if err := d.seedCategories(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("заполнить категории: %w", err)
	}
	return d, nil
}

func (d *DB) migrate() error {
	var version int
	if err := d.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("прочитать версию схемы: %w", err)
	}
	if version > len(migrations) {
		return fmt.Errorf("база новее приложения: версия %d", version)
	}

	for i := version; i < len(migrations); i++ {
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		for _, stmt := range migrations[i] {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("миграция %d: %w", i+1, err)
			}
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("записать версию схемы: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) seedCategories() error {
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, name := range defaultCategories {
		if _, err := tx.Exec(`INSERT INTO categories (name, sort_order) VALUES (?, ?)`, name, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

func parseTime(s string) time.Time {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
