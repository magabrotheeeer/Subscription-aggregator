// Package update реализует HTTP-обработчик для обновления данных подписки пользователя по ID.
//
// Handler принимает JSON-запрос с обновлёнными данными подписки, валидирует их,
// извлекает имя пользователя и ID из контекста и URL-параметров,
// вызывает бизнес-логику обновления через сервис и возвращает количество обновлённых записей в формате JSON.
//
// В случае ошибок формирует соответствующие HTTP-ответы с описанием проблемы.
package update

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/api/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// Handler отвечает за обработку запросов на обновление подписки.
type Handler struct {
	log      *slog.Logger        // Логгер для ведения журналов и ошибок
	service  Service             // Сервис бизнес-логики обновления подписок
	validate *validator.Validate // Валидатор для проверки входных данных
}

// Service описывает интерфейс бизнес-логики обновления подписки.
type Service interface {
	UpdateEntry(ctx context.Context, req models.DummyEntry, id int, username string) (int, error)
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
// @Summary Обновить подписку по ID
// @Description Обновляет данные существующей подписки пользователя по идентификатору.
// @Tags Subscriptions
// @Accept  json
// @Produce  json
// @Param id path int true "ID подписки"
// @Param request body models.DummyEntry true "Обновлённые данные подписки"
// @Success 200 {object} map[string]any "Успешное обновление"
// @Failure 400 {object} response.ErrorResponse "Некорректный ID или JSON"
// @Failure 401 {object} response.ErrorResponse "Пользователь не авторизован"
// @Failure 422 {object} response.ErrorResponse "Ошибка валидации"
// @Failure 404 {object} response.ErrorResponse "Подписка не найдена"
// @Failure 500 {object} response.ErrorResponse "Ошибка сервера при обновлении"
// @Router /subscriptions/{id} [put]
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.update.New"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var req models.DummyEntry
	var err error

	err = render.DecodeJSON(r.Body, &req)
	if err != nil {
		log.Error("failed to decode request body", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, response.Error("failed to decode request"))
		return
	}
	log.Info("request body decoded", slog.Any("request", req))

	if err = h.validate.Struct(req); err != nil {
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

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("failed to decode id from url", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, response.Error("failed to decode id from url"))
		return
	}

	counter, err := h.service.UpdateEntry(r.Context(), req, id, username)
	if err != nil {
		log.Error("failed to update subscription", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("could not update subscription"))
		return
	}

	log.Info("success to update subscription", slog.Any("updated count", counter))
	render.JSON(w, r, response.OKWithData(map[string]any{
		"updated_count": counter,
	}))
}
