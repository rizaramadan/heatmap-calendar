// Package e2e provides end-to-end testing infrastructure for the load-calendar application.
//
// This package contains setup utilities and test runners for comprehensive E2E tests
// that exercise the full application stack including database, HTTP API, and browser UI.
package e2e

import (
	"context"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gti/heatmap-internal/e2e/helpers"
	"github.com/gti/heatmap-internal/internal/database"
	"github.com/gti/heatmap-internal/internal/handler"
	"github.com/gti/heatmap-internal/internal/middleware"
	"github.com/gti/heatmap-internal/internal/repository"
	"github.com/gti/heatmap-internal/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// TestEnv holds all resources needed for E2E testing.
//
// Create a TestEnv using Setup() and clean up with Teardown().
// The helpers (DB, API, Browser) provide the only allowed operations
// for E2E tests as documented in README.md.
type TestEnv struct {
	// DB provides direct PostgreSQL access for test setup and verification.
	DB *helpers.DBHelper

	// API provides HTTP client for testing API endpoints.
	API *helpers.APIClient

	// Browser provides headless browser for UI testing (nil until StartBrowser called).
	Browser *helpers.Browser

	// Pool is the underlying database connection pool.
	Pool *pgxpool.Pool

	// ServerURL is the base URL of the test server.
	ServerURL string

	// server is the Echo instance (internal).
	server *echo.Echo

	// shutdownCh signals server shutdown.
	shutdownCh chan struct{}
}

// Config holds E2E test configuration.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string for the test database.
	// Defaults to TEST_DATABASE_URL env var or localhost test database.
	DatabaseURL string

	// APIKey is the API key for protected endpoints.
	// Defaults to TEST_API_KEY env var or "test-api-key".
	APIKey string

	// StartBrowser controls whether to initialize the headless browser.
	// Set to false for API-only tests to speed up test execution.
	StartBrowser bool
}

// DefaultConfig returns the default test configuration.
//
// Configuration is loaded from environment variables:
//   - TEST_DATABASE_URL: PostgreSQL connection string
//   - TEST_API_KEY: API key for protected endpoints
func DefaultConfig() Config {
	return Config{
		DatabaseURL:  getEnvOrDefault("TEST_DATABASE_URL", "postgres://localhost:5432/load_calendar_test?sslmode=disable"),
		APIKey:       getEnvOrDefault("TEST_API_KEY", "test-api-key"),
		StartBrowser: false,
	}
}

// Setup initializes the E2E test environment.
//
// This function:
//   - Connects to the test database
//   - Runs migrations to ensure schema exists
//   - Starts the application server on a random port
//   - Initializes API client pointing to the test server
//   - Optionally starts a headless browser
//
// Always call Teardown() when done, typically with defer:
//
//	env, err := e2e.Setup(e2e.DefaultConfig())
//	if err != nil {
//	    t.Fatal(err)
//	}
//	defer env.Teardown()
func Setup(cfg Config) (*TestEnv, error) {
	ctx := context.Background()

	// Connect to test database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Run migrations
	if err := db.RunMigrations(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Find available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	serverURL := fmt.Sprintf("http://localhost:%d", port)

	env := &TestEnv{
		DB:         helpers.NewDBHelper(db.Pool),
		Pool:       db.Pool,
		ServerURL:  serverURL,
		shutdownCh: make(chan struct{}),
	}

	// Start server
	if err := env.startServer(db, cfg.APIKey, port); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to start test server: %w", err)
	}

	// Initialize API client
	env.API = helpers.NewAPIClient(serverURL)
	env.API.SetHeader("x-api-key", cfg.APIKey)

	// Optionally start browser
	if cfg.StartBrowser {
		browser, err := helpers.NewBrowser()
		if err != nil {
			env.Teardown()
			return nil, fmt.Errorf("failed to start browser: %w", err)
		}
		env.Browser = browser
	}

	// Wait for server to be ready
	if err := env.waitForServer(5 * time.Second); err != nil {
		env.Teardown()
		return nil, fmt.Errorf("server failed to start: %w", err)
	}

	return env, nil
}

// StartBrowser initializes the headless browser if not already started.
//
// Call this method when you need browser testing but didn't enable it in Config.
// The browser is automatically closed by Teardown().
func (env *TestEnv) StartBrowser() error {
	if env.Browser != nil {
		return nil
	}

	browser, err := helpers.NewBrowser()
	if err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}
	env.Browser = browser
	return nil
}

// Teardown releases all test resources.
//
// This function:
//   - Stops the test server
//   - Closes the browser (if started)
//   - Closes the database connection
//
// Always call Teardown() after Setup(), typically with defer.
func (env *TestEnv) Teardown() {
	// Stop server
	if env.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = env.server.Shutdown(ctx)
	}

	// Close browser
	if env.Browser != nil {
		_ = env.Browser.Close()
	}

	// Close database
	if env.Pool != nil {
		env.Pool.Close()
	}
}

// CleanupTestData removes all test data from the database.
//
// Call this between tests to ensure a clean state:
//
//	func TestSomething(t *testing.T) {
//	    env.CleanupTestData()
//	    // ... test code ...
//	}
//
// This truncates all tables but preserves the schema.
func (env *TestEnv) CleanupTestData(ctx context.Context) error {
	// Truncate all tables in reverse dependency order
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
		_, err := env.DB.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	return nil
}

