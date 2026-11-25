package ui

import (
	"testing"

	"github.com/mymmrac/telego"

	"github.com/1daniil-dev/helpdesk-telegram-bot/internal/domain"
)

func callbacks(kb *telego.InlineKeyboardMarkup) []string {
	if kb == nil {
		return nil
	}
	var out []string
	for _, row := range kb.InlineKeyboard {
		for _, button := range row {
			out = append(out, button.CallbackData)
		}
	}
	return out
}

func has(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func TestReplyMenu(t *testing.T) {
	user := ReplyMenu(false)
	if len(user.Keyboard) != 1 {
		t.Fatalf("user rows = %d, want 1", len(user.Keyboard))
	}
	if user.Keyboard[0][0].Text != BtnNewTicket || user.Keyboard[0][1].Text != BtnMyTickets {
		t.Fatalf("user buttons = %+v", user.Keyboard[0])
	}
	if !user.IsPersistent || !user.ResizeKeyboard {
		t.Fatal("reply menu must be persistent and resized")
	}

	admin := ReplyMenu(true)
	if len(admin.Keyboard) != 2 {
		t.Fatalf("admin rows = %d, want 2", len(admin.Keyboard))
	}
	if admin.Keyboard[1][0].Text != BtnOpenTickets || admin.Keyboard[1][1].Text != BtnAdmin {
		t.Fatalf("admin buttons = %+v", admin.Keyboard[1])
	}
}

func TestCategoryPicker(t *testing.T) {
	kb := CategoryPicker([]domain.Category{
		{ID: 1, Name: "Принтер"},
		{ID: 2, Name: "Почта"},
		{ID: 3, Name: "Другое"},
	})
	cbs := callbacks(kb)
	for _, want := range []string{"tcat:1", "tcat:2", "tcat:3", "menu:main"} {
		if !has(cbs, want) {
			t.Fatalf("callback %q missing in %v", want, cbs)
		}
	}
	if len(kb.InlineKeyboard) != 3 {
		t.Fatalf("rows = %d, want 3", len(kb.InlineKeyboard))
	}
}

func TestTicketActions(t *testing.T) {
	assignee := int64(7)

	newTicket := domain.Ticket{ID: 5, Status: domain.TicketNew}
	cbs := callbacks(TicketActions(newTicket, 7, false))
	if !has(cbs, "take:5") || !has(cbs, "done:5") {
		t.Fatalf("new ticket callbacks = %v", cbs)
	}

	inProgress := domain.Ticket{ID: 5, Status: domain.TicketInProgress, AssigneeID: &assignee}
	cbs = callbacks(TicketActions(inProgress, 7, false))
	if !has(cbs, "done:5") || has(cbs, "take:5") {
		t.Fatalf("assignee callbacks = %v", cbs)
	}
	if kb := TicketActions(inProgress, 8, false); kb != nil {
		t.Fatalf("other admin got actions: %+v", kb)
	}
	if !has(callbacks(TicketActions(inProgress, 8, true)), "done:5") {
		t.Fatal("owner must be able to close in-progress ticket")
	}

	done := domain.Ticket{ID: 5, Status: domain.TicketDone}
	if kb := TicketActions(done, 7, true); kb != nil {
		t.Fatalf("done ticket got actions: %+v", kb)
	}
}

func TestMenus(t *testing.T) {
	if cbs := callbacks(MyTicketsMenu()); !has(cbs, "tmy") || !has(cbs, "menu:main") {
		t.Fatalf("my tickets menu = %v", cbs)
	}
	if cbs := callbacks(OpenTicketsMenu()); !has(cbs, "topen") || !has(cbs, "adm:menu") {
		t.Fatalf("open tickets menu = %v", cbs)
	}

	ownerMenu := callbacks(AdminMenu(true))
	for _, want := range []string{"adm:list", "cats:menu", "topen", "menu:main"} {
		if !has(ownerMenu, want) {
			t.Fatalf("owner admin menu missing %q: %v", want, ownerMenu)
		}
	}
	adminMenu := callbacks(AdminMenu(false))
	if has(adminMenu, "adm:list") || has(adminMenu, "cats:menu") {
		t.Fatalf("non-owner sees owner buttons: %v", adminMenu)
	}

	cbs := callbacks(AdminsMenu(
		[]domain.User{
			{ID: 9, FullName: "Владелец", Role: domain.RoleAdmin},
			{ID: 2, FullName: "Админ", Role: domain.RoleAdmin},
		},
		[]domain.Invite{{ID: 4, Code: "HD7K2M9Q"}},
		map[int64]struct{}{9: {}},
	))
	for _, want := range []string{"adm:del:2", "inv:revoke:4", "adm:add", "adm:menu"} {
		if !has(cbs, want) {
			t.Fatalf("admins menu missing %q: %v", want, cbs)
		}
	}
	if has(cbs, "adm:del:9") {
		t.Fatal("owner must not have remove button")
	}
}
