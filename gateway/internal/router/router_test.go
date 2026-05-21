package router

import (
	"testing"
	"time"

	"github.com/jetkzu/jetkzu/gateway/internal/handlers"
	"github.com/jetkzu/jetkzu/pkg/jwt"
	"go.uber.org/zap"
)

func TestNewRegistersRoutes(t *testing.T) {
	h := handlers.New(nil, time.Second)
	jm := jwt.New("test-secret", time.Hour)

	if got := New(h, jm, zap.NewNop()); got == nil {
		t.Fatal("expected router")
	}
}
