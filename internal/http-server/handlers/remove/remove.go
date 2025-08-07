package remove

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
)

type StorageEntryDeleter interface {
	RemoveSubscriptionEntry(ctx context.Context, id int) (int64, error)
}

type CacheEntryDeleter interface {
	Invalidate(key string) error
}

// @Summary Удалить подписку по ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Уникальный ID подписки"
// @Success 200 {object} map[string]interface{} "Количество удалённых записей"
// @Failure 400 {object} response.Response "Ошибка валидации"
// @Failure 404 {object} response.Response "Подписка не найдена"
// @Router /subscriptions/{id} [delete]
func New(ctx context.Context, log *slog.Logger, deleterStorage StorageEntryDeleter, deleterCache CacheEntryDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.remove.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var err error

		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Error("failed to decode id from url", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to decode id from url"))

			return
		}

		var counter int64
		cacheKey := fmt.Sprintf("subscription:%d", id)
		err = deleterCache.Invalidate(cacheKey)
		if err != nil {
			log.Error("failed to remove entry/entrys from cache", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
			render.JSON(w, r, response.Error("failed to remove from cache"))
		}
		log.Info("deleted entry/entrys from cache", "count", counter)

		counter, err = deleterStorage.RemoveSubscriptionEntry(ctx, id)
		if err != nil {
			log.Error("failed to remove entry/entrys from storage", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
			render.JSON(w, r, response.Error("failed to remove"))
			return
		}
		log.Info("deleted entry/entrys", "count", counter)
		render.JSON(w, r, response.StatusOKWithData(map[string]any{
			"deleted_count": counter,
		}))

	}
}
