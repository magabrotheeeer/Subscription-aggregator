package list

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/auth"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type List interface {
	ListSubscriptionEntrys(ctx context.Context, username string, limit, offset int) ([]*subs.SubscriptionEntry, error)
}

// @Summary Получить список всех подписок
// @Description Возвращает список подписок с поддержкой пагинации
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param limit query int false "Количество элементов на странице" default(10)
// @Param offset query int false "Смещение от начала списка" default(0)
// @Success 200 {object} map[string]interface{} "Количество и список подписок"
// @Failure 500 {object} response.Response "Ошибка сервера"
// @Router /subscriptions/list [get]
func New(ctx context.Context, log *slog.Logger, list List) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.list.New"

		log = log.With(
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
		username, ok := r.Context().Value(auth.UserKey).(string)
		if !ok || username == "" {
			log.Error("username not found in context")
			render.JSON(w, r, response.Error("unauthorized"))
			return
		}
		res, err := list.ListSubscriptionEntrys(ctx, username, limit, offset)
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
}
