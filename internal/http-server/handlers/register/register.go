// Package register предоставляет HTTP‑обработчик для регистрации нового пользователя.
// Обработчик принимает логин и пароль в формате JSON, валидирует их,
// хэширует пароль, вызывает бизнес‑логику для создания пользователя
// и возвращает данные о созданном пользователе в ответе.
package register

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/auth"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// Request содержит данные, необходимые для регистрации нового пользователя.
// Поля проходят проверку на обязательность и минимальную/максимальную длину.
type Request struct {
	Username string `json:"username" validate:"required,min=6,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

// Registration определяет контракт для сервиса регистрации пользователей.
// Реализация должна сохранять нового пользователя с указанным username и хэшем пароля,
// возвращая уникальный идентификатор созданной записи или ошибку.
type Registration interface {
	RegisterUser(ctx context.Context, username, passwordHash string) (int, error)
}

// New возвращает HTTP‑обработчик, который обрабатывает POST‑запрос для регистрации нового пользователя.
// Логика работы:
//  1. Декодирует JSON‑тело запроса в структуру RegisterRequest.
//  2. Валидирует входные данные.
//  3. Хэширует пароль.
//  4. Вызывает сервис регистрации пользователя.
//  5. Возвращает клиенту информацию о созданном пользователе.
//
// @Summary Регистрация нового пользователя
// @Tags auth
// @Accept  json
// @Produce json
// @Param   registerRequest body RegisterRequest true "Данные для регистрации (username, password)"
// @Success 200 {object} response.Response "Пользователь успешно создан"
// @Failure 400 {object} response.Response "Ошибка валидации или некорректный запрос"
// @Failure 500 {object} response.Response "Внутренняя ошибка сервера"
// @Router /register [post]
func New(ctx context.Context, log *slog.Logger, registration Registration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.register.New"
		var err error
		var registerRequest Request

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		err = render.DecodeJSON(r.Body, &registerRequest)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, response.Error("failed to decode request"))
			return
		}
		log.Info("request body decoded", slog.Any("request", registerRequest))

		if err := validator.New().Struct(registerRequest); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("Invalid request", sl.Err(err))
			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		log.Info("all fields are validated")

		hash, err := auth.GetHash(registerRequest.Password)
		if err != nil {
			log.Error("failed to register new user", sl.Err(err))
			render.JSON(w, r, response.Error("failed to register new user"))
			return
		}

		id, err := registration.RegisterUser(ctx, registerRequest.Username, hash)
		if err != nil {
			log.Error("failed to register new user", sl.Err(err))
			render.JSON(w, r, response.Error("failed to register new user"))
			return
		}

		log.Info("created new user", "username", registerRequest.Username)
		render.JSON(w, r, response.StatusOKWithData(map[string]any{
			"username": registerRequest.Username,
			"message":  "user created succesfully",
			"id":       id,
		}))
	}
}
