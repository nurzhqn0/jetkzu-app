package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

const CorrelationIDKey = "correlation_id"

func New(service string) *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.InitialFields = map[string]any{"service": service}
	cfg.DisableStacktrace = true
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}
	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return l
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func CorrelationIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}

func From(ctx context.Context, base *zap.Logger) *zap.Logger {
	if id := CorrelationIDFrom(ctx); id != "" {
		return base.With(zap.String(CorrelationIDKey, id))
	}
	return base
}
