package rabbitmq

import (
	"context"
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

// ConsumerMessage создает потребителя сообщений из очереди RabbitMQ.
func ConsumerMessage(ctx context.Context, ch *amqp.Channel, queueName string, handler func([]byte) error) error {
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
		for {
			select {
			case d, ok := <-delivery:
				if !ok {
					return
				}
				sem <- struct{}{}
				go func(delivery amqp.Delivery) {
					defer func() { <-sem }()
					if err := handler(delivery.Body); err != nil {
						if nackErr := delivery.Nack(false, true); nackErr != nil {
							log.Printf("failed to nack message: %v", nackErr)
						}
						return
					}
					if ackErr := delivery.Ack(false); ackErr != nil {
						log.Printf("failed to ack message: %v", ackErr)
					}
				}(d)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}
