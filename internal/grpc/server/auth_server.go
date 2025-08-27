// Package server реализует gRPC-сервер для авторизационного сервиса.
//
// AuthServer обрабатывает gRPC-запросы регистрации, входа и валидации JWT токенов.
// Логирует операции и ошибки, делегирует бизнес-логику объекту AuthService.
package server

import (
	"context"
	"log/slog"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/gen"
	services "github.com/magabrotheeeer/subscription-aggregator/internal/services/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthServer реализует gRPC-сервис авторизации
type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	authService *services.AuthService
	log         *slog.Logger
}

// NewAuthServer создает новый экземпляр AuthServer с указанным сервисом аутентификации и логгером.
func NewAuthServer(authService *services.AuthService, logger *slog.Logger) *AuthServer {
	return &AuthServer{
		authService: authService,
		log:         logger,
	}
}

// Register создает нового пользователя
func (s *AuthServer) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	s.log.Info("Register request", slog.String("username", req.Username))

	_, err := s.authService.Register(ctx, req.Email, req.Username, req.Password)
	if err != nil {
		s.log.Error("Register failed",
			slog.String("username", req.Username),
			slog.Any("error", err),
		)
		return nil, status.Errorf(codes.Internal, "registration failed: %v", err)
	}
	return &authpb.RegisterResponse{
		Success: true,
		Message: "user created successfully",
	}, nil
}

// Login проверяет пользователя и генерирует JWT
func (s *AuthServer) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	s.log.Info("Login request", slog.String("username", req.Username))

	token, refresh, role, err := s.authService.Login(ctx, req.Username, req.Password)
	if err != nil {
		s.log.Error("Login failed",
			slog.String("username", req.Username),
			slog.Any("error", err),
		)
		return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	return &authpb.LoginResponse{
		Token:        token,
		RefreshToken: refresh,
		Role:         role,
	}, nil
}

// ValidateToken проверяет валидность JWT и возвращает данные пользователя
func (s *AuthServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	s.log.Info("ValidateToken request")

	user, role, valid, err := s.authService.ValidateToken(ctx, req.Token)
	if err != nil || !valid {
		s.log.Error("Invalid token", slog.Any("error", err))
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	return &authpb.ValidateTokenResponse{
		Username: user.Username,
		Role:     role,
		Valid:    valid,
		Useruid:  user.UUID,
	}, nil
}
