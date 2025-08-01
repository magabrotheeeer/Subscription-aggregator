package remove

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
)

type Deleter interface {
	RemoveSubscriptionEntry(ctx context.Context, id int) (int64, error)
}

// @Summary Удалить подписки
// @Description Удаляет подписки пользователя по service_name или все подписки пользователя если service_name не указан
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param data body subs.FilterRemoverSubscriptionEntry true "Фильтры для удаления подписок"
// @Success 200 {object} map[string]interface{} "deleted_count: число удаленных записей"
// @Failure 400 {object} map[string]interface{} "Ошибка валидации данных"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /subscriptions [delete]
func New(ctx context.Context, log *slog.Logger, deleter Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.remove.New"

		log.With(
			slog.String("op", op),
			slog.String("requires_id", middleware.GetReqID(r.Context())),
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

		counter, err = deleter.RemoveSubscriptionEntry(ctx, id)
		if err != nil {
			log.Error("failed to remove entry/entrys", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
			render.JSON(w, r, response.Error("failed to remove"))
			return
		}
		log.Info("deleted entry/entrys", "count", counter)
		render.JSON(w, r, response.StatusOKWithData(map[string]interface{}{
			"deleted_count": counter,
		}))

	}
}
