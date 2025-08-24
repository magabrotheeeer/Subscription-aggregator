// Package login реализует HTTP-обработчик для запросов аутентификации пользователей.
//
// В нём определяется структура Request для входных данных, выполняется декодирование JSON,
// проверка и валидация полей, а также делегирование операции входа (login) клиенту gRPC Service.
// При успешной аутентификации возвращается JSON с JWT и refresh-токеном;
// в случае ошибок формируются соответствующие HTTP-ответы.
package login

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/gen"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// Request — структура входных данных для авторизации.
//
// Username должен быть строкой длиной от 3 до 50 символов, пароль — минимум 6 символов.
type Request struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

// Handler обрабатывает HTTP-запросы для авторизации.
type Handler struct {
	log        *slog.Logger        // Логгер для записи операций и ошибок
	authClient Service             // Клиент для вызова gRPC-сервиса аутентификации
	validate   *validator.Validate // Валидатор для проверки входных данных
}

// Service описывает интерфейс бизнес-логики аутентификации.
//
// Включает метод Login для входа пользователя по username и password.
type Service interface {
	Login(ctx context.Context, username, password string) (*authpb.LoginResponse, error)
}

// New создает новый экземпляр Handler с указанными логгером и клиентом аутентификации.
//
// Инициализирует валидатор для проверки структур.
func New(log *slog.Logger, authClient Service) *Handler {
	return &Handler{
		log:        log,
		authClient: authClient,
		validate:   validator.New(),
	}
}

// ServeHTTP godoc
// @Summary Авторизация пользователя
// @Description Аутентифицирует пользователя по имени и паролю. Возвращает JWT и refresh-токен.
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param request body Request true "Учетные данные пользователя"
// @Success 200 {object} map[string]any "Успешная авторизация"
// @Failure 400 {object} response.ErrorResponse "Некорректный JSON"
// @Failure 422 {object} response.ErrorResponse "Ошибка валидации"
// @Failure 401 {object} response.ErrorResponse "Неверные учетные данные"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /login [post]
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.auth.login"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("failed to decode request body", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, response.Error("invalid request body"))
		return
	}
	log.Info("request body decoded", slog.Any("request", req))

	if err := h.validate.Struct(req); err != nil {
		log.Error("validation failed", sl.Err(err))
		w.WriteHeader(http.StatusUnprocessableEntity)
		render.JSON(w, r, response.ValidationError(err.(validator.ValidationErrors)))
		return
	}
	log.Info("all fields are validated")

	grpcResp, err := h.authClient.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		log.Error("login failed", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("invalid credentials"))
		return
	}

	log.Info("login success", slog.Any("username", req.Username))
	render.JSON(w, r, response.OKWithData(map[string]any{
		"token":         grpcResp.Token,
		"refresh_token": grpcResp.RefreshToken,
		"role":          grpcResp.Role,
		"username":      req.Username,
	}))
}
