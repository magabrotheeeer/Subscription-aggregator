package rabbitmq

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumerMessage_HandleMessages(t *testing.T) {
	// Skip RabbitMQ tests in CI due to networking issues
	if os.Getenv("SKIP_RABBITMQ_TESTS") == SkipRabbitMQTestsEnv {
		t.Skip("Skipping RabbitMQ tests in CI")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var amqpURI string
	var cleanup func()

	// Check if we're in CI environment with external RabbitMQ
	if testRabbitMQURL := os.Getenv("TEST_RABBITMQ_URL"); testRabbitMQURL != "" {
		t.Logf("Using external RabbitMQ service: %s", testRabbitMQURL)
		amqpURI = testRabbitMQURL
		cleanup = func() {} // No cleanup needed for external service
	} else {
		t.Log("Using testcontainers for RabbitMQ")
		// Use testcontainers for local development
		rmqContainer, containerCleanup := SetupRabbitMQContainer(ctx, t)
		cleanup = containerCleanup

		var err error
		amqpURI, err = GetAmqpURI(ctx, rmqContainer)
		require.NoError(t, err)
	}
	defer cleanup()

	// Подключаемся
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

	queueName := "consumer-test"
	_, err = ch.QueueDeclare(
		queueName,
		false, false, false, false, nil,
	)
	require.NoError(t, err)

	// Синхронизация через WaitGroup
	var wg sync.WaitGroup
	wg.Add(2)

	received := make([]string, 0)
	var mu sync.Mutex

	handler := func(body []byte) error {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, string(body))
		wg.Done()
		return nil
	}

	// Запуск консьюмера
	err = ConsumerMessage(ctx, ch, queueName, handler)
	require.NoError(t, err)

	// Публикуем 2 сообщения
	for _, msg := range []string{"hello", "world"} {
		err := ch.Publish(
			"", queueName, false, false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(msg),
			},
		)
		require.NoError(t, err)
	}

	// Ждем пока все сообщения обработаются
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for messages to be processed")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, []string{"hello", "world"}, received)
}

func TestConsumerMessage_HandlerErrorTriggersNack(t *testing.T) {
	// Skip RabbitMQ tests in CI due to networking issues
	if os.Getenv("SKIP_RABBITMQ_TESTS") == SkipRabbitMQTestsEnv {
		t.Skip("Skipping RabbitMQ tests in CI")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var amqpURI string
	var cleanup func()

	// Check if we're in CI environment with external RabbitMQ
	if testRabbitMQURL := os.Getenv("TEST_RABBITMQ_URL"); testRabbitMQURL != "" {
		t.Logf("Using external RabbitMQ service: %s", testRabbitMQURL)
		amqpURI = testRabbitMQURL
		cleanup = func() {} // No cleanup needed for external service
	} else {
		t.Log("Using testcontainers for RabbitMQ")
		// Use testcontainers for local development
		rmqContainer, containerCleanup := SetupRabbitMQContainer(ctx, t)
		cleanup = containerCleanup

		var err error
		amqpURI, err = GetAmqpURI(ctx, rmqContainer)
		require.NoError(t, err)
	}
	defer cleanup()

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

	queueName := "nack-test"
	_, err = ch.QueueDeclare(queueName, false, false, false, false, nil)
	require.NoError(t, err)

	// Handler всегда возвращает ошибку → сообщения должны возвращаться в очередь
	handler := func(_ []byte) error {
		return fmt.Errorf("fail")
	}

	err = ConsumerMessage(ctx, ch, queueName, handler)
	require.NoError(t, err)

	// Публикуем сообщение
	err = ch.Publish("", queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("bad"),
	})
	require.NoError(t, err)

	// Проверяем повторную доставку
	deliveries, err := ch.Consume(queueName, "test-consumer", true, false, false, false, nil)
	require.NoError(t, err)

	select {
	case d := <-deliveries:
		assert.Equal(t, "bad", string(d.Body))
	case <-time.After(10 * time.Second):
		t.Fatal("Did not receive requeued message after Nack")
	}
}
