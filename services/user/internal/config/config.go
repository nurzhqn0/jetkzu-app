package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	GRPCPort    int
	HTTPPort    int
	DatabaseURL string
	NATSURL     string
	JWTSecret   string
	JWTTTL      time.Duration
}

func Load() (*Config, error) {
	c := &Config{
		GRPCPort:    envInt("GRPC_PORT", 50051),
		HTTPPort:    envInt("HTTP_PORT", 8081),
		DatabaseURL: envStr("DATABASE_URL", "postgres://jetkzu:jetkzu@postgres:5432/jetkzu_users?sslmode=disable"),
		NATSURL:     envStr("NATS_URL", "nats://nats:4222"),
		JWTSecret:   envStr("JWT_SECRET", "dev-secret-change-me"),
		JWTTTL:      envDuration("JWT_TTL", 24*time.Hour),
	}
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET required")
	}
	return c, nil
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
func envDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
