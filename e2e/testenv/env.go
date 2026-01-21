// Package testenv provides ephemeral test infrastructure using testcontainers.
//
// This package manages the complete E2E test environment including:
//   - Ephemeral PostgreSQL container via testcontainers-go
//   - Application service subprocess
//   - Test isolation and cleanup utilities
//
// Example usage:
//
//	func TestMain(m *testing.M) {
//	    env, err := testenv.Setup(context.Background(), testenv.DefaultConfig())
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer env.Teardown()
//
//	    os.Exit(m.Run())
//	}
package testenv

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gti/heatmap-internal/e2e/helpers"
	"github.com/gti/heatmap-internal/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestEnv holds all resources for E2E testing.
//
// This struct is safe for use across parallel tests when using
// proper isolation (separate data per test, cleanup between tests).
type TestEnv struct {
	// Postgres is the ephemeral PostgreSQL container.
	Postgres *PostgresContainer

	// Service is the running application service.
	Service *Service

	// DB provides database helper for tests.
	DB *helpers.DBHelper

	// API provides HTTP client for tests (pre-configured with API key).
	API *helpers.APIClient

	// Pool provides direct database access.
	Pool *pgxpool.Pool

	// Config holds the environment configuration.
	Config EnvConfig

	// mu protects browser lazy initialization.
	mu sync.Mutex

	// browser is lazily initialized.
	browser *helpers.Browser

	// cleanupFuncs holds cleanup functions in reverse order.
	cleanupFuncs []func()
}

// EnvConfig holds configuration for the test environment.
type EnvConfig struct {
	// Postgres holds PostgreSQL container configuration.
	Postgres PostgresConfig

	// Service holds service configuration.
	Service ServiceConfig

	// SkipService skips starting the service (for DB-only tests).
	SkipService bool

	// ExternalDatabaseURL is an optional external database URL to use instead of testcontainers.
	// If set, testcontainers will be skipped. Useful for CI environments without Docker.
	ExternalDatabaseURL string
}

// DefaultConfig returns the default test environment configuration.
func DefaultConfig() EnvConfig {
	return EnvConfig{
		Postgres:            DefaultPostgresConfig(),
		Service:             DefaultServiceConfig(),
		SkipService:         false,
		ExternalDatabaseURL: os.Getenv("TEST_DATABASE_URL"),
	}
}

// Setup initializes the complete E2E test environment.
//
// This function:
//  1. Starts an ephemeral PostgreSQL container (or uses external database if provided)
//  2. Runs database migrations
//  3. Starts the application service connected to the database
//  4. Initializes test helpers (DB, API)
//
// Always call Teardown() when done:
//
//	env, err := testenv.Setup(ctx, testenv.DefaultConfig())
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer env.Teardown()
func Setup(ctx context.Context, cfg EnvConfig) (*TestEnv, error) {
	env := &TestEnv{
		Config:       cfg,
		cleanupFuncs: make([]func(), 0),
	}

	var pool *pgxpool.Pool
	var dbURL string

	// Use external database if provided, otherwise start testcontainer
	if cfg.ExternalDatabaseURL != "" {
		// Connect to external database
		db, err := database.New(cfg.ExternalDatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to external database: %w", err)
		}

		// Run migrations
		if err := db.RunMigrations(ctx); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to run migrations on external database: %w", err)
		}

		pool = db.Pool
		dbURL = cfg.ExternalDatabaseURL
		env.addCleanup(func() {
			if pool != nil {
				pool.Close()
			}
		})
	} else {
		// Start PostgreSQL container
		pg, pgCleanup, err := StartPostgres(ctx, cfg.Postgres)
		if err != nil {
			return nil, fmt.Errorf("failed to start postgres: %w", err)
		}
		env.addCleanup(pgCleanup)
		env.Postgres = pg
		pool = pg.Pool
		dbURL = pg.ConnectionString
	}

	env.Pool = pool

	// Initialize DB helper
	env.DB = helpers.NewDBHelper(pool)

	// Start service unless skipped
	if !cfg.SkipService {
		// Configure service with database URL
		svcCfg := cfg.Service
		svcCfg.DatabaseURL = dbURL

		svc, svcCleanup, err := StartService(ctx, svcCfg)
		if err != nil {
			env.Teardown()
			return nil, fmt.Errorf("failed to start service: %w", err)
		}
		env.addCleanup(svcCleanup)
		env.Service = svc

		// Initialize API client
		env.API = helpers.NewAPIClient(svc.URL)
		env.API.SetHeader("x-api-key", svcCfg.APIKey)
	}

	return env, nil
}

