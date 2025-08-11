package countsum

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/mware"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
)

type CounterSum interface {
	CountSumSubscriptionEntrys(ctx context.Context, entry SubscriptionFilterSum) (float64, error)
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

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var filterReq DummySubscriptionFilterSum

		err := render.DecodeJSON(r.Body, &filterReq)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, response.Error("failed to decode request"))
			return
		}
		log.Info("request body decoded", slog.Any("request", filterReq))

		if err := validator.New().Struct(filterReq); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("Invalid request", sl.Err(err))
			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		log.Info("all fields are validated")

		var filter SubscriptionFilterSum

		startDate, err := time.Parse("01-2006", filterReq.StartDate)
		if err != nil {
			log.Error("failed to convert, field: startdate", sl.Err(err))
			render.JSON(w, r, response.Error("failed to convert, field: startdate"))
			return
		}
		var endDate *time.Time
		if filterReq.EndDate == "" {
			filter.EndDate = nil
		} else {
			endDate, err := time.Parse("01-2006", filterReq.EndDate)
			if err != nil {
				log.Error("failed to convert, field: enddate", sl.Err(err))
				render.JSON(w, r, response.Error("failed to convert, field: enddate"))
				return
			}
			filter.EndDate = &endDate
		}
		if filterReq.ServiceName == "" {
			filter.ServiceName = nil
		} else {
			filter.ServiceName = &filterReq.ServiceName
		}

		username, ok := r.Context().Value(mware.UserKey).(string)
		if !ok || username == "" {
			log.Error("username not found in context")
			render.JSON(w, r, response.Error("unauthorized"))
			return
		}
		filter.Username = username
		filter.StartDate = startDate
		filter.EndDate = endDate

		res, err := counterSum.CountSumSubscriptionEntrys(ctx, filter)
		if err != nil {
			log.Error("failed to sum", sl.Err(err))
			render.JSON(w, r, response.Error("failed to sum"))
			return
		}
		log.Info("sum of subscriptions", "sum", res)
		render.JSON(w, r, response.StatusOKWithData(map[string]any{
			"sum_of_subscriptions": res,
		}))
	}
}
