package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

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