// Teardown releases all test resources in reverse order.
//
// This function:
//  1. Stops the application service
//  2. Closes database connections
//  3. Terminates the PostgreSQL container
//  4. Closes the browser (if started)
func (env *TestEnv) Teardown() {
	// Close browser if started
	env.mu.Lock()
	if env.browser != nil {
		_ = env.browser.Close()
	}
	env.mu.Unlock()

	// Run cleanup functions in reverse order
	for i := len(env.cleanupFuncs) - 1; i >= 0; i-- {
		env.cleanupFuncs[i]()
	}
}

// CleanupTestData removes all data from tables.
//
// Call this between tests for isolation:
//
//	func TestSomething(t *testing.T) {
//	    env.CleanupTestData(ctx)
//	    // ... test with clean state ...
//	}
func (env *TestEnv) CleanupTestData(ctx context.Context) error {
	if env.Postgres != nil {
		return env.Postgres.TruncateAllTables(ctx)
	}

	// Fallback for external database: manually truncate tables
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
		_, err := env.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	return nil
}

// Browser returns the browser helper, initializing it lazily.
//
// The browser is shared across tests and closed during Teardown.
// For parallel tests, consider creating separate browser instances.
func (env *TestEnv) Browser() (*helpers.Browser, error) {
	env.mu.Lock()
	defer env.mu.Unlock()

	if env.browser != nil {
		return env.browser, nil
	}

	browser, err := helpers.NewBrowser()
	if err != nil {
		return nil, fmt.Errorf("failed to start browser: %w", err)
	}

	env.browser = browser
	return browser, nil
}

// ServiceURL returns the base URL of the running service.
func (env *TestEnv) ServiceURL() string {
	if env.Service == nil {
		return ""
	}
	return env.Service.URL
}

// SeedTestEntity creates a test entity in the database.
func (env *TestEnv) SeedTestEntity(ctx context.Context, id, title, entityType string, capacity float64) error {
	_, err := env.DB.Exec(ctx,
		"INSERT INTO load_calendar_data.entities (id, title, type, default_capacity) VALUES ($1, $2, $3, $4)",
		id, title, entityType, capacity)
	return err
}

// SeedTestLoad creates a test load with an assignment.
func (env *TestEnv) SeedTestLoad(ctx context.Context, externalID, title, assignee string, date string, weight float64) error {
	var loadID int
	err := env.Pool.QueryRow(ctx,
		"INSERT INTO load_calendar_data.loads (external_id, title, source, date) VALUES ($1, $2, 'test', $3::date) RETURNING id",
		externalID, title, date).Scan(&loadID)
	if err != nil {
		return fmt.Errorf("failed to create load: %w", err)
	}

	_, err = env.DB.Exec(ctx,
		"INSERT INTO load_calendar_data.load_assignments (load_id, person_email, weight) VALUES ($1, $2, $3)",
		loadID, assignee, weight)
	return err
}

// NewIsolatedEnv creates a new test environment for parallel test isolation.
//
// Each isolated environment shares the same PostgreSQL container but gets
// its own API client. Tests should use unique identifiers for their data.
//
// Example:
//
//	func TestParallel(t *testing.T) {
//	    t.Parallel()
//	    iso := env.NewIsolatedEnv(t.Name())
//	    // Use iso.API and iso.DB with test-specific data
//	}
func (env *TestEnv) NewIsolatedEnv(testName string) *IsolatedEnv {
	return &IsolatedEnv{
		parent:   env,
		TestName: testName,
		DB:       env.DB,
		API:      helpers.NewAPIClient(env.ServiceURL()),
		Pool:     env.Pool,
	}
}

// IsolatedEnv provides test isolation for parallel tests.
type IsolatedEnv struct {
	// parent is the shared test environment.
	parent *TestEnv

	// TestName identifies this test for unique data prefixes.
	TestName string

	// DB provides database access.
	DB *helpers.DBHelper

	// API provides HTTP client (separate instance per test).
	API *helpers.APIClient

	// Pool provides direct database access.
	Pool *pgxpool.Pool
}

// UniqueID creates a test-specific unique identifier.
//
// Use this to create unique entity IDs for parallel tests:
//
//	entityID := iso.UniqueID("user@example.com")
//	// Returns something like "TestParallel_user@example.com"
func (iso *IsolatedEnv) UniqueID(base string) string {
	return fmt.Sprintf("%s_%s", iso.TestName, base)
}

// Cleanup removes all data created by this test.
//
// This deletes entities matching the test's unique prefix.
func (iso *IsolatedEnv) Cleanup(ctx context.Context) error {
	pattern := iso.TestName + "_%"

	// Delete in reverse dependency order
	_, err := iso.DB.Exec(ctx, "DELETE FROM load_calendar_data.entities WHERE id LIKE $1", pattern)
	return err
}

// addCleanup adds a cleanup function to be called during Teardown.
func (env *TestEnv) addCleanup(fn func()) {
	env.cleanupFuncs = append(env.cleanupFuncs, fn)
}
