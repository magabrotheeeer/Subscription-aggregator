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

type Handler struct {
	log     *slog.Logger
	service Service
}

type Service interface {
	Remove(ctx context.Context, id int) (int, error)
}

func New(log *slog.Logger, service Service) *Handler {
	return &Handler{
		log:     log,
		service: service,
	}
}

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

	log.Info("success to delete subscription", slog.Any("deleted entrys:", res))
	render.JSON(w, r, response.StatusOKWithData(map[string]any{
		"deleted_count": res,
	}))
}
