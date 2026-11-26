package ui

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func esc(s string) string {
	return html.EscapeString(s)
}

func StartText(u domain.User, isAdmin bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "👋 Привет, <b>%s</b>!\n\n", esc(u.FullName))
	b.WriteString("Это «Техподдержка» — бот заявок в IT-отдел.\n")
	b.WriteString("Если что-то не работает — нажми «🆘 Нужна помощь», выбери категорию и опиши проблему. Администратор получит заявку и придёт на помощь.\n\n")
	b.WriteString("📋 «Мои заявки» — статусы твоих обращений.\n\n")
	b.WriteString("Кнопки меню — под строкой ввода 👇")
	if isAdmin {
		b.WriteString("\n\n🛠 Ты администратор: новые заявки приходят сюда, в личку.")
	}
	return b.String()
}

func MainMenuText() string {
	return "🏠 <b>Главное меню</b>\n\nКнопки — под строкой ввода 👇"
}

func HelpText(isAdmin bool) string {
	var b strings.Builder
	b.WriteString("ℹ️ <b>Как это работает</b>\n\n")
	b.WriteString("1. «🆘 Нужна помощь» → выбери категорию.\n")
	b.WriteString("2. Опиши проблему текстом (или нажми «Пропустить»).\n")
	b.WriteString("3. Администратор получит заявку в личку и возьмёт её в работу.\n")
	b.WriteString("4. Когда проблему решат, заявка станет «Решена» — тебе придёт уведомление.\n\n")
	b.WriteString("📋 «Мои заявки» — список твоих обращений со статусами.")
	if isAdmin {
		b.WriteString("\n\n🛠 «Открытые заявки» — все новые и взятые в работу.\n")
		b.WriteString("⚙️ «Админка» — управление. Владельцу доступны админы и категории.")
	}
	b.WriteString("\n\nКнопки меню находятся под строкой ввода 👇")
	return b.String()
}

func AdminWelcomeText(u domain.User) string {
	return fmt.Sprintf("🎉 <b>Готово!</b> Теперь ты администратор, %s.\n\nНовые заявки будут приходить сюда, в личку. Бери их в работу кнопкой «🔧 Взять в работу» и закрывай кнопкой «✅ Готово».", esc(u.FullName))
}

func AdminDeniedText() string {
	return "⛔ Это раздел администраторов."
}

func OwnerDeniedText() string {
	return "⛔ Это действие доступно только владельцу бота."
}

func ChooseCategoryText() string {
	return "🆘 <b>Новая заявка</b>\n\nВыбери категорию проблемы:"
}

func EnterCommentText(category string) string {
	return fmt.Sprintf("📂 Категория: <b>%s</b>\n\n✏️ Опиши проблему одним сообщением — это поможет администратору быстрее разобраться.\n\nМожно нажать «Пропустить», если описание не нужно.", esc(category))
}

func TicketPreview(category, comment string) string {
	var b strings.Builder
	b.WriteString("🧾 <b>Проверь заявку</b>\n\n")
	fmt.Fprintf(&b, "📂 Категория: <b>%s</b>\n", esc(category))
	if strings.TrimSpace(comment) == "" {
		b.WriteString("💬 Описание: <i>без описания</i>\n")
	} else {
		fmt.Fprintf(&b, "💬 Описание:\n%s\n", esc(clipText(comment, 1000)))
	}
	b.WriteString("\nОтправить администраторам?")
	return b.String()
}

func TicketCreatedText(id int64) string {
	return fmt.Sprintf("✅ <b>Заявка #%d создана.</b>\n\nАдминистраторы получили уведомление. Когда её возьмут в работу или решат — я сообщу.", id)
}

func TicketTakenByUser(id int64, adminName string) string {
	return fmt.Sprintf("🔧 Твоя заявка <b>#%d</b> принята в работу.\n👨‍💻 Администратор: <b>%s</b>", id, esc(adminName))
}

func TicketDoneByUser(id int64) string {
	return fmt.Sprintf("✅ Твоя заявка <b>#%d</b> решена. Спасибо за обращение!", id)
}

