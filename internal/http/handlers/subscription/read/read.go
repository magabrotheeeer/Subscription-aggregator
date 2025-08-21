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

type Handler struct {
	log      *slog.Logger
	service  Service
	validate *validator.Validate
}

type Service interface {
	Read(ctx context.Context, id int) (*models.Entry, error)
}

func New(log *slog.Logger, service Service) *Handler {
	return &Handler{
		log:      log,
		service:  service,
		validate: validator.New(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.read.New"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("failed to decode id from url", sl.Err(err))
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
