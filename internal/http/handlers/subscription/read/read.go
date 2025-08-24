// Package read реализует HTTP-обработчик для получения конкретной подписки по ID.
//
// Handler извлекает ID из URL-параметров, вызывает бизнес-логику для чтения подписки по идентификатору
// и возвращает данные подписки в JSON-формате.
//
// В случае ошибок формирует соответствующие HTTP-ответы с описанием проблемы.
package read

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// Handler обрабатывает запросы на получение подписки по уникальному идентификатору.
type Handler struct {
	log      *slog.Logger        // Логгер для записи информации и ошибок
	service  Service             // Сервис бизнес-логики для получения подписки по ID
	validate *validator.Validate // Валидатор (в текущей реализации не используется)
}

// Service описывает интерфейс бизнес-логики чтения подписки.
type Service interface {
	Read(ctx context.Context, id int) (*models.Entry, error)
}

// New создает новый Handler с переданным логгером и сервисом.
func New(log *slog.Logger, service Service) *Handler {
	return &Handler{
		log:      log,
		service:  service,
		validate: validator.New(),
	}
}

// ServeHTTP godoc
// @Summary Получить подписку по ID
// @Description Возвращает подписку по её уникальному идентификатору.
// @Tags Subscriptions
// @Accept  json
// @Produce  json
// @Param id path int true "ID подписки"
// @Success 200 {object} map[string]any "Успешный ответ с данными"
// @Failure 400 {object} response.ErrorResponse "Некорректный ID"
// @Failure 404 {object} response.ErrorResponse "Подписка не найдена"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /subscriptions/{id} [get]
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.read.New"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("failed to decode id from url", sl.Err(err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, response.Error("failed to decode id from url"))
		return
	}

	res, err := h.service.Read(r.Context(), id)
	if err != nil {
		log.Error("failed to read subscription", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("could not read subscription"))
		return
	}

	log.Info("success to read subscriptions", slog.Any("entry", res))
	render.JSON(w, r, response.StatusOKWithData(map[string]any{
		"entry": res,
	}))
}
