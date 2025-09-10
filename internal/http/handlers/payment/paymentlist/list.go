// Package paymentlist обрабатывает получение списка платежных методов.
package paymentlist

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

// Service определяет интерфейс для работы с платежами.
type Service interface {
	ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error)
}

// Handler обрабатывает запросы на получение списка платежных методов.
type Handler struct {
	log            *slog.Logger // Логгер для записи информации и ошибок
	paymentService Service
	validate       *validator.Validate // Валидатор структуры входящих данных
}

// New создает новый экземпляр Handler.
func New(log *slog.Logger, ps Service) *Handler {
	return &Handler{
		log:            log,
		paymentService: ps,
		validate:       validator.New(),
	}
}

// ServeHTTP godoc
// @Summary Получить список платежных токенов
// @Description Возвращает список всех платежных токенов пользователя
// @Tags Payments
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]any "Список платежных токенов"
// @Failure 401 {object} response.ErrorResponse "Пользователь не авторизован"
// @Failure 500 {object} response.ErrorResponse "Ошибка сервера при получении токенов"
// @Router /payments/tokens [get]
// @Security BearerAuth
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.payment.list"
	log := h.log.With(slog.String("op", op))

	userUID, ok := r.Context().Value(middlewarectx.UserUID).(string)
	if !ok || userUID == "" {
		log.Error("user UID not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}

	paymentTokens, err := h.paymentService.ListPaymentTokens(r.Context(), userUID)
	if err != nil {
		log.Error("failed to get payment tokens", sl.Err(err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, response.Error("internal error"))
		return
	}

	log.Info("list tokens", "count", len(paymentTokens))
	render.JSON(w, r, response.OKWithData(map[string]any{
		"list_count":     len(paymentTokens),
		"payment tokens": paymentTokens,
	}))
}
