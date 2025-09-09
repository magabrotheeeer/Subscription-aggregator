package sender

import (
	"context"
	"log/slog"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/rabbitmq"
	"github.com/magabrotheeeer/subscription-aggregator/internal/lib/smtp"
	senderservice "github.com/magabrotheeeer/subscription-aggregator/internal/services/sender"
	"github.com/magabrotheeeer/subscription-aggregator/internal/storage"
	"github.com/streadway/amqp"
)

type App struct {
	conn          *amqp.Connection
	ch            *amqp.Channel
	senderService *senderservice.SenderService
	logger        *slog.Logger
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*App, error) {
	db, err := storage.New(cfg.StorageConnectionString)
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
		conn.Close()
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