// SeedTestEntity creates a test entity (person or group) in the database.
//
// This is a convenience method for common test setup:
//
//	err := env.SeedTestEntity(ctx, "test@example.com", "Test User", "person", 5.0)
func (env *TestEnv) SeedTestEntity(ctx context.Context, id, title, entityType string, capacity float64) error {
	_, err := env.DB.Exec(ctx,
		"INSERT INTO load_calendar_data.entities (id, title, type, default_capacity) VALUES ($1, $2, $3, $4)",
		id, title, entityType, capacity)
	return err
}

// SeedTestLoad creates a test load with assignment in the database.
//
// This is a convenience method for common test setup:
//
//	err := env.SeedTestLoad(ctx, "ext-123", "Test Load", "test@example.com", time.Now(), 2.5)
func (env *TestEnv) SeedTestLoad(ctx context.Context, externalID, title, assignee string, date time.Time, weight float64) error {
	var loadID int
	err := env.Pool.QueryRow(ctx,
		"INSERT INTO load_calendar_data.loads (external_id, title, source, date) VALUES ($1, $2, 'test', $3) RETURNING id",
		externalID, title, date).Scan(&loadID)
	if err != nil {
		return fmt.Errorf("failed to create load: %w", err)
	}

	_, err = env.DB.Exec(ctx,
		"INSERT INTO load_calendar_data.load_assignments (load_id, person_email, weight) VALUES ($1, $2, $3)",
		loadID, assignee, weight)
	return err
}

// startServer initializes and starts the Echo server.
func (env *TestEnv) startServer(db *database.DB, apiKey string, port int) error {
	// Initialize repositories
	entityRepo := repository.NewEntityRepository(db.Pool)
	groupRepo := repository.NewGroupRepository(db.Pool)
	capacityRepo := repository.NewCapacityRepository(db.Pool)
	loadRepo := repository.NewLoadRepository(db.Pool)

	// Initialize services
	webhookService := service.NewWebhookService("", loadRepo, capacityRepo) // No webhook in tests
	heatmapService := service.NewHeatmapService(entityRepo, capacityRepo, loadRepo, groupRepo)
	loadService := service.NewLoadService(loadRepo, entityRepo, webhookService)
	authService := service.NewAuthService(db.Pool, "", "") // No Lark in tests
	capacityService := service.NewCapacityService(entityRepo, capacityRepo)

	// Load templates
	templates, err := loadTestTemplates()
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Initialize handlers
	heatmapHandler := handler.NewHeatmapHandler(heatmapService, entityRepo, templates)
	apiHandler := handler.NewAPIHandler(loadService, entityRepo, groupRepo)
	authHandler := handler.NewAuthHandler(authService, entityRepo, templates)
	capacityHandler := handler.NewCapacityHandler(capacityService, templates)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Optional session auth
	e.Use(middleware.SessionAuthOptional(authService))

	// Public routes
	e.GET("/", heatmapHandler.Index)
	e.GET("/login", authHandler.LoginPage)

	// Auth routes
	e.POST("/auth/request-otp", authHandler.RequestOTP)
	e.POST("/auth/verify-otp", authHandler.VerifyOTP)
	e.POST("/auth/logout", authHandler.Logout)

	// Protected routes (require session)
	protected := e.Group("")
	protected.Use(middleware.SessionAuth(authService))
	protected.GET("/my-capacity", capacityHandler.MyCapacityPage)
	protected.POST("/api/my-capacity", capacityHandler.UpdateMyCapacity)
	protected.DELETE("/api/my-capacity/override/:date", capacityHandler.DeleteMyCapacityOverride)

	// Public API routes
	e.GET("/api/entities", apiHandler.ListEntities)
	e.GET("/api/entities/:id", apiHandler.GetEntity)
	e.GET("/api/heatmap/:entity", heatmapHandler.GetHeatmapPartial)
	e.GET("/api/heatmap/:entity/day/:date", heatmapHandler.GetDayDetails)

	// Protected API routes
	apiProtected := e.Group("/api")
	apiProtected.Use(middleware.APIKeyAuth(apiKey))
	apiProtected.POST("/loads/upsert", apiHandler.UpsertLoad)
	apiProtected.POST("/loads/:id/assignees", apiHandler.AddAssigneesToLoad)
	apiProtected.DELETE("/loads/:id/assignees/:email", apiHandler.RemoveAssigneeFromLoad)
	apiProtected.POST("/entities", apiHandler.CreateEntity)
	apiProtected.DELETE("/entities/:id", apiHandler.DeleteEntity)
	apiProtected.GET("/groups/:id/members", apiHandler.GetGroupMembers)
	apiProtected.POST("/groups/:id/members", apiHandler.AddGroupMember)
	apiProtected.DELETE("/groups/:id/members/:member", apiHandler.RemoveGroupMember)

	// Static files
	e.Static("/static", "static")

	env.server = e

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", port)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Test server error: %v\n", err)
		}
	}()

	return nil
}

// waitForServer polls the server until it responds or timeout.
func (env *TestEnv) waitForServer(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(env.ServerURL + "/api/entities")
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("server did not respond within %v", timeout)
}

// loadTestTemplates loads HTML templates for the test server.
func loadTestTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 3:04 PM")
		},
	}

	templates, err := template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	if err != nil {
		return nil, err
	}

	templates, err = templates.ParseGlob("templates/partials/*.html")
	if err != nil {
		return nil, err
	}

	return templates, nil
}

// getEnvOrDefault returns the environment variable value or the default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
