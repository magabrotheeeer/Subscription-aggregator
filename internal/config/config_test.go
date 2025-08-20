package config

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutput перехватывает вывод log.Fatal
func captureOutput(f func()) (string, bool) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	oldFlags := log.Flags()
	log.SetFlags(0)
	defer func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(oldFlags)
	}()

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		f()
	}()

	return buf.String(), panicked
}

func TestConfig_String(t *testing.T) {
	cfg := &Config{
		Env:                     "test",
		GRPCAuthAddress:         "localhost:50051",
		StorageConnectionString: "postgres://user:pass@localhost:5432/db",
		RedisConnection: RedisConnection{
			AddressRedis: "localhost:6379",
			Password:     "redis_pass",
			User:         "redis_user",
			DB:           1,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			TimeoutRedis: 10 * time.Second,
		},
		HTTPServer: HTTPServer{
			AddressHTTP: ":8080",
			TimeoutHTTP: 30 * time.Second,
			IdleTimeout: 60 * time.Second,
		},
		JWTToken: JWTToken{
			JWTSecretKey: "secret_key_123",
			TokenTTL:     24 * time.Hour,
		},
	}

	result := cfg.String()

	assert.Contains(t, result, "Env: test")
	assert.Contains(t, result, "StorageConnectionString: postgres://user:pass@localhost:5432/db")
	assert.Contains(t, result, "Addr: localhost:6379")
	assert.Contains(t, result, "Password: redis_pass")
	assert.Contains(t, result, "Address: :8080")
	assert.Contains(t, result, "JWTSecretKey: secret_key_123")
	assert.Contains(t, result, "TokenTTL: 24h0m0s")
}

func TestMustLoad_ValidConfig(t *testing.T) {
	// Создаем временный конфиг файл
	configContent := `
env: test
grpc_auth_address: "localhost:50051"
storage_connection_string: "postgres://user:pass@localhost:5432/test"
redis_connection:
  addressredis: "localhost:6379"
  password: "redis_pass"
  user: "redis_user"
  db: 1
  max_retries: 3
  dial_timeout: 5s
  timeoutredis: 10s
http_server:
  addresshttp: ":8080"
  timeouthttp: 30s
  idle_timeout: 60s
jwttoken:
  jwt_secret_key: "test_secret_key"
  token_ttl: 24h
`

	tmpFile, err := os.CreateTemp("", "test_config_*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Устанавливаем переменную окружения
	originalPath := os.Getenv("CONFIG_PATH")
	defer os.Setenv("CONFIG_PATH", originalPath)

	os.Setenv("CONFIG_PATH", tmpFile.Name())

	// Не должно быть ошибок
	output, panicked := captureOutput(func() {
		cfg := MustLoad()

		assert.Equal(t, "test", cfg.Env)
		assert.Equal(t, "localhost:50051", cfg.GRPCAuthAddress)
		assert.Equal(t, "postgres://user:pass@localhost:5432/test", cfg.StorageConnectionString)
		assert.Equal(t, "localhost:6379", cfg.AddressRedis)
		assert.Equal(t, "redis_pass", cfg.Password)
		assert.Equal(t, "redis_user", cfg.User)
		assert.Equal(t, 1, cfg.DB)
		assert.Equal(t, 3, cfg.MaxRetries)
		assert.Equal(t, 5*time.Second, cfg.DialTimeout)
		assert.Equal(t, 10*time.Second, cfg.TimeoutRedis)
		assert.Equal(t, ":8080", cfg.AddressHTTP)
		assert.Equal(t, 30*time.Second, cfg.TimeoutHTTP)
		assert.Equal(t, 60*time.Second, cfg.IdleTimeout)
		assert.Equal(t, "test_secret_key", cfg.JWTSecretKey)
		assert.Equal(t, 24*time.Hour, cfg.TokenTTL)
	})

	assert.Empty(t, output)
	assert.False(t, panicked)
}

func TestConfig_DefaultValues(t *testing.T) {
	// Создаем минимальный конфиг
	configContent := `
env: test
grpc_auth_address: "localhost:50051"
storage_connection_string: "postgres://localhost:5432/test"
redis_connection:
  addressredis: "localhost:6379"
http_server:
  addresshttp: ":8080"
jwttoken:
  jwt_secret_key: "test_secret"
`

	tmpFile, err := os.CreateTemp("", "minimal_config_*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Устанавливаем переменную окружения
	originalPath := os.Getenv("CONFIG_PATH")
	defer os.Setenv("CONFIG_PATH", originalPath)

	os.Setenv("CONFIG_PATH", tmpFile.Name())

	output, panicked := captureOutput(func() {
		cfg := MustLoad()

		// Проверяем что обязательные поля установлены
		assert.Equal(t, "test", cfg.Env)
		assert.Equal(t, "localhost:50051", cfg.GRPCAuthAddress)
		assert.Equal(t, "localhost:6379", cfg.AddressRedis)
		assert.Equal(t, ":8080", cfg.AddressHTTP)
		assert.Equal(t, "test_secret", cfg.JWTSecretKey)

		// Проверяем значения по умолчанию для необязательных полей
		assert.Equal(t, "", cfg.Password)
		assert.Equal(t, "", cfg.User)
		assert.Equal(t, 0, cfg.DB)
		assert.Equal(t, 0, cfg.MaxRetries)
		assert.Equal(t, time.Duration(0), cfg.DialTimeout)
		assert.Equal(t, time.Duration(0), cfg.TimeoutRedis)
		assert.Equal(t, time.Duration(0), cfg.TimeoutHTTP)
		assert.Equal(t, time.Duration(0), cfg.IdleTimeout)
		assert.Equal(t, time.Duration(0), cfg.TokenTTL)
	})

	assert.Empty(t, output)
	assert.False(t, panicked)
}
