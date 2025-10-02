package health

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/cache"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
	"github.com/streadway/amqp"
)

type Handler struct {
	log     *slog.Logger
	storage *storage.Storage
	rabbit  *amqp.Connection
	cache   *cache.Cache
}

func New(log *slog.Logger, storage *storage.Storage, rabbit *amqp.Connection, cache *cache.Cache) *Handler {
	return &Handler{
		log: log,
		storage: storage,
		rabbit: rabbit,
		cache: cache,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.subscription.health"
	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, response.OKWithData(map[string]any{
		"status": "ok",
	}))
}
