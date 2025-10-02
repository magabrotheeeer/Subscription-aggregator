// Package auth предоставляет сервис аутентификации.
package auth

import (
	"context"
	"log/slog"
	"net"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/gen"
	"github.com/magabrotheeeer/subscription-aggregator/internal/grpc/server"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	authservices "github.com/magabrotheeeer/subscription-aggregator/internal/services/auth"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage/repository"
	"google.golang.org/grpc"
)

// App представляет приложение аутентификации.
type App struct {
	grpcServer *grpc.Server
	listener   net.Listener
	logger     *slog.Logger
}

// New создает новый экземпляр приложения аутентификации.
func New(_ context.Context, cfg *config.Config, logger *slog.Logger) (*App, error) {
	db, err := repository.New(cfg.StorageConnectionString)
	if err != nil {
		return nil, err
	}

	jwtMaker := jwt.NewJWTMaker(cfg.JWTSecretKey, cfg.TokenTTL)
	authService := authservices.NewAuthService(db, jwtMaker)

	lis, err := net.Listen("tcp", cfg.GRPCAuthAddress)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()

	authpb.RegisterAuthServiceServer(grpcServer, server.NewAuthServer(authService, logger))

	return &App{
		grpcServer: grpcServer,
		listener:   lis,
		logger:     logger,
	}, nil
}

// Run запускает приложение аутентификации.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.logger.Info("Auth gRPC service listening on", slog.String("address", a.listener.Addr().String()))
		errCh <- a.grpcServer.Serve(a.listener)
	}()

	select {
	case <-ctx.Done():
		a.grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}
