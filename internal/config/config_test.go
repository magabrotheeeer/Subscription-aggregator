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
	defer func() {
		err = os.Remove(tmpFile.Name())
		require.NoError(t, err)
	}()

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)

	// Устанавливаем переменную окружения
	originalPath := os.Getenv("CONFIG_PATH")
	defer func() {
		err = os.Setenv("CONFIG_PATH", originalPath)
		require.NoError(t, err)
	}()

	err = os.Setenv("CONFIG_PATH", tmpFile.Name())
	require.NoError(t, err)

	// Не должно быть ошибок
	output, panicked := captureOutput(func() {
		cfg := MustLoad()

		assert.Equal(t, "test", cfg.Env)
		assert.Equal(t, "localhost:50051", cfg.GRPCAuthAddress)
		assert.Equal(t, "postgres://user:pass@localhost:5432/test", cfg.StorageConnectionString)
		assert.Equal(t, "localhost:6379", cfg.RedisAddress)
		assert.Equal(t, "redis_pass", cfg.RedisPassword)
		assert.Equal(t, "redis_user", cfg.RedisUser)
		assert.Equal(t, 1, cfg.RedisDB)
		assert.Equal(t, 3, cfg.RedisMaxRetries)
		assert.Equal(t, 5*time.Second, cfg.RedisDialTimeout)
		assert.Equal(t, 10*time.Second, cfg.RedisTimeoutRedis)
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
	defer func() {
		err = os.Remove(tmpFile.Name())
		require.NoError(t, err)
	}()

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)

	// Устанавливаем переменную окружения
	originalPath := os.Getenv("CONFIG_PATH")
	defer func() {
		err = os.Setenv("CONFIG_PATH", originalPath)
		require.NoError(t, err)
	}()

	err = os.Setenv("CONFIG_PATH", tmpFile.Name())
	require.NoError(t, err)

	output, panicked := captureOutput(func() {
		cfg := MustLoad()

		// Проверяем что обязательные поля установлены
		assert.Equal(t, "test", cfg.Env)
		assert.Equal(t, "localhost:50051", cfg.GRPCAuthAddress)
		assert.Equal(t, "localhost:6379", cfg.RedisAddress)
		assert.Equal(t, ":8080", cfg.AddressHTTP)
		assert.Equal(t, "test_secret", cfg.JWTSecretKey)

		// Проверяем значения по умолчанию для необязательных полей
		assert.Equal(t, "", cfg.RedisPassword)
		assert.Equal(t, "", cfg.RedisUser)
		assert.Equal(t, 0, cfg.RedisDB)
		assert.Equal(t, 0, cfg.RedisMaxRetries)
		assert.Equal(t, time.Duration(0), cfg.RedisDialTimeout)
		assert.Equal(t, time.Duration(0), cfg.RedisTimeoutRedis)
		assert.Equal(t, time.Duration(0), cfg.TimeoutHTTP)
		assert.Equal(t, time.Duration(0), cfg.IdleTimeout)
		assert.Equal(t, time.Duration(0), cfg.TokenTTL)
	})

	assert.Empty(t, output)
	assert.False(t, panicked)
}
