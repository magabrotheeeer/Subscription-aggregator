// Package read предоставляет HTTP‑обработчик для получения данных одной подписки по ID.
// Обработчик сначала пытается найти данные в кэше, а при отсутствии — запрашивает из хранилища,
// и возвращает результат в формате JSON.
package read

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

// StorageEntryReader определяет контракт для чтения подписки по её уникальному ID из хранилища.
type StorageEntryReader interface {
	ReadSubscriptionEntry(ctx context.Context, id int) (*subs.Entry, error)
}

// CacheEntryReader определяет контракт для получения данных подписки из кэша по ключу.
type CacheEntryReader interface {
	Get(key string, result any) (bool, error)
}

// New возвращает HTTP‑обработчик, который обрабатывает GET‑запрос на получение подписки по ID.
// Логика работы:
//  1. Считывает ID подписки из пути запроса.
//  2. Пытается найти данные в кэше.
//  3. Если в кэше нет — запрашивает из хранилища.
//  4. Возвращает данные подписки в формате JSON.
//
// @Summary Получить подписку по ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Уникальный ID подписки"
// @Success 200 {object} subs.SubscriptionEntry "Подписка"
// @Failure 400 {object} response.Response "Неверный ID"
// @Failure 404 {object} response.Response "Подписка не найдена"
// @Router /subscriptions/{id} [get]
func New(ctx context.Context, log *slog.Logger, readerStorage StorageEntryReader, readerCache CacheEntryReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.read.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			log.Error("failed to decode id from url", sl.Err(err))
			render.JSON(w, r, response.Error("failed to decode id from url"))
			return
		}

		var res *subs.Entry
		cacheKey := fmt.Sprintf("subscription:%d", id)

		found, err := readerCache.Get(cacheKey, &res)
		if err != nil {
			log.Error("failed to read from cache", sl.Err(err))
			render.JSON(w, r, response.Error("internal error"))
			return
		}

		if found {
			log.Info("read entry/entrys from cache", "count", 1)
		} else {
			res, err = readerStorage.ReadSubscriptionEntry(ctx, id)
			if err != nil {
				log.Error("failed to read entry/entrys", sl.Err(err))
				render.JSON(w, r, response.Error("failed to read"))
				return
			}
			log.Info("read entry/entrys from storage", "count", 1)
		}

		render.JSON(w, r, response.StatusOKWithData(map[string]any{
			"entries": res,
		}))
	}
}
