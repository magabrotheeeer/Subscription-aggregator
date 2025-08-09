package postgresql

import (
	"context"
	"os"
	"testing"
	"time"

	subs "github.com/magabrotheeeer/subscription-aggregator/internal/subscription"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStorage(t *testing.T) *Storage {
	connStr := os.Getenv("TEST_POSTGRES")
	storage, err := New(connStr)
	require.NoError(t, err, "failed to connect test db")

	_, err = storage.Db.Exec(context.Background(), "TRUNCATE subscriptions, users CASCADE;")
	require.NoError(t, err, "failed to truncate table")
	return storage
}

func TestCreateSubscriptionsEntry(t *testing.T) {
	storage := setupStorage(t)

	end := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	entry := subs.SubscriptionEntry{
		ServiceName: "yandex-plus",
		Price:       500,
		Username:    "testuser",
		StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     &end,
	}

	ctx := context.Background()
	id, err := storage.CreateSubscriptionEntry(ctx, entry)
	require.NoError(t, err)
	assert.Greater(t, id, 0, "id should be greater than 0")

	got, err := storage.ReadSubscriptionEntry(ctx, id)
	require.NoError(t, err)

	assert.Equal(t, entry, *got)
}
