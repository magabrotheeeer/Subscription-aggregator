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

// @Summary Получить подписки по фильтру
// @Description Возвращает список подписок пользователя согласно заданным фильтрам
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param data body subs.FilterReaderSubscriptionEntry true "Фильтры для поиска подписок"
// @Success 200 {object} map[string]interface{} "read_count: число, entries: массив подписок"
// @Failure 400 {object} map[string]interface{} "Ошибка валидации данных"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /subscriptions/filter [post]
func New(ctx context.Context, log *slog.Logger, reader Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reade.New"

		log.With(
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

		render.JSON(w, r, response.StatusOKWithData(map[string]interface{}{
			"entries": res,
		}))
	}
}