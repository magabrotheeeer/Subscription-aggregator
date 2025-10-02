// Package response содержит вспомогательные типы и функции для формирования
// унифицированных JSON‑ответов HTTP‑обработчиков. Пакет упрощает возврат
// успешных ответов, ошибок и сообщений валидации в едином формате.
package response

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator"
)

const (
	// StatusOK — значение статуса для успешного ответа.
	StatusOK = "OK"
	// StatusError — значение статуса для ответа с ошибкой.
	StatusError = "Error"
)

// OKResponse описывает стандартную структуру JSON‑ответа сервера.
// Поле Status — статус запроса ("OK").
// Поле Data — данные ответа (опционально, при успехе).
type OKResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
}

// ErrorResponse описывает стандартную структуру JSON‑ответа сервера.
// Поле Status — статус запроса ("Error").
// Поле Error  — сообщение ошибки ответа.
type ErrorResponse struct {
	Status string `json:"status" example:"Error"`
	Error  string `json:"error" example:"invalid request body"`
}

// OKWithData возвращает успешный Response с переданными данными.
func OKWithData(data any) OKResponse {
	return OKResponse{
		Status: StatusOK,
		Data:   data,
	}
}

// Error возвращает Response с ошибкой и переданным сообщением.
func Error(msg string) ErrorResponse {
	return ErrorResponse{
		Status: StatusError,
		Error:  msg,
	}
}

// ValidationError формирует Response со статусом Error на основе ошибок валидации.
// Каждое нарушение формируется в человеко‑читаемый текст, объединённый через запятую.
func ValidationError(errs validator.ValidationErrors) ErrorResponse {
	var errsMsgs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s is a required field", err.Field()))
		case "alphanum":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s can contain only numbers and letters", err.Field()))
		case "numeric":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s can contain only numbers", err.Field()))
		case "uuid":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s can contain only uuid", err.Field()))
		case "datetime=01-2006":
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s can contain only date in format 01-2006", err.Field()))
		default:
			errsMsgs = append(errsMsgs, fmt.Sprintf("field %s is not a valid", err.Field()))
		}
	}
	return ErrorResponse{
		Status: StatusError,
		Error:  strings.Join(errsMsgs, ", "),
	}
}
