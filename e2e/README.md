# E2E Testing Infrastructure

This directory contains end-to-end testing infrastructure for the load-calendar application.

## Prerequisites

Before running E2E tests, ensure you have the following installed:

- **Go 1.25.1+** - Required Go version (check with `go version`)
- **Docker** - Required for running PostgreSQL test containers
  - Install from [docker.com](https://docs.docker.com/get-docker/)
  - Verify with `docker --version` and `docker ps`
- **Docker Compose** (optional) - For fallback database setup
  - Usually included with Docker Desktop
  - Verify with `docker-compose --version`

### Quick Setup Check

```bash
# Verify all prerequisites
go version              # Should show go1.25.1 or higher
docker --version        # Should show Docker version
docker ps               # Should connect to Docker daemon
```

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

## How to Run E2E Tests Locally

### Method 1: Using Test Scripts (Recommended)

The easiest way to run E2E tests is using the provided shell scripts:

```bash
# Run E2E tests only
./scripts/test-e2e.sh

# Run with verbose output
VERBOSE=1 ./scripts/test-e2e.sh

# Run with race detector
RACE=1 ./scripts/test-e2e.sh

# Run specific test by pattern
./scripts/test-e2e.sh TestSmoke

# Run all tests (unit + E2E) in parallel
./scripts/test-all.sh

# Run with coverage reporting
./scripts/test-e2e.sh
./scripts/merge-coverage.sh  # Merge all coverage reports
open coverage/coverage.html  # View combined coverage
```

### Method 2: Using Makefile

```bash
# From e2e directory
cd e2e

# Run all E2E tests
make test-e2e

# Run with verbose output (10m timeout)
make test-e2e-verbose

# Run API tests only (no browser)
make test-api

# Run smoke test only
make test-smoke

# Run with race detector
make test-race

# Run with coverage reporting
make test-coverage

# Use external docker-compose database
make docker-up           # Start database
make test-external-db    # Run tests
make docker-down         # Stop database
```

### Method 3: Direct Go Test Command

```bash
# Run all E2E tests with automatic testcontainers
go test -v -tags=e2e ./e2e/tests/...

# Run with custom database URL
TEST_DATABASE_URL="postgres://user:pass@localhost:5432/load_calendar_test?sslmode=disable" \
  go test -v -tags=e2e ./e2e/tests/...

# Run with browser tests enabled
E2E_START_BROWSER=true go test -v -tags=e2e ./e2e/tests/...

# Run specific test
go test -v -tags=e2e ./e2e/tests -run TestSmoke

# Run with coverage
go test -v -tags=e2e -covermode=atomic -coverprofile=coverage/e2e.out ./e2e/tests/...

# Run with race detector
go test -v -tags=e2e -race ./e2e/tests/...

# Run with timeout
go test -v -tags=e2e -timeout=10m ./e2e/tests/...
```

### Test Database Options

E2E tests support two database setup modes:

#### Option A: Automatic Testcontainers (Default)

Tests automatically spin up ephemeral PostgreSQL containers:

```bash
# No setup needed - just run tests
./scripts/test-e2e.sh
```

**Benefits:**
- Automatic container lifecycle management
- Parallel test execution safe
- No manual database cleanup needed
- Isolated test environment

**Requirements:**
- Docker daemon running
- Sufficient Docker permissions

#### Option B: External Docker Compose Database

Use a persistent test database via docker-compose:

```bash
# Start database
cd e2e
make docker-up

# Run tests
make test-external-db

# Stop database
make docker-down
```

**Configuration** (`e2e/docker-compose.test.yml`):
- PostgreSQL 16-alpine on port 5433
- Database: `load_calendar_test`
- User: `test_user` / Password: `test_pass`
- Connection: `postgres://test_user:test_pass@localhost:5433/load_calendar_test?sslmode=disable`

**When to use:**
- Docker socket unavailable
- Running tests in restricted environments
- Debugging database state between runs

## Coverage Reports

### Coverage File Locations

After running tests, coverage reports are stored in the `coverage/` directory:

```
coverage/
├── unit.out              # Unit test coverage profile
├── unit.html             # Unit test coverage HTML report
├── e2e.out               # E2E test coverage profile
├── e2e-service/          # Binary coverage from instrumented service
│   └── covmeta.*         # Coverage metadata files
├── e2e-service.out       # Converted E2E service coverage
├── combined.out          # Merged coverage from all sources
└── coverage.html         # Combined coverage HTML report (open this!)
```

### Viewing Coverage

```bash
# Generate and merge all coverage reports
./scripts/test-all.sh        # Run all tests
./scripts/merge-coverage.sh  # Merge coverage

# View combined coverage in browser
open coverage/coverage.html  # macOS
xdg-open coverage/coverage.html  # Linux

# View coverage in terminal
go tool cover -func=coverage/combined.out

# View coverage by package
go tool cover -func=coverage/combined.out | grep "^github.com/gti/heatmap-internal"

# Find lowest coverage packages
go tool cover -func=coverage/combined.out | sort -k3 -n | head -10
```

### Coverage Workflow

1. **Unit Tests** generate `coverage/unit.out`
   ```bash
   ./scripts/test-unit.sh
   ```

2. **E2E Tests** generate two coverage sources:
   ```bash
   ./scripts/test-e2e.sh
   # Creates: coverage/e2e.out (test coverage)
   # Creates: coverage/e2e-service/ (service binary coverage)
   ```

3. **Merge Coverage** combines all sources:
   ```bash
   ./scripts/merge-coverage.sh
   # Converts binary coverage to text format
   # Merges unit.out + e2e.out + e2e-service.out
   # Outputs: coverage/combined.out and coverage/coverage.html
   ```

### CI Coverage Reports

In GitHub Actions CI:
- Coverage artifacts are uploaded for each job
- Combined coverage report posted as PR comment
- Coverage summary shown in workflow summary
- HTML reports available for download (30-day retention)

## Troubleshooting

### Docker Issues

#### "Cannot connect to Docker daemon"

```bash
# Check Docker is running
docker ps

# Start Docker daemon (Linux)
sudo systemctl start docker

# Start Docker Desktop (macOS/Windows)
# Open Docker Desktop application

# Verify connection
docker run hello-world
```

#### "Permission denied while trying to connect to Docker"

```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker  # Activate group without logout

# Verify permissions
docker ps
```

#### "Testcontainers failed to start"

```bash
# Use fallback docker-compose instead
cd e2e
make docker-up
make test-external-db

# Or set explicit database URL
TEST_DATABASE_URL="postgres://test_user:test_pass@localhost:5433/load_calendar_test?sslmode=disable" \
  go test -v -tags=e2e ./e2e/tests/...
```

### Port Conflicts

#### "Port 5432 already in use"

```bash
# Check what's using the port
sudo lsof -i :5432  # Linux/macOS
netstat -ano | findstr :5432  # Windows

# Option 1: Stop conflicting service
sudo systemctl stop postgresql  # If system PostgreSQL

# Option 2: Use docker-compose (uses port 5433)
cd e2e
make docker-up
make test-external-db
```

#### "Port 8080 already in use"

```bash
# Check what's using port 8080
sudo lsof -i :8080

# Kill the process or use a different port
# Tests will automatically find available port
```

### Test Failures

#### "Database migration failed"

```bash
# Verify database is accessible
psql postgres://test_user:test_pass@localhost:5433/load_calendar_test

# Check migration files exist
ls -la migrations/

# Try manual migration
migrate -path migrations -database "postgres://test_user:test_pass@localhost:5433/load_calendar_test?sslmode=disable" up
```

#### "API key authentication failed"

```bash
# Ensure TEST_API_KEY is set
export TEST_API_KEY=test-api-key

# Or set in test command
TEST_API_KEY=your-key ./scripts/test-e2e.sh
```

#### "Browser tests timing out"

```bash
# Increase timeout
go test -v -tags=e2e -timeout=20m ./e2e/tests/...

# Run without browser tests
E2E_START_BROWSER=false ./scripts/test-e2e.sh

# Check browser dependencies (Linux)
# go-rod uses bundled Chromium, but may need:
sudo apt-get install -y chromium-browser
```

#### "Race detector failures"

```bash
# Race conditions detected
# Fix by adding proper synchronization (mutexes, channels)
# Or run without race detector temporarily:
./scripts/test-e2e.sh  # Race detector off by default

# To enable:
RACE=1 ./scripts/test-e2e.sh
```

### Coverage Issues

#### "Coverage files not found"

```bash
# Ensure coverage directory exists
mkdir -p coverage

# Run tests with coverage first
./scripts/test-all.sh

# Then merge
./scripts/merge-coverage.sh
```

#### "gocovmerge not found"

```bash
# Install gocovmerge
go install github.com/wadey/gocovmerge@latest

# Verify installation
which gocovmerge

# Add to PATH if needed
export PATH=$PATH:$(go env GOPATH)/bin
```

#### "Coverage percentage seems wrong"

```bash
# Clear old coverage data
rm -rf coverage/*

# Run fresh test suite
./scripts/test-all.sh
./scripts/merge-coverage.sh

# Verify each component
go tool cover -func=coverage/unit.out | tail -1
go tool cover -func=coverage/e2e.out | tail -1
go tool cover -func=coverage/combined.out | tail -1
```

### General Issues

#### "Go version mismatch"

```bash
# Check Go version
go version  # Should be 1.25.1+

# Install correct version
# Visit: https://go.dev/dl/

# Or use version manager (e.g., gvm, asdf)
```

#### "Dependencies not found"

```bash
# Download dependencies
go mod download

# Verify dependencies
go mod verify

# Tidy up if needed
go mod tidy
```

#### "Tests hang indefinitely"

```bash
# Set explicit timeout
go test -v -tags=e2e -timeout=5m ./e2e/tests/...

# Run with verbose to see where it hangs
VERBOSE=1 ./scripts/test-e2e.sh

# Check for deadlocks in test logs
```

#### "Clean test state between runs"

```bash
# Stop all test containers
docker stop $(docker ps -aq --filter "name=testcontainers")

# Remove test volumes
docker volume prune -f

# Clear coverage data
rm -rf coverage/*

# Reset docker-compose database
cd e2e
make docker-down
make docker-up
```

## Getting Help

If you encounter issues not covered here:

1. **Check test logs**: Run with `VERBOSE=1` to see detailed output
2. **Verify prerequisites**: Ensure Docker and Go are properly installed
3. **Review test code**: Check `/e2e/tests/` for test examples
4. **Check CI logs**: GitHub Actions runs show detailed test execution
5. **Read helper code**: `/e2e/helpers/` contains implementation details

## Dependencies

- `github.com/go-rod/rod` - Headless browser automation
- `github.com/stretchr/testify` - Assertion library
- `github.com/jackc/pgx/v5` - PostgreSQL driver (from main app)
- `github.com/testcontainers/testcontainers-go` - Ephemeral test containers
- `github.com/testcontainers/testcontainers-go/modules/postgres` - PostgreSQL testcontainer
