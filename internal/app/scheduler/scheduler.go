// Package scheduler содержит логику планировщика для обработки подписок.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/cache"
	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/rabbitmq"
	schedulerservice "github.com/magabrotheeeer/subscription-aggregator/internal/services/scheduler"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
	"github.com/streadway/amqp"
)

// App представляет приложение планировщика.
type App struct {
	schedulerService *schedulerservice.SchedulerService
	conn             *amqp.Connection
	ch               *amqp.Channel
	logger           *slog.Logger
}

func waitForDB(db *storage.Storage) error {
	for i := 0; i < 10; i++ {
		err := storage.CheckDatabaseReady(db)
		if err == nil {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("database not ready after retries")
}

// New создает новый экземпляр приложения планировщика.
func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*App, error) {
	conn, err := rabbitmq.Connect(cfg.RabbitMQURL, cfg.RabbitMQMaxRetries, cfg.RabbitMQRetryDelay)
	if err != nil {
		return nil, fmt.Errorf("failed to connect RabbitMQ: %w", err)
	}

	queues := rabbitmq.GetNotificationQueues()
	ch, err := rabbitmq.SetupChannel(conn, queues)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			logger.Error("failed to close connection", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to setup RabbitMQ channel: %w", err)
	}

	db, err := storage.New(cfg.StorageConnectionString)
	if err != nil {
		if closeErr := ch.Close(); closeErr != nil {
			logger.Error("failed to close channel", "error", closeErr)
		}
		if closeErr := conn.Close(); closeErr != nil {
			logger.Error("failed to close connection", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to connect storage: %w", err)
	}

	if err := waitForDB(db); err != nil {
		if closeErr := ch.Close(); closeErr != nil {
			logger.Error("failed to close channel", "error", closeErr)
		}
		if closeErr := conn.Close(); closeErr != nil {
			logger.Error("failed to close connection", "error", closeErr)
		}
		return nil, err
	}

	cacheRedis, err := cache.InitServer(ctx, cfg.RedisConnection)
	if err != nil {
		if closeErr := ch.Close(); closeErr != nil {
			logger.Error("failed to close channel", "error", closeErr)
		}
		if closeErr := conn.Close(); closeErr != nil {
			logger.Error("failed to close connection", "error", closeErr)
		}
		return nil, fmt.Errorf("cache not initialized: %w", err)
	}

	schedulerService := schedulerservice.NewSchedulerService(db, cacheRedis, logger)

	return &App{
		schedulerService: schedulerService,
		conn:             conn,
		ch:               ch,
		logger:           logger,
	}, nil
}

// Run запускает планировщик.
func (a *App) Run(ctx context.Context) error {
	go a.schedulerService.FindExpiringSubscriptionsDueTomorrow(ctx, a.ch)
	go a.schedulerService.FindExpiringSubscriptionsDueToday(ctx, a.ch)

	<-ctx.Done()

	a.logger.Info("shutting down scheduler service")

	if err := a.ch.Close(); err != nil {
		a.logger.Error("failed to close channel", slog.Any("err", err))
	}

	if err := a.conn.Close(); err != nil {
		a.logger.Error("failed to close connection", slog.Any("err", err))
	}

	return nil
}
