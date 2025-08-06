package read

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type Reader interface {
	ReadSubscriptionEntry(ctx context.Context, id int) ([]*subs.SubscriptionEntry, error)
}

// @Summary Получить подписку по ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Уникальный ID подписки"
// @Success 200 {object} subs.SubscriptionEntry "Подписка"
// @Failure 400 {object} response.Response "Неверный ID"
// @Failure 404 {object} response.Response "Подписка не найдена"
// @Router /subscriptions/{id} [get]
func New(ctx context.Context, log *slog.Logger, reader Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.read.New"

		log = log.With(
			"op", op,
			"requires_id", middleware.GetReqID(r.Context()),
		)

		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Error("failed to decode id from url", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to decode id from url"))

			return
		}

		res, err := reader.ReadSubscriptionEntry(ctx, id)
		if err != nil {
			log.Error("failed to read entry/entrys", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
			render.JSON(w, r, response.Error("failed to read"))
			return
		}
		log.Info("read entry/entrys", "count", len(res))

		render.JSON(w, r, response.StatusOKWithData(map[string]any{
			"entries": res,
		}))
	}
}

