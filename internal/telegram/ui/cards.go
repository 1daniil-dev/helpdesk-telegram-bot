package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func FormatTime(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("02.01.2006 15:04")
}

func Person(name, username string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "администратор"
	}
	if username != "" {
		return esc(name) + " (@" + esc(username) + ")"
	}
	return esc(name)
}

func UserLine(name, username string, id int64) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Без имени"
	}
	if username != "" {
		return fmt.Sprintf("%s (@%s)", esc(name), esc(username))
	}
	return fmt.Sprintf("%s (id %d)", esc(name), id)
}

func TicketCard(t domain.Ticket, loc *time.Location) string {
	var b strings.Builder

	fmt.Fprintf(&b, "<b>🆘 Заявка #%d</b>\n", t.ID)
	fmt.Fprintf(&b, "📂 Категория: <b>%s</b>\n", esc(t.CategoryName))
	fmt.Fprintf(&b, "👤 Автор: %s\n", UserLine(t.UserName, t.UserUsername, t.UserID))
	fmt.Fprintf(&b, "🕒 Создана: %s\n", FormatTime(t.CreatedAt, loc))

	if comment := strings.TrimSpace(t.Comment); comment != "" {
		fmt.Fprintf(&b, "\n💬 <b>Описание:</b>\n%s\n", esc(clipText(comment, 3000)))
	}

	b.WriteString("\n")
	switch t.Status {
	case domain.TicketInProgress:
		fmt.Fprintf(&b, "🔧 <b>В работе</b>: %s\n", Person(t.AssigneeName, t.AssigneeUsername))
		if t.TakenAt != nil {
			fmt.Fprintf(&b, "🕒 Взята: %s\n", FormatTime(*t.TakenAt, loc))
		}
	case domain.TicketDone:
		b.WriteString("✅ <b>Решена</b>\n")
		if t.AssigneeName != "" {
			fmt.Fprintf(&b, "👨‍💻 Исполнитель: %s\n", Person(t.AssigneeName, t.AssigneeUsername))
		}
		if t.ClosedAt != nil {
			fmt.Fprintf(&b, "🕒 Закрыта: %s\n", FormatTime(*t.ClosedAt, loc))
		}
	default:
		b.WriteString("🟡 <b>Новая</b> — ждёт администратора\n")
	}

	return b.String()
}

func clipText(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "…"
}
