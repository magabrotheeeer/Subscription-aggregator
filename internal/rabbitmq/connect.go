package rabbitmq

import (
	"fmt"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/config"
	"github.com/streadway/amqp"
)

func Connect(cfg config.Config) (*amqp.Connection, error) {
	const op = "rabbitmq.ConnectRabbitMQ"
	var conn *amqp.Connection
	var err error

	for i := 0; i < cfg.RabbitMQMaxRetries; i++ {
		conn, err = amqp.Dial(cfg.RabbitMQURL)
		if err == nil {
			return conn, nil
		}
		time.Sleep(cfg.RabbitMQRetryDelay)
	}

	return nil, fmt.Errorf("%s: %w", op, err)
}
