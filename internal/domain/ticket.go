package domain

import "time"

type TicketStatus string

const (
	TicketNew        TicketStatus = "new"
	TicketInProgress TicketStatus = "in_progress"
	TicketDone       TicketStatus = "done"
)

type Ticket struct {
	ID               int64
	UserID           int64
	UserName         string
	UserUsername     string
	CategoryID       int64
	CategoryName     string
	Comment          string
	Status           TicketStatus
	AssigneeID       *int64
	AssigneeName     string
	AssigneeUsername string
	CreatedAt        time.Time
	TakenAt          *time.Time
	ClosedAt         *time.Time
}

func (t Ticket) IsOpen() bool {
	return t.Status == TicketNew || t.Status == TicketInProgress
}

func (t Ticket) IsDone() bool {
	return t.Status == TicketDone
}

type TicketCard struct {
	AdminID   int64
	MessageID int
}
