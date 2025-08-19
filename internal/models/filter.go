// Package countsum содержит структуры данных, используемые для подсчёта
// суммарной стоимости подписок пользователя за выбранный период.
// Здесь определены как структуры для внутреннего использования в бизнес‑логике,
// так и для приёма данных из JSON‑запросов.
package models

import "time"

// FilterSum представляет параметры фильтрации, которые передаются в слой доступа к данным.
// Используется при подсчёте суммы подписок за определённый период.
type FilterSum struct {
	Username    string    // Имя пользователя
	ServiceName *string   // Название сервиса (nil, если фильтра по сервису нет)
	StartDate   time.Time // Дата начала периода
	EndDate     time.Time // Дата окончания периода
}

// DummyFilterSum используется для приёма параметров фильтра из JSON‑запроса
// до их валидации и преобразования в FilterSum. Даты приходят строками.
type DummyFilterSum struct {
	ServiceName string `json:"service_name,omitempty" validate:"omitempty"`        // Название сервиса (опционально)
	StartDate   string `json:"start_date" validate:"required"` // Дата начала периода
	EndDate     string `json:"end_date" validate:"required"`   // Дата окончания периода
}
