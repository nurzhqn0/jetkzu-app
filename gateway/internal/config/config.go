package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort  int
	JWTSecret string

	UserAddr         string
	DriverAddr       string
	RideAddr         string
	PaymentAddr      string
	NotificationAddr string

	GRPCTimeout time.Duration
}

func Load() *Config {
	return &Config{
		HTTPPort:         envInt("HTTP_PORT", 8080),
		JWTSecret:        envStr("JWT_SECRET", "dev-secret-change-me"),
		UserAddr:         envStr("USER_SERVICE_ADDR", "user-service:50051"),
		DriverAddr:       envStr("DRIVER_SERVICE_ADDR", "driver-service:50052"),
		RideAddr:         envStr("RIDE_SERVICE_ADDR", "ride-service:50053"),
		PaymentAddr:      envStr("PAYMENT_SERVICE_ADDR", "payment-service:50054"),
		NotificationAddr: envStr("NOTIFICATION_SERVICE_ADDR", "notification-service:50055"),
		GRPCTimeout:      envDur("GRPC_TIMEOUT", 5*time.Second),
	}
}

func envStr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
func envDur(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
