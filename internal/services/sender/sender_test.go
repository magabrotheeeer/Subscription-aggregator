package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/api/handlers/payment/paymentwebhook"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/smtp"
	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetUser(ctx context.Context, userUID string) (*models.User, error) {
	args := m.Called(ctx, userUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

type MockTransport struct {
	mock.Mock
}

func (m *MockTransport) Connect() (smtp.Client, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(smtp.Client), args.Error(1)
}

func (m *MockTransport) GetSMTPUser() string {
	args := m.Called()
	return args.String(0)
}

type MockSMTPClient struct {
	mock.Mock
}

func (m *MockSMTPClient) Mail(from string) error {
	args := m.Called(from)
	return args.Error(0)
}

func (m *MockSMTPClient) Rcpt(to string) error {
	args := m.Called(to)
	return args.Error(0)
}

func (m *MockSMTPClient) Data() (io.WriteCloser, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.WriteCloser), args.Error(1)
}

func (m *MockSMTPClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSMTPClient) Quit() error {
	args := m.Called()
	return args.Error(0)
}

type MockSMTPWriter struct {
	mock.Mock
}

func (m *MockSMTPWriter) Write(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *MockSMTPWriter) Close() error {
	args := m.Called()
	return args.Error(0)
}

func newNoopLogger() *slog.Logger {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return slog.New(h)
}

func TestSenderService_SendInfoExpiringSubscription(t *testing.T) {
	tests := []struct {
		name          string
		body          []byte
		setupMocks    func(*MockTransport)
		expectedError bool
		errorMessage  string
	}{
		{
			name: "success - send expiring subscription email",
			body: []byte(`{"email":"test@example.com","username":"testuser","service_name":"Netflix","end_date":"2024-01-01T00:00:00Z","price":500}`),
			setupMocks: func(t *MockTransport) {
				mockClient := new(MockSMTPClient)
				mockWriter := new(MockSMTPWriter)

				t.On("GetSMTPUser").Return("sender@example.com")
				t.On("Connect").Return(mockClient, nil).Once()
				mockClient.On("Mail", "sender@example.com").Return(nil).Once()
				mockClient.On("Rcpt", "test@example.com").Return(nil).Once()
				mockClient.On("Data").Return(mockWriter, nil).Once()
				mockWriter.On("Write", mock.AnythingOfType("[]uint8")).Return(100, nil).Once()
				mockWriter.On("Close").Return(nil).Once()
				mockClient.On("Quit").Return(nil).Once()
				mockClient.On("Close").Return(nil).Once()
			},
			expectedError: false,
		},
		{
			name: "invalid JSON",
			body: []byte(`invalid json`),
			setupMocks: func(_ *MockTransport) {
				// No transport calls expected for invalid JSON
			},
			expectedError: true,
			errorMessage:  "error unmarshalling message",
		},
		{
			name: "SMTP connection error",
			body: []byte(`{"email":"test@example.com","username":"testuser","service_name":"Netflix","end_date":"2024-01-01T00:00:00Z","price":500}`),
			setupMocks: func(t *MockTransport) {
				t.On("GetSMTPUser").Return("sender@example.com")
				t.On("Connect").Return(nil, errors.New("connection error")).Once()
			},
			expectedError: true,
			errorMessage:  "connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			transport := new(MockTransport)
			service := NewSenderService(repo, newNoopLogger(), transport)

			tt.setupMocks(transport)

			err := service.SendInfoExpiringSubscription(tt.body)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}

			transport.AssertExpectations(t)
		})
	}
}

func TestSenderService_SendInfoExpiringTrialPeriodSubscription(t *testing.T) {
	tests := []struct {
		name          string
		body          []byte
		setupMocks    func(*MockTransport)
		expectedError bool
		errorMessage  string
	}{
		{
			name: "success - send trial period expiring email",
			body: []byte(`{"uuid":"user123","email":"test@example.com","username":"testuser"}`),
			setupMocks: func(t *MockTransport) {
				mockClient := new(MockSMTPClient)
				mockWriter := new(MockSMTPWriter)

				t.On("GetSMTPUser").Return("sender@example.com")
				t.On("Connect").Return(mockClient, nil).Once()
				mockClient.On("Mail", "sender@example.com").Return(nil).Once()
				mockClient.On("Rcpt", "test@example.com").Return(nil).Once()
				mockClient.On("Data").Return(mockWriter, nil).Once()
				mockWriter.On("Write", mock.AnythingOfType("[]uint8")).Return(100, nil).Once()
				mockWriter.On("Close").Return(nil).Once()
				mockClient.On("Quit").Return(nil).Once()
				mockClient.On("Close").Return(nil).Once()
			},
			expectedError: false,
		},
		{
			name: "invalid JSON",
			body: []byte(`invalid json`),
			setupMocks: func(_ *MockTransport) {
				// No transport calls expected for invalid JSON
			},
			expectedError: true,
			errorMessage:  "error unmarshalling message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			transport := new(MockTransport)
			service := NewSenderService(repo, newNoopLogger(), transport)

			tt.setupMocks(transport)

			err := service.SendInfoExpiringTrialPeriodSubscription(tt.body)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}

			transport.AssertExpectations(t)
		})
	}
}

