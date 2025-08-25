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

	sem := make(chan struct{}, 10)
	go func() {
		for d := range delivery {
			sem <- struct{}{}
			go func(delivery amqp.Delivery) {
				defer func() { <-sem }()
				if err := handler(delivery.Body); err != nil {
					delivery.Nack(false, true)
					return
				}
				delivery.Ack(false)
			}(d)
		}
	}()
	return nil
}
