package domain

import "strings"

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID       int64
	FullName string
	Username string
	Role     Role
}

func (u User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u User) DisplayName() string {
	name := strings.TrimSpace(u.FullName)
	if name == "" {
		return "Без имени"
	}
	return name
}

type Profile struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
}

func (p Profile) FullName() string {
	name := strings.TrimSpace(p.FirstName + " " + p.LastName)
	if name == "" {
		name = p.Username
	}
	return name
}
