// Package response содержит вспомогательные типы и функции для формирования
// унифицированных JSON‑ответов HTTP‑обработчиков. Пакет упрощает возврат
// успешных ответов, ошибок и сообщений валидации в едином формате.
package response

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator"
)

// Response описывает стандартную структуру JSON‑ответа сервера.
// Поле Status — статус запроса ("OK" или "Error").
// Поле Error — текст ошибки (опционально, при неуспехе).
// Поле Data — данные ответа (опционально, при успехе).
type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Data   any    `json:"data,omitempty"`
}

// ErrorResponse — структура ошибки для Swagger-документации.
// Используется в аннотациях @Failure как возвращаемый тип ошибки.
type ErrorResponse struct {
	Status string `json:"status" example:"Error"`
	Error  string `json:"error" example:"invalid request body"`
}

const (
	// StatusOK — значение статуса для успешного ответа.
	StatusOK = "OK"
	// StatusError — значение статуса для ответа с ошибкой.
	StatusError = "Error"
)

// StatusOKWithData возвращает успешный Response с переданными данными.
func StatusOKWithData(data any) Response {
	return Response{
		Status: StatusOK,
		Data:   data,
	}
}

// TODO: сделать корректные статусы возвратов

// Error возвращает Response с ошибкой и переданным сообщением.
func Error(msg string) ErrorResponse {
	return ErrorResponse{
		Status: StatusError,
		Error:  msg,
	}
}

// ValidationError формирует Response со статусом Error на основе ошибок валидации.
// Каждое нарушение формируется в человеко‑читаемый текст, объединённый через запятую.
func ValidationError(errs validator.ValidationErrors) Response {
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
	return Response{
		Status: StatusError,
		Error:  strings.Join(errsMsgs, ", "),
	}
}
