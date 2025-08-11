package list_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/list"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/mware"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/response"
	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
)

type mockList struct {
	ListFunc func(ctx context.Context, username string, limit, offset int) ([]*subs.Entry, error)
}

func (m *mockList) ListSubscriptionEntrys(ctx context.Context, username string, limit, offset int) ([]*subs.Entry, error) {
	return m.ListFunc(ctx, username, limit, offset)
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }

func makeLogger() *slog.Logger {
	return slog.New(discardHandler{})
}

func TestListHandler(t *testing.T) {
	baseCtx := context.WithValue(context.Background(), mware.UserKey, "testuser")

	t.Run("success", func(t *testing.T) {
		expected := []*subs.Entry{
			{ServiceName: "Netflix", Price: 1299},
			{ServiceName: "YouTube Premium", Price: 399},
		}

		mock := &mockList{
			ListFunc: func(ctx context.Context, username string, limit, offset int) ([]*subs.Entry, error) {
				require.Equal(t, "testuser", username)
				require.Equal(t, 10, limit)
				require.Equal(t, 0, offset)
				return expected, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/subscriptions/list", nil)
		req = req.WithContext(baseCtx)
		w := httptest.NewRecorder()

		handler := list.New(context.Background(), makeLogger(), mock)
		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp response.Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, response.StatusOK, resp.Status)

		data, ok := resp.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(len(expected)), data["list_count"])
	})

	t.Run("unauthorized - no username in ctx", func(t *testing.T) {
		mock := &mockList{
			ListFunc: func(ctx context.Context, username string, limit, offset int) ([]*subs.Entry, error) {
				t.Fatal("ListSubscriptionEntrys should not be called when unauthorized")
				return nil, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/subscriptions/list", nil)
		// без user в context
		w := httptest.NewRecorder()

		handler := list.New(context.Background(), makeLogger(), mock)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "unauthorized")
	})

	t.Run("storage returns error", func(t *testing.T) {
		mock := &mockList{
			ListFunc: func(ctx context.Context, username string, limit, offset int) ([]*subs.Entry, error) {
				return nil, errors.New("db error")
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/subscriptions/list", nil)
		req = req.WithContext(baseCtx)
		w := httptest.NewRecorder()

		handler := list.New(context.Background(), makeLogger(), mock)
		handler.ServeHTTP(w, req)

		assert.Contains(t, w.Body.String(), "failed to list")
	})
}
