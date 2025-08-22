package rabbitmq

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

func Connect(connection string, retries int, delay time.Duration) (*amqp.Connection, error) {
	const op = "rabbitmq.ConnectRabbitMQ"
	var conn *amqp.Connection
	var err error

	for i := 0; i < retries; i++ {
		conn, err = amqp.Dial(connection)
		if err == nil {
			return conn, nil
		}
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("%s: %w", op, err)
}

func SetupChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	const op = "rabbitmq.SetupChannel"
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	err = ch.ExchangeDeclare(
		"notifications",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	_, err = ch.QueueDeclare(
		"notifications.upcoming",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	err = ch.QueueBind("notifications.upcoming", "upcoming", "notifications", false, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return ch, err
}
