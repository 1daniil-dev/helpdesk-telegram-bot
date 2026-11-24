package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken string
	OwnerIDs map[int64]struct{}
	Location *time.Location
	DBPath   string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	if strings.TrimSpace(os.Getenv("BOT_TOKEN")) == "" {
		if exe, err := os.Executable(); err == nil {
			_ = godotenv.Load(filepath.Join(filepath.Dir(exe), ".env"))
		}
	}

	token := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	if token == "" {
		return nil, errors.New("BOT_TOKEN не задан: заполни файл .env")
	}

	owners := make(map[int64]struct{})
	parts := strings.FieldsFunc(os.Getenv("OWNER_IDS"), func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	for _, part := range parts {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("OWNER_IDS: %q не является Telegram ID", part)
		}
		owners[id] = struct{}{}
	}
	if len(owners) == 0 {
		return nil, errors.New("OWNER_IDS пуст: укажи свой Telegram ID")
	}

	tz := strings.TrimSpace(os.Getenv("TIMEZONE"))
	if tz == "" {
		tz = "Europe/Moscow"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("TIMEZONE %q: %w", tz, err)
	}

	dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
	if dbPath == "" {
		dbPath = filepath.Join("data", "helpdesk.db")
	}

	return &Config{
		BotToken: token,
		OwnerIDs: owners,
		Location: loc,
		DBPath:   dbPath,
	}, nil
}

func (c *Config) IsOwner(id int64) bool {
	_, ok := c.OwnerIDs[id]
	return ok
}
