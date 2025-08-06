package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type JWTMaker interface {
	GenerateToken(username string) (string, error)
	ParseToken(tokenStr string) (*jwt.RegisteredClaims, error)
}

type JWTMakerImpl struct {
	secretKey string
	tokenTTL  time.Duration
}

func NewJWTMaker(seckerKey string, ttl time.Duration) *JWTMakerImpl {
	return &JWTMakerImpl{
		secretKey: seckerKey,
		tokenTTL:  ttl,
	}
}

func (j *JWTMakerImpl) GenerateToken(username string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   username,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.tokenTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

func (j *JWTMakerImpl) ParseToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	const op = "auth.parsetoken"
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{},
		func(token *jwt.Token) (any, error) {
			return []byte(j.secretKey), nil
		})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return claims, nil
}

func GetHash(password string) (string, error) {
	const op = "auth.gethash"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return string(hashedPassword), nil
}

func CompareHash(originalHash, externalPassword string) error {
	const op = "auth.comparehash"
	err := bcrypt.CompareHashAndPassword([]byte(originalHash), []byte(externalPassword))
	if err == nil {
		return nil
	} else {
		return fmt.Errorf("%s: %w", op, err)
	}
}
