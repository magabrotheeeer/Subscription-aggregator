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

type PaymentService interface {
	ListPaymentTokens(ctx context.Context, userUID string) ([]*models.PaymentToken, error)
}

type Handler struct {
	log            *slog.Logger // Логгер для записи информации и ошибок
	paymentService PaymentService
	validate       *validator.Validate // Валидатор структуры входящих данных
}

func New(log *slog.Logger, ps PaymentService) *Handler {
	return &Handler{
		log:            log,
		paymentService: ps,
		validate:       validator.New(),
	}
}

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
	}

	log.Info("list tokens", "count", len(paymentTokens))
	render.JSON(w, r, response.OKWithData(map[string]any{
		"list_count":     len(paymentTokens),
		"payment tokens": paymentTokens,
	}))
}
