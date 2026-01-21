// Package testenv provides ephemeral test infrastructure using testcontainers.
package testenv

import (
	"context"
	"fmt"
	"time"

	"github.com/gti/heatmap-internal/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer holds the ephemeral PostgreSQL container and connection.
type PostgresContainer struct {
	// Container is the testcontainers container instance.
	Container testcontainers.Container

	// Pool is the connection pool to the container database.
	Pool *pgxpool.Pool

	// ConnectionString is the PostgreSQL connection URL.
	ConnectionString string

	// Host is the container host (for service connection).
	Host string

	// Port is the mapped PostgreSQL port.
	Port string
}

// PostgresConfig holds configuration for the PostgreSQL container.
type PostgresConfig struct {
	// Image is the PostgreSQL Docker image (default: postgres:16-alpine).
	Image string

	// Database is the database name (default: test_db).
	Database string

	// Username is the database user (default: test_user).
	Username string

	// Password is the database password (default: test_pass).
	Password string
}

// DefaultPostgresConfig returns default PostgreSQL container configuration.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Image:    "postgres:16-alpine",
		Database: "load_calendar_test",
		Username: "test_user",
		Password: "test_pass",
	}
}

// StartPostgres spins up an ephemeral PostgreSQL container for testing.
//
// The container runs PostgreSQL 16 (or configured version) with the
// project's schema migrations applied automatically.
//
// Returns the container wrapper with connection pool and cleanup function.
// Always call the cleanup function when done, typically with defer:
//
//	pg, cleanup, err := StartPostgres(ctx, DefaultPostgresConfig())
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer cleanup()
func StartPostgres(ctx context.Context, cfg PostgresConfig) (*PostgresContainer, func(), error) {
	// Start PostgreSQL container
	container, err := postgres.Run(ctx,
		cfg.Image,
		postgres.WithDatabase(cfg.Database),
		postgres.WithUsername(cfg.Username),
		postgres.WithPassword(cfg.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	// Get connection string
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	// Get host and port for external connections
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		container.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Connect to database
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		container.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		container.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Run migrations using the project's migration code
	db := &database.DB{Pool: pool}
	if err := db.RunMigrations(ctx); err != nil {
		pool.Close()
		container.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	pg := &PostgresContainer{
		Container:        container,
		Pool:             pool,
		ConnectionString: connStr,
		Host:             host,
		Port:             mappedPort.Port(),
	}

	cleanup := func() {
		pool.Close()
		container.Terminate(context.Background())
	}

	return pg, cleanup, nil
}

// TruncateAllTables removes all data from tables while preserving schema.
//
// Use this between tests to ensure clean state:
//
//	func TestSomething(t *testing.T) {
//	    pg.TruncateAllTables(ctx)
//	    // ... test code ...
//	}
func (pg *PostgresContainer) TruncateAllTables(ctx context.Context) error {
	tables := []string{
		"load_calendar_data.sessions",
		"load_calendar_data.otp_records",
		"load_calendar_data.load_assignments",
		"load_calendar_data.loads",
		"load_calendar_data.capacity_overrides",
		"load_calendar_data.group_members",
		"load_calendar_data.entities",
	}

	for _, table := range tables {
		_, err := pg.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	return nil
}

// ExecSQL executes raw SQL against the test database.
//
// Use for test setup and verification:
//
//	_, err := pg.ExecSQL(ctx, "INSERT INTO entities (id, title, type) VALUES ($1, $2, $3)", id, title, "person")
func (pg *PostgresContainer) ExecSQL(ctx context.Context, sql string, args ...interface{}) error {
	_, err := pg.Pool.Exec(ctx, sql, args...)
	return err
}

// QueryRowSQL queries a single row from the test database.
//
// Use for test verification:
//
//	var count int
//	err := pg.QueryRowSQL(ctx, "SELECT COUNT(*) FROM entities", &count)
func (pg *PostgresContainer) QueryRowSQL(ctx context.Context, sql string, dest ...interface{}) error {
	return pg.Pool.QueryRow(ctx, sql).Scan(dest...)
}
