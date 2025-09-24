package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDataFactory содержит методы для создания тестовых данных
type TestDataFactory struct {
	storage *Storage
}

// NewTestDataFactory создает новую фабрику тестовых данных
func NewTestDataFactory(storage *Storage) *TestDataFactory {
	return &TestDataFactory{storage: storage}
}

// CreateUser создает тестового пользователя
func (f *TestDataFactory) CreateUser(t *testing.T, userUID, username, email, passwordHash, role string) {
	_, err := f.storage.DB.Exec(`INSERT INTO users (uid, username, email, password_hash, role) 
		VALUES ($1, $2, $3, $4, $5)`,
		userUID, username, email, passwordHash, role)
	require.NoError(t, err)
}

// CreateUserWithSubscription создает пользователя с полными данными подписки
func (f *TestDataFactory) CreateUserWithSubscription(t *testing.T, userUID, username, email, passwordHash, role string,
	trialEndDate, subscriptionExpiry time.Time, subscriptionStatus string) {
	_, err := f.storage.DB.Exec(`INSERT INTO users 
		(uid, username, email, password_hash, role, trial_end_date, subscription_status, subscription_expiry)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userUID, username, email, passwordHash, role, trialEndDate, subscriptionStatus, subscriptionExpiry)
	require.NoError(t, err)
}

// CreateSubscription создает тестовую подписку
func (f *TestDataFactory) CreateSubscription(t *testing.T, serviceName string, price float64, username string,
	startDate time.Time, counterMonths int, userUID string, nextPaymentDate time.Time, isActive bool) int {
	var id int
	err := f.storage.DB.QueryRow(`INSERT INTO subscriptions 
		(service_name, price, username, start_date, counter_months, user_uid, next_payment_date, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		serviceName, price, username, startDate, counterMonths, userUID, nextPaymentDate, isActive).Scan(&id)
	require.NoError(t, err)
	return id
}

// CreatePaymentToken создает тестовый токен платежа
func (f *TestDataFactory) CreatePaymentToken(t *testing.T, userUID, token string) {
	_, err := f.storage.DB.Exec(`INSERT INTO yookassa_payment_tokens (user_uid, token) 
		VALUES ($1, $2)`,
		userUID, token)
	require.NoError(t, err)
}

// TestUserData содержит стандартные тестовые данные пользователя
type TestUserData struct {
	UID                string
	Username           string
	Email              string
	PasswordHash       string
	Role               string
	TrialEndDate       *time.Time
	SubscriptionStatus string
	SubscriptionExpiry *time.Time
}

// GetTestUserData возвращает стандартные тестовые данные пользователя
func GetTestUserData() TestUserData {
	uid := uuid.New().String()
	trialEnd := time.Now().AddDate(0, 0, 7)
	subscriptionExpiry := time.Now().AddDate(0, 1, 0)

	return TestUserData{
		UID:                uid,
		Username:           "testuser",
		Email:              "test@example.com",
		PasswordHash:       "hashedpassword",
		Role:               "user",
		TrialEndDate:       &trialEnd,
		SubscriptionStatus: "trial",
		SubscriptionExpiry: &subscriptionExpiry,
	}
}

// TestSubscriptionData содержит стандартные тестовые данные подписки
type TestSubscriptionData struct {
	ServiceName     string
	Price           float64
	Username        string
	StartDate       time.Time
	CounterMonths   int
	UserUID         string
	NextPaymentDate time.Time
	IsActive        bool
}

// GetTestSubscriptionData возвращает стандартные тестовые данные подписки
func GetTestSubscriptionData(userUID string) TestSubscriptionData {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	return TestSubscriptionData{
		ServiceName:     "Netflix",
		Price:           1000.0,
		Username:        "testuser",
		StartDate:       startDate,
		CounterMonths:   12,
		UserUID:         userUID,
		NextPaymentDate: startDate,
		IsActive:        true,
	}
}

// TestVerification содержит общие функции для проверки результатов тестов
type TestVerification struct {
	storage *Storage
}

// NewTestVerification создает новый объект для проверки результатов
func NewTestVerification(storage *Storage) *TestVerification {
	return &TestVerification{storage: storage}
}

// VerifyUserExists проверяет существование пользователя в БД
func (v *TestVerification) VerifyUserExists(t *testing.T, userUID string) {
	var count int
	err := v.storage.DB.QueryRow("SELECT COUNT(*) FROM users WHERE uid = $1", userUID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// VerifySubscriptionExists проверяет существование подписки в БД
func (v *TestVerification) VerifySubscriptionExists(t *testing.T, subscriptionID int) {
	var count int
	err := v.storage.DB.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE id = $1", subscriptionID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// VerifySubscriptionDeleted проверяет удаление подписки из БД
func (v *TestVerification) VerifySubscriptionDeleted(t *testing.T, subscriptionID int) {
	var count int
	err := v.storage.DB.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE id = $1", subscriptionID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

// VerifySubscriptionData проверяет данные подписки
func (v *TestVerification) VerifySubscriptionData(t *testing.T, subscriptionID int, expectedServiceName string,
	expectedPrice float64, expectedCounterMonths int) {
	var serviceName string
	var price float64
	var counterMonths int
	err := v.storage.DB.QueryRow("SELECT service_name, price, counter_months FROM subscriptions WHERE id = $1", subscriptionID).
		Scan(&serviceName, &price, &counterMonths)
	require.NoError(t, err)
	require.Equal(t, expectedServiceName, serviceName)
	require.Equal(t, expectedPrice, price)
	require.Equal(t, expectedCounterMonths, counterMonths)
}

// VerifyUserSubscriptionStatus проверяет статус подписки пользователя
func (v *TestVerification) VerifyUserSubscriptionStatus(t *testing.T, userUID, expectedStatus string) {
	var status string
	err := v.storage.DB.QueryRow("SELECT subscription_status FROM users WHERE uid = $1", userUID).
		Scan(&status)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, status)
}

// setupTestDatabase создает тестовую БД с контейнером PostgreSQL
func setupTestDatabase(t *testing.T) (*Storage, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		).WithDeadline(3 * time.Minute),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start container")

	// Добавляем задержку для полной инициализации PostgreSQL
	time.Sleep(3 * time.Second)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get port")

	connStr := fmt.Sprintf("postgres://testuser:testpass@localhost:%s/testdb?sslmode=disable", port.Port())

	// Пробуем подключиться несколько раз с ретраями
	var storage *Storage
	for range 10 {
		storage, err = New(connStr)
		if err == nil {
			// Проверяем, что подключение действительно работает
			err = storage.DB.Ping()
			if err == nil {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	require.NoError(t, err, "Failed to create storage after retries")

	// Создаем таблицы
	_, err = storage.DB.Exec(`
        DROP TABLE IF EXISTS yookassa_payments CASCADE;
        DROP TABLE IF EXISTS yookassa_payment_tokens CASCADE;
        DROP TABLE IF EXISTS subscriptions CASCADE;
        DROP TABLE IF EXISTS users CASCADE;
        
        CREATE EXTENSION IF NOT EXISTS "pgcrypto";
        
        CREATE TABLE users (
            uid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            username TEXT NOT NULL UNIQUE,
            email TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            role TEXT NOT NULL DEFAULT 'user',
            trial_end_date DATE,
            subscription_status TEXT DEFAULT 'trial',
            subscription_expiry DATE
        );
        
        CREATE TABLE subscriptions (
			id SERIAL PRIMARY KEY,
            service_name TEXT NOT NULL,
            price FLOAT NOT NULL,
            username TEXT NOT NULL,
            start_date DATE NOT NULL,
            counter_months INT NOT NULL,
            user_uid UUID REFERENCES users(uid),
            next_payment_date DATE,
            is_active BOOLEAN DEFAULT true
        );
        
        CREATE TABLE yookassa_payment_tokens (
            id SERIAL PRIMARY KEY,
            user_uid UUID NOT NULL REFERENCES users(uid),
            token TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        
        CREATE TABLE yookassa_payments (
            id SERIAL PRIMARY KEY,
            user_uid UUID NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
            subscription_id INTEGER REFERENCES subscriptions(id) ON DELETE SET NULL,
            payment_id VARCHAR(255) NOT NULL,
            amount BIGINT NOT NULL,
            currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
            status VARCHAR(50) NOT NULL,
            payment_token_id INTEGER REFERENCES yookassa_payment_tokens(id) ON DELETE SET NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
        
        CREATE INDEX idx_subscriptions_username ON subscriptions(username);
        CREATE INDEX idx_subscriptions_user_uid ON subscriptions(user_uid);
        CREATE INDEX idx_subscriptions_next_payment_date ON subscriptions(next_payment_date);
        CREATE INDEX idx_yookassa_payments_user_uid ON yookassa_payments(user_uid);
        CREATE INDEX idx_yookassa_payments_subscription_id ON yookassa_payments(subscription_id);
    `)
	require.NoError(t, err, "Failed to create tables")

	cleanup := func() {
		if storage != nil && storage.DB != nil {
			_ = storage.DB.Close()
		}
		if postgresContainer != nil {
			_ = postgresContainer.Terminate(ctx)
		}
	}

	return storage, cleanup
}
