package yookassa

import "time"

// CreatePaymentRequest представляет запрос на создание платежа.
type CreatePaymentRequest struct {
	Amount struct {
		Value    string `json:"value"`    // сумма, например "200.00"
		Currency string `json:"currency"` // валюта, например "RUB"
	} `json:"amount"`
	PaymentToken string            `json:"payment_token"`      // токен карты (payment_method_token)
	Metadata     map[string]string `json:"metadata,omitempty"` // дополнительная инфа: user_uid, subscription_id
}

// CreatePaymentResponse представляет ответ на создание платежа.
type CreatePaymentResponse struct {
	ID     string `json:"id"`     // ID платежа в ЮKassa
	Status string `json:"status"` // статус платежа, например "succeeded"
	Amount struct {
		Value    string `json:"value"`    // сумма
		Currency string `json:"currency"` // валюта
	} `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

// Amount представляет денежную сумму.
type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}
