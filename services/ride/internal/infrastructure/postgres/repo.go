package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jetkzu/jetkzu/services/ride/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// CreateWithHistory inserts ride and initial status history in one transaction.
func (r *Repo) CreateWithHistory(ctx context.Context, ride *domain.Ride) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var driverID any
	if ride.DriverID != "" {
		driverID = ride.DriverID
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO rides (id, passenger_id, driver_id, pickup_lat, pickup_lng,
		    dropoff_lat, dropoff_lng, status, price, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10)`,
		ride.ID, ride.PassengerID, driverID, ride.PickupLat, ride.PickupLng,
		ride.DropoffLat, ride.DropoffLng, ride.Status, ride.Price, ride.CreatedAt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO ride_status_history (ride_id, status, reason, changed_at)
		VALUES ($1, $2, '', $3)`, ride.ID, ride.Status, ride.CreatedAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repo) GetByID(ctx context.Context, id string) (*domain.Ride, error) {
	return r.queryOne(ctx, `SELECT id, passenger_id, COALESCE(driver_id::text,''), pickup_lat, pickup_lng,
		dropoff_lat, dropoff_lng, status, price, created_at, updated_at FROM rides WHERE id::text=$1`, id)
}

func (r *Repo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Ride, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, passenger_id, COALESCE(driver_id::text,''), pickup_lat, pickup_lng,
		    dropoff_lat, dropoff_lng, status, price, created_at, updated_at
		FROM rides WHERE passenger_id::text=$1 OR driver_id::text=$1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Ride
	for rows.Next() {
		ride := &domain.Ride{}
		if err := rows.Scan(&ride.ID, &ride.PassengerID, &ride.DriverID, &ride.PickupLat, &ride.PickupLng,
			&ride.DropoffLat, &ride.DropoffLng, &ride.Status, &ride.Price, &ride.CreatedAt, &ride.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ride)
	}
	return out, rows.Err()
}

func (r *Repo) ListByDriver(ctx context.Context, driverID string, limit, offset int) ([]*domain.Ride, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, passenger_id, COALESCE(driver_id::text,''), pickup_lat, pickup_lng,
		    dropoff_lat, dropoff_lng, status, price, created_at, updated_at
		FROM rides WHERE driver_id::text=$1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, driverID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRides(rows)
}

func (r *Repo) ListActive(ctx context.Context, limit, offset int) ([]*domain.Ride, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, passenger_id, COALESCE(driver_id::text,''), pickup_lat, pickup_lng,
		    dropoff_lat, dropoff_lng, status, price, created_at, updated_at
		FROM rides WHERE status NOT IN ($1,$2)
		ORDER BY created_at DESC LIMIT $3 OFFSET $4`, domain.StatusCompleted, domain.StatusCancelled, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRides(rows)
}

func scanRides(rows pgx.Rows) ([]*domain.Ride, error) {
	var out []*domain.Ride
	for rows.Next() {
		ride := &domain.Ride{}
		if err := rows.Scan(&ride.ID, &ride.PassengerID, &ride.DriverID, &ride.PickupLat, &ride.PickupLng,
			&ride.DropoffLat, &ride.DropoffLng, &ride.Status, &ride.Price, &ride.CreatedAt, &ride.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ride)
	}
	return out, rows.Err()
}

func (r *Repo) ListHistory(ctx context.Context, rideID string, limit int) ([]domain.StatusHistory, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT ride_id::text,status,reason,changed_at FROM ride_status_history
		WHERE ride_id::text=$1 ORDER BY changed_at DESC LIMIT $2`, rideID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StatusHistory
	for rows.Next() {
		var h domain.StatusHistory
		if err := rows.Scan(&h.RideID, &h.Status, &h.Reason, &h.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *Repo) queryOne(ctx context.Context, q string, args ...any) (*domain.Ride, error) {
	ride := &domain.Ride{}
	err := r.pool.QueryRow(ctx, q, args...).Scan(&ride.ID, &ride.PassengerID, &ride.DriverID,
		&ride.PickupLat, &ride.PickupLng, &ride.DropoffLat, &ride.DropoffLng,
		&ride.Status, &ride.Price, &ride.CreatedAt, &ride.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRideNotFound
		}
		return nil, err
	}
	return ride, nil
}

// UpdateStatusWithHistory enforces FSM and writes status row atomically.
func (r *Repo) UpdateStatusWithHistory(ctx context.Context, id, newStatus, reason string) (*domain.Ride, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string
	if err := tx.QueryRow(ctx, `SELECT status FROM rides WHERE id::text=$1 FOR UPDATE`, id).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRideNotFound
		}
		return nil, err
	}
	if !domain.CanTransition(current, newStatus) {
		return nil, domain.ErrInvalidTransition
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `UPDATE rides SET status=$1, updated_at=$2 WHERE id::text=$3`, newStatus, now, id); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO ride_status_history (ride_id, status, reason, changed_at)
		VALUES ($1,$2,$3,$4)`, id, newStatus, reason, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *Repo) AssignDriver(ctx context.Context, rideID, driverID string) (*domain.Ride, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string
	if err := tx.QueryRow(ctx, `SELECT status FROM rides WHERE id::text=$1 FOR UPDATE`, rideID).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRideNotFound
		}
		return nil, err
	}
	if current != domain.StatusRequested {
		return nil, domain.ErrInvalidTransition
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `UPDATE rides SET driver_id=$1, status=$2, updated_at=$3 WHERE id::text=$4`,
		driverID, domain.StatusDriverAssigned, now, rideID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO ride_status_history (ride_id, status, reason, changed_at)
		VALUES ($1,$2,$3,$4)`, rideID, domain.StatusDriverAssigned, "driver "+driverID, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, rideID)
}

func (r *Repo) SavePriceEstimation(ctx context.Context, rideID string, price, distance float64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ride_price_estimations (ride_id, price, distance_km, created_at)
		VALUES ($1,$2,$3, now())`, rideID, price, distance)
	return err
}
