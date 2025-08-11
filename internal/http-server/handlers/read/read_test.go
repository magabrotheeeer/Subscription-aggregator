package read_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/read"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type mockStorage struct {
	ReadFunc func(ctx context.Context, id int) (*subs.SubscriptionEntry, error)
}

func (m *mockStorage) ReadSubscriptionEntry(ctx context.Context, id int) (*subs.SubscriptionEntry, error) {
	return m.ReadFunc(ctx, id)
}

type mockCache struct {
	GetFunc func(key string, result any) (bool, error)
}

func (m *mockCache) Get(key string, result any) (bool, error) {
	return m.GetFunc(key, result)
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }

func makeLogger() *slog.Logger {
	return slog.New(discardHandler{})
}

func TestReadHandler(t *testing.T) {
	ctx := context.Background()
	exampleEntry := &subs.SubscriptionEntry{
		ServiceName: "Netflix",
		Username:    "testuser",
		Price:       899,
		StartDate:   time.Now(),
	}

	t.Run("success from cache", func(t *testing.T) {
		storage := &mockStorage{
			ReadFunc: func(ctx context.Context, id int) (*subs.SubscriptionEntry, error) {
				t.Fatal("storage should not be called when found in cache")
				return nil, nil
			},
		}
		cache := &mockCache{
			GetFunc: func(key string, result any) (bool, error) {
				require.Equal(t, "subscription:42", key)
				ptr := result.(**subs.SubscriptionEntry)
				*ptr = exampleEntry
				return true, nil
			},
		}

		req := newGetRequest("/subscriptions/42", "42")
		w := httptest.NewRecorder()

		handler := read.New(ctx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp response.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, response.StatusOK, resp.Status)
		assert.Equal(t, "Netflix", resp.Data.(map[string]any)["entries"].(map[string]any)["ServiceName"])
	})

	t.Run("success from storage", func(t *testing.T) {
		storage := &mockStorage{
			ReadFunc: func(ctx context.Context, id int) (*subs.SubscriptionEntry, error) {
				require.Equal(t, 42, id)
				return exampleEntry, nil
			},
		}
		cache := &mockCache{
			GetFunc: func(key string, result any) (bool, error) {
				return false, nil // не найдено в кэше
			},
		}

		req := newGetRequest("/subscriptions/42", "42")
		w := httptest.NewRecorder()

		handler := read.New(ctx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Netflix")
	})

	t.Run("invalid id", func(t *testing.T) {
		storage := &mockStorage{}
		cache := &mockCache{}

		req := newGetRequest("/subscriptions/abc", "abc")
		w := httptest.NewRecorder()

		handler := read.New(ctx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to decode id from url")
	})

	t.Run("cache error", func(t *testing.T) {
		storage := &mockStorage{}
		cache := &mockCache{
			GetFunc: func(key string, result any) (bool, error) {
				return false, errors.New("cache down")
			},
		}

		req := newGetRequest("/subscriptions/42", "42")
		w := httptest.NewRecorder()

		handler := read.New(ctx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "internal error")
	})

	t.Run("storage error", func(t *testing.T) {
		storage := &mockStorage{
			ReadFunc: func(ctx context.Context, id int) (*subs.SubscriptionEntry, error) {
				return nil, errors.New("db error")
			},
		}
		cache := &mockCache{
			GetFunc: func(key string, result any) (bool, error) {
				return false, nil
			},
		}

		req := newGetRequest("/subscriptions/42", "42")
		w := httptest.NewRecorder()

		handler := read.New(ctx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to read")
	})
}

func newGetRequest(url, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return req
}
