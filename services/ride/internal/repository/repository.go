package repository

import (
	"context"

	"github.com/jetkzu/jetkzu/services/ride/internal/domain"
)

type RideRepository interface {
	CreateWithHistory(ctx context.Context, r *domain.Ride) error
	GetByID(ctx context.Context, id string) (*domain.Ride, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Ride, error)
	ListByDriver(ctx context.Context, driverID string, limit, offset int) ([]*domain.Ride, error)
	ListActive(ctx context.Context, limit, offset int) ([]*domain.Ride, error)
	ListHistory(ctx context.Context, rideID string, limit int) ([]domain.StatusHistory, error)
	UpdateStatusWithHistory(ctx context.Context, id, newStatus, reason string) (*domain.Ride, error)
	AssignDriver(ctx context.Context, rideID, driverID string) (*domain.Ride, error)
	SavePriceEstimation(ctx context.Context, rideID string, price, distance float64) error
}
