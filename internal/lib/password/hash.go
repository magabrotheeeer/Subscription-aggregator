// Package password реализует функции для безопасного хеширования и проверки паролей.
//
// GetHash создает bcrypt-хеш пароля для безопасного хранения.
// CompareHash сравнивает исходный bcrypt-хеш с введённым паролем, проверяя их соответствие.
package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// GetHash принимает пароль пользователя и возвращает его bcrypt‑хэш.
//
// Используется для безопасного хранения паролей в базе данных.
func GetHash(password string) (string, error) {
	const op = "auth.GetHash"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return string(hashedPassword), nil
}

// CompareHash сравнивает bcrypt‑хэш с введённым паролем.
//
// Возвращает nil, если пароль соответствует хэшу, иначе — ошибку.
func CompareHash(originalHash, externalPassword string) error {
	const op = "auth.CompareHash"
	if err := bcrypt.CompareHashAndPassword([]byte(originalHash), []byte(externalPassword)); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
