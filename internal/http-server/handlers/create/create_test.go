package create_test

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/create"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/mware"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type mockStorage struct {
	CreateFunc func(ctx context.Context, entry subs.Entry) (int, error)
}

func (m *mockStorage) CreateSubscriptionEntry(ctx context.Context, entry subs.Entry) (int, error) {
	return m.CreateFunc(ctx, entry)
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

func TestCreateHandler(t *testing.T) {
	ctx := context.WithValue(context.Background(), mware.UserKey, "testuser")

	t.Run("success", func(t *testing.T) {
		dummy := subs.DummyEntry{
			ServiceName: "Netflix",
			Price:       1200,
			StartDate:   "01-2024",
			EndDate:     "02-2024",
		}
		body, _ := json.Marshal(dummy)

		storage := &mockStorage{
			CreateFunc: func(ctx context.Context, entry subs.Entry) (int, error) {
				require.Equal(t, "Netflix", entry.ServiceName)
				require.Equal(t, "testuser", entry.Username)
				return 42, nil
			},
		}
		cache := &mockCache{
			SetFunc: func(key string, value any, exp time.Duration) error {
				require.Contains(t, key, "subscription:42")
				return nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(body))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler := create.New(context.Background(), makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		var resp response.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		assert.Equal(t, response.StatusOK, resp.Status)
		assert.Equal(t, float64(42), resp.Data.(map[string]any)["last added id"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		storage := &mockStorage{
			CreateFunc: func(ctx context.Context, entry subs.Entry) (int, error) {
				t.Fatal("storage should not be called on invalid JSON")
				return 0, nil
			},
		}
		cache := &mockCache{
			SetFunc: func(key string, v any, exp time.Duration) error {
				t.Fatal("cache should not be called on invalid JSON")
				return nil
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader([]byte("{bad json")))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler := create.New(context.Background(), makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to decode request")
	})

	t.Run("validation error", func(t *testing.T) {
		storage := &mockStorage{
			CreateFunc: func(ctx context.Context, entry subs.Entry) (int, error) {
				t.Fatal("storage should not be called on validation error")
				return 0, nil
			},
		}
		cache := &mockCache{
			SetFunc: func(key string, v any, exp time.Duration) error {
				t.Fatal("cache should not be called on validation error")
				return nil
			},
		}

		dummy := subs.DummyEntry{
			Price:     1200,
			StartDate: "01-2024",
			EndDate:   "02-2024",
		}
		body, _ := json.Marshal(dummy)

		req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(body))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler := create.New(context.Background(), makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "is a required field")
	})

	t.Run("invalid startDate", func(t *testing.T) {
		storage := &mockStorage{
			CreateFunc: func(ctx context.Context, entry subs.Entry) (int, error) {
				t.Fatal("storage should not be called on invalid startDate")
				return 0, nil
			},
		}
		cache := &mockCache{}

		dummy := subs.DummyEntry{
			ServiceName: "Netflix",
			Price:       1200,
			StartDate:   "wrong-date",
			EndDate:     "02-2024",
		}
		body, _ := json.Marshal(dummy)

		req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(body))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler := create.New(context.Background(), makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to convert, field: startdate")
	})

	t.Run("invalid endDate", func(t *testing.T) {
		storage := &mockStorage{
			CreateFunc: func(ctx context.Context, entry subs.Entry) (int, error) {
				t.Fatal("storage should not be called on invalid endDate")
				return 0, nil
			},
		}
		cache := &mockCache{}

		dummy := subs.DummyEntry{
			ServiceName: "Netflix",
			Price:       1200,
			StartDate:   "01-2024",
			EndDate:     "bad-date",
		}
		body, _ := json.Marshal(dummy)

		req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(body))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler := create.New(context.Background(), makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to convert, field: enddate")
	})

	t.Run("no username in context", func(t *testing.T) {
		storage := &mockStorage{
			CreateFunc: func(ctx context.Context, entry subs.Entry) (int, error) {
				t.Fatal("storage should not be called when username missing")
				return 0, nil
			},
		}
		cache := &mockCache{}

		dummy := subs.DummyEntry{
			ServiceName: "Netflix",
			Price:       1200,
			StartDate:   "01-2024",
		}
		body, _ := json.Marshal(dummy)

		req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := create.New(context.Background(), makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "unauthorized")
	})

	t.Run("storage error", func(t *testing.T) {
		storage := &mockStorage{
			CreateFunc: func(ctx context.Context, entry subs.Entry) (int, error) {
				return 0, errors.New("db error")
			},
		}
		cache := &mockCache{}

		dummy := subs.DummyEntry{
			ServiceName: "Netflix",
			Price:       1200,
			StartDate:   "01-2024",
		}
		body, _ := json.Marshal(dummy)

		req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(body))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler := create.New(context.Background(), makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to save")
	})

	t.Run("cache error but still success", func(t *testing.T) {
		storage := &mockStorage{
			CreateFunc: func(ctx context.Context, entry subs.Entry) (int, error) {
				return 42, nil
			},
		}
		cache := &mockCache{
			SetFunc: func(key string, value any, exp time.Duration) error {
				return errors.New("cache down")
			},
		}

		dummy := subs.DummyEntry{
			ServiceName: "Netflix",
			Price:       1200,
			StartDate:   "01-2024",
		}
		body, _ := json.Marshal(dummy)

		req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(body))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler := create.New(context.Background(), makeLogger(), storage, cache)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "last added id")
	})
}
