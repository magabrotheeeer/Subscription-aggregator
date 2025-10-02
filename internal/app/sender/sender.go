// Package sender содержит логику отправки уведомлений.
package sender

import (
	"context"
	"log/slog"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/smtp"
	"github.com/magabrotheeeer/subscription-aggregator/internal/rabbitmq"
	senderservice "github.com/magabrotheeeer/subscription-aggregator/internal/services/sender"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage/repository"
	"github.com/streadway/amqp"
)

// App представляет приложение отправителя уведомлений.
type App struct {
	conn          *amqp.Connection
	ch            *amqp.Channel
	senderService *senderservice.SenderService
	logger        *slog.Logger
}

// New создает новый экземпляр приложения отправителя.
func New(_ context.Context, cfg *config.Config, logger *slog.Logger) (*App, error) {
	db, err := repository.New(cfg.StorageConnectionString)
	if err != nil {
		return nil, err
	}
	conn, err := rabbitmq.Connect(cfg.RabbitMQURL, cfg.RabbitMQMaxRetries, cfg.RabbitMQRetryDelay)
	if err != nil {
		return nil, err
	}

	queues := rabbitmq.GetNotificationQueues()
	ch, err := rabbitmq.SetupChannel(conn, queues)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			logger.Error("failed to close connection", "error", closeErr)
		}
		return nil, err
	}

	newTransport := smtp.NewTransport(cfg, logger)
	senderService := senderservice.NewSenderService(db, logger, newTransport)

	return &App{
		conn:          conn,
		ch:            ch,
		senderService: senderService,
		logger:        logger,
	}, nil
}

// Run запускает отправитель уведомлений.
func (a *App) Run(ctx context.Context) error {
	err := rabbitmq.ConsumerMessage(ctx, a.ch, "subscription_expiring_queue", a.senderService.SendInfoExpiringSubscription)
	if err != nil {
		a.logger.Error("failed to start subscription_expiring_queue consumer", slog.Any("err", err))
		return err
	}

	err = rabbitmq.ConsumerMessage(ctx, a.ch, "trial_expiring_queue", a.senderService.SendInfoExpiringTrialPeriodSubscription)
	if err != nil {
		a.logger.Error("failed to start trial_expiring_queue consumer", slog.Any("err", err))
		return err
	}

	<-ctx.Done()
	a.logger.Info("Sender service shutting down gracefully")

	if err := a.ch.Close(); err != nil {
		a.logger.Error("failed to close channel", slog.Any("err", err))
	}

	if err := a.conn.Close(); err != nil {
		a.logger.Error("failed to close connection", slog.Any("err", err))
	}

	return nil
}
