package rabbitmq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishMessage(t *testing.T) {
	ctx := context.Background()
	rmqContainer, cleanup := SetupRabbitMQContainer(ctx, t)
	defer cleanup()

	amqpURI, err := GetAmqpURI(ctx, rmqContainer)
	require.NoError(t, err)

	conn, err := Connect(amqpURI, 3, time.Second)
	require.NoError(t, err)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("failed to close connection: %v", err)
		}
	}()

	ch, err := conn.Channel()
	require.NoError(t, err)
	defer func() {
		if err := ch.Close(); err != nil {
			t.Errorf("failed to close channel: %v", err)
		}
	}()

	queueName := "publish-test"
	_, err = ch.QueueDeclare(queueName, false, false, false, false, nil)
	require.NoError(t, err)

	type TestMsg struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	t.Run("success publish and consume", func(t *testing.T) {
		msg := TestMsg{ID: 1, Name: "Hello"}

		// Публикуем сообщение
		err = PublishMessage(ch, "", queueName, msg)
		require.NoError(t, err)

		// Читаем из очереди
		deliveries, err := ch.Consume(queueName, "test-consumer", true, false, false, false, nil)
		require.NoError(t, err)

		select {
		case d := <-deliveries:
			var got TestMsg
			err := json.Unmarshal(d.Body, &got)
			require.NoError(t, err)
			assert.Equal(t, msg, got)
			assert.Equal(t, "application/json", d.ContentType)
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("marshal error", func(t *testing.T) {
		// В json marshal нельзя сериализовать канал
		badMsg := struct {
			Ch chan int `json:"ch"`
		}{
			Ch: make(chan int),
		}

		err := PublishMessage(ch, "", queueName, badMsg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rabbitmq.PublishMessage")
	})
}

func TestPublishMessage_ToExchangeWithRoutingKey(t *testing.T) {
	ctx := context.Background()
	rmqContainer, cleanup := SetupRabbitMQContainer(ctx, t)
	defer cleanup()

	amqpURI, err := GetAmqpURI(ctx, rmqContainer)
	require.NoError(t, err)

	conn, err := Connect(amqpURI, 3, time.Second)
	require.NoError(t, err)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("failed to close connection: %v", err)
		}
	}()

	ch, err := conn.Channel()
	require.NoError(t, err)
	defer func() {
		if err := ch.Close(); err != nil {
			t.Errorf("failed to close channel: %v", err)
		}
	}()

	// Создаем direct exchange
	exchangeName := "test-exchange"
	err = ch.ExchangeDeclare(exchangeName, "direct", true, false, false, false, nil)
	require.NoError(t, err)

	queueName := "route-test"
	_, err = ch.QueueDeclare(queueName, false, false, false, false, nil)
	require.NoError(t, err)

	// биндим очередь к exchange с routing key
	routingKey := "rk"
	err = ch.QueueBind(queueName, routingKey, exchangeName, false, nil)
	require.NoError(t, err)

	msg := map[string]any{"ok": true}

	// публикуем сообщение в обменник
	err = PublishMessage(ch, exchangeName, routingKey, msg)
	require.NoError(t, err)

	deliveries, err := ch.Consume(queueName, "test-consumer2", true, false, false, false, nil)
	require.NoError(t, err)

	select {
	case d := <-deliveries:
		var got map[string]any
		err := json.Unmarshal(d.Body, &got)
		require.NoError(t, err)
		assert.Equal(t, msg["ok"], got["ok"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message via exchange")
	}
}
