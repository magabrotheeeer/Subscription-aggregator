package update

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/mware"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type StorageEntryUpdater interface {
	UpdateSubscriptionEntry(ctx context.Context, entry subs.SubscriptionEntry, id int) (int64, error)
}

type CacheEntryUpdater interface {
	Set(key string, value any, expiration time.Duration) error
}

// @Summary Обновить подписку по ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Уникальный ID подписки"
// @Param data body subs.DummySubscriptionEntry true "Данные для обновления подписки"
// @Success 200 {object} map[string]interface{} "Обновлено записей"
// @Failure 400 {object} response.Response "Ошибка валидации"
// @Failure 404 {object} response.Response "Подписка не найдена"
// @Router /subscriptions/{id} [put]
func New(ctx context.Context, log *slog.Logger, updaterStorage StorageEntryUpdater, updaterCache CacheEntryUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.update.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var dummyReq subs.DummySubscriptionEntry
		var err error

		err = render.DecodeJSON(r.Body, &dummyReq)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, response.Error("failed to decode request"))
			return
		}
		log.Info("request body decoded", slog.Any("request", dummyReq))

		if err = validator.New().Struct(dummyReq); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("Invalid request", sl.Err(err))
			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		log.Info("all fields are validated")

		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Error("failed to decode id from url", sl.Err(err))
			render.JSON(w, r, response.Error("failed to decode id from url"))
			return
		}

		var req subs.SubscriptionEntry

		startDate, err2 := time.Parse("01-2006", dummyReq.StartDate)
		if err2 != nil {
			log.Error("failed to convert, field: startdate", sl.Err(err2))
			render.JSON(w, r, response.Error("failed to convert, field: startdate"))
			return
		}
		if dummyReq.EndDate != "" {
			endDate, err2 := time.Parse("01-2006", dummyReq.EndDate)
			if err2 != nil {
				log.Error("failed to convert, field: enddate", sl.Err(err2))
				render.JSON(w, r, response.Error("failed to convert, field: enddate"))
				return
			}
			req.EndDate = &endDate
		} else {
			req.EndDate = nil
		}

		var counter int64

		req.ServiceName = dummyReq.ServiceName
		req.Username = r.Context().Value(mware.UserKey).(string)
		req.Price = dummyReq.Price
		req.StartDate = startDate

		counter, err = updaterStorage.UpdateSubscriptionEntry(ctx, req, id)
		if err != nil {
			log.Error("failed to update entry/entrys in storage", sl.Err(err))
			render.JSON(w, r, response.Error("failed to update"))
			return
		}

		log.Info("update entry/entrys in storage", "count", counter)

		cacheKey := fmt.Sprintf("subscription:%d", id)

		if err := updaterCache.Set(cacheKey, req, time.Hour); err != nil {
			log.Warn("failed to update entry/entrys in cache", sl.Err(err))
		}
		log.Info("cache updated", slog.String("key", cacheKey))

		render.JSON(w, r, response.StatusOKWithData(map[string]any{
			"updated_count": counter,
		}))
	}
}
