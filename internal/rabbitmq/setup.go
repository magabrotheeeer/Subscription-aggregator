package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
)

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
