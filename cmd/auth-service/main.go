package main

import (
	"log"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/proto"
	"github.com/magabrotheeeer/subscription-aggregator/internal/grpc/server"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	"github.com/magabrotheeeer/subscription-aggregator/internal/services"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

func main() {
	config := config.MustLoad()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting auth-service", slog.String("env", config.Env))
	logger.Debug("debug messages are enabled")

	// Подключаем базу пользователей
	userRepo, err := storage.New(config.StorageConnectionString)
	if err != nil {
		log.Fatalf("failed to init user repo: %v", err)
	}

	// JWT Maker
	jwtMaker := jwt.NewJWTMaker(config.JWTSecretKey, config.TokenTTL)

	// Бизнес-логика Auth
	authService := services.NewAuthService(userRepo, jwtMaker)

	// gRPC сервер
	lis, err := net.Listen("tcp", config.GRPCAuthAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, server.NewAuthServer(authService, logger))

	log.Printf("Auth gRPC service listening on %s", config.GRPCAuthAddress)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
