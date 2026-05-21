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

	driverv1 "github.com/jetkzu/jetkzu/gen/go/driver/v1"
	"github.com/jetkzu/jetkzu/pkg/logger"
	"github.com/jetkzu/jetkzu/pkg/metrics"
	"github.com/jetkzu/jetkzu/pkg/natsbus"
	"github.com/jetkzu/jetkzu/pkg/postgres"
	pkgredis "github.com/jetkzu/jetkzu/pkg/redis"
	"github.com/jetkzu/jetkzu/services/driver/internal/config"
	driverGRPC "github.com/jetkzu/jetkzu/services/driver/internal/delivery/grpc"
	driverpg "github.com/jetkzu/jetkzu/services/driver/internal/infrastructure/postgres"
	driverredis "github.com/jetkzu/jetkzu/services/driver/internal/infrastructure/redis"
	"github.com/jetkzu/jetkzu/services/driver/internal/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	log := logger.New("driver-service")
	defer log.Sync() //nolint:errcheck

	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("postgres", zap.Error(err))
	}
	defer pool.Close()

	rc, err := pkgredis.Connect(ctx, cfg.RedisAddr)
	if err != nil {
		log.Fatal("redis", zap.Error(err))
	}
	defer rc.Close()

	bus, err := natsbus.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatal("nats", zap.Error(err))
	}
	defer bus.Close()

	repo := driverpg.New(pool)
	cache := driverredis.New(rc)
	uc := usecase.New(repo, cache, bus)
	handler := driverGRPC.NewHandler(uc)

	// Subscribe to ride.requested and auto-assign.
	if _, err := bus.Subscribe(natsbus.SubjectRideRequested, "driver-service",
		func(c context.Context, data []byte) error {
			var ev struct {
				RideID    string  `json:"ride_id"`
				PickupLat float64 `json:"pickup_lat"`
				PickupLng float64 `json:"pickup_lng"`
			}
			if err := json.Unmarshal(data, &ev); err != nil {
				return err
			}
			metrics.NATSEvents.WithLabelValues("driver", natsbus.SubjectRideRequested, "in").Inc()
			if _, err := uc.AssignToRide(c, ev.RideID, ev.PickupLat, ev.PickupLng); err != nil {
				log.Warn("auto-assign failed", zap.String("ride_id", ev.RideID), zap.Error(err))
			}
			return nil
		}); err != nil {
		log.Fatal("subscribe ride.requested", zap.Error(err))
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(metrics.UnaryServerInterceptor("driver")))
	driverv1.RegisterDriverServiceServer(grpcServer, handler)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatal("listen", zap.Error(err))
	}

	metricsSrv := metrics.ServeHealthAndMetrics(fmt.Sprintf(":%d", cfg.HTTPPort))
	log.Info("driver-service starting", zap.Int("grpc_port", cfg.GRPCPort))

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
