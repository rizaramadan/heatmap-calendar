# E2E Testing Infrastructure

This directory contains end-to-end testing infrastructure for the load-calendar application.

## Allowed Operations

E2E tests can **ONLY** use these four operation types:

### 1. Database Operations (`db.Query()` / `db.Exec()`)

Direct PostgreSQL operations for test setup and verification.

```go
db := helpers.NewDBHelper(pool)

// Query: SELECT operations
rows, err := db.Query(ctx, "SELECT id, name FROM entities WHERE type = $1", "person")

// Exec: INSERT, UPDATE, DELETE operations
tag, err := db.Exec(ctx, "DELETE FROM entities WHERE id = $1", testEntityID)
```

**Use cases:**
- Insert test fixtures before tests
- Verify database state after operations
- Clean up test data after tests

### 2. API Operations (`api.Call()`)

HTTP requests to test the service API.

```go
api := helpers.NewAPIClient("http://localhost:8080")
api.SetHeader("x-api-key", "test-key")

// GET request
resp, err := api.Call("GET", "/api/entities", nil)

// POST request with body
resp, err := api.Call("POST", "/api/loads/upsert", map[string]interface{}{
    "name": "Test Load",
    "effort": 5,
})

// DELETE request
resp, err := api.Call("DELETE", "/api/my-capacity/override/2024-01-15", nil)

// Check response
if resp.StatusCode != 200 {
    t.Errorf("expected 200, got %d", resp.StatusCode)
}
```

**Use cases:**
- Test API endpoints
- Verify response status codes and bodies
- Test authentication and authorization

### 3. Browser Operations (`browser.Navigate()` / `Click()` / `Fill()` / `Text()` / `Wait()`)

Headless browser automation for UI testing.

```go
browser, err := helpers.NewBrowser()
defer browser.Close()

// Navigate to page
err = browser.Navigate("http://localhost:8080/login")

// Fill form fields
err = browser.Fill("input[name='email']", "test@example.com")

// Click buttons
err = browser.Click("button[type='submit']")

// Wait for dynamic content
err = browser.Wait(".success-message")

// Get text content
text, err := browser.Text("h1.page-title")
```

**Use cases:**
- Test UI flows and user interactions
- Verify page content and element states
- Test HTMX-powered dynamic updates

### 4. Assertions (`assert.X()`)

Value assertions for test verification.

```go
a := helpers.NewAssert(t)

a.Equal(200, resp.StatusCode)
a.NotNil(user)
a.NoError(err)
a.Contains(body, "success")
a.True(isActive)
a.Len(items, 3)
```

**Available assertions:**
- `Equal(expected, actual)` / `NotEqual(expected, actual)`
- `Nil(object)` / `NotNil(object)`
- `True(value)` / `False(value)`
- `NoError(err)` / `Error(err)`
- `Contains(s, substr)` / `NotContains(s, substr)`
- `Len(object, length)`
- `Empty(object)` / `NotEmpty(object)`

## Test Environment

The `TestEnv` struct provides access to all test resources:

```go
type TestEnv struct {
    DB        *helpers.DBHelper   // Database helper
    API       *helpers.APIClient  // API client (pre-configured with API key)
    Browser   *helpers.Browser    // Browser helper (nil unless enabled)
    Pool      *pgxpool.Pool       // Raw database pool
    ServerURL string              // Base URL of test server
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_DATABASE_URL` | PostgreSQL connection string | `postgres://localhost:5432/load_calendar_test?sslmode=disable` |
| `TEST_API_KEY` | API key for protected endpoints | `test-api-key` |
| `E2E_START_BROWSER` | Enable browser tests | `false` |

## Writing E2E Tests

### Directory Structure

```
e2e/
  helpers/
    db.go        # Database helper
    api.go       # API client helper
    browser.go   # Browser automation helper
    assert.go    # Assertion helpers
  tests/
    auth_test.go      # Authentication tests
    heatmap_test.go   # Heatmap feature tests
    capacity_test.go  # Capacity management tests
  setup.go       # Test setup utilities
  e2e_test.go    # Main test runner
```

### Test File Template

```go
package e2e

import (
    "context"
    "testing"

    "github.com/gti/heatmap-internal/e2e/helpers"
)

func TestFeatureName(t *testing.T) {
    // Get test environment
    env := GetTestEnv()
    a := helpers.NewAssert(t)
    ctx := context.Background()

    // Clean state before test
    env.CleanupTestData(ctx)

    // Arrange: Set up test data using convenience methods
    err := env.SeedTestEntity(ctx, "test@example.com", "Test User", "person", 5.0)
    a.NoError(err)

    // Act: Perform the operation
    resp, err := env.API.Call("GET", "/api/entities", nil)
    a.NoError(err)

    // Assert: Verify results
    a.Equal(200, resp.StatusCode)
    a.Contains(resp.String(), "Test User")
}

func TestWithBrowser(t *testing.T) {
    env := GetTestEnv()
    a := helpers.NewAssert(t)

    // Start browser if not already started
    err := env.StartBrowser()
    a.NoError(err)

    // Navigate and interact
    err = env.Browser.Navigate(env.ServerURL + "/login")
    a.NoError(err)

    text, err := env.Browser.Text("h1")
    a.NoError(err)
    a.Contains(text, "Login")
}
```

### Convenience Methods

The `TestEnv` provides convenience methods for common operations:

```go
// Clean all test data between tests
env.CleanupTestData(ctx)

// Seed a test entity (person or group)
env.SeedTestEntity(ctx, "test@example.com", "Test User", "person", 5.0)

// Seed a test load with assignment
env.SeedTestLoad(ctx, "ext-123", "Test Load", "test@example.com", time.Now(), 2.5)

// Start browser on demand
env.StartBrowser()
```

### Running Tests

```bash
# Run all E2E tests (requires test database)
TEST_DATABASE_URL="postgres://user:pass@localhost:5432/load_calendar_test?sslmode=disable" \
  go test -v ./e2e/...

# Run with browser tests enabled
E2E_START_BROWSER=true go test -v ./e2e/...

# Run specific test
go test -v ./e2e -run TestFeatureName
```

## Dependencies

- `github.com/go-rod/rod` - Headless browser automation
- `github.com/stretchr/testify` - Assertion library
- `github.com/jackc/pgx/v5` - PostgreSQL driver (from main app)
