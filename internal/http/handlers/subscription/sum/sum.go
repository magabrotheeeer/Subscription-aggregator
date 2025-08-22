// Package sum реализует HTTP-обработчик для подсчёта общей суммы подписок пользователя.
//
// Handler принимает JSON-запрос с фильтром, валидирует его, извлекает имя пользователя из контекста,
// вызывает бизнес-логику подсчёта суммы через сервис и возвращает результат в JSON-формате.
//
// В случае ошибок формируются соответствующие HTTP-ответы с описанием проблемы.
package sum

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

// Handler управляет HTTP-запросами на подсчёт суммы подписок.
//
// Использует логгер для журналирования, сервис для бизнес-логики и валидатор для проверки структуры запроса.
type Handler struct {
	log      *slog.Logger        // Логгер для записи информации и ошибок
	service  Service             // Сервис бизнес-логики для подсчёта суммы с фильтрами
	validate *validator.Validate // Валидатор структуры входящих данных
}

// Service описывает интерфейс бизнес-логики подсчёта суммы подписок с фильтрами.
type Service interface {
	CountSumWithFilter(ctx context.Context, username string, req models.DummyFilterSum) (float64, error)
}

// New создаёт новый Handler с переданным логгером и сервисом подсчёта.
func New(log *slog.Logger, service Service) *Handler {
	return &Handler{
		log:      log,
		service:  service,
		validate: validator.New(),
	}
}

// ServeHTTP обрабатывает HTTP-запрос к эндпоинту подсчёта суммы подписок.
//
// Выполняет:
// - Декодирование JSON с фильтром из тела запроса.
// - Валидацию фильтра.
// - Извлечение имени пользователя из контекста запроса.
// - Вызов сервиса подсчёта суммы.
// - Возврат результата или ошибочного ответа в формате JSON.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.subscription.countsum"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var req models.DummyFilterSum
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("failed to decode request body", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, response.Error("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		log.Error("validation failed", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(err.(validator.ValidationErrors)))
		return
	}

	username, ok := r.Context().Value(middlewarectx.User).(string)
	if !ok || username == "" {
		log.Error("username not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}

	sum, err := h.service.CountSumWithFilter(r.Context(), username, req)
	if err != nil {
		log.Error("failed to calculate sum", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("could not calculate sum"))
		return
	}

	log.Info("success to calculate sum", slog.Any("sum", sum))
	render.JSON(w, r, response.StatusOKWithData(map[string]any{
		"sum_of_subscriptions": sum,
	}))
}
