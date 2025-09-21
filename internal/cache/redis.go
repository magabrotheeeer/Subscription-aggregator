// Package cache предоставляет функционал для работы с кэшированием данных в Redis.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/redis/go-redis/v9"
)

// Cache представляет собой обёртку над клиентом Redis.
// Содержит подключение к базе данных Redis.
type Cache struct {
	DB *redis.Client
}

// InitServer инициализирует подключение к Redis и возвращает структуру Cache.
// Настройки подключения берутся из переданного конфигурационного объекта cfg.
// При невозможности подключения возвращается ошибка.
func InitServer(ctx context.Context, cfg config.RedisConnection) (*Cache, error) {
	const op = "cache.InitServer"

	db := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddress,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		Username:     cfg.RedisUser,
		MaxRetries:   cfg.RedisMaxRetries,
		DialTimeout:  cfg.RedisDialTimeout,
		ReadTimeout:  cfg.RedisTimeoutRedis,
		WriteTimeout: cfg.RedisTimeoutRedis,
	})

	if err := db.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Cache{DB: db}, nil
}

// Get получает значение из Redis по ключу key и записывает его в result.
// Возвращает true, если значение найдено, и false — если ключ отсутствует.
// В случае ошибки возвращает её.
func (c *Cache) Get(key string, result any) (bool, error) {
	const op = "cache.Get"

	val, err := c.DB.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	err = json.Unmarshal([]byte(val), result)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

// Set записывает значение value по ключу key в Redis.
// expiration задаёт время жизни ключа. При ошибке возвращает её.
func (c *Cache) Set(key string, value any, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.DB.Set(context.Background(), key, jsonData, expiration).Err()
}

// Invalidate удаляет значение из Redis по указанному ключу key.
// Возвращает ошибку, если операция удаления не удалась.
func (c *Cache) Invalidate(key string) error {
	res, err := c.DB.Del(context.Background(), key).Result()
	if err != nil {
		return err
	}
	if res == 0 {
		return fmt.Errorf("key %q not found", key)
	}
	return nil
}
