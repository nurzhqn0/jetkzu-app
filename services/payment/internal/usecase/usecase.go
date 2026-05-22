package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jetkzu/jetkzu/pkg/natsbus"
	"github.com/jetkzu/jetkzu/services/payment/internal/domain"
	"github.com/jetkzu/jetkzu/services/payment/internal/repository"
)

type Publisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

type UseCase struct {
	repo repository.PaymentRepository
	bus  Publisher
}

func New(repo repository.PaymentRepository, bus Publisher) *UseCase {
	return &UseCase{repo: repo, bus: bus}
}

type CreateInput struct {
	RideID string
	UserID string
	Amount float64
	Method string
}

func (uc *UseCase) Create(ctx context.Context, in CreateInput) (*domain.Payment, error) {
	if in.Method == "" {
		in.Method = "card"
	}
	p := &domain.Payment{
		ID:        uuid.NewString(),
		RideID:    in.RideID,
		UserID:    in.UserID,
		Amount:    in.Amount,
		Currency:  "KZT",
		Status:    domain.StatusPending,
		Method:    in.Method,
		CreatedAt: time.Now().UTC(),
	}
	if err := uc.repo.CreateWithEvent(ctx, p, "created"); err != nil {
		return nil, err
	}
	_ = uc.bus.Publish(ctx, natsbus.SubjectPaymentCreated, map[string]any{
		"payment_id": p.ID, "ride_id": p.RideID, "user_id": p.UserID, "amount": p.Amount,
	})
	return p, nil
}

func (uc *UseCase) Get(ctx context.Context, id string) (*domain.Payment, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *UseCase) GetByRide(ctx context.Context, rideID string) (*domain.Payment, error) {
	return uc.repo.GetByRide(ctx, rideID)
}

func (uc *UseCase) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Payment, error) {
	return uc.repo.ListByUser(ctx, userID, limit, offset)
}

func (uc *UseCase) ListFailed(ctx context.Context, limit, offset int) ([]*domain.Payment, error) {
	return uc.repo.ListByStatus(ctx, domain.StatusFailed, limit, offset)
}

func (uc *UseCase) Receipt(ctx context.Context, id string) (*domain.Payment, string, error) {
	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	return p, "Receipt for ride " + p.RideID + " paid with " + p.Method, nil
}

func (uc *UseCase) ValidateMethod(method string) (bool, string) {
	switch method {
	case "card", "cash", "wallet":
		return true, "payment method is supported"
	default:
		return false, "supported methods: card, cash, wallet"
	}
}

// Process simulates a payment provider call: always succeeds in mock mode.
func (uc *UseCase) Process(ctx context.Context, id string) (*domain.Payment, error) {
	p, err := uc.repo.UpdateStatusWithEvent(ctx, id, domain.StatusSucceeded, "processed", "mock_provider_ok")
	if err != nil {
		return nil, err
	}
	_ = uc.bus.Publish(ctx, natsbus.SubjectPaymentSucceeded, map[string]any{
		"payment_id": p.ID, "ride_id": p.RideID, "user_id": p.UserID, "amount": p.Amount,
	})
	return p, nil
}

func (uc *UseCase) CreateRefundRequest(ctx context.Context, id, reason string) (*domain.Payment, string, error) {
	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	_ = uc.bus.Publish(ctx, "payment.refund_requested", map[string]any{
		"payment_id": id, "ride_id": p.RideID, "user_id": p.UserID, "reason": reason,
	})
	return p, uuid.NewString(), nil
}

func (uc *UseCase) Refund(ctx context.Context, id, reason string) (*domain.Payment, error) {
	p, err := uc.repo.UpdateStatusWithEvent(ctx, id, domain.StatusRefunded, "refunded", reason)
	if err != nil {
		return nil, err
	}
	_ = uc.bus.Publish(ctx, natsbus.SubjectPaymentRefunded, map[string]any{
		"payment_id": p.ID, "ride_id": p.RideID, "user_id": p.UserID, "amount": p.Amount, "reason": reason,
	})
	return p, nil
}
