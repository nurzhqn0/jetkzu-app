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

	paymentv1 "github.com/jetkzu/jetkzu/gen/go/payment/v1"
	"github.com/jetkzu/jetkzu/pkg/logger"
	"github.com/jetkzu/jetkzu/pkg/metrics"
	"github.com/jetkzu/jetkzu/pkg/natsbus"
	"github.com/jetkzu/jetkzu/pkg/postgres"
	"github.com/jetkzu/jetkzu/services/payment/internal/config"
	paymentgrpc "github.com/jetkzu/jetkzu/services/payment/internal/delivery/grpc"
	paymentpg "github.com/jetkzu/jetkzu/services/payment/internal/infrastructure/postgres"
	"github.com/jetkzu/jetkzu/services/payment/internal/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	log := logger.New("payment-service")
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

	repo := paymentpg.New(pool)
	uc := usecase.New(repo, bus)
	handler := paymentgrpc.NewHandler(uc)

	// ride.completed -> create payment + auto-process (mock)
	if _, err := bus.Subscribe(natsbus.SubjectRideCompleted, "payment-service",
		func(c context.Context, data []byte) error {
			var ev struct {
				RideID      string  `json:"ride_id"`
				PassengerID string  `json:"passenger_id"`
				Price       float64 `json:"price"`
			}
			if err := json.Unmarshal(data, &ev); err != nil {
				return err
			}
			metrics.NATSEvents.WithLabelValues("payment", natsbus.SubjectRideCompleted, "in").Inc()
			p, err := uc.Create(c, usecase.CreateInput{
				RideID: ev.RideID, UserID: ev.PassengerID, Amount: ev.Price, Method: "card",
			})
			if err != nil {
				log.Warn("auto-create payment", zap.Error(err))
				return nil
			}
			if _, err := uc.Process(c, p.ID); err != nil {
				log.Warn("auto-process payment", zap.Error(err))
			}
			return nil
		}); err != nil {
		log.Fatal("subscribe ride.completed", zap.Error(err))
	}

	if _, err := bus.Subscribe(natsbus.SubjectRideCancelled, "payment-service",
		func(c context.Context, data []byte) error {
			var ev struct {
				RideID string `json:"ride_id"`
				Reason string `json:"reason"`
			}
			if err := json.Unmarshal(data, &ev); err != nil {
				return err
			}
			metrics.NATSEvents.WithLabelValues("payment", natsbus.SubjectRideCancelled, "in").Inc()
			p, err := uc.GetByRide(c, ev.RideID)
			if err != nil {
				return nil
			}
			if _, err := uc.Refund(c, p.ID, ev.Reason); err != nil {
				log.Warn("auto-refund", zap.Error(err))
			}
			return nil
		}); err != nil {
		log.Fatal("subscribe ride.cancelled", zap.Error(err))
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(metrics.UnaryServerInterceptor("payment")))
	paymentv1.RegisterPaymentServiceServer(grpcServer, handler)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatal("listen", zap.Error(err))
	}
	metricsSrv := metrics.ServeHealthAndMetrics(fmt.Sprintf(":%d", cfg.HTTPPort))
	log.Info("payment-service starting", zap.Int("grpc_port", cfg.GRPCPort))

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
