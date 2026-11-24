package domain

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation")
	ErrForbidden    = errors.New("forbidden")
	ErrAlreadyTaken = errors.New("ticket already taken")
	ErrTicketDone   = errors.New("ticket already done")
	ErrCloseDenied  = errors.New("close denied")

	ErrInviteNotFound = errors.New("invite not found")
	ErrInviteUsed     = errors.New("invite used")
	ErrInviteExpired  = errors.New("invite expired")
	ErrInviteRevoked  = errors.New("invite revoked")
	ErrAlreadyAdmin   = errors.New("already admin")
)
