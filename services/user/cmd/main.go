package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	userv1 "github.com/jetkzu/jetkzu/gen/go/user/v1"
	"github.com/jetkzu/jetkzu/pkg/jwt"
	"github.com/jetkzu/jetkzu/pkg/logger"
	"github.com/jetkzu/jetkzu/pkg/metrics"
	"github.com/jetkzu/jetkzu/pkg/natsbus"
	"github.com/jetkzu/jetkzu/pkg/postgres"
	userconfig "github.com/jetkzu/jetkzu/services/user/internal/config"
	usergrpc "github.com/jetkzu/jetkzu/services/user/internal/delivery/grpc"
	userpg "github.com/jetkzu/jetkzu/services/user/internal/infrastructure/postgres"
	"github.com/jetkzu/jetkzu/services/user/internal/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	log := logger.New("user-service")
	defer log.Sync() //nolint:errcheck

	cfg, err := userconfig.Load()
	if err != nil {
		log.Fatal("config", zap.Error(err))
	}

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

	repo := userpg.NewUserRepo(pool)
	jwtMgr := jwt.New(cfg.JWTSecret, cfg.JWTTTL)
	uc := usecase.New(repo, jwtMgr, bus)
	handler := usergrpc.NewHandler(uc)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(metrics.UnaryServerInterceptor("user")),
	)
	userv1.RegisterUserServiceServer(grpcServer, handler)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatal("listen", zap.Error(err))
	}

	metricsSrv := metrics.ServeHealthAndMetrics(fmt.Sprintf(":%d", cfg.HTTPPort))
	log.Info("user-service starting", zap.Int("grpc_port", cfg.GRPCPort), zap.Int("http_port", cfg.HTTPPort))

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
