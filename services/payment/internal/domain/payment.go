package domain

import (
	"errors"
	"time"
)

const (
	StatusPending   = "pending"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusRefunded  = "refunded"
)

var (
	ErrPaymentNotFound   = errors.New("payment not found")
	ErrInvalidTransition = errors.New("invalid payment status transition")
)

type Payment struct {
	ID        string
	RideID    string
	UserID    string
	Amount    float64
	Currency  string
	Status    string
	Method    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func CanTransition(from, to string) bool {
	switch from {
	case StatusPending:
		return to == StatusSucceeded || to == StatusFailed || to == StatusRefunded
	case StatusSucceeded:
		return to == StatusRefunded
	}
	return false
}
