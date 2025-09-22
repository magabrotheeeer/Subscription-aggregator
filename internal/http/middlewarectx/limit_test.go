package middlewarectx

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func newNoopLoggerLimit() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestRateLimitMiddleware(t *testing.T) {
	logger := newNoopLoggerLimit()
	middleware := RateLimitMiddleware(logger)

	// Создаем тестовый handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	t.Run("allows requests within rate limit", func(t *testing.T) {
		originalLimiter := limiter
		limiter = rate.NewLimiter(10, 10)
		defer func() { limiter = originalLimiter }()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		for range 10 {
			w = httptest.NewRecorder()
			middleware(testHandler).ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "success", w.Body.String())
		}
	})

	t.Run("blocks requests exceeding rate limit", func(t *testing.T) {
		originalLimiter := limiter
		limiter = rate.NewLimiter(1, 1)
		defer func() { limiter = originalLimiter }()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		w := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())

		w = httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, `"too many requests"`+"\n", w.Body.String())
	})

	t.Run("allows requests after rate limit reset", func(t *testing.T) {
		originalLimiter := limiter
		limiter = rate.NewLimiter(1, 1)
		defer func() { limiter = originalLimiter }()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		w := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		w = httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		time.Sleep(1 * time.Second)

		w = httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
	})
}

func TestRateLimitMiddleware_ConcurrentRequests(t *testing.T) {
	logger := newNoopLoggerLimit()
	middleware := RateLimitMiddleware(logger)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	t.Run("handles concurrent requests correctly", func(t *testing.T) {
		originalLimiter := limiter
		limiter = rate.NewLimiter(5, 5)
		defer func() { limiter = originalLimiter }()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		results := make(chan int, 10)

		for range 10 {
			go func() {
				w := httptest.NewRecorder()
				middleware(testHandler).ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		successCount := 0
		rateLimitedCount := 0
		for range 10 {
			select {
			case code := <-results:
				switch code {
				case http.StatusOK:
					successCount++
				case http.StatusTooManyRequests:
					rateLimitedCount++
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timeout waiting for concurrent requests")
			}
		}

		assert.Equal(t, 5, successCount)
		assert.Equal(t, 5, rateLimitedCount)
	})
}

func TestRateLimitMiddleware_DifferentEndpoints(t *testing.T) {
	logger := newNoopLoggerLimit()
	middleware := RateLimitMiddleware(logger)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	t.Run("rate limit applies globally across all endpoints", func(t *testing.T) {
		originalLimiter := limiter
		limiter = rate.NewLimiter(2, 2)
		defer func() { limiter = originalLimiter }()

		endpoints := []string{"/api/v1/users", "/api/v1/subscriptions", "/api/v1/payments"}

		successCount := 0
		rateLimitedCount := 0

		for i := range 6 {
			endpoint := endpoints[i%len(endpoints)]
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()

			middleware(testHandler).ServeHTTP(w, req)

			switch w.Code {
			case http.StatusOK:
				successCount++
			case http.StatusTooManyRequests:
				rateLimitedCount++
			}
		}

		assert.Equal(t, 2, successCount)
		assert.Equal(t, 4, rateLimitedCount)
	})
}

func TestRateLimitMiddleware_ResponseFormat(t *testing.T) {
	logger := newNoopLoggerLimit()
	middleware := RateLimitMiddleware(logger)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	t.Run("returns correct response format when rate limited", func(t *testing.T) {
		originalLimiter := limiter
		limiter = rate.NewLimiter(1, 1)
		defer func() { limiter = originalLimiter }()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		w := httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		w = httptest.NewRecorder()
		middleware(testHandler).ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, `"too many requests"`+"\n", w.Body.String())
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})
}

func TestRateLimitMiddleware_HandlerNotCalledWhenRateLimited(t *testing.T) {
	logger := newNoopLoggerLimit()
	middleware := RateLimitMiddleware(logger)

	originalLimiter := limiter
	limiter = rate.NewLimiter(1, 1)
	defer func() { limiter = originalLimiter }()

	var handlerCalled bool
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	handlerCalled = false
	w := httptest.NewRecorder()
	middleware(testHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled, "Handler should be called for first request")

	handlerCalled = false
	w = httptest.NewRecorder()
	middleware(testHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.False(t, handlerCalled, "Handler should not be called when rate limited")
}

func TestRateLimitMiddleware_WithDifferentHTTPMethods(t *testing.T) {
	logger := newNoopLoggerLimit()
	middleware := RateLimitMiddleware(logger)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("success")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	t.Run("rate limit applies to all HTTP methods", func(t *testing.T) {
		originalLimiter := limiter
		limiter = rate.NewLimiter(3, 3)
		defer func() { limiter = originalLimiter }()

		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

		successCount := 0
		rateLimitedCount := 0

		for i := range 6 {
			method := methods[i%len(methods)]
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			middleware(testHandler).ServeHTTP(w, req)

			switch w.Code {
			case http.StatusOK:
				successCount++
			case http.StatusTooManyRequests:
				rateLimitedCount++
			}
		}

		assert.Equal(t, 3, successCount)
		assert.Equal(t, 3, rateLimitedCount)
	})
}
