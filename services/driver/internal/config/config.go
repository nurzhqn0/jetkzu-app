package config

import (
	"os"
	"strconv"
)

type Config struct {
	GRPCPort    int
	HTTPPort    int
	DatabaseURL string
	RedisAddr   string
	NATSURL     string
}

func Load() *Config {
	return &Config{
		GRPCPort:    envInt("GRPC_PORT", 50052),
		HTTPPort:    envInt("HTTP_PORT", 8082),
		DatabaseURL: envStr("DATABASE_URL", "postgres://jetkzu:jetkzu@postgres:5432/jetkzu_drivers?sslmode=disable"),
		RedisAddr:   envStr("REDIS_ADDR", "redis:6379"),
		NATSURL:     envStr("NATS_URL", "nats://nats:4222"),
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
