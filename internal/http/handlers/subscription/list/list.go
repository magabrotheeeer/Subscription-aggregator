package list

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
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
	if err != nil || limit <= 0 {
		offset = 0
	}
	username, ok := r.Context().Value(middlewarectx.User).(string)
	if !ok || username == "" {
		log.Error("username not found in context")
		render.JSON(w, r, response.Error("unauthorized"))
		return
	}
	role, ok := r.Context().Value(middlewarectx.Role).(string)
	if !ok || role == "" {
		log.Error("role not found in context")
		render.JSON(w, r, response.Error("unauthorized"))
	}
	res, err := h.service.List(r.Context(), username, role, limit, offset)
	if err != nil {
		log.Error("failed to list entrys", sl.Err(err))
		render.JSON(w, r, response.Error("failed to list"))
		return
	}

	log.Info("list entrys", "count", len(res))
	render.JSON(w, r, response.StatusOKWithData(map[string]any{
		"list_count": len(res),
		"entries":    res,
	}))
}
