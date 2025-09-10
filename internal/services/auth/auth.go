// Package services содержит логику бизнес-уровня для работы с пользователями и аутентификацией.
package services

import (
	"context"
	"errors"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/password"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// UserRepository описывает контракт для работы с пользователями в базе данных.
type UserRepository interface {
	// RegisterUser сохраняет нового пользователя и возвращает его ID.
	RegisterUser(ctx context.Context, user models.User) (string, error)

	// GetUserByUsername возвращает пользователя по имени или ошибку, если не найден.
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
}

// AuthService отвечает за регистрацию, авторизацию и валидацию JWT.
type AuthService struct {
	users    UserRepository
	jwtMaker jwt.Maker
}

// NewAuthService создает новый экземпляр AuthService.
func NewAuthService(users UserRepository, jwtMaker jwt.Maker) *AuthService {
	return &AuthService{
		users:    users,
		jwtMaker: jwtMaker,
	}
}

// Register создает нового пользователя с хэшированием пароля и дефолтной ролью "user".
func (s *AuthService) Register(ctx context.Context, email, username, rawPassword string) (string, error) {
	hashed, err := password.GetHash(rawPassword)
	if err != nil {
		return "", err
	}
	trialEndDate := time.Now().UTC().AddDate(0, 1, 0)
	user := &models.User{
		Email:              email,
		Username:           username,
		PasswordHash:       hashed,
		Role:               "user", // дефолтная роль при регистрации
		TrialEndDate:       &trialEndDate,
		SubscriptionStatus: "trial",
	}
	return s.users.RegisterUser(ctx, *user)
}

// Login проверяет пароль пользователя и генерирует JWT (доступ + refresh token).
func (s *AuthService) Login(ctx context.Context, username, rawPassword string) (token, refresh, role string, err error) {
	user, err := s.users.GetUserByUsername(ctx, username)
	if err != nil {
		return "", "", "", err
	}
	if err := password.CompareHash(user.PasswordHash, rawPassword); err != nil {
		return "", "", "", errors.New("invalid credentials")
	}
	token, err = s.jwtMaker.GenerateToken(user.Username, user.Role, user.UUID)
	if err != nil {
		return "", "", "", err
	}
	refresh = "refresh-token-placeholder"
	return token, refresh, user.Role, nil
}

// ValidateToken проверяет JWT и возвращает информацию о пользователе, роль и признак валидности.
func (s *AuthService) ValidateToken(_ context.Context, token string) (*models.User, string, bool, error) {
	claims, err := s.jwtMaker.ParseToken(token)
	if err != nil {
		return nil, "", false, err
	}
	user := &models.User{
		Username: claims.Username,
		Role:     claims.Role,
		UUID:     claims.UserUID,
	}
	return user, claims.Role, true, nil
}
