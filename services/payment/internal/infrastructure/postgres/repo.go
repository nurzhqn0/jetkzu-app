package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jetkzu/jetkzu/services/payment/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) CreateWithEvent(ctx context.Context, p *domain.Payment, event string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO payments (id, ride_id, user_id, amount, currency, status, method, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)`,
		p.ID, p.RideID, p.UserID, p.Amount, p.Currency, p.Status, p.Method, p.CreatedAt)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO payment_events (payment_id, event, payload, created_at) VALUES ($1,$2,$3,now())`,
		p.ID, event, p.Status); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repo) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	return r.scan(ctx, `SELECT id,ride_id,user_id,amount,currency,status,method,created_at,updated_at FROM payments WHERE id=$1`, id)
}

func (r *Repo) GetByRide(ctx context.Context, rideID string) (*domain.Payment, error) {
	return r.scan(ctx, `SELECT id,ride_id,user_id,amount,currency,status,method,created_at,updated_at FROM payments WHERE ride_id=$1 ORDER BY created_at DESC LIMIT 1`, rideID)
}

func (r *Repo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id,ride_id,user_id,amount,currency,status,method,created_at,updated_at
		FROM payments WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPayments(rows)
}

func (r *Repo) ListByStatus(ctx context.Context, statusVal string, limit, offset int) ([]*domain.Payment, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id,ride_id,user_id,amount,currency,status,method,created_at,updated_at
		FROM payments WHERE status=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, statusVal, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPayments(rows)
}

func scanPayments(rows pgx.Rows) ([]*domain.Payment, error) {
	var out []*domain.Payment
	for rows.Next() {
		p := &domain.Payment{}
		if err := rows.Scan(&p.ID, &p.RideID, &p.UserID, &p.Amount, &p.Currency, &p.Status, &p.Method, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repo) scan(ctx context.Context, q string, args ...any) (*domain.Payment, error) {
	p := &domain.Payment{}
	err := r.pool.QueryRow(ctx, q, args...).Scan(&p.ID, &p.RideID, &p.UserID, &p.Amount, &p.Currency, &p.Status, &p.Method, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *Repo) UpdateStatusWithEvent(ctx context.Context, id, newStatus, event, reason string) (*domain.Payment, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string
	if err := tx.QueryRow(ctx, `SELECT status FROM payments WHERE id=$1 FOR UPDATE`, id).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	if !domain.CanTransition(current, newStatus) {
		return nil, domain.ErrInvalidTransition
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `UPDATE payments SET status=$1, updated_at=$2 WHERE id=$3`, newStatus, now, id); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO payment_events (payment_id, event, payload, created_at) VALUES ($1,$2,$3,$4)`,
		id, event, newStatus+":"+reason, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}
