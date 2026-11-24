package session

import (
	"sync"
	"time"
)

type State string

const (
	StateTicketComment  State = "ticket_comment"
	StateTicketConfirm  State = "ticket_confirm"
	StateCategoryAdd    State = "category_add"
	StateCategoryRename State = "category_rename"
)

type Data struct {
	State        State
	CategoryID   int64
	CategoryName string
	Comment      string
	UpdatedAt    time.Time
}

type Manager struct {
	mu   sync.Mutex
	ttl  time.Duration
	data map[int64]Data
}

func New(ttl time.Duration) *Manager {
	return &Manager{
		ttl:  ttl,
		data: make(map[int64]Data),
	}
}

func (m *Manager) Get(userID int64) (Data, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	d, ok := m.data[userID]
	if !ok {
		return Data{}, false
	}
	if time.Since(d.UpdatedAt) > m.ttl {
		delete(m.data, userID)
		return Data{}, false
	}
	return d, true
}

func (m *Manager) Set(userID int64, d Data) {
	d.UpdatedAt = time.Now()
	m.mu.Lock()
	m.data[userID] = d
	m.mu.Unlock()
}

func (m *Manager) Clear(userID int64) {
	m.mu.Lock()
	delete(m.data, userID)
	m.mu.Unlock()
}
