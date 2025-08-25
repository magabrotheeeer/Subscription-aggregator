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

	for range retries {
		conn, err = amqp.Dial(connection)
		if err == nil {
			return conn, nil
		}
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("%s: %w", op, err)
}

func SetupChannel(conn *amqp.Connection, queues []QueueConfig) (*amqp.Channel, error) {
	const op = "rabbitmq.SetupChannel"

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err := ch.Qos(10, 0, false); err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	err = ch.ExchangeDeclare(
		"notifications", // exchange
		"direct",        // тип
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for _, q := range queues {
		_, err := ch.QueueDeclare(
			q.QueueName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to declare queue %s: %w", op, q.QueueName, err)
		}

		err = ch.QueueBind(
			q.QueueName,
			q.RoutingKey,
			"notifications",
			false,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to bind queue %s with routing key %s: %w", op, q.QueueName, q.RoutingKey, err)
		}
	}

	return ch, nil
}
