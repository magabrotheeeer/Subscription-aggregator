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
	ServiceName string `json:"service_name" validator:"required,alphanum"`
	Price       int    `json:"price" validator:"required,numeric"`
	StartDate   string `json:"start_date" validator:"required,datetime=01-2006"`
	EndDate     string `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}
