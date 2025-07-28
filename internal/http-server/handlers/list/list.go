package list

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type List interface {
	ListSubscriptionEntrys(ctx context.Context) ([]*subs.ListSubscriptionEntrys, error)
}

func New(ctx context.Context, log *slog.Logger, list List) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.list.New"

		log.With(
			"op", op,
			"requires_id", middleware.GetReqID(r.Context()),
		)

		res, err := list.ListSubscriptionEntrys(ctx)
		if err != nil {
			log.Error("failed to list entrys", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
			render.JSON(w, r, response.Error("failed to list"))
			return
		}

		log.Info("list entrys", "count", len(res))
		render.JSON(w, r, response.StatusOKWithData(map[string]interface{}{
			"list_count": len(res),
			"entries": res,
		}))
	}
}