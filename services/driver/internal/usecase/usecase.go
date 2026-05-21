package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jetkzu/jetkzu/pkg/validator"
	"github.com/jetkzu/jetkzu/services/driver/internal/domain"
	"github.com/jetkzu/jetkzu/services/driver/internal/repository"
)

type Publisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

type UseCase struct {
	repo  repository.DriverRepository
	cache repository.LocationCache
	bus   Publisher
}

func New(repo repository.DriverRepository, cache repository.LocationCache, bus Publisher) *UseCase {
	return &UseCase{repo: repo, cache: cache, bus: bus}
}

func (uc *UseCase) Register(ctx context.Context, userID, license string) (*domain.Driver, error) {
	if err := validator.NotEmpty("user_id", userID); err != nil {
		return nil, err
	}
	if err := validator.NotEmpty("license_number", license); err != nil {
		return nil, err
	}
	d := &domain.Driver{
		ID:            uuid.NewString(),
		UserID:        userID,
		LicenseNumber: license,
		Status:        domain.StatusOffline,
		CreatedAt:     time.Now().UTC(),
	}
	if err := uc.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	_ = uc.cache.SetStatus(ctx, d.ID, d.Status)
	return d, nil
}

func (uc *UseCase) AddVehicle(ctx context.Context, v *domain.Vehicle) (*domain.Vehicle, error) {
	if err := validator.NotEmpty("plate_number", v.PlateNumber); err != nil {
		return nil, err
	}
	v.ID = uuid.NewString()
	if err := uc.repo.AddVehicle(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (uc *UseCase) UpdateVehicle(ctx context.Context, v *domain.Vehicle) (*domain.Vehicle, error) {
	if err := validator.NotEmpty("vehicle_id", v.ID); err != nil {
		return nil, err
	}
	if err := validator.NotEmpty("plate_number", v.PlateNumber); err != nil {
		return nil, err
	}
	return uc.repo.UpdateVehicle(ctx, v)
}

func (uc *UseCase) DeleteVehicle(ctx context.Context, vehicleID string) (bool, error) {
	if err := uc.repo.DeleteVehicle(ctx, vehicleID); err != nil {
		return false, err
	}
	return true, nil
}

func (uc *UseCase) UpdateStatus(ctx context.Context, id, status string) (*domain.Driver, error) {
	if err := validator.OneOf("status", status, domain.StatusOnline, domain.StatusOffline, domain.StatusBusy); err != nil {
		return nil, domain.ErrInvalidStatus
	}
	if err := uc.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}
	_ = uc.cache.SetStatus(ctx, id, status)
	return uc.repo.GetByID(ctx, id)
}

func (uc *UseCase) UpdateLocation(ctx context.Context, id string, lat, lng float64) error {
	if err := uc.repo.UpdateLocation(ctx, id, lat, lng); err != nil {
		return err
	}
	if err := uc.cache.SetLocation(ctx, id, lat, lng); err != nil {
		return err
	}
	_ = uc.bus.Publish(ctx, "driver.location_updated", map[string]any{
		"driver_id": id, "latitude": lat, "longitude": lng,
	})
	return nil
}

func (uc *UseCase) GetLocation(ctx context.Context, id string) (*domain.Driver, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *UseCase) FindNearest(ctx context.Context, lat, lng, radiusKm float64, limit int) ([]domain.NearbyDriver, error) {
	if radiusKm <= 0 {
		radiusKm = 5
	}
	return uc.cache.FindNearest(ctx, lat, lng, radiusKm, limit)
}

func (uc *UseCase) List(ctx context.Context, status string, limit, offset int) ([]*domain.Driver, error) {
	if status != "" {
		if err := validator.OneOf("status", status, domain.StatusOnline, domain.StatusOffline, domain.StatusBusy); err != nil {
			return nil, domain.ErrInvalidStatus
		}
	}
	return uc.repo.List(ctx, status, limit, offset)
}

func (uc *UseCase) ListAvailable(ctx context.Context, limit, offset int) ([]*domain.Driver, error) {
	return uc.repo.List(ctx, domain.StatusOnline, limit, offset)
}

func (uc *UseCase) AssignToRide(ctx context.Context, rideID string, pickupLat, pickupLng float64) (string, error) {
	candidates, err := uc.cache.FindNearest(ctx, pickupLat, pickupLng, 10, 5)
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", domain.ErrNoDriverNearby
	}
	chosen := candidates[0]
	if err := uc.repo.UpdateStatus(ctx, chosen.DriverID, domain.StatusBusy); err != nil {
		return "", err
	}
	_ = uc.cache.SetStatus(ctx, chosen.DriverID, domain.StatusBusy)
	if err := uc.repo.RecordAssignment(ctx, chosen.DriverID, rideID); err != nil {
		return "", err
	}
	_ = uc.bus.Publish(ctx, "driver.assigned", map[string]any{
		"ride_id": rideID, "driver_id": chosen.DriverID,
	})
	return chosen.DriverID, nil
}

func (uc *UseCase) Get(ctx context.Context, id string) (*domain.Driver, *domain.Vehicle, error) {
	d, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	v, err := uc.repo.GetVehicle(ctx, id)
	if err != nil {
		return d, nil, err
	}
	return d, v, nil
}

func (uc *UseCase) StatusHistory(ctx context.Context, driverID string, limit int) ([]domain.StatusHistory, error) {
	return uc.repo.ListStatusHistory(ctx, driverID, limit)
}

func (uc *UseCase) SetRating(ctx context.Context, driverID string, rating float64) (float64, error) {
	if _, err := uc.repo.GetByID(ctx, driverID); err != nil {
		return 0, err
	}
	if rating < 1 {
		rating = 1
	}
	if rating > 5 {
		rating = 5
	}
	return rating, nil
}
