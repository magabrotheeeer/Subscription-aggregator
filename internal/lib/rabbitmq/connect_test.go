package rabbitmq

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SetupRabbitMQContainer(ctx context.Context, t *testing.T) (testcontainers.Container, func()) {
	req := testcontainers.ContainerRequest{
		Image:        "rabbitmq:3-management",
		ExposedPorts: []string{"5672/tcp", "15672/tcp"},
		Env: map[string]string{
			"RABBITMQ_DEFAULT_USER":   "guest",
			"RABBITMQ_DEFAULT_PASS":   "guest",
			"RABBITMQ_DEFAULT_VHOST":  "/",
			"RABBITMQ_LOOPBACK_USERS": "",
		},
		WaitingFor: wait.ForListeningPort("5672/tcp").
			WithStartupTimeout(2 * time.Minute),
	}

	rmqContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	cleanup := func() {
		err := rmqContainer.Terminate(ctx)
		if err != nil {
			t.Logf("failed to terminate rabbitmq container: %v", err)
		}
	}

	return rmqContainer, cleanup
}

func GetAmqpURI(ctx context.Context, container testcontainers.Container) (string, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}
	port, err := container.MappedPort(ctx, "5672/tcp")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port.Port()), nil
}

func TestConnectAndSetupChannel(t *testing.T) {
	ctx := context.Background()

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

	tests := []struct {
		name       string
		amqpURI    string
		queues     []QueueConfig
		wantErr    bool
		errMessage string
	}{
		{
			name:    "valid connection and setup",
			amqpURI: amqpURI,
			queues: []QueueConfig{
				{QueueName: "test-queue", RoutingKey: "test-key"},
			},
			wantErr: false,
		},
		{
			name:    "invalid AMQP URI",
			amqpURI: "amqp://invalid:invalid@localhost:5672/",
			queues:  []QueueConfig{},
			wantErr: true,
		},
		{
			name:    "empty queues list",
			amqpURI: amqpURI,
			queues:  []QueueConfig{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt // локальная копия для параллельных вызовов
		t.Run(tt.name, func(t *testing.T) {
			conn, err := Connect(tt.amqpURI, 3, time.Second)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, conn)
			defer func() {
				if err := conn.Close(); err != nil {
					t.Errorf("failed to close connection: %v", err)
				}
			}()

			ch, err := SetupChannel(conn, tt.queues)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, ch)

			for _, q := range tt.queues {
				queue, err := ch.QueueInspect(q.QueueName)
				require.NoError(t, err)
				assert.Equal(t, q.QueueName, queue.Name)
			}
		})
	}
}
