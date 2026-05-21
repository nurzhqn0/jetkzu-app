package config

import (
	"os"
	"strconv"
)

type Config struct {
	GRPCPort    int
	HTTPPort    int
	DatabaseURL string
	NATSURL     string

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
}

func Load() *Config {
	return &Config{
		GRPCPort:    envInt("GRPC_PORT", 50055),
		HTTPPort:    envInt("HTTP_PORT", 8085),
		DatabaseURL: envStr("DATABASE_URL", "postgres://jetkzu:jetkzu@postgres:5432/jetkzu_notifications?sslmode=disable"),
		NATSURL:     envStr("NATS_URL", "nats://nats:4222"),
		SMTPHost:    envStr("SMTP_HOST", ""),
		SMTPPort:    envInt("SMTP_PORT", 587),
		SMTPUser:    envStr("SMTP_USERNAME", ""),
		SMTPPassword: envStr("SMTP_PASSWORD", ""),
		SMTPFrom:    envStr("SMTP_FROM", "no-reply@jetkzu.kz"),
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
