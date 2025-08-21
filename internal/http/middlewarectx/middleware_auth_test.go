package middlewarectx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authpb "github.com/magabrotheeeer/subscription-aggregator/internal/grpc/proto"
	"github.com/magabrotheeeer/subscription-aggregator/internal/http/middlewarectx"

	"io"
	"log/slog"
)

// Mock for AuthClient
type AuthClientMock struct {
	mock.Mock
}

func (m *AuthClientMock) ValidateToken(ctx context.Context, token string) (*authpb.ValidateTokenResponse, error) {
	args := m.Called(ctx, token)
	resp, _ := args.Get(0).(*authpb.ValidateTokenResponse)
	return resp, args.Error(1)
}

func newNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestJWTMiddleware(t *testing.T) {
	authMock := new(AuthClientMock)
	logger := newNoopLogger()

	handlerCalled := false

	// Test handler which checks context values
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		username := r.Context().Value(middlewarectx.User)
		role := r.Context().Value(middlewarectx.Role)
		assert.Equal(t, "testuser", username)
		assert.Equal(t, "user", role)
		w.WriteHeader(http.StatusOK)
	})

	middleware := middlewarectx.JWTMiddleware(authMock, logger)(nextHandler)

	tests := []struct {
		name           string
		authHeader     string
		mockResp       *authpb.ValidateTokenResponse
		mockErr        error
		wantStatusCode int
		wantCalled     bool
	}{
		{
			name:           "missing Authorization header",
			authHeader:     "",
			wantStatusCode: http.StatusUnauthorized,
			wantCalled:     false,
		},
		{
			name:           "invalid Authorization header prefix",
			authHeader:     "Basic sometoken",
			wantStatusCode: http.StatusUnauthorized,
			wantCalled:     false,
		},
		{
			name:           "token validation error",
			authHeader:     "Bearer token",
			mockResp:       nil,
			mockErr:        errors.New("some grpc error"),
			wantStatusCode: http.StatusUnauthorized,
			wantCalled:     false,
		},
		{
			name:           "token invalid",
			authHeader:     "Bearer token",
			mockResp:       &authpb.ValidateTokenResponse{Valid: false},
			mockErr:        nil,
			wantStatusCode: http.StatusUnauthorized,
			wantCalled:     false,
		},
		{
			name:           "valid token",
			authHeader:     "Bearer validtoken",
			mockResp:       &authpb.ValidateTokenResponse{Valid: true, Username: "testuser", Role: "user"},
			mockErr:        nil,
			wantStatusCode: http.StatusOK,
			wantCalled:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false
			authMock.ExpectedCalls = nil // reset calls
			authMock.Calls = nil
			if tt.mockResp != nil || tt.mockErr != nil {
				authMock.On("ValidateToken", mock.Anything, strings.TrimPrefix(tt.authHeader, "Bearer ")).
					Return(tt.mockResp, tt.mockErr).Once()
			}

			req := httptest.NewRequest(http.MethodGet, "/somepath", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()

			middleware.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
			assert.Equal(t, tt.wantCalled, handlerCalled)
			authMock.AssertExpectations(t)
		})
	}
}