func TestSenderService_SendInfoSuccessPayment(t *testing.T) {
	payload := &paymentwebhook.Payload{
		Object: struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Amount struct {
				Value    string `json:"value"`
				Currency string `json:"currency"`
			} `json:"amount"`
			PaymentMethod struct {
				ID string `json:"id"`
			} `json:"payment_method"`
			Metadata map[string]string `json:"metadata"`
		}{
			ID:     "payment123",
			Status: "succeeded",
			Amount: struct {
				Value    string `json:"value"`
				Currency string `json:"currency"`
			}{
				Value:    "100.00",
				Currency: "RUB",
			},
			PaymentMethod: struct {
				ID string `json:"id"`
			}{
				ID: "card123",
			},
			Metadata: map[string]string{
				"user_uid": "user123",
			},
		},
	}

	user := &models.User{
		UUID:     "user123",
		Email:    "test@example.com",
		Username: "testuser",
	}

	tests := []struct {
		name          string
		payload       *paymentwebhook.Payload
		setupMocks    func(*MockRepository, *MockTransport)
		expectedError bool
		errorMessage  string
	}{
		{
			name:    "success - send success payment email",
			payload: payload,
			setupMocks: func(r *MockRepository, t *MockTransport) {
				mockClient := new(MockSMTPClient)
				mockWriter := new(MockSMTPWriter)

				r.On("GetUser", mock.Anything, "user123").Return(user, nil).Once()
				t.On("GetSMTPUser").Return("sender@example.com")
				t.On("Connect").Return(mockClient, nil).Once()
				mockClient.On("Mail", "sender@example.com").Return(nil).Once()
				mockClient.On("Rcpt", "test@example.com").Return(nil).Once()
				mockClient.On("Data").Return(mockWriter, nil).Once()
				mockWriter.On("Write", mock.AnythingOfType("[]uint8")).Return(100, nil).Once()
				mockWriter.On("Close").Return(nil).Once()
				mockClient.On("Quit").Return(nil).Once()
				mockClient.On("Close").Return(nil).Once()
			},
			expectedError: false,
		},
		{
			name:    "repository error",
			payload: payload,
			setupMocks: func(r *MockRepository, _ *MockTransport) {
				r.On("GetUser", mock.Anything, "user123").Return(nil, errors.New("user not found")).Once()
			},
			expectedError: true,
			errorMessage:  "failed to get username: user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			transport := new(MockTransport)
			service := NewSenderService(repo, newNoopLogger(), transport)

			tt.setupMocks(repo, transport)

			err := service.SendInfoSuccessPayment(tt.payload)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
			transport.AssertExpectations(t)
		})
	}
}

