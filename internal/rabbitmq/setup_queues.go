package rabbitmq

// QueueConfig содержит конфигурацию очереди RabbitMQ.
type QueueConfig struct {
	QueueName  string
	RoutingKey string
}

// GetNotificationQueues возвращает конфигурацию очередей для уведомлений.
func GetNotificationQueues() []QueueConfig {
	return []QueueConfig{
		{QueueName: "subscription_expiring_queue", RoutingKey: "subscription.expiring.tomorrow"},
		{QueueName: "trial_expiring_queue", RoutingKey: "subscription.trial.expiring"},
	}
}
