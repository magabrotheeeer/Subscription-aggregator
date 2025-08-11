// Package login предоставляет HTTP‑обработчик для авторизации пользователя.
// Обработчик принимает логин и пароль, валидирует входные данные,
// проверяет наличие пользователя и корректность пароля,
// а затем генерирует JWT‑токен при успешной аутентификации.
package login

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
	"github.com/magabrotheeeer/subscription-aggregator/internal/user"
)

// Request содержит данные для авторизации пользователя.
// Поля валидируются на минимальную и максимальную длину.
type Request struct {
	Username string `json:"username" validate:"required,min=6,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserGetter определяет контракт для получения данных пользователя
// по его имени пользователя (username).
type UserGetter interface {
	GetUserByUsername(ctx context.Context, username string) (*user.User, error)
}

// New возвращает HTTP‑обработчик, который обрабатывает POST‑запрос на авторизацию пользователя.
// Обработчик декодирует JSON с логином и паролем, валидирует данные,
// проверяет существование пользователя и соответствие пароля,
// генерирует JWT‑токен и возвращает его в ответе.
//
// @Summary Авторизация пользователя (логин)
// @Tags auth
// @Accept  json
// @Produce json
// @Param   loginRequest body LoginRequest true "Данные для входа (username, password)"
// @Success 200 {object} map[string]string "JWT токен в поле token"
// @Failure 400 {object} response.Response "Ошибка валидации или некорректный запрос"
// @Failure 401 {object} response.Response "Некорректный пользователь или пароль"
// @Failure 500 {object} response.Response "Внутренняя ошибка сервера"
// @Router /login [post]
func New(ctx context.Context, log *slog.Logger, userGetter UserGetter, jwtMaker auth.JWTMaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.login.New"
		var err error
		var loginRequest Request

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		err = render.DecodeJSON(r.Body, &loginRequest)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, response.Error("failed to decode request"))
			return
		}
		log.Info("request body decoded", slog.Any("request", loginRequest))

		if err := validator.New().Struct(loginRequest); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("Invalid request", sl.Err(err))
			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		log.Info("all fields are validated")

		user, err := userGetter.GetUserByUsername(ctx, loginRequest.Username)
		if err != nil {
			log.Error("incorrect user or password", sl.Err(err))
			render.JSON(w, r, response.Error("incorrect user or password"))
			return
		}

		err = auth.CompareHash(user.PasswordHash, loginRequest.Password)
		if err != nil {
			log.Error("incorrect user or password", sl.Err(err))
			render.JSON(w, r, response.Error("incorrect user or password"))
			return
		}

		token, err := jwtMaker.GenerateToken(user.Username)
		if err != nil {
			log.Error("could not generate token", sl.Err(err))
			render.JSON(w, r, response.Error("could not generate token"))
			return
		}
		log.Info("created token", "token", token)
		render.JSON(w, r, response.StatusOKWithData(map[string]any{
			"token": token,
		}))
	}
}
