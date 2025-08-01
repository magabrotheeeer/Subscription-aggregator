package countsum

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type CounterSum interface {
	CountSumSubscriptionEntrys(ctx context.Context, entry subs.SubscriptionEntry, id int) (float64, error)
}

// @Summary Подсчитать сумму подписок за период для пользователя
// @Description Подсчитывает суммарную стоимость подписок по фильтрам из тела запроса
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Идентификатор пользователя (UUID)"
// @Param data body subs.DummySubscriptionEntry true "Фильтры и даты для подсчёта суммы подписок"
// @Success 200 {object} map[string]interface{} "Сумма подписок"
// @Failure 400 {object} response.Response "Ошибка валидации"
// @Failure 500 {object} response.Response "Ошибка сервера"
// @Router /subscriptions/sum/{id} [post]
func New(ctx context.Context, log *slog.Logger, counterSum CounterSum) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.countersum.New"

		log.With(
			"op", op,
			"requires_id", middleware.GetReqID(r.Context()),
		)

		var dummyReq subs.DummySubscriptionEntry

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

			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		log.Info("all fields are validated")

		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Error("failed to decode id from url", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to decode id from url"))

			return
		}

		startDate, err := time.Parse("01-2006", dummyReq.StartDate)
		if err != nil {
			log.Error("failed to convert, field: startdate", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to convert, field: startdate"))

			return
		}
		endDate, err := time.Parse("01-2006", dummyReq.EndDate)
		if err != nil {
			log.Error("failed to convert, field: enddate", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error())})

			render.JSON(w, r, response.Error("failed to convert, field: enddate"))

			return
		}

		var req subs.SubscriptionEntry

		req.ServiceName = dummyReq.ServiceName
		req.UserID = dummyReq.UserID
		req.Price = dummyReq.Price
		req.StartDate = startDate
		req.EndDate = &endDate

		res, err := counterSum.CountSumSubscriptionEntrys(ctx, req, id)
		if err != nil {
			log.Error("failed to sum", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
			render.JSON(w, r, response.Error("failed to sum"))
			return
		}
		log.Info("sum of subscriptions", "sum", res)
		render.JSON(w, r, response.StatusOKWithData(map[string]interface{}{
			"sum_of_subscriptions": res,
		}))
	}
}