package subscription

import "time"

// Структура для хранения данных при первичном обращении к серверу
type CreaterSubscriptionEntry struct {
	ServiceName string     `json:"service_name" validator:"required,alphanum"`
	Price       int        `json:"price" validator:"required,numeric"`
	UserID      string     `json:"user_id" validator:"required,uuid"`
	StartDate   time.Time  `json:"start_date" validator:"required,datetime=01-2006"`
	EndDate     *time.Time `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}

// Структура для хранения данных, полученных в формате JSON
type DummyCreaterSubscriptionEntry struct {
	ServiceName string `json:"service_name" validator:"required,alphanum"`
	Price       int    `json:"price" validator:"required,numeric"`
	UserID      string `json:"user_id" validator:"required,uuid"`
	StartDate   string `json:"start_date" validator:"required,datetime=01-2006"`
	EndDate     string `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}

// Структура для фильтрации данных для remove
type FilterRemoveSubscriptionEntry struct {
	ServiceName string `json:"service_name" validator:"omitempty,alphanum"`
	UserID      string `json:"user_id" validator:"required,uuid"`
}

// Структура для фильтрации данных для read
type FilterReadSubscriptionEntry struct {
	ServiceName string     `json:"service_name,omitempty" validator:"omitempty,alphanum"`
	Price       int        `json:"price,omitempty" validator:"omitempty,numeric"`
	UserID      string     `json:"user_id" validator:"required,uuid"`
	StartDate   time.Time  `json:"start_date,omitempty" validator:"omitempty,datetime=01-2006"`
	EndDate     *time.Time `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}

// Структура для фильтрации данных для update
type FilterUpdateSubscriptionEntry struct {
	ServiceName string     `json:"service_name" validator:"required,alphanum"`
	Price       int        `json:"price" validator:"omitempty,numeric"`
	UserID      string     `json:"user_id" validator:"required,uuid"`
	StartDate   time.Time  `json:"start_date" validator:"omitempty,datetime=01-2006"`
	EndDate     *time.Time `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}

type DummyFilterUpdateSubscriptionEntry struct {
	ServiceName string `json:"service_name" validator:"required,alphanum"`
	Price       int    `json:"price" validator:"omitempty,numeric"`
	UserID      string `json:"user_id" validator:"required,uuid"`
	StartDate   string `json:"start_date" validator:"omitempty,datetime=01-2006"`
	EndDate     string `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}

type ListSubscriptionEntrys struct {
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      string     `json:"user_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}
