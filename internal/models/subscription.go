// Package models содержит доменные структуры, описывающие подписку,
// а также вспомогательные типы для работы с данными из внешних источников (например, JSON-запросы).
package models

import "time"

// Entry представляет собой основную модель подписки,
// используемую в бизнес-логике и хранилище.
// Все даты хранятся в формате time.Time, поле EndDate может быть nil —
// это означает отсутствие даты окончания (подписка бессрочная).
type Entry struct {
	ServiceName   string    // Название сервиса подписки
	Price         int       // Цена подписки за месяц
	Username      string    // Имя пользователя, которому принадлежит подписка
	StartDate     time.Time // Дата начала подписки
	CounterMonths int       // Количество месяцев
}

// DummyEntry используется для приёма данных из JSON-запроса,
// прежде чем конвертировать их в SubscriptionEntry.
// Даты приходят в виде строк, чтобы их можно было валидировать и парсить вручную.
type DummyEntry struct {
	ServiceName   string `json:"service_name" validate:"required"`   // Название сервиса
	Price         int    `json:"price" validate:"required,gt=0"`     // Цена (>0)
	StartDate     string `json:"start_date" validate:"required"`     // Дата начала в формате 01-2006
	CounterMonths int    `json:"counter_months" validate:"required"` // Количество месяцев
}

type EntryInfo struct {
	Email       string
	Username    string
	ServiceName string
	EndDate     time.Time
	Price       int
}
