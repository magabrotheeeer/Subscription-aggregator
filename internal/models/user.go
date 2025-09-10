// Package models содержит доменную модель пользователя системы,
// включающую данные учётной записи, хэш пароля и дату создания.
// Структура используется в бизнес‑логике и при работе с хранилищем.
package models

import "time"

// User представляет зарегистрированного пользователя системы.
type User struct {
	UUID               string     // Уникальный идентификатор пользователя
	Email              string     // Электронная почта
	Username           string     // Имя пользователя (уникальное)
	PasswordHash       string     // Хэш пароля пользователя
	Role               string     // Роль пользователя, admin или user
	TrialEndDate       *time.Time // Дата истечения пробного периода
	SubscriptionExpire *time.Time // Дата истечения оплаченной подписки на сервис
	SubscriptionStatus string
}
