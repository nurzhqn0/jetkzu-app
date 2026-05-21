package repository

import (
	"context"

	"github.com/jetkzu/jetkzu/services/user/internal/domain"
)

type UserRepository interface {
	CreateWithVerification(ctx context.Context, u *domain.User, vt *domain.VerificationToken) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, role string, limit, offset int) ([]*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	UpdateRole(ctx context.Context, userID, role string) (*domain.User, error)
	ReplaceVerification(ctx context.Context, vt *domain.VerificationToken) error
	VerifyEmail(ctx context.Context, userID, token string) error
}
