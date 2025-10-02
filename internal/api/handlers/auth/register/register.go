// Package register реализует HTTP-обработчик для регистрации новых пользователей.
package register

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"

	"github.com/magabrotheeeer/subscription-aggregator/internal/api/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// Request — входные данные для регистрации
type Request struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
	Email    string `json:"email" validate:"required"`
}

// Handler обрабатывает HTTP-запросы входа пользователей.
//
// Включает логгер для записи операций, клиент для вызова gRPC-сервиса аутентификации
// и валидатор для проверки входящих данных.
type Handler struct {
	log                 *slog.Logger
	authClient          AuthService
	subscriptionService SubscriptionService
	validate            *validator.Validate
}

// AuthService определяет методы бизнес-логики для работы с пользователями.
//
// В данном случае включает регистрацию пользователя с учётом
// email, имени пользователя и пароля.
type AuthService interface {
	Register(ctx context.Context, email, username, password string) (string, error)
}

// SubscriptionService определяет интерфейс для работы с подписками.
type SubscriptionService interface {
	CreateEntrySubscriptionAggregator(ctx context.Context, userName, userUID string) (int, error)
}

// New создает новый экземпляр Handler с заданным логгером и клиентом аутентификации.
//
// Инициализирует валидатор для проверки входных данных запросов.
func New(log *slog.Logger, authClient AuthService, subscriptionService SubscriptionService) *Handler {
	return &Handler{
		log:                 log,
		authClient:          authClient,
		subscriptionService: subscriptionService,
		validate:            validator.New(),
	}
}

// ServeHTTP godoc
// @Summary Регистрация нового пользователя
// @Description Создает нового пользователя по email, username и password
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param request body Request true "Данные нового пользователя"
// @Success 200 {object} map[string]any "Успешная регистрация"
// @Failure 400 {object} response.ErrorResponse "Некорректный JSON"
// @Failure 422 {object} response.ErrorResponse "Ошибка валидации данных"
// @Failure 500 {object} response.ErrorResponse "Ошибка сервера при регистрации"
// @Router /register [post]
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.auth.register"

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

	userUID, err := h.authClient.Register(r.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		log.Error("registration failed", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("failed to register user"))
		return
	}
	_, err = h.subscriptionService.CreateEntrySubscriptionAggregator(r.Context(), req.Username, userUID)
	if err != nil {
		log.Error("failed to create entry with trial period", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("failed to create entry with trial period"))
		return
	}

	log.Info("register success", slog.Any("username", req.Username), slog.Any("email", req.Email))
	render.JSON(w, r, response.OKWithData(map[string]any{
		"message":  "user created successfully",
		"username": req.Username,
		"email":    req.Email,
	}))
}
