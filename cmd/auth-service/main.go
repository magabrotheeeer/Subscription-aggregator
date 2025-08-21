// Пакет main содержит точку входа для запуска gRPC-сервиса аутентификации.
//
// Он выполняет следующие действия:
// 1. Загружает конфигурацию приложения.
// 2. Настраивает логгер с уровнем отладки.
// 3. Инициализирует подключение к базе данных.
// 4. Создает JWT-генератор с указанным секретным ключом и TTL.
// 5. Строит бизнес-логику аутентификации (AuthService).
// 6. Запускает gRPC-сервер и регистрирует AuthService.
// 7. Обрабатывает системные сигналы для корректного завершения работы.
package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/proto"
	"github.com/magabrotheeeer/subscription-aggregator/internal/grpc/server"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/services"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

func main() {
	config := config.MustLoad()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting auth-service", slog.String("env", config.Env))
	logger.Debug("debug messages are enabled")

	// Подключаем базу пользователей
	db, err := storage.New(config.StorageConnectionString)
	if err != nil {
		logger.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	// JWT Maker
	jwtMaker := jwt.NewJWTMaker(config.JWTSecretKey, config.TokenTTL)

	// Бизнес-логика Auth
	authService := services.NewAuthService(db, jwtMaker)

	// gRPC сервер
	lis, err := net.Listen("tcp", config.GRPCAuthAddress)
	if err != nil {
		logger.Error("failed to listen", slog.String("address", config.GRPCAuthAddress))
		os.Exit(1)
	}
	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, server.NewAuthServer(authService, logger))

	logger.Info("Auth gRPC service listening on", slog.String("port:", config.GRPCAuthAddress))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("failed to listen port", sl.Err(err))
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	grpcServer.GracefulStop()
	logger.Info("application stopped")
}
