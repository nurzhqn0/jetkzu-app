package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jetkzu/jetkzu/pkg/natsbus"
	"github.com/jetkzu/jetkzu/services/notification/internal/domain"
	"github.com/jetkzu/jetkzu/services/notification/internal/infrastructure/smtp"
	"github.com/jetkzu/jetkzu/services/notification/internal/repository"
)

type Publisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

type UseCase struct {
	repo   repository.NotificationRepository
	sender smtp.Sender
	bus    Publisher
}

func New(repo repository.NotificationRepository, sender smtp.Sender, bus Publisher) *UseCase {
	return &UseCase{repo: repo, sender: sender, bus: bus}
}

func (uc *UseCase) SendEmail(ctx context.Context, userID, to, subject, body string) (*domain.Notification, error) {
	n := &domain.Notification{
		ID:        uuid.NewString(),
		UserID:    userID,
		Channel:   domain.ChannelEmail,
		To:        to,
		Subject:   subject,
		Body:      body,
		Status:    domain.StatusQueued,
		CreatedAt: time.Now().UTC(),
		SentAt:    time.Now().UTC(),
	}
	if err := uc.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	st, sendErr := uc.sender.Send(ctx, to, subject, body)
	finalStatus := st
	if sendErr != nil {
		finalStatus = domain.StatusFailed
	}
	updated, err := uc.repo.UpdateStatus(ctx, n.ID, finalStatus)
	if err != nil {
		return nil, err
	}
	_ = uc.bus.Publish(ctx, natsbus.SubjectNotificationSent, map[string]any{
		"notification_id": updated.ID, "user_id": updated.UserID, "status": updated.Status,
	})
	return updated, nil
}

func (uc *UseCase) SendRideReceipt(ctx context.Context, userID, to, rideID, paymentID string, amount float64) (*domain.Notification, error) {
	subject := "JetKZu ride receipt"
	body := "Ride " + rideID + " payment " + paymentID + " completed for KZT " + formatAmount(amount)
	return uc.SendEmail(ctx, userID, to, subject, body)
}

func (uc *UseCase) History(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error) {
	return uc.repo.ListByUser(ctx, userID, limit, offset)
}

func (uc *UseCase) Unread(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error) {
	return uc.repo.ListUnread(ctx, userID, limit, offset)
}

func (uc *UseCase) CountUnread(ctx context.Context, userID string) (int, error) {
	return uc.repo.CountUnread(ctx, userID)
}

func (uc *UseCase) MarkRead(ctx context.Context, id string) (*domain.Notification, error) {
	return uc.repo.UpdateStatus(ctx, id, domain.StatusRead)
}

func (uc *UseCase) MarkAllRead(ctx context.Context, userID string) (int, error) {
	return uc.repo.MarkAllRead(ctx, userID)
}

func (uc *UseCase) Resend(ctx context.Context, id string) (*domain.Notification, error) {
	n, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.SendEmail(ctx, n.UserID, n.To, n.Subject, n.Body)
}

func (uc *UseCase) Delete(ctx context.Context, id string) (bool, error) {
	if err := uc.repo.Delete(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

func formatAmount(v float64) string {
	cents := int(v*100 + 0.5)
	whole := cents / 100
	frac := cents % 100
	if frac < 10 {
		return itoa(whole) + ".0" + itoa(frac)
	}
	return itoa(whole) + "." + itoa(frac)
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
