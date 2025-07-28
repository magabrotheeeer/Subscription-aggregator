package update

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

type Updater interface {
	UpdateSubscriptionEntryPriceByServiceName(ctx context.Context, entry subs.FilterUpdaterSubscriptionEntry) (int64, error)
	UpdateSubscriptionEntryDateByServiceName(ctx context.Context, entry subs.FilterUpdaterSubscriptionEntry) (int64, error)
}

// @Summary Обновить подписку
// @Description Обновляет цену или даты подписки пользователя по service_name. Если указана цена - обновляется цена, иначе обновляются даты
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param data body subs.DummyFilterUpdaterSubscriptionEntry true "Данные для обновления подписки"
// @Success 200 {object} map[string]interface{} "updated_count: число обновленных записей"
// @Failure 400 {object} map[string]interface{} "Ошибка валидации данных или конвертации дат"
// @Failure 500 {object} map[string]interface{} "Внутренняя ошибка сервера"
// @Router /api/v1/subscriptions [put]
func New(ctx context.Context, log *slog.Logger, update Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.update.New"

		log.With(
			"op", op,
			"requires_id", middleware.GetReqID(r.Context()),
		)

		var dummyReq subs.DummyFilterUpdaterSubscriptionEntry
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

		if err = validator.New().Struct(dummyReq); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("Invalid request", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})

			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		log.Info("all fields are validated")

		var counter int64
		var req subs.FilterUpdaterSubscriptionEntry
		req.ServiceName = dummyReq.ServiceName
		req.UserID = dummyReq.UserID

		if dummyReq.Price != 0 {
			req.Price = dummyReq.Price
			counter, err = update.UpdateSubscriptionEntryPriceByServiceName(ctx, req)
		}else {
			startDate, err2 := time.Parse("01-2006", dummyReq.StartDate)
			if err2 != nil {
				log.Error("failed to convert, field: startdate", slog.Attr{
					Key:   "err",
					Value: slog.StringValue(err2.Error())})

				render.JSON(w, r, response.Error("failed to convert, field: startdate"))
				return
			}
			endDate, err2 := time.Parse("01-2006", dummyReq.EndDate)
			if err2 != nil {
				log.Error("failed to convert, field: enddate", slog.Attr{
					Key:   "err",
					Value: slog.StringValue(err2.Error())})

				render.JSON(w, r, response.Error("failed to convert, field: enddate"))
				return
			}
			req.StartDate = startDate
			req.EndDate = &endDate
			counter, err = update.UpdateSubscriptionEntryDateByServiceName(ctx, req)
		}
		if err != nil {
			log.Error("failed to update entry/entrys", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
			render.JSON(w, r, response.Error("failed to update"))
			return
		}

		log.Info("update entry/entrys", "count", counter)
		render.JSON(w, r, response.StatusOKWithData(map[string]interface{}{
			"updated_count": counter,
		}))
	}
}