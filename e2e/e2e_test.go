// Package e2e provides end-to-end testing infrastructure for the load-calendar application.
package e2e

import (
	"os"
	"testing"
)

// testEnv is the shared test environment for all E2E tests.
var testEnv *TestEnv

// TestMain is the entry point for E2E tests.
//
// It handles test setup and teardown including:
//   - Connecting to the test database
//   - Running migrations
//   - Starting the test server
//   - Running all E2E tests
//   - Cleaning up resources
//
// Environment variables:
//   - TEST_DATABASE_URL: PostgreSQL connection string (default: localhost test db)
//   - TEST_API_KEY: API key for protected endpoints (default: "test-api-key")
//   - E2E_START_BROWSER: Set to "true" to enable browser tests (default: false)
func TestMain(m *testing.M) {
	// Configure test environment
	cfg := DefaultConfig()

	// Check if browser tests are requested
	if os.Getenv("E2E_START_BROWSER") == "true" {
		cfg.StartBrowser = true
	}

	// Setup test environment
	var err error
	testEnv, err = Setup(cfg)
	if err != nil {
		// Can't use t.Fatal in TestMain, so print and exit
		println("Failed to setup E2E test environment:", err.Error())
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown
	testEnv.Teardown()

	os.Exit(code)
}

// GetTestEnv returns the shared test environment.
//
// Use this in test files to access the test helpers:
//
//	func TestSomething(t *testing.T) {
//	    env := e2e.GetTestEnv()
//	    a := helpers.NewAssert(t)
//	    ctx := context.Background()
//
//	    // Clean state
//	    env.CleanupTestData(ctx)
//
//	    // Setup test data
//	    env.SeedTestEntity(ctx, "test@example.com", "Test User", "person", 5.0)
//
//	    // Test API
//	    resp, err := env.API.Call("GET", "/api/entities", nil)
//	    a.NoError(err)
//	    a.Equal(200, resp.StatusCode)
//	}
func GetTestEnv() *TestEnv {
	return testEnv
}
