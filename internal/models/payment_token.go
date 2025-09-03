package models

import "time"

type PaymentToken struct {
	ID        int
	UserUID   string
	Token     string
	CreatedAt time.Time
}