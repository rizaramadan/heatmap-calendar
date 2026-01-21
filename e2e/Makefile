.PHONY: test test-e2e test-e2e-verbose test-api test-browser clean docker-up docker-down help

# Default target
all: test-e2e

# ============================================================================
# E2E Test Targets
# ============================================================================

# Run all E2E tests (requires Docker)
test-e2e:
	@echo "Running E2E tests..."
	@echo "Note: This requires Docker to be running"
	cd .. && go test -tags=e2e -v ./e2e/tests/...

# Run E2E tests with verbose output and longer timeout
test-e2e-verbose:
	cd .. && go test -tags=e2e -v -timeout=10m ./e2e/tests/...

# Run only API tests (faster, no browser)
test-api:
	cd .. && go test -tags=e2e -v -run="TestAPI" ./e2e/tests/...

# Run smoke test only
test-smoke:
	cd .. && go test -tags=e2e -v -run="TestSmoke" ./e2e/tests/...

# Run tests with race detection
test-race:
	cd .. && go test -tags=e2e -race -v ./e2e/tests/...

# Run tests with coverage
test-coverage:
	cd .. && go test -tags=e2e -v -coverprofile=e2e/coverage.out ./e2e/tests/...
	cd .. && go tool cover -html=e2e/coverage.out -o e2e/coverage.html
	@echo "Coverage report: e2e/coverage.html"

# ============================================================================
# Docker Compose Fallback
# ============================================================================

# Start test database using docker-compose (fallback for testcontainers)
docker-up:
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@echo "Test database ready at localhost:5433"

# Stop test database
docker-down:
	docker-compose -f docker-compose.test.yml down -v

# Run tests using docker-compose database (without testcontainers)
test-external-db: docker-up
	TEST_DATABASE_URL="postgres://test_user:test_pass@localhost:5433/load_calendar_test?sslmode=disable" \
	SKIP_TESTCONTAINERS=true \
	cd .. && go test -tags=e2e -v ./e2e/tests/...
	$(MAKE) docker-down

# ============================================================================
# Development Helpers
# ============================================================================

# Build the test binary for debugging
build-test:
	cd .. && go test -tags=e2e -c -o e2e/tests/e2e.test ./e2e/tests/

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html
	rm -f tests/*.test
	rm -rf ../tmp/e2e-server

# Check if Docker is available
check-docker:
	@docker info > /dev/null 2>&1 && echo "Docker is available" || echo "Docker is NOT available"

# ============================================================================
# Help
# ============================================================================

help:
	@echo "E2E Test Commands:"
	@echo ""
	@echo "  make test-e2e         - Run all E2E tests (requires Docker)"
	@echo "  make test-e2e-verbose - Run with verbose output and 10m timeout"
	@echo "  make test-smoke       - Run smoke test only"
	@echo "  make test-api         - Run API tests only (no browser)"
	@echo "  make test-race        - Run with race detector"
	@echo "  make test-coverage    - Run with coverage report"
	@echo ""
	@echo "Docker Compose Fallback:"
	@echo ""
	@echo "  make docker-up        - Start test PostgreSQL via docker-compose"
	@echo "  make docker-down      - Stop test PostgreSQL"
	@echo "  make test-external-db - Run tests using docker-compose database"
	@echo ""
	@echo "Development:"
	@echo ""
	@echo "  make build-test       - Build test binary for debugging"
	@echo "  make clean            - Clean test artifacts"
	@echo "  make check-docker     - Check if Docker is available"
