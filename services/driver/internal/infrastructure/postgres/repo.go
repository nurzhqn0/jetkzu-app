package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jetkzu/jetkzu/services/driver/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Create(ctx context.Context, d *domain.Driver) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO drivers (id, user_id, license_number, status, latitude, longitude, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		d.ID, d.UserID, d.LicenseNumber, d.Status, d.Latitude, d.Longitude, d.CreatedAt)
	return err
}

func (r *Repo) GetByID(ctx context.Context, id string) (*domain.Driver, error) {
	d := &domain.Driver{}
	err := r.pool.QueryRow(ctx, `SELECT id,user_id,license_number,status,latitude,longitude,created_at FROM drivers WHERE id=$1`, id).
		Scan(&d.ID, &d.UserID, &d.LicenseNumber, &d.Status, &d.Latitude, &d.Longitude, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDriverNotFound
		}
		return nil, err
	}
	return d, nil
}

// UpdateStatus writes the new status and records history inside a transaction.
func (r *Repo) UpdateStatus(ctx context.Context, id, statusVal string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ct, err := tx.Exec(ctx, `UPDATE drivers SET status=$1 WHERE id=$2`, statusVal, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrDriverNotFound
	}
	if _, err = tx.Exec(ctx, `INSERT INTO driver_status_history (driver_id, status, changed_at) VALUES ($1,$2,$3)`,
		id, statusVal, time.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repo) UpdateLocation(ctx context.Context, id string, lat, lng float64) error {
	ct, err := r.pool.Exec(ctx, `UPDATE drivers SET latitude=$1, longitude=$2 WHERE id=$3`, lat, lng, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrDriverNotFound
	}
	return nil
}

func (r *Repo) List(ctx context.Context, statusVal string, limit, offset int) ([]*domain.Driver, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := `SELECT id,user_id,license_number,status,latitude,longitude,created_at FROM drivers`
	args := []any{}
	if statusVal != "" {
		query += ` WHERE status=$1`
		args = append(args, statusVal)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Driver
	for rows.Next() {
		d := &domain.Driver{}
		if err := rows.Scan(&d.ID, &d.UserID, &d.LicenseNumber, &d.Status, &d.Latitude, &d.Longitude, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repo) ListStatusHistory(ctx context.Context, driverID string, limit int) ([]domain.StatusHistory, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT driver_id::text,status,changed_at FROM driver_status_history
		WHERE driver_id=$1 ORDER BY changed_at DESC LIMIT $2`, driverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StatusHistory
	for rows.Next() {
		var h domain.StatusHistory
		if err := rows.Scan(&h.DriverID, &h.Status, &h.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *Repo) RecordAssignment(ctx context.Context, driverID, rideID string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO driver_assignments (driver_id, ride_id, assigned_at) VALUES ($1,$2,$3)`,
		driverID, rideID, time.Now().UTC())
	return err
}

func (r *Repo) AddVehicle(ctx context.Context, v *domain.Vehicle) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO vehicles (id, driver_id, plate_number, make, model, year, color)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		v.ID, v.DriverID, v.PlateNumber, v.Make, v.Model, v.Year, v.Color)
	return err
}

func (r *Repo) UpdateVehicle(ctx context.Context, v *domain.Vehicle) (*domain.Vehicle, error) {
	ct, err := r.pool.Exec(ctx, `
		UPDATE vehicles SET plate_number=$1, make=$2, model=$3, year=$4, color=$5 WHERE id=$6`,
		v.PlateNumber, v.Make, v.Model, v.Year, v.Color, v.ID)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, domain.ErrDriverNotFound
	}
	out := &domain.Vehicle{}
	err = r.pool.QueryRow(ctx, `SELECT id,driver_id,plate_number,make,model,year,color FROM vehicles WHERE id=$1`, v.ID).
		Scan(&out.ID, &out.DriverID, &out.PlateNumber, &out.Make, &out.Model, &out.Year, &out.Color)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repo) DeleteVehicle(ctx context.Context, vehicleID string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM vehicles WHERE id=$1`, vehicleID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrDriverNotFound
	}
	return nil
}

func (r *Repo) GetVehicle(ctx context.Context, driverID string) (*domain.Vehicle, error) {
	v := &domain.Vehicle{}
	err := r.pool.QueryRow(ctx, `SELECT id,driver_id,plate_number,make,model,year,color FROM vehicles WHERE driver_id=$1 LIMIT 1`, driverID).
		Scan(&v.ID, &v.DriverID, &v.PlateNumber, &v.Make, &v.Model, &v.Year, &v.Color)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return v, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
