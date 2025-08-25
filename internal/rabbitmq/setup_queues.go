package rabbitmq

type QueueConfig struct {
	QueueName  string
	RoutingKey string
}

func GetNotificationQueues() []QueueConfig {
	return []QueueConfig{
		{QueueName: "notification.upcoming", RoutingKey: "upcoming"},
		// при необходимости дополнительные очереди для других воркеров
	}
}
