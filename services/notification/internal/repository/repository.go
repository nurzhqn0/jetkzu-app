package repository

import (
	"context"

	"github.com/jetkzu/jetkzu/services/notification/internal/domain"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *domain.Notification) error
	UpdateStatus(ctx context.Context, id, status string) (*domain.Notification, error)
	GetByID(ctx context.Context, id string) (*domain.Notification, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error)
	ListUnread(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error)
	CountUnread(ctx context.Context, userID string) (int, error)
	MarkAllRead(ctx context.Context, userID string) (int, error)
	Delete(ctx context.Context, id string) error
}
