package smtp

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestMockSender(t *testing.T) {
	log := zaptest.NewLogger(t)
	m := NewMock(log)
	status, err := m.Send(context.Background(), "to@example.com", "Hi", "body")
	if err != nil {
		t.Fatalf("mock should not fail: %v", err)
	}
	if status != "mock_sent" {
		t.Fatalf("expected mock_sent, got %s", status)
	}
}
