// Package jwt реализует генерацию и парсинг JWT токенов с пользовательскими claim полями.
//
// JWTMaker определяет интерфейс для создания и проверки JWT токенов с username и role.
// JWTMakerImpl — конкретная реализация с использованием секретного ключа и срока
package jwt

import (
	"time"
)

// Maker описывает интерфейс для генерации и парсинга JWT токенов.
//
// Методы позволяют создавать токен с указанием username и роли,
// а также разбирать токен и извлекать из него кастомные данные.
type Maker interface {
	// GenerateToken теперь принимает username и role
	GenerateToken(username, role, useruid string) (string, error)
	// ParseToken возвращает *CustomClaims с username и role
	ParseToken(tokenStr string) (*CustomClaims, error)
}

// MakerImpl реализует интерфейс JWTMaker с использованием секретного ключа
// и времени жизни токена (TTL).
type MakerImpl struct {
	secretKey string        // Секретный ключ для подписи токенов.
	tokenTTL  time.Duration // Время жизни токена.
}

// NewJWTMaker создаёт новый экземпляр JWTMakerImpl на основе секретного ключа и TTL.
func NewJWTMaker(secretKey string, ttl time.Duration) *MakerImpl {
	return &MakerImpl{
		secretKey: secretKey,
		tokenTTL:  ttl,
	}
}
