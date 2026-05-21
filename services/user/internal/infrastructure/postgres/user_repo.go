package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jetkzu/jetkzu/services/user/internal/domain"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

// CreateWithVerification inserts user and verification token atomically.
func (r *UserRepo) CreateWithVerification(ctx context.Context, u *domain.User, vt *domain.VerificationToken) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, phone, role, email_verified, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)
	`, u.ID, u.Email, u.PasswordHash, u.FullName, u.Phone, u.Role, u.EmailVerified, u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailAlreadyTaken
		}
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO verification_tokens (user_id, token, expires_at, created_at)
		VALUES ($1,$2,$3,now())
	`, vt.UserID, vt.Token, vt.ExpiresAt)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.scan(ctx, `SELECT id,email,password_hash,full_name,phone,role,email_verified,created_at,updated_at FROM users WHERE id=$1`, id)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scan(ctx, `SELECT id,email,password_hash,full_name,phone,role,email_verified,created_at,updated_at FROM users WHERE email=$1`, email)
}

func (r *UserRepo) List(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := `SELECT id,email,password_hash,full_name,phone,role,email_verified,created_at,updated_at FROM users`
	args := []any{}
	if role != "" {
		query += ` WHERE role=$1`
		args = append(args, role)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Phone, &u.Role, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserRepo) scan(ctx context.Context, q string, args ...any) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, q, args...).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Phone, &u.Role, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	u.UpdatedAt = time.Now().UTC()
	ct, err := r.pool.Exec(ctx, `
		UPDATE users SET full_name=$1, phone=$2, updated_at=$3 WHERE id=$4
	`, u.FullName, u.Phone, u.UpdatedAt, u.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	ct, err := r.pool.Exec(ctx, `UPDATE users SET password_hash=$1, updated_at=now() WHERE id=$2`, passwordHash, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) UpdateRole(ctx context.Context, userID, role string) (*domain.User, error) {
	ct, err := r.pool.Exec(ctx, `UPDATE users SET role=$1, updated_at=now() WHERE id=$2`, role, userID)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, domain.ErrUserNotFound
	}
	return r.GetByID(ctx, userID)
}

func (r *UserRepo) ReplaceVerification(ctx context.Context, vt *domain.VerificationToken) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM verification_tokens WHERE user_id=$1`, vt.UserID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO verification_tokens (user_id, token, expires_at, created_at)
		VALUES ($1,$2,$3,now())`, vt.UserID, vt.Token, vt.ExpiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// VerifyEmail consumes a token and flips email_verified inside one transaction.
func (r *UserRepo) VerifyEmail(ctx context.Context, userID, token string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var expires time.Time
	err = tx.QueryRow(ctx, `SELECT expires_at FROM verification_tokens WHERE user_id=$1 AND token=$2`, userID, token).Scan(&expires)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInvalidToken
		}
		return err
	}
	if time.Now().After(expires) {
		return domain.ErrInvalidToken
	}

	if _, err = tx.Exec(ctx, `UPDATE users SET email_verified=true, updated_at=now() WHERE id=$1`, userID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM verification_tokens WHERE user_id=$1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
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

func isUniqueViolation(err error) bool {
	return err != nil && (containsCode(err.Error(), "23505") || containsCode(err.Error(), "duplicate key"))
}
func containsCode(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
