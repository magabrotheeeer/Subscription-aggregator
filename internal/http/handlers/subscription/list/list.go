// Package list реализует HTTP-обработчик для получения списка подписок пользователя с пагинацией.
//
// Handler извлекает параметры limit и offset из query строки, получает имя пользователя и роль из контекста,
// вызывает бизнес-логику получения списка подписок через сервис и возвращает результат в JSON-формате.
//
// При ошибках возвращает соответствующие HTTP-статусы и описания ошибок в ответах.
package list

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// Handler обрабатывает запросы на получение списка подписок.
//
// Использует логгер для ведения журнала, сервис бизнес-логики для выборки данных
// и валидатор (хотя в текущей реализации не применяется для параметров).
type Handler struct {
	log      *slog.Logger        // Логгер для записи информации и ошибок
	service  Service             // Сервис бизнес-логики получения списка записей
	validate *validator.Validate // Валидатор входных параметров (не используется в ServeHTTP)
}

// Service описывает интерфейс бизнес-логики получения списка подписок с параметрами пагинации и фильтрации.
type Service interface {
	List(ctx context.Context, username, role string, limit, offset int) ([]*models.Entry, error)
}

// New создает новый Handler с переданными логгером и бизнес-сервисом.
func New(log *slog.Logger, service Service) *Handler {
	return &Handler{
		log:      log,
		service:  service,
		validate: validator.New(),
	}
}

// ServeHTTP обрабатывает HTTP-запрос на получение списка подписок.
//
// Выполняет:
// - Парсинг параметров limit и offset из query строки с дефолтными значениями.
// - Извлечение имени пользователя и роли из контекста запроса.
// - Вызов сервиса получения списка подписок.
// - Формирование JSON-ответа с результатом или ошибкой.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.list.New"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offsetStr := r.URL.Query().Get("offset")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	username, ok := r.Context().Value(middlewarectx.User).(string)
	if !ok || username == "" {
		log.Error("username not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}
	role, ok := r.Context().Value(middlewarectx.Role).(string)
	if !ok || role == "" {
		log.Error("role not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}
	res, err := h.service.List(r.Context(), username, role, limit, offset)
	if err != nil {
		log.Error("failed to list entries", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("failed to list"))
		return
	}

	log.Info("list entries", "count", len(res))
	render.JSON(w, r, response.StatusOKWithData(map[string]any{
		"list_count": len(res),
		"entries":    res,
	}))
}
