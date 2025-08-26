package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNotificationQueues(t *testing.T) {
	queues := GetNotificationQueues()

	require.NotEmpty(t, queues, "queues list should not be empty")

	// Проверка первой очереди
	first := queues[0]
	assert.Equal(t, "notification.upcoming", first.QueueName)
	assert.Equal(t, "upcoming", first.RoutingKey)

	// Проверка уникальности QueueName
	seen := map[string]bool{}
	for _, q := range queues {
		assert.Falsef(t, seen[q.QueueName], "duplicate queue name: %s", q.QueueName)
		seen[q.QueueName] = true
	}
}
