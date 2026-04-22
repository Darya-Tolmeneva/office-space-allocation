package domain

import "time"

// UserRole describes the authorization level of a user.
type UserRole string

const (
	UserRoleMember UserRole = "member"
	UserRoleAdmin  UserRole = "admin"
)

// UserStatus describes whether a user can access the system.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// User represents an authenticated actor in the system.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Role         UserRole
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
