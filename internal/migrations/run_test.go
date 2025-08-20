package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func getTestDB(t *testing.T) (*sql.DB, func()) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	require.NoError(t, err)

	dsn, err := pgContainer.ConnectionString(ctx)
	require.NoError(t, err)

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}

	return db, cleanup
}

func getMigrationsPath(t *testing.T) string {
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	migrationsPath := filepath.Join(projectRoot, "migrations")
	t.Logf("Migrations path: %s", migrationsPath)
	return migrationsPath
}

func TestRunMigrations(t *testing.T) {
	db, cleanup := getTestDB(t)
	defer cleanup()

	migrationsPath := getMigrationsPath(t)

	err := Run(db, migrationsPath)
	require.NoError(t, err)

	var tablesCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public'
	`).Scan(&tablesCount)
	require.NoError(t, err)
	require.Greater(t, tablesCount, 0, "Should have tables after migration")

	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'users'
		)
	`).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists, "Table 'users' should exist")

	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = 'subscriptions'
		)
	`).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists, "Table 'subscriptions' should exist")

	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE schemaname = 'public' 
			AND tablename = 'subscriptions' 
			AND indexname = 'idx_subscriptions_username'
		)
	`).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists, "Index should exist")

	var adminCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&adminCount)
	require.NoError(t, err)
	require.Equal(t, 1, adminCount, "Should have one admin user")
}

func TestMigrationIdempotency(t *testing.T) {
	db, cleanup := getTestDB(t)
	defer cleanup()

	migrationsPath := getMigrationsPath(t)

	err := Run(db, migrationsPath)
	require.NoError(t, err)

	err = Run(db, migrationsPath)
	require.True(t, err == nil || err.Error() == "no change",
		"Running migrations twice should not fail. Got error: %v", err)

	var adminCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&adminCount)
	require.NoError(t, err)
	require.Equal(t, 1, adminCount, "Should still have one admin user after second run")
}
