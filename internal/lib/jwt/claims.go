package jwt

import (
	"time"
)

type JWTMaker interface {
	// GenerateToken теперь принимает username и role
	GenerateToken(username, role string) (string, error)
	// ParseToken возвращает *CustomClaims с username и role
	ParseToken(tokenStr string) (*CustomClaims, error)
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
