package read

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

type Reader interface {
	ReadSubscriptionEntryByUserID(ctx context.Context, entry subs.FilterReaderSubscriptionEntry) ([]*subs.FilterReaderSubscriptionEntry, error)
}

func New(ctx context.Context, log *slog.Logger, reader Reader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reade.New"

		log.With(
			"op", op,
			"requires_id", middleware.GetReqID(r.Context()),
		)
		var req subs.FilterReaderSubscriptionEntry

		err := render.DecodeJSON(r.Body, &req)
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

		res, err := reader.ReadSubscriptionEntryByUserID(ctx, req)
		
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
			"read_count": len(res),
			"entries": res,
		}))
	}
}