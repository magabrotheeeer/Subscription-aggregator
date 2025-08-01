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
	CreateSubscriptionEntry(ctx context.Context, entry subs.SubscriptionEntry) (int, error)
}

// @Summary Создать подписку
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param data body subs.DummyCreaterSubscriptionEntry true "Новая подписка"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /subscriptions [post]
func New(ctx context.Context, log *slog.Logger, creater Creater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.create.New"

		log = log.With(
			slog.String("op", op),
			slog.String("requires_id", middleware.GetReqID(r.Context())),
		)

		var dummyReq subs.DummySubscriptionEntry
		var err error

		err = render.DecodeJSON(r.Body, &dummyReq)
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

			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		log.Info("all fields are validated")

		var req subs.SubscriptionEntry

		startDate, err := time.Parse("01-2006", dummyReq.StartDate)
		if err != nil {
			log.Error("failed to convert, field: startdate", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to convert, field: startdate"))

			return
		}

		if dummyReq.EndDate == "" {
			req.EndDate = nil
		} else {
			endDate, err := time.Parse("01-2006", dummyReq.EndDate)
			if err != nil {
				log.Error("failed to convert, field: enddate", slog.Attr{
					Key:   "err",
					Value: slog.StringValue(err.Error())})

				render.JSON(w, r, response.Error("failed to convert, field: enddate"))

				return
			}
			req.EndDate = &endDate
		}

		req.ServiceName = dummyReq.ServiceName
		req.UserID = dummyReq.UserID
		req.Price = dummyReq.Price
		req.StartDate = startDate

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