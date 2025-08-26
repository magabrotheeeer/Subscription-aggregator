package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/sl"
	"github.com/magabrotheeeer/subscription-aggregator/internal/rabbitmq"
	services "github.com/magabrotheeeer/subscription-aggregator/internal/services/notification-sender"
	"github.com/magabrotheeeer/subscription-aggregator/internal/smtp"
)

func main() {
	ctx := context.Background()
	cfg := config.MustLoad()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Info("starting notification-sender", slog.String("env", cfg.Env))
	conn, err := rabbitmq.Connect(cfg.RabbitMQURL, cfg.RabbitMQMaxRetries, cfg.RabbitMQRetryDelay)
	if err != nil {
		logger.Error("failed to connect to RabbitMQ", sl.Err(err))
		os.Exit(1)
	}
	logger.Info("succes to connect to RabbitMQ:", slog.String("URL", cfg.RabbitMQURL))
	defer func() {
		_ = conn.Close()
	}()

	queues := rabbitmq.GetNotificationQueues()
	ch, err := rabbitmq.SetupChannel(conn, queues)
	if err != nil {
		logger.Error("failed to setup RabbitMQ channel", sl.Err(err))
		os.Exit(1)
	}
	logger.Info("success to setup RabbitMQ channel")
	defer func() {
		_ = ch.Close()
	}()

	newTransport := smtp.NewTransport(cfg, logger)

	senderService := services.NewSenderService(cfg, logger, newTransport)

	err = rabbitmq.ConsumerMessage(ctx, ch, "notification.upcoming", senderService.SendInfoExpiringSubscription)
	if err != nil {
		logger.Error("failed to start consumer", sl.Err(err))
		os.Exit(1)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	<-sigterm

	logger.Info("Notification sender shutting down gracefully")
}
