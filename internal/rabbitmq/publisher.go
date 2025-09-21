package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/streadway/amqp"
)

// PublishMessage публикует сообщение в RabbitMQ.
func PublishMessage(ch *amqp.Channel, exchange string, routingkey string, message any) error {
	const op = "rabbitmq.PublishMessage"
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = ch.Publish(
		exchange,
		routingkey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
