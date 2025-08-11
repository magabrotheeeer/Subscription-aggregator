package countsum_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	countsum "github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/count_sum"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/mware"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
)

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }

func makeLogger() *slog.Logger { return slog.New(discardHandler{}) }

type mockCounterSum struct {
	CountSumFunc func(ctx context.Context, entry countsum.FilterSum) (float64, error)
}

func (m *mockCounterSum) CountSumSubscriptionEntrys(ctx context.Context, entry countsum.FilterSum) (float64, error) {
	return m.CountSumFunc(ctx, entry)
}

func newRequestWithUser(body []byte, username string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/subscriptions/sum/123", bytes.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), mware.UserKey, username))
}

func TestCountSumHandler(t *testing.T) {
	ctx := context.Background()

	validReq := countsum.DummyFilterSum{
		ServiceName: "Netflix",
		StartDate:   "01-2024",
		EndDate:     "02-2024",
	}

	t.Run("success", func(t *testing.T) {
		body, _ := json.Marshal(validReq)

		mock := &mockCounterSum{
			CountSumFunc: func(ctx context.Context, entry countsum.FilterSum) (float64, error) {
				require.Equal(t, "Netflix", *entry.ServiceName)
				require.Equal(t, "testuser", entry.Username)
				return 123.45, nil
			},
		}

		req := newRequestWithUser(body, "testuser")
		w := httptest.NewRecorder()

		handler := countsum.New(ctx, makeLogger(), mock)
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp response.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, response.StatusOK, resp.Status)
		assert.Equal(t, 123.45, resp.Data.(map[string]any)["sum_of_subscriptions"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mock := &mockCounterSum{
			CountSumFunc: func(ctx context.Context, entry countsum.FilterSum) (float64, error) {
				t.Fatal("should not be called")
				return 0, nil
			},
		}

		req := newRequestWithUser([]byte("{bad-json"), "testuser")
		w := httptest.NewRecorder()

		handler := countsum.New(ctx, makeLogger(), mock)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to decode request")
	})

	t.Run("validation error", func(t *testing.T) {
		badReq := countsum.DummyFilterSum{
			ServiceName: "Netflix",
			StartDate:   "",
			EndDate:     "02-2024",
		}
		body, _ := json.Marshal(badReq)

		mock := &mockCounterSum{
			CountSumFunc: func(ctx context.Context, entry countsum.FilterSum) (float64, error) {
				t.Fatal("CountSumSubscriptionEntrys should not be called on validation error")
				return 0, nil
			},
		}

		req := newRequestWithUser(body, "testuser")
		w := httptest.NewRecorder()

		handler := countsum.New(ctx, makeLogger(), mock)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "is a required field")
	})

	t.Run("invalid startDate", func(t *testing.T) {
		badDate := validReq
		badDate.StartDate = "wrong-date"
		body, _ := json.Marshal(badDate)

		mock := &mockCounterSum{}
		req := newRequestWithUser(body, "testuser")
		w := httptest.NewRecorder()

		handler := countsum.New(ctx, makeLogger(), mock)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to convert, field: startdate")
	})

	t.Run("invalid endDate", func(t *testing.T) {
		badDate := validReq
		badDate.EndDate = "wrong-date"
		body, _ := json.Marshal(badDate)

		mock := &mockCounterSum{}
		req := newRequestWithUser(body, "testuser")
		w := httptest.NewRecorder()

		handler := countsum.New(ctx, makeLogger(), mock)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to convert, field: enddate")
	})

	t.Run("unauthorized - no username", func(t *testing.T) {
		body, _ := json.Marshal(validReq)

		mock := &mockCounterSum{}
		req := httptest.NewRequest(http.MethodPost, "/subscriptions/sum/123", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler := countsum.New(ctx, makeLogger(), mock)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "unauthorized")
	})

	t.Run("storage error", func(t *testing.T) {
		body, _ := json.Marshal(validReq)

		mock := &mockCounterSum{
			CountSumFunc: func(ctx context.Context, entry countsum.FilterSum) (float64, error) {
				return 0, errors.New("db error")
			},
		}

		req := newRequestWithUser(body, "testuser")
		w := httptest.NewRecorder()

		handler := countsum.New(ctx, makeLogger(), mock)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to sum")
	})
}
