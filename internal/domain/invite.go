package domain

import "time"

type Invite struct {
	ID        int64
	Code      string
	CreatedBy int64
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedBy    *int64
	UsedAt    *time.Time
	Revoked   bool
}

func (i Invite) IsActive(now time.Time) bool {
	return !i.Revoked && i.UsedBy == nil && now.Before(i.ExpiresAt)
}