func NoCategoriesText() string {
	return "🗂 Категории пока не настроены. Напиши администратору напрямую."
}

func SessionExpiredText() string {
	return "⌛ Время ввода истекло. Начни заново из меню."
}

func CancelText() string {
	return "❌ Действие отменено."
}

func UnknownText() string {
	return "🤔 Не понял сообщение. Вот меню:"
}

func TextOnlyText() string {
	return "Пока принимаю только текст: опиши проблему словами."
}

func ErrorText() string {
	return "⚠️ Что-то пошло не так. Попробуй ещё раз чуть позже."
}

func MyTicketsEmptyText() string {
	return "📋 У тебя пока нет заявок.\nЕсли нужна помощь — нажми «🆘 Нужна помощь»."
}

func MyTicketsText(tickets []domain.Ticket, loc *time.Location) string {
	var b strings.Builder
	b.WriteString("📋 <b>Мои заявки</b>\n\n")
	for _, t := range tickets {
		fmt.Fprintf(&b, "<b>#%d</b> %s — %s — %s\n",
			t.ID, statusLabel(t.Status), esc(t.CategoryName), FormatTime(t.CreatedAt, loc))
	}
	b.WriteString("\n🟡 новая · 🔧 в работе · ✅ решена")
	return b.String()
}

func OpenTicketsEmptyText() string {
	return "🎉 Открытых заявок нет."
}

func OpenTicketsText(tickets []domain.Ticket, loc *time.Location) string {
	var b strings.Builder
	b.WriteString("🛠 <b>Открытые заявки</b>\n\n")
	for _, t := range tickets {
		fmt.Fprintf(&b, "<b>#%d</b> %s — <b>%s</b>\n", t.ID, statusLabel(t.Status), esc(t.CategoryName))
		fmt.Fprintf(&b, "    👤 %s · %s", Person(t.UserName, t.UserUsername), FormatTime(t.CreatedAt, loc))
		if t.Status == domain.TicketInProgress && t.AssigneeName != "" {
			fmt.Fprintf(&b, " · 🔧 %s", Person(t.AssigneeName, t.AssigneeUsername))
		}
		b.WriteString("\n")
	}
	b.WriteString("\nКарточки с кнопками приходят в личку отдельными сообщениями.")
	return b.String()
}

func AdminMenuText(u domain.User, isOwner bool) string {
	if isOwner {
		return fmt.Sprintf("⚙️ <b>Админка</b>\n\n👤 Ты владелец: %s\n\nЗдесь управление администраторами и категориями, а также все открытые заявки.", esc(u.FullName))
	}
	return fmt.Sprintf("⚙️ <b>Админка</b>\n\n👤 %s\n\nОткрытые заявки — в разделе ниже. Управление админами и категориями доступно владельцу.", esc(u.FullName))
}

