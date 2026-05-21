package repository

import (
	"context"

	"github.com/jetkzu/jetkzu/services/driver/internal/domain"
)

type DriverRepository interface {
	Create(ctx context.Context, d *domain.Driver) error
	GetByID(ctx context.Context, id string) (*domain.Driver, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateLocation(ctx context.Context, id string, lat, lng float64) error
	RecordAssignment(ctx context.Context, driverID, rideID string) error
	List(ctx context.Context, status string, limit, offset int) ([]*domain.Driver, error)
	ListStatusHistory(ctx context.Context, driverID string, limit int) ([]domain.StatusHistory, error)
	AddVehicle(ctx context.Context, v *domain.Vehicle) error
	UpdateVehicle(ctx context.Context, v *domain.Vehicle) (*domain.Vehicle, error)
	DeleteVehicle(ctx context.Context, vehicleID string) error
	GetVehicle(ctx context.Context, driverID string) (*domain.Vehicle, error)
}

type LocationCache interface {
	SetLocation(ctx context.Context, driverID string, lat, lng float64) error
	SetStatus(ctx context.Context, driverID, status string) error
	GetStatus(ctx context.Context, driverID string) (string, error)
	FindNearest(ctx context.Context, lat, lng, radiusKm float64, limit int) ([]domain.NearbyDriver, error)
}
