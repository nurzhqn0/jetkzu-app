package domain

import (
	"errors"
	"time"
)

const (
	ChannelEmail = "email"

	StatusQueued   = "queued"
	StatusSent     = "sent"
	StatusMockSent = "mock_sent"
	StatusFailed   = "failed"
	StatusRead     = "read"
)

var ErrNotFound = errors.New("notification not found")

type Notification struct {
	ID        string
	UserID    string
	Channel   string
	To        string
	Subject   string
	Body      string
	Status    string
	CreatedAt time.Time
	SentAt    time.Time
}
