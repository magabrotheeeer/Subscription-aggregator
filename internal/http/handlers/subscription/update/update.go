package update

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
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
		render.JSON(w, r, response.Error("failed to decode request"))
		return
	}
	log.Info("request body decoded", slog.Any("request", req))

	if err = h.validate.Struct(req); err != nil {
		log.Error("validation failed", sl.Err(err))
		render.JSON(w, r, response.ValidationError(err.(validator.ValidationErrors)))
		return
	}
	log.Info("all fields are validated")

	username, ok := r.Context().Value(middlewarectx.UserKey).(string)
	if !ok || username == "" {
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("failed to decode id from url", sl.Err(err))
		render.JSON(w, r, response.Error("failed to decode id from url"))
		return
	}

	counter, err := h.service.Update(r.Context(), req, id, username)
	if err != nil {
		log.Error("failed to update subscription", sl.Err(err))
		render.JSON(w, r, response.Error("could not update subscription"))
		return
	}

	render.JSON(w, r, response.StatusOKWithData(map[string]any{
		"updated_count": counter,
	}))
}
