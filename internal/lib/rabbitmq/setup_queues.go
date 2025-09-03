package rabbitmq

type QueueConfig struct {
	QueueName  string
	RoutingKey string
}

func GetNotificationQueues() []QueueConfig {
	return []QueueConfig{
		{QueueName: "subscription_expiring_queue", RoutingKey: "subscription.expiring.tomorrow"},
		{QueueName: "trial_expiring_queue", RoutingKey: "subscription.trial.expiring"},
	}
}
