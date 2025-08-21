package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/magabrotheeeer/subscription-aggregator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDb(t *testing.T) (*Storage, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		).WithDeadline(3 * time.Minute),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start container")

	// Добавляем задержку для полной инициализации PostgreSQL
	time.Sleep(3 * time.Second)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get port")

	connStr := fmt.Sprintf("postgres://testuser:testpass@localhost:%s/testdb?sslmode=disable", port.Port())

	// Пробуем подключиться несколько раз с ретраями
	var storage *Storage
	for i := 0; i < 10; i++ {
		storage, err = New(connStr)
		if err == nil {
			// Проверяем, что подключение действительно работает
			err = storage.Db.Ping()
			if err == nil {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	require.NoError(t, err, "Failed to create storage after retries")

	// Создаем таблицы
	_, err = storage.Db.Exec(`
        DROP TABLE IF EXISTS subscriptions CASCADE;
        DROP TABLE IF EXISTS users CASCADE;
        
        CREATE TABLE subscriptions (
            id SERIAL PRIMARY KEY,
            service_name TEXT NOT NULL,
            price INT NOT NULL,
            username TEXT NOT NULL,
            start_date DATE NOT NULL,
            counter_months INT NOT NULL
        );
    `)
	require.NoError(t, err, "Failed to create subscription table")

	_, err = storage.Db.Exec(`
        CREATE TABLE users (
            id SERIAL PRIMARY KEY,
            username TEXT NOT NULL UNIQUE,
            email TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            role TEXT NOT NULL DEFAULT 'user'
        );	
    `)
	require.NoError(t, err, "Failed to create user table")

	cleanup := func() {
		if storage != nil && storage.Db != nil {
			storage.Db.Close()
		}
		if postgresContainer != nil {
			postgresContainer.Terminate(ctx)
		}
	}

	return storage, cleanup
}

func TestStorage_Create(t *testing.T) {
	type args struct {
		ctx   context.Context
		entry models.Entry
	}

	tests := []struct {
		name   string
		args   args
		wantID int
		verify func(t *testing.T, s *Storage, id int)
	}{
		{
			name: "successful create entry",
			args: args{
				ctx: context.Background(),
				entry: models.Entry{
					ServiceName:   "Spotify",
					Price:         500,
					Username:      "testuser",
					StartDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					CounterMonths: 6,
				},
			},
			wantID: 1,
			verify: func(t *testing.T, s *Storage, id int) {
				var count int
				err := s.Db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE id = $1", id).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDb(t)
			defer cleanup()

			gotID, err := storage.Create(tt.args.ctx, tt.args.entry)
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, gotID)
			tt.verify(t, storage, gotID)
		})
	}
}

func TestStorage_Remove(t *testing.T) {
	type args struct {
		ctx context.Context
		id  int
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name             string
		args             args
		wantRowsAffected int
		wantError        bool
		setup            func(s *Storage)
		verify           func(t *testing.T, s *Storage, id int)
	}{
		{
			name: "successful delete entry",
			args: args{
				ctx: context.Background(),
				id:  1,
			},
			wantRowsAffected: 1,
			wantError:        false,
			setup: func(s *Storage) {
				_, err := s.Db.Exec(`INSERT INTO subscriptions
					(service_name, price, username, start_date, counter_months)
				VALUES($1, $2, $3, $4, $5)`,
					"Netflix", 1000, "testuser", startDate, 5)
				require.NoError(t, err)
			},
			verify: func(t *testing.T, s *Storage, id int) {
				var count int
				err := s.Db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE id = $1", id).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "invalid id",
			args: args{
				ctx: context.Background(),
				id:  9999,
			},
			wantRowsAffected: 0,
			wantError:        true,
			setup: func(s *Storage) {
				_, err := s.Db.Exec(`INSERT INTO subscriptions
					(service_name, price, username, start_date, counter_months)
				VALUES($1, $2, $3, $4, $5)`,
					"Netflix", 1000, "testuser", startDate, 5)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDb(t)
			defer cleanup()
			tt.setup(storage)
			gotRowsAffected, err := storage.Remove(tt.args.ctx, tt.args.id)

			require.NoError(t, err)
			assert.Equal(t, tt.wantRowsAffected, gotRowsAffected)
			if tt.name == "successful delete entry" {
				tt.verify(t, storage, 1)
			}
		})
	}
}

func TestStorage_Read(t *testing.T) {
	type args struct {
		ctx context.Context
		id  int
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		args    args
		want    *models.Entry
		wantErr bool
		setup   func(s *Storage)
	}{
		{
			name: "successful read existing entry",
			args: args{
				ctx: context.Background(),
				id:  1,
			},
			want: &models.Entry{
				ServiceName:   "Netflix",
				Price:         1000,
				Username:      "testuser",
				StartDate:     startDate,
				CounterMonths: 12,
			},
			wantErr: false,
			setup: func(s *Storage) {
				_, err := s.Db.Exec(`INSERT INTO subscriptions 
                    (service_name, price, username, start_date, counter_months)
                    VALUES ($1, $2, $3, $4, $5)`,
					"Netflix", 1000, "testuser", startDate, 12)
				require.NoError(t, err)
			},
		},
		{
			name: "read non-existing entry",
			args: args{
				ctx: context.Background(),
				id:  999,
			},
			want:    nil,
			wantErr: true,
			setup:   func(_ *Storage) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDb(t)
			defer cleanup()

			tt.setup(storage)

			got, err := storage.Read(tt.args.ctx, tt.args.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, tt.want.ServiceName, got.ServiceName)
			assert.Equal(t, tt.want.Price, got.Price)
			assert.Equal(t, tt.want.Username, got.Username)
			assert.True(t, tt.want.StartDate.Equal(got.StartDate))
			assert.Equal(t, tt.want.CounterMonths, got.CounterMonths)
		})
	}
}

func TestStorage_CountSum(t *testing.T) {
	type args struct {
		ctx    context.Context
		filter models.FilterSum
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	netflixService := "Netflix"

	tests := []struct {
		name      string
		args      args
		wantTotal float64
		wantErr   bool
		setup     func(s *Storage)
	}{
		{
			name: "count sum for single subscription",
			args: args{
				ctx: context.Background(),
				filter: models.FilterSum{
					Username:      "testuser",
					ServiceName:   nil, // Нет фильтра по service_name
					StartDate:     startDate,
					CounterMonths: 12,
				},
			},
			wantTotal: 12000.0, // 1000.0 * 12 месяцев
			wantErr:   false,
			setup: func(s *Storage) {
				_, err := s.Db.Exec(`INSERT INTO subscriptions 
                    (service_name, price, username, start_date, counter_months)
                    VALUES ($1, $2, $3, $4, $5)`,
					"Netflix", 1000.0, "testuser", startDate, 12)
				require.NoError(t, err)
			},
		},
		{
			name: "count sum for filtered service",
			args: args{
				ctx: context.Background(),
				filter: models.FilterSum{
					Username:      "testuser",
					ServiceName:   &netflixService, // Фильтр по Netflix
					StartDate:     startDate,
					CounterMonths: 12,
				},
			},
			wantTotal: 12000.0, // 1000.0 * 12 месяцев (только Netflix)
			wantErr:   false,
			setup: func(s *Storage) {
				_, err := s.Db.Exec(`INSERT INTO subscriptions 
                    (service_name, price, username, start_date, counter_months)
                    VALUES ($1, $2, $3, $4, $5)`,
					"Netflix", 1000.0, "testuser", startDate, 12)
				require.NoError(t, err)

				_, err = s.Db.Exec(`INSERT INTO subscriptions 
                    (service_name, price, username, start_date, counter_months)
                    VALUES ($1, $2, $3, $4, $5)`,
					"Spotify", 1000.0, "testuser", startDate, 12)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDb(t)
			defer cleanup()

			tt.setup(storage)

			gotTotal, err := storage.CountSum(tt.args.ctx, tt.args.filter)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.InDelta(t, tt.wantTotal, gotTotal, 0.001)
		})
	}
}

func TestStorage_RegisterUser(t *testing.T) {
	type args struct {
		ctx  context.Context
		user models.User
	}

	tests := []struct {
		name    string
		args    args
		wantID  int
		wantErr bool
		verify  func(t *testing.T, s *Storage, id int)
		setup   func(s *Storage)
	}{
		{
			name: "successful register user",
			args: args{
				ctx: context.Background(),
				user: models.User{
					Email:        "test@example.com",
					Username:     "testuser",
					PasswordHash: "hashedpassword",
					Role:         "user",
				},
			},
			wantID:  1,
			wantErr: false,
			verify: func(t *testing.T, s *Storage, id int) {
				var count int
				err := s.Db.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", id).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "register user with duplicate username",
			args: args{
				ctx: context.Background(),
				user: models.User{
					Email:        "test2@example.com",
					Username:     "testuser", // Дубликат
					PasswordHash: "hashedpassword2",
					Role:         "user",
				},
			},
			wantID:  0,
			wantErr: true,
			setup: func(s *Storage) {
				_, err := s.Db.Exec(`INSERT INTO users 
                    (email, username, password_hash, role)
                    VALUES ($1, $2, $3, $4)`,
					"test@example.com", "testuser", "hashedpassword", "user")
				require.NoError(t, err)
			},
			verify: func(_ *testing.T, _ *Storage, _ int) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDb(t)
			defer cleanup()

			if tt.setup != nil {
				tt.setup(storage)
			}

			gotID, err := storage.RegisterUser(tt.args.ctx, tt.args.user)

			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantID, gotID)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantID, gotID)

			tt.verify(t, storage, gotID)
		})
	}
}

