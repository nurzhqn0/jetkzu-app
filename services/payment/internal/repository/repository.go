package repository

import (
	"context"

	"github.com/jetkzu/jetkzu/services/payment/internal/domain"
)

type PaymentRepository interface {
	CreateWithEvent(ctx context.Context, p *domain.Payment, event string) error
	GetByID(ctx context.Context, id string) (*domain.Payment, error)
	GetByRide(ctx context.Context, rideID string) (*domain.Payment, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error)
	ListByStatus(ctx context.Context, status string, limit, offset int) ([]*domain.Payment, error)
	UpdateStatusWithEvent(ctx context.Context, id, newStatus, event, reason string) (*domain.Payment, error)
}
