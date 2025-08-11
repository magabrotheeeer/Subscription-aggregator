package subscription

import "time"

// Оригинальная структура для хранения данных
type SubscriptionEntry struct {
	ServiceName string
	Price       int
	Username    string
	StartDate   time.Time
	EndDate     *time.Time
}

// Структура для хранения данных, полученных в формате JSON, для конвертации времени для create
type DummySubscriptionEntry struct {
	ServiceName string `json:"service_name" validate:"required"`
	Price       int    `json:"price" validate:"required,gt=0"`
	StartDate   string `json:"start_date" validate:"required"`
	EndDate     string `json:"end_date,omitempty" validate:"omitempty"`
}