func TestStorage_GetUserByUsername(t *testing.T) {
	type args struct {
		ctx      context.Context
		username string
	}

	tests := []struct {
		name    string
		args    args
		want    *models.User
		wantErr bool
		setup   func(s *Storage)
	}{
		{
			name: "successful get user by username",
			args: args{
				ctx:      context.Background(),
				username: "testuser",
			},
			want: &models.User{
				ID:           1,
				Email:        "test@example.com",
				Username:     "testuser",
				PasswordHash: "hashedpassword",
				Role:         "user",
			},
			wantErr: false,
			setup: func(s *Storage) {
				_, err := s.Db.Exec(`INSERT INTO users 
                    (email, username, password_hash, role)
                    VALUES ($1, $2, $3, $4)`,
					"test@example.com", "testuser", "hashedpassword", "user")
				require.NoError(t, err)
			},
		},
		{
			name: "get non-existing user",
			args: args{
				ctx:      context.Background(),
				username: "nonexistent",
			},
			want:    nil,
			wantErr: true,
			setup:   func(_ *Storage) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, cleanup := setupTestDb(t)
			defer cleanup()

			tt.setup(storage)

			got, err := storage.GetUserByUsername(tt.args.ctx, tt.args.username)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, tt.want.ID, got.ID)
			assert.Equal(t, tt.want.Email, got.Email)
			assert.Equal(t, tt.want.Username, got.Username)
			assert.Equal(t, tt.want.PasswordHash, got.PasswordHash)
			assert.Equal(t, tt.want.Role, got.Role)
		})
	}
}
