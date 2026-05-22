package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jetkzu/jetkzu/services/notification/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) Create(ctx context.Context, n *domain.Notification) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (id, user_id, channel, recipient, subject, body, status, created_at, sent_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		n.ID, n.UserID, n.Channel, n.To, n.Subject, n.Body, n.Status, n.CreatedAt, n.SentAt)
	return err
}

func (r *Repo) UpdateStatus(ctx context.Context, id, statusVal string) (*domain.Notification, error) {
	now := time.Now().UTC()
	ct, err := r.pool.Exec(ctx, `UPDATE notifications SET status=$1, sent_at=$2 WHERE id=$3`, statusVal, now, id)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, domain.ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *Repo) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	n := &domain.Notification{}
	err := r.pool.QueryRow(ctx, `
		SELECT id,user_id,channel,recipient,subject,body,status,created_at, COALESCE(sent_at, created_at)
		FROM notifications WHERE id=$1`, id).
		Scan(&n.ID, &n.UserID, &n.Channel, &n.To, &n.Subject, &n.Body, &n.Status, &n.CreatedAt, &n.SentAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return n, nil
}

func (r *Repo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id,user_id,channel,recipient,subject,body,status,created_at, COALESCE(sent_at, created_at)
		FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Notification
	for rows.Next() {
		n := &domain.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Channel, &n.To, &n.Subject, &n.Body, &n.Status, &n.CreatedAt, &n.SentAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repo) ListUnread(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id,user_id,channel,recipient,subject,body,status,created_at, COALESCE(sent_at, created_at)
		FROM notifications WHERE user_id=$1 AND status<>$2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		userID, domain.StatusRead, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Notification
	for rows.Next() {
		n := &domain.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Channel, &n.To, &n.Subject, &n.Body, &n.Status, &n.CreatedAt, &n.SentAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repo) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id=$1 AND status<>$2`, userID, domain.StatusRead).Scan(&count)
	return count, err
}

func (r *Repo) MarkAllRead(ctx context.Context, userID string) (int, error) {
	ct, err := r.pool.Exec(ctx, `UPDATE notifications SET status=$1 WHERE user_id=$2 AND status<>$1`, domain.StatusRead, userID)
	if err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM notifications WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
