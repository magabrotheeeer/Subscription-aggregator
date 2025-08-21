// Package jwt реализует генерацию и парсинг JWT токенов с пользовательскими claim полями.
//
// CustomClaims расширяет стандартные claims JWT, добавляя username и роль пользователя.
//
// Методы GenerateToken и ParseToken реализуют создание и валидацию JWT токена с заданными claims.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims описывает пользовательские данные, хранящиеся в JWT.
type CustomClaims struct {
	Username             string `json:"username"` // Имя пользователя
	Role                 string `json:"role"`     // Роль пользователя
	jwt.RegisteredClaims        // Встроенные стандартные claims JWT (ExpiresAt, IssuedAt и пр.)
}

// GenerateToken создает JWT токен с заданными username и role, подписывая его секретным ключом.
//
// Время жизни токена определяется полем tokenTTL.
func (j *MakerImpl) GenerateToken(username, role string) (string, error) {
	claims := CustomClaims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.tokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// ParseToken парсит JWT токен, проверяет его подпись и валидность,
// возвращает CustomClaims с данными, если токен корректен.
func (j *MakerImpl) ParseToken(tokenStr string) (*CustomClaims, error) {
	const op = "jwt.ParseToken"
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(_ *jwt.Token) (any, error) {
		return []byte(j.secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("%s: invalid token", op)
	}
	return claims, nil
}
