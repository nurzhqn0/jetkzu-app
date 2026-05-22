package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jetkzu/jetkzu/gateway/internal/clients"
	"github.com/jetkzu/jetkzu/gateway/internal/config"
	"github.com/jetkzu/jetkzu/gateway/internal/handlers"
	"github.com/jetkzu/jetkzu/gateway/internal/router"
	"github.com/jetkzu/jetkzu/pkg/jwt"
	"github.com/jetkzu/jetkzu/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	log := logger.New("api-gateway")
	defer log.Sync() //nolint:errcheck

	cfg := config.Load()

	cli, err := clients.Connect(cfg.UserAddr, cfg.DriverAddr, cfg.RideAddr, cfg.PaymentAddr, cfg.NotificationAddr)
	if err != nil {
		log.Fatal("connect upstream", zap.Error(err))
	}
	defer cli.Close()

	jm := jwt.New(cfg.JWTSecret, 24*time.Hour)
	h := handlers.New(cli, cfg.GRPCTimeout)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router.New(h, jm, log),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("api-gateway listening", zap.Int("port", cfg.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("listen", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
