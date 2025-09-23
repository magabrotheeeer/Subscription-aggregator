package health

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
)

type Handler struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Handler {
	return &Handler{
		log: log,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.subscription.health"
	w.WriteHeader(http.StatusOK)	
	render.JSON(w, r, response.OKWithData(map[string]any{
		"status": "ok",
	}))
}