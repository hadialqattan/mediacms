package domain

import "time"

type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleEditor UserRole = "editor"
)

func IsValidUserRole(role string) bool {
	switch role {
	case string(UserRoleAdmin), string(UserRoleEditor):
		return true
	default:
		return false
	}
}

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         UserRole
	CreatedAt    time.Time
}