func AdminsListText(admins []domain.User, invites []domain.Invite, owners map[int64]struct{}, now time.Time) string {
	var b strings.Builder
	b.WriteString("👥 <b>Администраторы</b>\n\n")

	seen := make(map[int64]bool, len(admins))
	for _, a := range admins {
		seen[a.ID] = true
		suffix := ""
		if _, ok := owners[a.ID]; ok {
			suffix = " — владелец"
		}
		fmt.Fprintf(&b, "• %s%s\n", UserLine(a.FullName, a.Username, a.ID), suffix)
	}

	var missing []int64
	for id := range owners {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
	for _, id := range missing {
		fmt.Fprintf(&b, "• владелец ID <code>%d</code> (ещё не запускал бота)\n", id)
	}

	if len(invites) == 0 {
		b.WriteString("\nАктивных приглашений нет.")
	} else {
		b.WriteString("\n🕒 <b>Активные приглашения</b>\n")
		for _, inv := range invites {
			fmt.Fprintf(&b, "• <code>%s</code> — осталось %s\n", inv.Code, remaining(inv.ExpiresAt, now))
		}
	}

	b.WriteString("\n\n📌 Снять админа или отозвать код — кнопками ниже.")
	return b.String()
}

func InviteCreatedText(code, link string, expires time.Time, loc *time.Location) string {
	var b strings.Builder
	b.WriteString("🔑 <b>Приглашение администратора создано</b>\n\n")
	fmt.Fprintf(&b, "Код: <code>%s</code>\n", esc(code))
	fmt.Fprintf(&b, "Действует до: %s (24 часа)\n", FormatTime(expires, loc))
	if link != "" {
		fmt.Fprintf(&b, "\nСсылка:\n%s\n", esc(link))
	}
	b.WriteString("\nОтправь новому админу ссылку или сам код. Ему нужно открыть бота и вставить код в чат.")
	return b.String()
}

func RemoveAdminConfirmText(u domain.User) string {
	return fmt.Sprintf("⚠️ Снять права администратора с %s?\n\nЗаявки «в работе» вернутся в общий список, новые карточки перестанут приходить.", UserLine(u.FullName, u.Username, u.ID))
}

func CategoriesText(cats []domain.Category, note string) string {
	var b strings.Builder
	b.WriteString("🗂 <b>Категории заявок</b>\n\n")
	for _, c := range cats {
		state := ""
		if !c.Active {
			state = " <i>(скрыта)</i>"
		}
		fmt.Fprintf(&b, "• %s%s\n", esc(c.Name), state)
	}
	b.WriteString("\n✏️ — переименовать, 🙈/👁 — скрыть или показать.")
	if note != "" {
		b.WriteString("\n\n")
		b.WriteString(note)
	}
	return b.String()
}

func AddCategoryPrompt() string {
	return "➕ Отправь название новой категории (2–40 символов)."
}

func RenameCategoryPrompt(name string) string {
	return fmt.Sprintf("✏️ Отправь новое название для категории «%s» (2–40 символов).", esc(name))
}

func CategoryAddedNote(name string) string {
	return fmt.Sprintf("✅ Категория «%s» добавлена.", esc(name))
}

func CategoryRenamedNote(name string) string {
	return fmt.Sprintf("✅ Категория переименована в «%s».", esc(name))
}

func CategoryNameInvalidText() string {
	return "⚠️ Название должно быть от 2 до 40 символов. Отправь ещё раз."
}

func CategoryExistsText() string {
	return "⚠️ Такая категория уже есть. Отправь другое название."
}

func TakeConflictText(t domain.Ticket) string {
	switch {
	case t.IsDone():
		return fmt.Sprintf("Заявка #%d уже решена.", t.ID)
	case t.AssigneeName != "":
		return fmt.Sprintf("⚠️ Заявка #%d уже в работе у %s.", t.ID, Person(t.AssigneeName, t.AssigneeUsername))
	default:
		return fmt.Sprintf("⚠️ Не удалось взять заявку #%d. Обнови список.", t.ID)
	}
}

func CloseDeniedText(t domain.Ticket) string {
	if t.AssigneeName != "" {
		return fmt.Sprintf("Закрыть заявку #%d может только тот, кто её взял (%s), или владелец.", t.ID, Person(t.AssigneeName, t.AssigneeUsername))
	}
	return fmt.Sprintf("Закрыть заявку #%d может только владелец.", t.ID)
}

func AlreadyDoneText(id int64) string {
	return fmt.Sprintf("Заявка #%d уже решена.", id)
}

func remaining(exp, now time.Time) string {
	d := exp.Sub(now)
	if d <= 0 {
		return "истёк"
	}
	if d < time.Hour {
		m := int(math.Ceil(d.Minutes()))
		if m < 1 {
			m = 1
		}
		return fmt.Sprintf("%d мин", m)
	}
	return fmt.Sprintf("%d ч", int(math.Ceil(d.Hours())))
}

func statusLabel(status domain.TicketStatus) string {
	switch status {
	case domain.TicketNew:
		return "🟡 новая"
	case domain.TicketInProgress:
		return "🔧 в работе"
	case domain.TicketDone:
		return "✅ решена"
	default:
		return string(status)
	}
}
