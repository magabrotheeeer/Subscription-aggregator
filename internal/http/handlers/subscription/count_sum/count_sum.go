package countsum

import (
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
	"github.com/magabrotheeeer/subscription-aggregator/internal/services"
)

type Handler struct {
	log      *slog.Logger
	service  *services.SubscriptionService
	validate *validator.Validate
}

func New(log *slog.Logger, service *services.SubscriptionService) *Handler {
	return &Handler{
		log:      log,
		service:  service,
		validate: validator.New(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.subscription.countsum"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var req models.DummyFilterSum
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("failed to decode request body", sl.Err(err))
		render.JSON(w, r, response.Error("invalid request body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		log.Error("validation failed", sl.Err(err))
		render.JSON(w, r, response.ValidationError(err.(validator.ValidationErrors)))
		return
	}

	username, ok := r.Context().Value(middlewarectx.UserKey).(string)
	if !ok || username == "" {
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}

	sum, err := h.service.CountSumWithFilter(r.Context(), username, req)
	if err != nil {
		log.Error("failed to calculate sum", sl.Err(err))
		render.JSON(w, r, response.Error("could not calculate sum"))
		return
	}

	log.Info("success to calculate sum", slog.Any("sum", sum))
	render.JSON(w, r, response.StatusOKWithData(map[string]any{
		"sum_of_subscriptions": sum,
	}))
}
