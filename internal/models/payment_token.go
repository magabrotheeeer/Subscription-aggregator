package models

import "time"

// PaymentToken представляет токен платежного метода пользователя.
type PaymentToken struct {
	ID        int       `json:"id"`
	UserUID   string    `json:"user_uid"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
}
