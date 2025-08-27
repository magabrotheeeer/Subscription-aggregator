package paymentprovider

import "time"

// Запрос на создание платёжного метода (tokenization)
type CreatePaymentMethodRequest struct {
	CardCryptogramPacket string `json:"CardCryptogramPacket" validate:"required"`
}

// Ответ CloudPayments при сохранении платёжного метода
type CreatePaymentMethodResponse struct {
	Success bool   `json:"Success"`
	Message string `json:"Message"`
	Model   struct {
		CardId       int64     `json:"CardId"` // ID сохранённой карты
		CardFirstSix string    `json:"CardFirstSix"`
		CardLastFour string    `json:"CardLastFour"`
		CardExpDate  string    `json:"CardExpDate"`
		CreatedDate  time.Time `json:"CreatedDate"`
	} `json:"Model,omitempty"`
}
