package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	ridev1 "github.com/jetkzu/jetkzu/gen/go/ride/v1"
	"github.com/jetkzu/jetkzu/pkg/logger"
	"github.com/jetkzu/jetkzu/pkg/metrics"
	"github.com/jetkzu/jetkzu/pkg/natsbus"
	"github.com/jetkzu/jetkzu/pkg/postgres"
	"github.com/jetkzu/jetkzu/services/ride/internal/config"
	ridegrpc "github.com/jetkzu/jetkzu/services/ride/internal/delivery/grpc"
	ridepg "github.com/jetkzu/jetkzu/services/ride/internal/infrastructure/postgres"
	"github.com/jetkzu/jetkzu/services/ride/internal/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	log := logger.New("ride-service")
	defer log.Sync() //nolint:errcheck

	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("postgres", zap.Error(err))
	}
	defer pool.Close()

	bus, err := natsbus.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatal("nats", zap.Error(err))
	}
	defer bus.Close()

	repo := ridepg.New(pool)
	uc := usecase.New(repo, bus)
	handler := ridegrpc.NewHandler(uc)

	if _, err := bus.Subscribe(natsbus.SubjectDriverAssigned, "ride-service",
		func(c context.Context, data []byte) error {
			var ev struct {
				RideID   string `json:"ride_id"`
				DriverID string `json:"driver_id"`
			}
			if err := json.Unmarshal(data, &ev); err != nil {
				return err
			}
			metrics.NATSEvents.WithLabelValues("ride", natsbus.SubjectDriverAssigned, "in").Inc()
			if _, err := uc.AssignDriver(c, ev.RideID, ev.DriverID); err != nil {
				log.Warn("assign driver", zap.String("ride_id", ev.RideID), zap.Error(err))
			}
			return nil
		}); err != nil {
		log.Fatal("subscribe driver.assigned", zap.Error(err))
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(metrics.UnaryServerInterceptor("ride")))
	ridev1.RegisterRideServiceServer(grpcServer, handler)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatal("listen", zap.Error(err))
	}
	metricsSrv := metrics.ServeHealthAndMetrics(fmt.Sprintf(":%d", cfg.HTTPPort))
	log.Info("ride-service starting", zap.Int("grpc_port", cfg.GRPCPort))

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("grpc serve", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	shutdownCtx, sc := context.WithTimeout(context.Background(), 5*time.Second)
	defer sc()
	_ = metricsSrv.Shutdown(shutdownCtx)
	grpcServer.GracefulStop()
}
