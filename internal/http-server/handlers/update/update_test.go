package update_test

import (
	"bytes"
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

	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/update"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/mware"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
)

type mockStorage struct {
	UpdateFunc func(context.Context, models.Entry, int) (int64, error)
}

func (m *mockStorage) UpdateSubscriptionEntry(ctx context.Context, entry models.Entry, id int) (int64, error) {
	return m.UpdateFunc(ctx, entry, id)
}

type mockCache struct {
	SetFunc func(key string, value any, expiration time.Duration) error
}

func (m *mockCache) Set(key string, value any, expiration time.Duration) error {
	return m.SetFunc(key, value, expiration)
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }

func makeLogger() *slog.Logger {
	return slog.New(discardHandler{})
}

func newPutRequest(url, id string, body []byte, username string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	ctx := req.Context()
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, mware.UserKey, username)
	return req.WithContext(ctx)
}

func TestUpdateHandler(t *testing.T) {
	baseCtx := context.Background()
	username := "testuser"

	t.Run("success", func(t *testing.T) {
		dummy := models.DummyEntry{
			ServiceName: "Netflix",
			Price:       999,
			StartDate:   "08-2025",
			EndDate:     "12-2025",
		}
		body, _ := json.Marshal(dummy)

		storage := &mockStorage{
			UpdateFunc: func(ctx context.Context, entry models.Entry, id int) (int64, error) {
				require.Equal(t, "Netflix", entry.ServiceName)
				require.Equal(t, 42, id)
				require.Equal(t, username, entry.Username)
				return 1, nil
			},
		}
		cache := &mockCache{
			SetFunc: func(key string, value any, expiration time.Duration) error {
				require.Equal(t, "subscription:42", key)
				return nil
			},
		}

		req := newPutRequest("/subscriptions/42", "42", body, username)
		w := httptest.NewRecorder()

		handler := update.New(baseCtx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp response.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, response.StatusOK, resp.Status)
		assert.Equal(t, float64(1), resp.Data.(map[string]any)["updated_count"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		storage := &mockStorage{
			UpdateFunc: func(context.Context, models.Entry, int) (int64, error) {
				t.Fatal("UpdateSubscriptionEntry should not be called on invalid JSON")
				return 0, nil
			},
		}
		cache := &mockCache{
			SetFunc: func(string, any, time.Duration) error {
				t.Fatal("Cache should not be called on invalid JSON")
				return nil
			},
		}

		req := newPutRequest("/subscriptions/42", "42", []byte("{bad json"), username)
		w := httptest.NewRecorder()

		handler := update.New(baseCtx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to decode request")
	})

	t.Run("validation error", func(t *testing.T) {
		dummy := models.DummyEntry{
			ServiceName: "", // required!
			Price:       999,
			StartDate:   "08-2025",
		}
		body, _ := json.Marshal(dummy)
		storage := &mockStorage{
			UpdateFunc: func(context.Context, models.Entry, int) (int64, error) {
				t.Fatal("Should not be called on validation error")
				return 0, nil
			},
		}
		cache := &mockCache{
			SetFunc: func(string, any, time.Duration) error {
				t.Fatal("Cache should not be called on validation error")
				return nil
			},
		}
		req := newPutRequest("/subscriptions/42", "42", body, username)
		w := httptest.NewRecorder()

		handler := update.New(baseCtx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "is a required field")
	})

	t.Run("invalid id in url", func(t *testing.T) {
		body, _ := json.Marshal(models.DummyEntry{
			ServiceName: "Test", Price: 1, StartDate: "08-2025",
		})
		storage := &mockStorage{}
		cache := &mockCache{}
		req := newPutRequest("/subscriptions/bad", "bad", body, username)
		w := httptest.NewRecorder()

		handler := update.New(baseCtx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)
		assert.Contains(t, w.Body.String(), "failed to decode id from url")
	})

	t.Run("startDate parse error", func(t *testing.T) {
		dummy := models.DummyEntry{
			ServiceName: "Test",
			Price:       1,
			StartDate:   "not-a-date",
		}
		body, _ := json.Marshal(dummy)
		storage := &mockStorage{
			UpdateFunc: func(context.Context, models.Entry, int) (int64, error) {
				t.Fatal("Should not be called on invalid startDate")
				return 0, nil
			},
		}
		cache := &mockCache{}
		req := newPutRequest("/subscriptions/42", "42", body, username)
		w := httptest.NewRecorder()

		handler := update.New(baseCtx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)
		assert.Contains(t, w.Body.String(), "failed to convert, field: startdate")
	})

	t.Run("endDate parse error", func(t *testing.T) {
		dummy := models.DummyEntry{
			ServiceName: "Test",
			Price:       1,
			StartDate:   "08-2025",
			EndDate:     "not-date",
		}
		body, _ := json.Marshal(dummy)
		storage := &mockStorage{
			UpdateFunc: func(context.Context, models.Entry, int) (int64, error) {
				t.Fatal("Should not be called on invalid endDate")
				return 0, nil
			},
		}
		cache := &mockCache{}
		req := newPutRequest("/subscriptions/42", "42", body, username)
		w := httptest.NewRecorder()

		handler := update.New(baseCtx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)
		assert.Contains(t, w.Body.String(), "failed to convert, field: enddate")
	})

	t.Run("storage update error", func(t *testing.T) {
		dummy := models.DummyEntry{
			ServiceName: "Netflix",
			Price:       999,
			StartDate:   "08-2025",
		}
		body, _ := json.Marshal(dummy)
		storage := &mockStorage{
			UpdateFunc: func(context.Context, models.Entry, int) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		cache := &mockCache{}
		req := newPutRequest("/subscriptions/42", "42", body, username)
		w := httptest.NewRecorder()

		handler := update.New(baseCtx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)
		assert.Contains(t, w.Body.String(), "failed to update")
	})

	t.Run("cache set error but still success", func(t *testing.T) {
		dummy := models.DummyEntry{
			ServiceName: "Netflix",
			Price:       999,
			StartDate:   "08-2025",
		}
		body, _ := json.Marshal(dummy)
		storage := &mockStorage{
			UpdateFunc: func(context.Context, models.Entry, int) (int64, error) {
				return 2, nil
			},
		}
		cache := &mockCache{
			SetFunc: func(string, any, time.Duration) error {
				return errors.New("cache down")
			},
		}
		req := newPutRequest("/subscriptions/42", "42", body, username)
		w := httptest.NewRecorder()

		handler := update.New(baseCtx, makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "updated_count")
	})
}
