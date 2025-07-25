package create

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type Creater interface {
	CreateSubscriptionEntry(ctx context.Context, serviceName string, price int,
		userID string, startDate time.Time, endDate time.Time) (int, error)
}

func New(log *slog.Logger, creater Creater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.create.New"

		log = log.With(
			slog.String("op", op),
			slog.String("requires_id", middleware.GetReqID(r.Context())),
		)

		var req subscription.SubscriptionEntry

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", slog.Attr{Key: "err", Value: slog.StringValue(err.Error())})

			//render.JSON(w, r, )
		}
	}
}
