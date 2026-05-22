package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	GRPCRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "jetkzu_grpc_requests_total", Help: "Total gRPC requests"},
		[]string{"service", "method", "code"},
	)
	GRPCLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "jetkzu_grpc_latency_seconds", Help: "gRPC handler latency"},
		[]string{"service", "method"},
	)
	NATSEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "jetkzu_nats_events_total", Help: "Total NATS events"},
		[]string{"service", "subject", "direction"},
	)
	HTTPRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "jetkzu_http_requests_total", Help: "HTTP requests via gateway"},
		[]string{"method", "path", "status"},
	)
)

func init() {
	prometheus.MustRegister(GRPCRequests, GRPCLatency, NATSEvents, HTTPRequests)
}

// ServeHealthAndMetrics starts an HTTP server exposing /health and /metrics
// in a goroutine. Returns the server so callers can shut it down.
func ServeHealthAndMetrics(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() { _ = srv.ListenAndServe() }()
	return srv
}

// UnaryServerInterceptor records counters and latency for gRPC handlers.
func UnaryServerInterceptor(service string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)
		GRPCRequests.WithLabelValues(service, info.FullMethod, code.String()).Inc()
		GRPCLatency.WithLabelValues(service, info.FullMethod).Observe(time.Since(start).Seconds())
		return resp, err
	}
}

// CodeAsString helps when manually labelling counters.
func CodeAsString(c int) string { return strconv.Itoa(c) }
