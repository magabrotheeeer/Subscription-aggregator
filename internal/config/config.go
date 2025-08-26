// Package config предоставялет структуры и функцию для парсинга и загрузки конфига
package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config общая структура для хранения настроек
type Config struct {
	Env                     string `yaml:"env"`
	GRPCAuthAddress         string `yaml:"grpc_auth_address"`
	StorageConnectionString string `yaml:"storage_connection_string"`
	RedisConnection         `yaml:"redis_connection"`
	HTTPServer              `yaml:"http_server"`
	JWTToken                `yaml:"jwttoken"`
	SMTP                    `yaml:"smtp"`
	RabbitMQ                `yaml:"rabbitmq"`
}

type RabbitMQ struct {
	RabbitMQURL        string        `yaml:"rabbitmq_url"`
	RabbitMQMaxRetries int           `yaml:"rabbitmq_maxretries"`
	RabbitMQRetryDelay time.Duration `yaml:"rabbitmq_retry_delay"`
}

type SMTP struct {
	SMTPHost string `yaml:"smtp_host"`
	SMTPPort string `yaml:"smtp_port"`
	SMTPUser string `yaml:"smtp_user"`
	SMTPPass string `yaml:"smtp_pass"`
}

// HTTPServer структура для настройки сервера
type HTTPServer struct {
	AddressHTTP string        `yaml:"addresshttp"`
	TimeoutHTTP time.Duration `yaml:"timeouthttp"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

// RedisConnection структура для настройки подключения к redis
type RedisConnection struct {
	RedisAddress      string        `yaml:"addressredis"`
	RedisPassword     string        `yaml:"password"`
	RedisUser         string        `yaml:"user"`
	RedisDB           int           `yaml:"db"`
	RedisMaxRetries   int           `yaml:"max_retries"`
	RedisDialTimeout  time.Duration `yaml:"dial_timeout"`
	RedisTimeoutRedis time.Duration `yaml:"timeoutredis"`
}

// JWTToken структура для работы с jwt-токеном
type JWTToken struct {
	JWTSecretKey string        `yaml:"jwt_secret_key"`
	TokenTTL     time.Duration `yaml:"token_ttl"`
}

// MustLoad функция для загрузки конфига, возвращает конфиг, сгенерированный из config/config.go
func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("file: %s - does not exist", configPath)
	}
	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}
	return &cfg
}
