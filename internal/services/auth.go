package services

import (
	"context"
	"errors"

	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/jwt"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/password"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// Интерфейс репозитория пользователей
type UserRepository interface {
	RegisterUser(ctx context.Context, user models.User) (int, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
}

// AuthService реализует бизнес-логику авторизации и аутентификации
type AuthService struct {
	users    UserRepository
	jwtMaker jwt.JWTMaker
}

func NewAuthService(users UserRepository, jwtMaker jwt.JWTMaker) *AuthService {
	return &AuthService{
		users:    users,
		jwtMaker: jwtMaker,
	}
}

// Register — создание нового пользователя с хэшированием пароля и дефолтной ролью "user"
func (s *AuthService) Register(ctx context.Context, username, rawPassword string) (int, error) {
	hashed, err := password.GetHash(rawPassword)
	if err != nil {
		return 0, err
	}
	user := &models.User{
		Username:     username,
		PasswordHash: hashed,
		Role:         "user", // дефолтная роль при регистрации
	}
	return s.users.RegisterUser(ctx, *user)
}

// Login — проверка пароля и генерация JWT с username и role
func (s *AuthService) Login(ctx context.Context, username, rawPassword string) (token, refresh, role string, err error) {
	user, err := s.users.GetUserByUsername(ctx, username)
	if err != nil {
		return "", "", "", err
	}
	// проверяем пароль
	if err := password.CompareHash(user.PasswordHash, rawPassword); err != nil {
		return "", "", "", errors.New("invalid credentials")
	}
	// генерируем токен с использованием username и role
	token, err = s.jwtMaker.GenerateToken(user.Username, user.Role)
	if err != nil {
		return "", "", "", err
	}
	// пока refresh-токен можно сделать простым заглушечным
	refresh = "refresh-token-placeholder"
	return token, refresh, user.Role, nil
}

// ValidateToken — проверка JWT и возврат username, роли и статуса валидности
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*models.User, string, bool, error) {
	claims, err := s.jwtMaker.ParseToken(token)
	if err != nil {
		return nil, "", false, err
	}
	user := &models.User{
		Username: claims.Username,
		Role:     claims.Role,
	}
	return user, claims.Role, true, nil
}
