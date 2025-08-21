package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
)

func ConsumerMessage(ch *amqp.Channel, queueName string, handler func([]byte) error) error {
	const op = "rabbitmq.ConsumerMessage"
	delivery, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	go func() {
		for d := range delivery {
			if err := handler(d.Body); err != nil {
				d.Nack(false, true)
				continue
			}
			d.Ack(false)
		}
	}()
	return nil
}
