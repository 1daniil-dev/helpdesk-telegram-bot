package ui

import (
	"strconv"

	"github.com/mymmrac/telego"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

const ParseModeHTML = "HTML"

const (
	BtnNewTicket   = "🆘 Нужна помощь"
	BtnMyTickets   = "📋 Мои заявки"
	BtnOpenTickets = "🛠 Открытые заявки"
	BtnAdmin       = "⚙️ Админка"
)

func ReplyMenu(isAdmin bool) *telego.ReplyKeyboardMarkup {
	rows := [][]telego.KeyboardButton{
		{{Text: BtnNewTicket}, {Text: BtnMyTickets}},
	}
	if isAdmin {
		rows = append(rows, []telego.KeyboardButton{{Text: BtnOpenTickets}, {Text: BtnAdmin}})
	}
	return &telego.ReplyKeyboardMarkup{
		Keyboard:       rows,
		IsPersistent:   true,
		ResizeKeyboard: true,
	}
}

func EmptyInline() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{}}
}

func CategoryPicker(cats []domain.Category) *telego.InlineKeyboardMarkup {
	var rows [][]telego.InlineKeyboardButton
	var row []telego.InlineKeyboardButton
	for _, c := range cats {
		row = append(row, telego.InlineKeyboardButton{
			Text:         clipText(c.Name, 30),
			CallbackData: "tcat:" + strconv.FormatInt(c.ID, 10),
		})
		if len(row) == 2 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, []telego.InlineKeyboardButton{{Text: "✖️ Закрыть", CallbackData: "menu:main"}})
	return &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func CommentPrompt() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
		{{Text: "⏭ Пропустить", CallbackData: "tskip"}},
		{{Text: "✖️ Отмена", CallbackData: "tcancel"}},
	}}
}

func ConfirmTicket() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
		{{Text: "✅ Отправить", CallbackData: "tsend"}},
		{{Text: "✖️ Отмена", CallbackData: "tcancel"}},
	}}
}

func TicketActions(t domain.Ticket, viewerID int64, isOwner bool) *telego.InlineKeyboardMarkup {
	id := strconv.FormatInt(t.ID, 10)

	switch t.Status {
	case domain.TicketNew:
		return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
			{{Text: "🔧 Взять в работу", CallbackData: "take:" + id}},
			{{Text: "✅ Готово", CallbackData: "done:" + id}},
		}}
	case domain.TicketInProgress:
		if isOwner || (t.AssigneeID != nil && *t.AssigneeID == viewerID) {
			return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
				{{Text: "✅ Готово", CallbackData: "done:" + id}},
			}}
		}
	}
	return nil
}

func MyTicketsMenu() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
		{{Text: "🔄 Обновить", CallbackData: "tmy"}},
		{{Text: "✖️ Закрыть", CallbackData: "menu:main"}},
	}}
}

func OpenTicketsMenu() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
		{{Text: "🔄 Обновить", CallbackData: "topen"}},
		{{Text: "⬅️ Назад", CallbackData: "adm:menu"}},
	}}
}

func AdminMenu(isOwner bool) *telego.InlineKeyboardMarkup {
	var rows [][]telego.InlineKeyboardButton
	if isOwner {
		rows = append(rows,
			[]telego.InlineKeyboardButton{{Text: "👥 Администраторы", CallbackData: "adm:list"}},
			[]telego.InlineKeyboardButton{{Text: "🗂 Категории", CallbackData: "cats:menu"}},
		)
	}
	rows = append(rows,
		[]telego.InlineKeyboardButton{{Text: "🛠 Открытые заявки", CallbackData: "topen"}},
		[]telego.InlineKeyboardButton{{Text: "✖️ Закрыть", CallbackData: "menu:main"}},
	)
	return &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func AdminsMenu(admins []domain.User, invites []domain.Invite, owners map[int64]struct{}) *telego.InlineKeyboardMarkup {
	var rows [][]telego.InlineKeyboardButton

	for _, a := range admins {
		if _, ok := owners[a.ID]; ok {
			continue
		}
		rows = append(rows, []telego.InlineKeyboardButton{{
			Text:         clipText("❌ Снять: "+a.FullName, 60),
			CallbackData: "adm:del:" + strconv.FormatInt(a.ID, 10),
		}})
	}
	for _, inv := range invites {
		rows = append(rows, []telego.InlineKeyboardButton{{
			Text:         "🗑 Отозвать код " + inv.Code,
			CallbackData: "inv:revoke:" + strconv.FormatInt(inv.ID, 10),
		}})
	}

	rows = append(rows,
		[]telego.InlineKeyboardButton{{Text: "➕ Добавить администратора", CallbackData: "adm:add"}},
		[]telego.InlineKeyboardButton{{Text: "⬅️ Назад", CallbackData: "adm:menu"}},
	)
	return &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func RemoveAdminConfirm(id int64) *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
		{{Text: "✅ Да, снять", CallbackData: "adm:delok:" + strconv.FormatInt(id, 10)}},
		{{Text: "⬅️ Отмена", CallbackData: "adm:list"}},
	}}
}

func CategoriesMenu(cats []domain.Category) *telego.InlineKeyboardMarkup {
	var rows [][]telego.InlineKeyboardButton
	for _, c := range cats {
		label := "🙈 Скрыть"
		if !c.Active {
			label = "👁 Показать"
		}
		rows = append(rows, []telego.InlineKeyboardButton{
			{Text: clipText("✏️ "+c.Name, 40), CallbackData: "cats:ren:" + strconv.FormatInt(c.ID, 10)},
			{Text: label, CallbackData: "cats:tgl:" + strconv.FormatInt(c.ID, 10)},
		})
	}
	rows = append(rows,
		[]telego.InlineKeyboardButton{{Text: "➕ Добавить категорию", CallbackData: "cats:add"}},
		[]telego.InlineKeyboardButton{{Text: "⬅️ Назад", CallbackData: "adm:menu"}},
	)
	return &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func BackToMain() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
		{{Text: "✖️ Закрыть", CallbackData: "menu:main"}},
	}}
}

func BackToAdmins() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
		{{Text: "⬅️ Назад", CallbackData: "adm:list"}},
	}}
}

func CancelToCategories() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{
		{{Text: "✖️ Отмена", CallbackData: "cats:menu"}},
	}}
}
