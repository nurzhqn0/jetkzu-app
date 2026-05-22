package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jetkzu/jetkzu/pkg/natsbus"
	"github.com/jetkzu/jetkzu/services/ride/internal/domain"
	"github.com/jetkzu/jetkzu/services/ride/internal/repository"
)

type Publisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

type UseCase struct {
	repo repository.RideRepository
	bus  Publisher
}

func New(repo repository.RideRepository, bus Publisher) *UseCase {
	return &UseCase{repo: repo, bus: bus}
}

type CreateRideInput struct {
	PassengerID string
	PickupLat   float64
	PickupLng   float64
	DropoffLat  float64
	DropoffLng  float64
}

func (uc *UseCase) Create(ctx context.Context, in CreateRideInput) (*domain.Ride, error) {
	existing, err := uc.repo.ListByUser(ctx, in.PassengerID, 100, 0)
	if err != nil {
		return nil, err
	}
	for _, ride := range existing {
		if domain.IsActiveStatus(ride.Status) {
			return nil, domain.ErrActiveRideExists
		}
	}

	price, dist := domain.EstimatePrice(in.PickupLat, in.PickupLng, in.DropoffLat, in.DropoffLng)
	ride := &domain.Ride{
		ID:          uuid.NewString(),
		PassengerID: in.PassengerID,
		PickupLat:   in.PickupLat,
		PickupLng:   in.PickupLng,
		DropoffLat:  in.DropoffLat,
		DropoffLng:  in.DropoffLng,
		Status:      domain.StatusRequested,
		Price:       price,
		CreatedAt:   time.Now().UTC(),
	}
	if err := uc.repo.CreateWithHistory(ctx, ride); err != nil {
		return nil, err
	}
	_ = uc.repo.SavePriceEstimation(ctx, ride.ID, price, dist)
	_ = uc.bus.Publish(ctx, natsbus.SubjectRideRequested, map[string]any{
		"ride_id":      ride.ID,
		"passenger_id": ride.PassengerID,
		"pickup_lat":   ride.PickupLat,
		"pickup_lng":   ride.PickupLng,
		"dropoff_lat":  ride.DropoffLat,
		"dropoff_lng":  ride.DropoffLng,
		"price":        ride.Price,
	})
	return ride, nil
}

func (uc *UseCase) Schedule(ctx context.Context, in CreateRideInput, scheduledAt time.Time) (*domain.Ride, error) {
	ride, err := uc.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	_ = uc.bus.Publish(ctx, "ride.scheduled", map[string]any{
		"ride_id": ride.ID, "passenger_id": ride.PassengerID, "scheduled_at": scheduledAt,
	})
	return ride, nil
}

func (uc *UseCase) EstimatePrice(_ context.Context, in CreateRideInput) (float64, float64) {
	return domain.EstimatePrice(in.PickupLat, in.PickupLng, in.DropoffLat, in.DropoffLng)
}

func (uc *UseCase) Get(ctx context.Context, id string) (*domain.Ride, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *UseCase) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Ride, error) {
	return uc.repo.ListByUser(ctx, userID, limit, offset)
}

func (uc *UseCase) ListActive(ctx context.Context, limit, offset int) ([]*domain.Ride, error) {
	return uc.repo.ListActive(ctx, limit, offset)
}

func (uc *UseCase) ListByDriver(ctx context.Context, driverID string, limit, offset int) ([]*domain.Ride, error) {
	return uc.repo.ListByDriver(ctx, driverID, limit, offset)
}

func (uc *UseCase) History(ctx context.Context, rideID string, limit int) ([]domain.StatusHistory, error) {
	return uc.repo.ListHistory(ctx, rideID, limit)
}

func (uc *UseCase) UpdateStatus(ctx context.Context, id, statusVal, reason string) (*domain.Ride, error) {
	ride, err := uc.repo.UpdateStatusWithHistory(ctx, id, statusVal, reason)
	if err != nil {
		return nil, err
	}
	_ = uc.bus.Publish(ctx, natsbus.SubjectRideStatusChanged, map[string]any{
		"ride_id": ride.ID, "status": ride.Status, "passenger_id": ride.PassengerID, "driver_id": ride.DriverID,
	})
	if statusVal == domain.StatusCompleted {
		_ = uc.bus.Publish(ctx, natsbus.SubjectRideCompleted, map[string]any{
			"ride_id": ride.ID, "passenger_id": ride.PassengerID, "driver_id": ride.DriverID, "price": ride.Price,
		})
	}
	if statusVal == domain.StatusCancelled {
		_ = uc.bus.Publish(ctx, natsbus.SubjectRideCancelled, map[string]any{
			"ride_id": ride.ID, "passenger_id": ride.PassengerID, "reason": reason, "price": ride.Price,
		})
	}
	return ride, nil
}

func (uc *UseCase) Cancel(ctx context.Context, id, reason string) (*domain.Ride, error) {
	return uc.UpdateStatus(ctx, id, domain.StatusCancelled, reason)
}

func (uc *UseCase) Complete(ctx context.Context, id string) (*domain.Ride, error) {
	return uc.UpdateStatus(ctx, id, domain.StatusCompleted, "completed by driver")
}

func (uc *UseCase) Accept(ctx context.Context, rideID, driverID string) (*domain.Ride, error) {
	return uc.AssignDriver(ctx, rideID, driverID)
}

func (uc *UseCase) Reject(ctx context.Context, rideID, driverID, reason string) (bool, error) {
	if _, err := uc.repo.GetByID(ctx, rideID); err != nil {
		return false, err
	}
	_ = uc.bus.Publish(ctx, "ride.rejected", map[string]any{
		"ride_id": rideID, "driver_id": driverID, "reason": reason,
	})
	return true, nil
}

func (uc *UseCase) Rate(ctx context.Context, rideID, userID string, rating int32, comment string) (int32, error) {
	if _, err := uc.repo.GetByID(ctx, rideID); err != nil {
		return 0, err
	}
	if rating < 1 {
		rating = 1
	}
	if rating > 5 {
		rating = 5
	}
	_ = uc.bus.Publish(ctx, "ride.rated", map[string]any{
		"ride_id": rideID, "user_id": userID, "rating": rating, "comment": comment,
	})
	return rating, nil
}

// AssignDriver is called via NATS when driver service picked someone.
func (uc *UseCase) AssignDriver(ctx context.Context, rideID, driverID string) (*domain.Ride, error) {
	ride, err := uc.repo.AssignDriver(ctx, rideID, driverID)
	if err != nil {
		return nil, err
	}
	_ = uc.bus.Publish(ctx, natsbus.SubjectRideStatusChanged, map[string]any{
		"ride_id": ride.ID, "status": ride.Status, "passenger_id": ride.PassengerID, "driver_id": ride.DriverID,
	})
	return ride, nil
}