func TestSenderService_SendInfoFailurePayment(t *testing.T) {
	payload := &paymentwebhook.Payload{
		Object: struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Amount struct {
				Value    string `json:"value"`
				Currency string `json:"currency"`
			} `json:"amount"`
			PaymentMethod struct {
				ID string `json:"id"`
			} `json:"payment_method"`
			Metadata map[string]string `json:"metadata"`
		}{
			ID:     "payment123",
			Status: "failed",
			Amount: struct {
				Value    string `json:"value"`
				Currency string `json:"currency"`
			}{
				Value:    "100.00",
				Currency: "RUB",
			},
			PaymentMethod: struct {
				ID string `json:"id"`
			}{
				ID: "card123",
			},
			Metadata: map[string]string{
				"user_uid": "user123",
			},
		},
	}

	user := &models.User{
		UUID:     "user123",
		Email:    "test@example.com",
		Username: "testuser",
	}

	tests := []struct {
		name          string
		payload       *paymentwebhook.Payload
		setupMocks    func(*MockRepository, *MockTransport)
		expectedError bool
		errorMessage  string
	}{
		{
			name:    "success - send failure payment email",
			payload: payload,
			setupMocks: func(r *MockRepository, t *MockTransport) {
				mockClient := new(MockSMTPClient)
				mockWriter := new(MockSMTPWriter)

				r.On("GetUser", mock.Anything, "user123").Return(user, nil).Once()
				t.On("GetSMTPUser").Return("sender@example.com")
				t.On("Connect").Return(mockClient, nil).Once()
				mockClient.On("Mail", "sender@example.com").Return(nil).Once()
				mockClient.On("Rcpt", "test@example.com").Return(nil).Once()
				mockClient.On("Data").Return(mockWriter, nil).Once()
				mockWriter.On("Write", mock.AnythingOfType("[]uint8")).Return(100, nil).Once()
				mockWriter.On("Close").Return(nil).Once()
				mockClient.On("Quit").Return(nil).Once()
				mockClient.On("Close").Return(nil).Once()
			},
			expectedError: false,
		},
		{
			name:    "repository error",
			payload: payload,
			setupMocks: func(r *MockRepository, _ *MockTransport) {
				r.On("GetUser", mock.Anything, "user123").Return(nil, errors.New("user not found")).Once()
			},
			expectedError: true,
			errorMessage:  "failed to get username: user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			transport := new(MockTransport)
			service := NewSenderService(repo, newNoopLogger(), transport)

			tt.setupMocks(repo, transport)

			err := service.SendInfoFailurePayment(tt.payload)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
			transport.AssertExpectations(t)
		})
	}
}

func TestSenderService_NewSenderService(t *testing.T) {
	repo := new(MockRepository)
	transport := new(MockTransport)
	logger := newNoopLogger()

	service := NewSenderService(repo, logger, transport)

	assert.NotNil(t, service)
	assert.Equal(t, repo, service.repo)
	assert.Equal(t, transport, service.transport)
	assert.Equal(t, logger, service.log)
}

func TestSenderService_SMTPErrorHandling(t *testing.T) {
	entryInfo := &models.EntryInfo{
		Email:       "test@example.com",
		Username:    "testuser",
		ServiceName: "Netflix",
		EndDate:     time.Now().Add(24 * time.Hour),
		Price:       500,
	}

	body, _ := json.Marshal(entryInfo)

	tests := []struct {
		name          string
		setupMocks    func(*MockTransport)
		expectedError bool
		errorMessage  string
	}{
		{
			name: "SMTP Mail error",
			setupMocks: func(t *MockTransport) {
				mockClient := new(MockSMTPClient)

				t.On("GetSMTPUser").Return("sender@example.com")
				t.On("Connect").Return(mockClient, nil).Once()
				mockClient.On("Mail", "sender@example.com").Return(errors.New("mail error")).Once()
				mockClient.On("Close").Return(nil).Once()
			},
			expectedError: true,
			errorMessage:  "mail error",
		},
		{
			name: "SMTP Rcpt error",
			setupMocks: func(t *MockTransport) {
				mockClient := new(MockSMTPClient)

				t.On("GetSMTPUser").Return("sender@example.com")
				t.On("Connect").Return(mockClient, nil).Once()
				mockClient.On("Mail", "sender@example.com").Return(nil).Once()
				mockClient.On("Rcpt", "test@example.com").Return(errors.New("rcpt error")).Once()
				mockClient.On("Close").Return(nil).Once()
			},
			expectedError: true,
			errorMessage:  "rcpt error",
		},
		{
			name: "SMTP Data error",
			setupMocks: func(t *MockTransport) {
				mockClient := new(MockSMTPClient)

				t.On("GetSMTPUser").Return("sender@example.com")
				t.On("Connect").Return(mockClient, nil).Once()
				mockClient.On("Mail", "sender@example.com").Return(nil).Once()
				mockClient.On("Rcpt", "test@example.com").Return(nil).Once()
				mockClient.On("Data").Return(nil, errors.New("data error")).Once()
				mockClient.On("Close").Return(nil).Once()
			},
			expectedError: true,
			errorMessage:  "data error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockRepository)
			transport := new(MockTransport)
			service := NewSenderService(repo, newNoopLogger(), transport)

			tt.setupMocks(transport)

			err := service.SendInfoExpiringSubscription(body)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMessage)

			transport.AssertExpectations(t)
		})
	}
}
