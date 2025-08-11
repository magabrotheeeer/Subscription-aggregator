package postgresql

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/http-server/auth"
	countsum "github.com/magabrotheeeer/subscription-aggregator/internal/http-server/handlers/count_sum"
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
	entry := subs.Entry{
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

func TestRemoveSusbscriptionEntry(t *testing.T) {
	storage := setupStorage(t)

	end := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	entry := subs.Entry{
		ServiceName: "yandex-plus",
		Price:       500,
		Username:    "testuser",
		StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     &end,
	}
	ctx := context.Background()
	id, err := storage.CreateSubscriptionEntry(ctx, entry)
	require.NoError(t, err)

	got, err := storage.ReadSubscriptionEntry(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, entry, *got)

	res, err := storage.RemoveSubscriptionEntry(ctx, id)

	assert.Equal(t, int64(1), res, "rows affected should be 1")
	require.NoError(t, err)
	_, err = storage.ReadSubscriptionEntry(ctx, id)
	require.Error(t, err, "expected an error because entry should be deleted")
}

func TestReadSubscriptionEntry(t *testing.T) {
	storage := setupStorage(t)

	entry := subs.Entry{
		ServiceName: "kinopoisk",
		Price:       500,
		Username:    "testuser",
		StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     nil,
	}
	ctx := context.Background()
	id, err := storage.CreateSubscriptionEntry(ctx, entry)
	require.NoError(t, err)

	t.Run("ok, found", func(t *testing.T) {
		got, err := storage.ReadSubscriptionEntry(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, entry, *got)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := storage.ReadSubscriptionEntry(ctx, id+1000)
		assert.Error(t, err)
	})
}

func TestListSubscriptionEntrys(t *testing.T) {
	storage := setupStorage(t)
	ctx := context.Background()

	subsToInsert := []subs.Entry{
		{ServiceName: "netflix", Price: 100, Username: "userA", StartDate: time.Now()},
		{ServiceName: "spotify", Price: 200, Username: "userA", StartDate: time.Now()},
		{ServiceName: "youtube", Price: 300, Username: "userA", StartDate: time.Now()},
		{ServiceName: "kinopoisk", Price: 400, Username: "userB", StartDate: time.Now()},
	}

	for _, entry := range subsToInsert {
		_, err := storage.CreateSubscriptionEntry(ctx, entry)
		require.NoError(t, err)
	}

	t.Run("filter by username", func(t *testing.T) {
		got, err := storage.ListSubscriptionEntrys(ctx, "userA", 10, 0)
		require.NoError(t, err)
		require.Len(t, got, 3)
		for _, sub := range got {
			assert.Equal(t, "userA", sub.Username)
		}
	})

	t.Run("limit and offset", func(t *testing.T) {
		got, err := storage.ListSubscriptionEntrys(ctx, "userA", 2, 0)
		require.NoError(t, err)
		require.Len(t, got, 2)

		got2, err := storage.ListSubscriptionEntrys(ctx, "userA", 2, 2)
		require.NoError(t, err)
		require.Len(t, got2, 1)
	})

	t.Run("no results", func(t *testing.T) {
		got, err := storage.ListSubscriptionEntrys(ctx, "unknown", 10, 0)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestCountSumSubscriptionEntrys(t *testing.T) {
	storage := setupStorage(t)
	ctx := context.Background()

	entries := []subs.Entry{
		{
			ServiceName: "kinopoisk",
			Price:       300,
			Username:    "katya",
			StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     ptrTime(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)),
		},
		{
			ServiceName: "youtube",
			Price:       200,
			Username:    "katya",
			StartDate:   time.Date(2024, 2, 17, 0, 0, 0, 0, time.UTC),
			EndDate:     nil, // открытая подписка
		},
		{
			ServiceName: "kinopoisk",
			Price:       500,
			Username:    "oleg",
			StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     nil,
		},
	}

	for _, entry := range entries {
		_, err := storage.CreateSubscriptionEntry(ctx, entry)
		require.NoError(t, err)
	}

	t.Run("sum all katya", func(t *testing.T) {
		filter := countsum.FilterSum{
			Username:    "katya",
			ServiceName: nil,
			StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     ptrTime(time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)),
		}
		total, err := storage.CountSumSubscriptionEntrys(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 1000.0, total)
	})

	t.Run("only kinopoisk for katya", func(t *testing.T) {
		filter := countsum.FilterSum{
			Username:    "katya",
			ServiceName: ptrString("kinopoisk"),
			StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     ptrTime(time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)),
		}
		total, err := storage.CountSumSubscriptionEntrys(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 600.0, total)
	})

	t.Run("filter by period", func(t *testing.T) {
		filter := countsum.FilterSum{
			Username:    "katya",
			ServiceName: nil,
			StartDate:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     ptrTime(time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)),
		}
		total, err := storage.CountSumSubscriptionEntrys(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 500.0, total)
	})

	t.Run("no results", func(t *testing.T) {
		filter := countsum.FilterSum{Username: "ghost"}
		total, err := storage.CountSumSubscriptionEntrys(ctx, filter)
		require.NoError(t, err)
		assert.Zero(t, total)
	})
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrString(s string) *string     { return &s }

func TestRegisterUser(t *testing.T) {
	storage := setupStorage(t)

	username := "testuser"
	password := "123456"
	hash, err := auth.GetHash(password)
	require.NoError(t, err)

	ctx := context.Background()
	id, err := storage.RegisterUser(ctx, username, hash)
	require.NoError(t, err)
	assert.Greater(t, id, 0, "id should be greater than 0")

	t.Run("repeat register should fail", func(t *testing.T) {
		_, err := storage.RegisterUser(ctx, username, hash)
		assert.Error(t, err)
	})

	got, err := storage.GetUserByUsername(ctx, username)
	require.NoError(t, err)
	assert.Equal(t, username, got.Username)
	assert.Equal(t, hash, got.PasswordHash)
}

func TestGetUserByUsername(t *testing.T) {
	storage := setupStorage(t)
	username := "testuser"
	password := "123456"
	hash, err := auth.GetHash(password)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = storage.RegisterUser(ctx, username, hash)
	require.NoError(t, err)

	t.Run("ok, found", func(t *testing.T) {
		user, err := storage.GetUserByUsername(ctx, username)
		require.NoError(t, err)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, hash, user.PasswordHash)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := storage.GetUserByUsername(ctx, "fakeuser")
		assert.Error(t, err)
	})
}
