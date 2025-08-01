package subscription

import "time"

// Структура для хранения данных при первичном обращении к серверу для create
type SubscriptionEntry struct {
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      string     `json:"user_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

// Структура для хранения данных, полученных в формате JSON, для конвертации времени для create
type DummySubscriptionEntry struct {
	ServiceName string `json:"service_name" validator:"required,alphanum"`
	Price       int    `json:"price" validator:"required,numeric"`
	UserID      string `json:"user_id" validator:"required,uuid"`
	StartDate   string `json:"start_date" validator:"required,datetime=01-2006"`
	EndDate     string `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}
