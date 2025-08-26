package rabbitmq

type QueueConfig struct {
	QueueName  string
	RoutingKey string
}

func GetNotificationQueues() []QueueConfig {
	return []QueueConfig{
		{QueueName: "notification.upcoming", RoutingKey: "upcoming"},
		{QueueName: "payment.due", RoutingKey: "due"},
	}
}
