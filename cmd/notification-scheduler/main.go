package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/rabbitmq"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
)

func main() {
	cfg := config.MustLoad()
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting notification-scheduler", slog.String("env", cfg.Env))
	conn, err := rabbitmq.Connect(cfg.RabbitMQURL, cfg.RabbitMQMaxRetries, cfg.RabbitMQRetryDelay)
	if err != nil {
		logger.Error("failed to connect to RabbitMQ", sl.Err(err))
		os.Exit(1)
	}
	logger.Info("succes to connect to RabbitMQ:", slog.String("URL", cfg.RabbitMQURL))
	defer func() {
		_ = conn.Close()
	}()

	ch, err := rabbitmq.SetupChannel(conn)
	if err != nil {
		logger.Error("failed to setup RabbitMQ channel", sl.Err(err))
		os.Exit(1)
	}
	logger.Info("success to setup RabbitMQ channel")
	defer func() {
		_ = ch.Close()
	}()
	db, err := storage.New(cfg.StorageConnectionString)
	if err != nil {
		logger.Error("failed to connect to storage", sl.Err(err))
		os.Exit(1)
	}
	defer func() {
		_ = db.Db.Close()
	}()

	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		logger.Info("starting service to find expiring subscriptions")
		entriesInfo, err := db.FindSubscriptionExpiringTomorrow(ctx)
		if err != nil {
			logger.Error("failed to find entries", sl.Err(err))
		}
		for _, entryInfo := range entriesInfo {
			err = rabbitmq.PublishMessage(ch, "notifications", "upcoming", entryInfo)
			if err != nil {
				logger.Error("failed to publish message", sl.Err(err))
			}
		}
	}
}
