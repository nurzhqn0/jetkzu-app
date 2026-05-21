package usecase

import (
	"context"
	"encoding/hex"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jetkzu/jetkzu/pkg/jwt"
	"github.com/jetkzu/jetkzu/pkg/validator"
	"github.com/jetkzu/jetkzu/services/user/internal/domain"
	"github.com/jetkzu/jetkzu/services/user/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type Publisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

type UserUseCase struct {
	repo repository.UserRepository
	jwt  *jwt.Manager
	bus  Publisher
}

func New(repo repository.UserRepository, jwtMgr *jwt.Manager, bus Publisher) *UserUseCase {
	return &UserUseCase{repo: repo, jwt: jwtMgr, bus: bus}
}

type RegisterInput struct {
	Email    string
	Password string
	FullName string
	Phone    string
	Role     string
}

type RegisterResult struct {
	User              *domain.User
	VerificationToken string
}

func (uc *UserUseCase) Register(ctx context.Context, in RegisterInput) (*RegisterResult, error) {
	if err := validator.Email(in.Email); err != nil {
		return nil, err
	}
	if err := validator.MinLen("password", in.Password, 6); err != nil {
		return nil, err
	}
	if err := validator.NotEmpty("full_name", in.FullName); err != nil {
		return nil, err
	}
	if in.Role == "" {
		in.Role = domain.RolePassenger
	}
	if err := validator.OneOf("role", in.Role, domain.RolePassenger, domain.RoleDriver, domain.RoleAdmin); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	u := &domain.User{
		ID:           uuid.NewString(),
		Email:        in.Email,
		PasswordHash: string(hash),
		FullName:     in.FullName,
		Phone:        in.Phone,
		Role:         in.Role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	token := randomToken(24)
	vt := &domain.VerificationToken{
		UserID:    u.ID,
		Token:     token,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := uc.repo.CreateWithVerification(ctx, u, vt); err != nil {
		return nil, err
	}

	_ = uc.bus.Publish(ctx, "user.registered", map[string]any{
		"user_id":            u.ID,
		"email":              u.Email,
		"full_name":          u.FullName,
		"role":               u.Role,
		"verification_token": token,
	})

	return &RegisterResult{User: u, VerificationToken: token}, nil
}

type LoginResult struct {
	User      *domain.User
	Token     string
	ExpiresAt time.Time
}

func (uc *UserUseCase) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	u, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, domain.ErrInvalidCredential
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredential
	}
	token, exp, err := uc.jwt.Issue(u.ID, u.Email, u.Role)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, Token: token, ExpiresAt: exp}, nil
}

func (uc *UserUseCase) Logout(_ context.Context, userID string) (bool, error) {
	return userID != "", nil
}

func (uc *UserUseCase) ValidateSession(ctx context.Context, token string) (*domain.User, bool, error) {
	claims, err := uc.jwt.Parse(token)
	if err != nil {
		return nil, false, nil
	}
	u, err := uc.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, false, err
	}
	return u, true, nil
}

func (uc *UserUseCase) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	return uc.repo.GetByID(ctx, userID)
}

func (uc *UserUseCase) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if err := validator.Email(email); err != nil {
		return nil, err
	}
	return uc.repo.GetByEmail(ctx, email)
}

func (uc *UserUseCase) List(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	if role != "" {
		if err := validator.OneOf("role", role, domain.RolePassenger, domain.RoleDriver, domain.RoleAdmin, domain.RoleInactive); err != nil {
			return nil, err
		}
	}
	return uc.repo.List(ctx, role, limit, offset)
}

type UpdateInput struct {
	UserID   string
	FullName string
	Phone    string
}

func (uc *UserUseCase) UpdateProfile(ctx context.Context, in UpdateInput) (*domain.User, error) {
	u, err := uc.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if in.FullName != "" {
		u.FullName = in.FullName
	}
	if in.Phone != "" {
		u.Phone = in.Phone
	}
	if err := uc.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (uc *UserUseCase) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) (bool, error) {
	u, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
		return false, domain.ErrInvalidCredential
	}
	if err := validator.MinLen("password", newPassword, 6); err != nil {
		return false, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	return true, uc.repo.UpdatePassword(ctx, userID, string(hash))
}

func (uc *UserUseCase) ResetPassword(ctx context.Context, email, newPassword string) (bool, error) {
	u, err := uc.GetByEmail(ctx, email)
	if err != nil {
		return false, err
	}
	if err := validator.MinLen("password", newPassword, 6); err != nil {
		return false, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	return true, uc.repo.UpdatePassword(ctx, u.ID, string(hash))
}

func (uc *UserUseCase) VerifyEmail(ctx context.Context, userID, token string) (bool, error) {
	if err := uc.repo.VerifyEmail(ctx, userID, token); err != nil {
		if err == domain.ErrInvalidToken {
			return false, nil
		}
		return false, err
	}
	_ = uc.bus.Publish(ctx, "user.email_verified", map[string]any{"user_id": userID})
	return true, nil
}

func (uc *UserUseCase) ResendVerification(ctx context.Context, userID string) (string, error) {
	u, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	token := randomToken(24)
	vt := &domain.VerificationToken{UserID: userID, Token: token, ExpiresAt: time.Now().UTC().Add(24 * time.Hour)}
	if err := uc.repo.ReplaceVerification(ctx, vt); err != nil {
		return "", err
	}
	_ = uc.bus.Publish(ctx, "user.verification_resent", map[string]any{
		"user_id": userID, "email": u.Email, "verification_token": token,
	})
	return token, nil
}

func (uc *UserUseCase) Deactivate(ctx context.Context, userID string) (*domain.User, error) {
	return uc.repo.UpdateRole(ctx, userID, domain.RoleInactive)
}

func (uc *UserUseCase) UpdateRole(ctx context.Context, userID, role string) (*domain.User, error) {
	if err := validator.OneOf("role", role, domain.RolePassenger, domain.RoleDriver, domain.RoleAdmin, domain.RoleInactive); err != nil {
		return nil, err
	}
	return uc.repo.UpdateRole(ctx, userID, role)
}

func randomToken(n int) string {
	b := make([]byte, n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	_, _ = r.Read(b)
	return hex.EncodeToString(b)
}
