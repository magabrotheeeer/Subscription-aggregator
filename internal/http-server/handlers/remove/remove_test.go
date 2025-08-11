package remove_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/remove"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
)

type mockStorage struct {
	RemoveFunc func(ctx context.Context, id int) (int64, error)
}

func (m *mockStorage) RemoveSubscriptionEntry(ctx context.Context, id int) (int64, error) {
	return m.RemoveFunc(ctx, id)
}

type mockCache struct {
	InvalidateFunc func(key string) error
}

func (m *mockCache) Invalidate(key string) error {
	return m.InvalidateFunc(key)
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }

func makeLogger() *slog.Logger {
	return slog.New(discardHandler{})
}

func newDeleteRequest(url, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return req
}

func TestRemoveHandler(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		storage := &mockStorage{
			RemoveFunc: func(ctx context.Context, id int) (int64, error) {
				require.Equal(t, 42, id)
				return 1, nil
			},
		}
		cache := &mockCache{
			InvalidateFunc: func(key string) error {
				require.Equal(t, "subscription:42", key)
				return nil
			},
		}

		req := newDeleteRequest("/subscriptions/42", "42")
		w := httptest.NewRecorder()

		handler := remove.New(ctx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp response.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, response.StatusOK, resp.Status)
		assert.Equal(t, float64(1), resp.Data.(map[string]any)["deleted_count"])
	})

	t.Run("invalid id", func(t *testing.T) {
		storage := &mockStorage{}
		cache := &mockCache{}

		req := newDeleteRequest("/subscriptions/abc", "abc")
		w := httptest.NewRecorder()

		handler := remove.New(ctx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to decode id from url")
	})

	t.Run("cache error but continue", func(t *testing.T) {
		storage := &mockStorage{
			RemoveFunc: func(ctx context.Context, id int) (int64, error) {
				return 1, nil
			},
		}
		cache := &mockCache{
			InvalidateFunc: func(key string) error {
				return errors.New("cache down")
			},
		}

		req := newDeleteRequest("/subscriptions/42", "42")
		w := httptest.NewRecorder()

		handler := remove.New(ctx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "deleted_count")
	})

	t.Run("storage error", func(t *testing.T) {
		storage := &mockStorage{
			RemoveFunc: func(ctx context.Context, id int) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		cache := &mockCache{
			InvalidateFunc: func(key string) error {
				return nil
			},
		}

		req := newDeleteRequest("/subscriptions/42", "42")
		w := httptest.NewRecorder()

		handler := remove.New(ctx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to remove")
	})
}
