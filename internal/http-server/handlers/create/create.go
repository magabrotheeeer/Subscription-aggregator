package create

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type Creater interface {
	CreateSubscriptionEntry(ctx context.Context, entry subs.CreaterSubscriptionEntry) (int, error)
}

func New(ctx context.Context, log *slog.Logger, creater Creater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.create.New"

		log = log.With(
			slog.String("op", op),
			slog.String("requires_id", middleware.GetReqID(r.Context())),
		)

		var dummyReq subs.DummyCreaterSubscriptionEntry

		err := render.DecodeJSON(r.Body, &dummyReq)
		if err != nil {
			log.Error("failed to decode request body", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to decode request"))

			return
		}
		log.Info("request body decoded", slog.Any("request", dummyReq))

		if err := validator.New().Struct(dummyReq); err != nil {
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

		startDate, err := time.Parse("01-2006", dummyReq.StartDate)
		if err != nil {
			log.Error("failed to convert, field: startdate", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to convert, field: startdate"))

			return
		}

		var req subs.CreaterSubscriptionEntry

		if dummyReq.EndDate != "" {
			endDate, err := time.Parse("01-2006", dummyReq.EndDate)
			if err != nil {
				log.Error("failed to convert, field: enddate", slog.Attr{
					Key:   "err",
					Value: slog.StringValue(err.Error())})

				render.JSON(w, r, response.Error("failed to convert, field: enddate"))

				return
			}

			req.EndDate = &endDate
		} else {
			req.EndDate = nil
		}
		req.StartDate = startDate
		req.UserID = dummyReq.UserID
		req.Price = dummyReq.Price
		req.ServiceName = dummyReq.ServiceName

		counter, err := creater.CreateSubscriptionEntry(ctx, req)
		if err != nil {
			log.Error("failed to create new entry", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
			render.JSON(w, r, response.Error("failed to save"))
			return
		}
		log.Info("created new entry", "entrys in table:", counter)
		render.JSON(w, r, response.StatusOKWithData(map[string]interface{}{
			"entrys_count": counter,
		}))

	}
}
