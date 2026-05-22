package domain

import (
	"errors"
	"time"
)

const (
	RolePassenger = "passenger"
	RoleDriver    = "driver"
	RoleAdmin     = "admin"
	RoleInactive  = "inactive"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyTaken = errors.New("email already taken")
	ErrInvalidCredential = errors.New("invalid email or password")
	ErrInvalidToken      = errors.New("invalid verification token")
)

type User struct {
	ID            string
	Email         string
	PasswordHash  string
	FullName      string
	Phone         string
	Role          string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type VerificationToken struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}
