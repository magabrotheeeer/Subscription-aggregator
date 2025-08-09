package postgresql

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) *Storage {
	connStr := os.Getenv("TEST_POSTGRES")
	storage, err := New(connStr)
	require.NoError(t, err, "failed to connect test db")

	_, err = storage.Db.Exec(context.Background(), "TRUNCATE susbcriptions, users CASCADE;")
	require.NoError(t, err, "failed to truncate table")
	return storage
}

func TestCreateSubscriptionsEntry(t *testing.T) {

}
