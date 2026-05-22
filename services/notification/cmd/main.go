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

	notifv1 "github.com/jetkzu/jetkzu/gen/go/notification/v1"
	"github.com/jetkzu/jetkzu/pkg/logger"
	"github.com/jetkzu/jetkzu/pkg/metrics"
	"github.com/jetkzu/jetkzu/pkg/natsbus"
	"github.com/jetkzu/jetkzu/pkg/postgres"
	"github.com/jetkzu/jetkzu/services/notification/internal/config"
	notifgrpc "github.com/jetkzu/jetkzu/services/notification/internal/delivery/grpc"
	notifpg "github.com/jetkzu/jetkzu/services/notification/internal/infrastructure/postgres"
	notifsmtp "github.com/jetkzu/jetkzu/services/notification/internal/infrastructure/smtp"
	"github.com/jetkzu/jetkzu/services/notification/internal/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	log := logger.New("notification-service")
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

	var sender notifsmtp.Sender
	if cfg.SMTPHost != "" && cfg.SMTPUser != "" {
		sender = notifsmtp.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
		log.Info("smtp sender enabled", zap.String("host", cfg.SMTPHost))
	} else {
		sender = notifsmtp.NewMock(log)
		log.Info("mock email sender enabled (SMTP not configured)")
	}

	repo := notifpg.New(pool)
	uc := usecase.New(repo, sender, bus)
	handler := notifgrpc.NewHandler(uc)

	// Subscribe to platform events and send notifications.
	subscribeAll(bus, log, uc)

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(metrics.UnaryServerInterceptor("notification")))
	notifv1.RegisterNotificationServiceServer(grpcServer, handler)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatal("listen", zap.Error(err))
	}
	metricsSrv := metrics.ServeHealthAndMetrics(fmt.Sprintf(":%d", cfg.HTTPPort))
	log.Info("notification-service starting", zap.Int("grpc_port", cfg.GRPCPort))

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

func subscribeAll(bus *natsbus.Bus, log *zap.Logger, uc *usecase.UseCase) {
	type genericEvent map[string]any

	mustSub := func(subj string, handler func(context.Context, genericEvent)) {
		if _, err := bus.Subscribe(subj, "notification-service", func(c context.Context, data []byte) error {
			var ev genericEvent
			if err := json.Unmarshal(data, &ev); err != nil {
				return err
			}
			metrics.NATSEvents.WithLabelValues("notification", subj, "in").Inc()
			handler(c, ev)
			return nil
		}); err != nil {
			log.Fatal("subscribe "+subj, zap.Error(err))
		}
	}

	str := func(m map[string]any, k string) string {
		if v, ok := m[k].(string); ok {
			return v
		}
		return ""
	}

	mustSub(natsbus.SubjectUserRegistered, func(c context.Context, e genericEvent) {
		uid := str(e, "user_id")
		email := str(e, "email")
		name := str(e, "full_name")
		_, _ = uc.SendEmail(c, uid, email, "Welcome to JetKZu",
			fmt.Sprintf("Hi %s, your JetKZu account is ready. Verification token: %s", name, str(e, "verification_token")))
	})

	mustSub(natsbus.SubjectRideRequested, func(c context.Context, e genericEvent) {
		_, _ = uc.SendEmail(c, str(e, "passenger_id"), "", "Ride requested",
			fmt.Sprintf("Your ride %s has been requested.", str(e, "ride_id")))
	})

	mustSub(natsbus.SubjectDriverAssigned, func(c context.Context, e genericEvent) {
		_, _ = uc.SendEmail(c, str(e, "driver_id"), "", "You have a new ride",
			fmt.Sprintf("Ride %s assigned to you.", str(e, "ride_id")))
	})

	mustSub(natsbus.SubjectRideCompleted, func(c context.Context, e genericEvent) {
		_, _ = uc.SendEmail(c, str(e, "passenger_id"), "", "Ride completed",
			fmt.Sprintf("Ride %s completed. Thanks for using JetKZu!", str(e, "ride_id")))
	})

	mustSub(natsbus.SubjectPaymentSucceeded, func(c context.Context, e genericEvent) {
		_, _ = uc.SendEmail(c, str(e, "user_id"), "", "Payment receipt",
			fmt.Sprintf("Payment for ride %s succeeded.", str(e, "ride_id")))
	})
}
