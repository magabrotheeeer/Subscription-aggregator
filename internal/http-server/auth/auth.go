// Package auth предоставляет инструменты для работы с аутентификацией на основе JWT.
// Содержит интерфейс и реализацию генерации/проверки JWT‑токенов, а также функции
// для хэширования паролей и их проверки.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JWTMaker определяет методы для генерации и разбора (валидации) JWT‑токенов.
type JWTMaker interface {
	// GenerateToken генерирует токен на основе имени пользователя.
	GenerateToken(username string) (string, error)
	// ParseToken проверяет токен и возвращает зарегистрированные claims.
	ParseToken(tokenStr string) (*jwt.RegisteredClaims, error)
}

// JWTMakerImpl реализует интерфейс JWTMaker с использованием секретного ключа
// и времени жизни токена (TTL).
type JWTMakerImpl struct {
	secretKey string        // Секретный ключ для подписи токенов.
	tokenTTL  time.Duration // Время жизни токена.
}

// NewJWTMaker создаёт новый экземпляр JWTMakerImpl на основе секретного ключа и TTL.
func NewJWTMaker(secretKey string, ttl time.Duration) *JWTMakerImpl {
	return &JWTMakerImpl{
		secretKey: secretKey,
		tokenTTL:  ttl,
	}
}

// GenerateToken генерирует JWT‑токен для указанного имени пользователя.
// В токен добавляются стандартные claims с датой выдачи и временем истечения.
func (j *JWTMakerImpl) GenerateToken(username string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   username,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.tokenTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// ParseToken разбирает и валидирует переданный JWT‑токен.
// Возвращает claims токена или ошибку, если токен некорректен либо просрочен.
func (j *JWTMakerImpl) ParseToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	const op = "auth.ParseToken"

	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(j.secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("%s: invalid token", op)
	}
	return claims, nil
}

// GetHash принимает пароль пользователя и возвращает его bcrypt‑хэш.
// Используется для безопасного хранения паролей.
func GetHash(password string) (string, error) {
	const op = "auth.GetHash"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return string(hashedPassword), nil
}

// CompareHash сравнивает bcrypt‑хэш с введённым паролем.
// Возвращает nil, если пароль соответствует хэшу, иначе — ошибку.
func CompareHash(originalHash, externalPassword string) error {
	const op = "auth.CompareHash"
	if err := bcrypt.CompareHashAndPassword([]byte(originalHash), []byte(externalPassword)); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
