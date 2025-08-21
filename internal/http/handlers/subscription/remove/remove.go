// Package remove реализует HTTP-обработчик для удаления подписки пользователя по ID.
//
// Handler извлекает ID из URL-параметров, вызывает бизнес-логику удаления через сервис
// и возвращает количество удалённых записей в JSON-формате.
//
// В случае ошибок формирует соответствующие HTTP-ответы с описанием проблемы.
package remove

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

// Handler обрабатывает HTTP-запросы на удаление подписки по идентификатору.
type Handler struct {
	log     *slog.Logger // Логгер для записи информации и ошибок
	service Service      // Сервис бизнес-логики для удаления подписки
}

// Service описывает интерфейс бизнес-логики удаления подписки.
type Service interface {
	Remove(ctx context.Context, id int) (int, error)
}

// New создает новый Handler с переданным логгером и сервисом.
func New(log *slog.Logger, service Service) *Handler {
	return &Handler{
		log:     log,
		service: service,
	}
}

// ServeHTTP обрабатывает HTTP-запрос на удаление подписки.
//
// Выполняет:
// - Парсинг ID из URL.
// - Вызов бизнес-логики удаления записи по ID.
// - Формирование JSON-ответа с количеством удалённых записей или ошибкой.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.subscription.remove"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Error("invalid id format", sl.Err(err))
		render.JSON(w, r, response.Error("invalid id"))
		return
	}

	res, err := h.service.Remove(r.Context(), id)
	if err != nil {
		log.Error("failed to delete subscription", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("failed to delete subscription"))
		return
	}

	log.Info("success to delete subscription", slog.Any("deleted entries", res))
	render.JSON(w, r, response.StatusOKWithData(map[string]any{
		"deleted_count": res,
	}))
}
