// Package create реализует HTTP-обработчик для создания новых подписок пользователя.
//
// Handler принимает JSON-запрос с данными подписки, валидирует их, извлекает имя пользователя из контекста,
// вызывает бизнес-логику создания подписки через сервис и возвращает ID созданной записи в JSON-формате.
//
// В случае ошибок формируются соответствующие HTTP-ответы с описанием проблемы.
package create

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// Handler управляет HTTP-запросами на создание новых подписок.
//
// Использует логгер для записи операций и ошибок,
// сервис бизнес-логики для создания подписки,
// а также валидатор для проверки структуры входных данных.
type Handler struct {
	log      *slog.Logger        // Логгер для записи информации и ошибок
	service  Service             // Сервис бизнес-логики для создания подписок
	validate *validator.Validate // Валидатор структуры входящих данных
}

// Service описывает интерфейс бизнес-логики создания подписки.
type Service interface {
	Create(ctx context.Context, userName string, req models.DummyEntry) (int, error)
}

// New создает новый Handler с переданными логгером и сервисом.
func New(log *slog.Logger, service Service) *Handler {
	return &Handler{
		log:      log,
		service:  service,
		validate: validator.New(),
	}
}

// ServeHTTP godoc
// @Summary Создать новую подписку
// @Description Создает новую подписку для текущего пользователя. Возвращает ID созданной записи.
// @Tags Subscriptions
// @Accept  json
// @Produce  json
// @Param request body models.DummyEntry true "Данные новой подписки"
// @Success 200 {object} map[string]any "Успешное создание подписки"
// @Failure 400 {object} response.ErrorResponse "Некорректный JSON"
// @Failure 401 {object} response.ErrorResponse "Пользователь не авторизован"
// @Failure 422 {object} response.ErrorResponse "Ошибка валидации"
// @Failure 500 {object} response.ErrorResponse "Ошибка сервера при создании подписки"
// @Router /subscriptions [post]
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.subscription.create"
	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var req models.DummyEntry
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("failed to decode request", sl.Err(err))
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

	username, ok := r.Context().Value(middlewarectx.User).(string)
	if !ok || username == "" {
		log.Error("username not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}

	id, err := h.service.Create(r.Context(), username, req)
	if err != nil {
		log.Error("failed to create subscription", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("could not create subscription"))
		return
	}

	log.Info("succes to create subscriptions", slog.Any("id", id))
	render.JSON(w, r, response.OKWithData(map[string]any{
		"last_added_id": id,
	}))
}
