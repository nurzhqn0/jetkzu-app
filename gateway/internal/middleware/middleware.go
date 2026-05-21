package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jetkzu/jetkzu/pkg/httpx"
	"github.com/jetkzu/jetkzu/pkg/jwt"
	"github.com/jetkzu/jetkzu/pkg/logger"
	"github.com/jetkzu/jetkzu/pkg/metrics"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type ctxKey string

const (
	UserIDKey   ctxKey = "user_id"
	UserRoleKey ctxKey = "user_role"
)

const HeaderCorrelationID = "X-Correlation-ID"

// CorrelationID middleware assigns or reuses a correlation id and stores it in context.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderCorrelationID)
		if id == "" {
			id = uuid.NewString()
		}
		ctx := logger.WithCorrelationID(r.Context(), id)
		w.Header().Set(HeaderCorrelationID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logging logs every request with status and latency.
func Logging(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(rw, r)
			cid := logger.CorrelationIDFrom(r.Context())
			log.Info("http",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rw.status),
				zap.Duration("dur", time.Since(start)),
				zap.String("correlation_id", cid),
			)
			metrics.HTTPRequests.WithLabelValues(r.Method, r.URL.Path, http.StatusText(rw.status)).Inc()
		})
	}
}

func CORS(next http.Handler) http.Handler {
	allowed := map[string]struct{}{
		"http://localhost:3000": {},
		"http://localhost:5173": {},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Correlation-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Correlation-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Auth verifies a JWT and injects user_id + role into context.
func Auth(jm *jwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			cid := logger.CorrelationIDFrom(r.Context())
			if h == "" || !strings.HasPrefix(h, "Bearer ") {
				httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token", cid)
				return
			}
			token := strings.TrimPrefix(h, "Bearer ")
			claims, err := jm.Parse(token)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid token", cid)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}

func UserRole(ctx context.Context) string {
	v, _ := ctx.Value(UserRoleKey).(string)
	return v
}

// GRPCContext propagates correlation id to gRPC metadata.
func GRPCContext(ctx context.Context) context.Context {
	cid := logger.CorrelationIDFrom(ctx)
	if cid == "" {
		return ctx
	}
	md := metadata.Pairs("x-correlation-id", cid)
	return metadata.NewOutgoingContext(ctx, md)
}
