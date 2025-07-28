package remove

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type Deleter interface {
	RemoveSubscriptionEntryByUserID(ctx context.Context, entry subs.FilterRemoveSubscriptionEntry) (int64, error)
	RemoveSubscriptionEntryByServiceName(ctx context.Context, entry subs.FilterRemoveSubscriptionEntry) (int64, error)
}

func New(ctx context.Context, log *slog.Logger, deleter Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.remove.New"

		log.With(
			slog.String("op", op),
			slog.String("requires_id", middleware.GetReqID(r.Context())),
		)
		var req subs.FilterRemoveSubscriptionEntry
		var err error


		err = render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to decode request"))

			return
		}
		log.Info("request body decoded", slog.Any("request", req))


		if err = validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("Invalid request", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})

			render.JSON(w, r, response.Error("Invalid request"))
			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		log.Info("all fields are validated")
		var counter int64

		if req.ServiceName != "" {
			counter, err = deleter.RemoveSubscriptionEntryByServiceName(ctx, req)
		} else {
			counter, err = deleter.RemoveSubscriptionEntryByUserID(ctx, req)
		}
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
