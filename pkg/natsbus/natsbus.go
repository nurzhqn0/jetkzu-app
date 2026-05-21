package natsbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Subjects used across the platform.
const (
	SubjectUserRegistered      = "user.registered"
	SubjectUserEmailVerified   = "user.email_verified"
	SubjectRideRequested       = "ride.requested"
	SubjectRideStatusChanged   = "ride.status_changed"
	SubjectRideCompleted       = "ride.completed"
	SubjectRideCancelled       = "ride.cancelled"
	SubjectDriverLocationUpd   = "driver.location_updated"
	SubjectDriverAssigned      = "driver.assigned"
	SubjectPaymentCreated      = "payment.created"
	SubjectPaymentSucceeded    = "payment.succeeded"
	SubjectPaymentRefunded     = "payment.refunded"
	SubjectNotificationSent    = "notification.sent"
	HeaderCorrelationID        = "X-Correlation-ID"
)

type Bus struct {
	nc *nats.Conn
}

func Connect(url string) (*Bus, error) {
	var nc *nats.Conn
	var err error
	deadline := time.Now().Add(30 * time.Second)
	for {
		nc, err = nats.Connect(url, nats.Timeout(5*time.Second), nats.MaxReconnects(-1))
		if err == nil {
			return &Bus{nc: nc}, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("nats not reachable: %w", err)
		}
		time.Sleep(1 * time.Second)
	}
}

func (b *Bus) Close() {
	if b.nc != nil {
		_ = b.nc.Drain()
	}
}

func (b *Bus) Publish(ctx context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	msg := &nats.Msg{Subject: subject, Data: data, Header: nats.Header{}}
	if cid, ok := ctx.Value(corrKey{}).(string); ok && cid != "" {
		msg.Header.Set(HeaderCorrelationID, cid)
	}
	return b.nc.PublishMsg(msg)
}

type Handler func(ctx context.Context, data []byte) error

func (b *Bus) Subscribe(subject, queue string, h Handler) (*nats.Subscription, error) {
	return b.nc.QueueSubscribe(subject, queue, func(m *nats.Msg) {
		ctx := context.Background()
		if cid := m.Header.Get(HeaderCorrelationID); cid != "" {
			ctx = WithCorrelationID(ctx, cid)
		}
		_ = h(ctx, m.Data)
	})
}

type corrKey struct{}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, corrKey{}, id)
}

func CorrelationIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(corrKey{}).(string)
	return v
}
