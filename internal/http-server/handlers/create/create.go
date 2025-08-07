package create

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type StorageEntryCreater interface {
	CreateSubscriptionEntry(ctx context.Context, entry subs.SubscriptionEntry) (int, error)
}

type CacheEntryCreator interface {
	Set(key string, value any, expiration time.Duration) error
}

// @Summary Создать подписку
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param data body subs.DummySubscriptionEntry true "Данные новой подписки"
// @Success 201 {object} response.Response "Успешное создание"
// @Failure 400 {object} response.Response "Ошибка валидации"
// @Router /subscriptions/ [post]
func New(ctx context.Context, log *slog.Logger, createrStorage StorageEntryCreater, createrCache CacheEntryCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.create.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
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

		id, err := createrStorage.CreateSubscriptionEntry(ctx, req)

		if err != nil {
			log.Error("failed to create new entry", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
			render.JSON(w, r, response.Error("failed to save"))
			return
		}
		log.Info("created new entry", "last added id:", id)

		cacheKey := fmt.Sprintf("subscription:%d", id)

		if err := createrCache.Set(cacheKey, req, time.Hour); err != nil {
			log.Warn("failed to add to cache", slog.Attr{
				Key:   "err",
				Value: slog.StringValue(err.Error()),
			})
		}
		log.Info("cache updated", slog.String("key", cacheKey))

		render.JSON(w, r, response.StatusOKWithData(map[string]any{
			"last added id": id,
		}))

	}
}
