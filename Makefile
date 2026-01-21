.PHONY: build run dev test clean docker-up docker-down docs install-swag \
        test-e2e test-e2e-verbose test-e2e-api test-e2e-smoke test-e2e-race test-e2e-coverage \
        e2e-docker-up e2e-docker-down test-e2e-external-db e2e-build-test e2e-clean e2e-check-docker

# ============================================================================
# Build & Run
# ============================================================================

# Install swag if not present
install-swag:
	@which swag > /dev/null || go install github.com/swaggo/swag/cmd/swag@latest

# Generate swagger docs
docs: install-swag
	$(shell go env GOPATH)/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Build the application (generates docs first)
build: docs
	go build -o bin/server ./cmd/server

# Run the application
run: build
	./bin/server

# Run in development mode with hot reload (requires air)
dev: docs
	@if command -v air > /dev/null; then \
		air; \
	elif [ -f ~/go/bin/air ]; then \
		~/go/bin/air; \
	else \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
		~/go/bin/air; \
	fi

# ============================================================================
# Unit Tests
# ============================================================================

# Run tests
test:
	go test -v ./...

# ============================================================================
# E2E Tests
# ============================================================================

# Prepare E2E test dependencies (check Docker is installed and running, or external database is provided)
prepare-e2e:
	@echo "Checking E2E test dependencies..."
	@if [ -n "$$TEST_DATABASE_URL" ]; then \
		echo "✅ Using external database: $$TEST_DATABASE_URL"; \
		echo "✅ All E2E dependencies are ready!"; \
	elif ! command -v docker > /dev/null 2>&1; then \
		echo "❌ Docker is not installed and no TEST_DATABASE_URL provided."; \
		echo "   Option 1: Install Docker Desktop from https://www.docker.com/products/docker-desktop/"; \
		echo "   Option 2: Set TEST_DATABASE_URL environment variable to use external database"; \
		exit 1; \
	elif ! docker info > /dev/null 2>&1; then \
		echo "❌ Docker is not running. Please start Docker Desktop or set TEST_DATABASE_URL."; \
		exit 1; \
	else \
		echo "✅ Docker is installed"; \
		echo "✅ Docker is running"; \
		echo "✅ All E2E dependencies are ready!"; \
	fi

# Run all E2E tests (requires Docker)
test-e2e: prepare-e2e
	@echo "Running E2E tests..."
	@echo "Note: This requires Docker to be running"
	go test -tags=e2e -v ./e2e/tests/...

# Run E2E tests with verbose output and longer timeout
test-e2e-verbose:
	go test -tags=e2e -v -timeout=10m ./e2e/tests/...

# Run only API tests (faster, no browser)
test-e2e-api:
	go test -tags=e2e -v -run="TestAPI" ./e2e/tests/...

# Run smoke test only
test-e2e-smoke:
	go test -tags=e2e -v -run="TestSmoke" ./e2e/tests/...

# Run E2E tests with race detection
test-e2e-race:
	go test -tags=e2e -race -v ./e2e/tests/...

# Run E2E tests with coverage
test-e2e-coverage:
	go test -tags=e2e -v -coverprofile=e2e/coverage.out ./e2e/tests/...
	go tool cover -html=e2e/coverage.out -o e2e/coverage.html
	@echo "Coverage report: e2e/coverage.html"

# ============================================================================
# E2E Docker Compose Fallback
# ============================================================================

# Start test database using docker-compose (fallback for testcontainers)
e2e-docker-up:
	docker-compose -f e2e/docker-compose.test.yml up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@echo "Test database ready at localhost:5433"

# Stop test database
e2e-docker-down:
	docker-compose -f e2e/docker-compose.test.yml down -v

# Run tests using docker-compose database (without testcontainers)
test-e2e-external-db: e2e-docker-up
	TEST_DATABASE_URL="postgres://test_user:test_pass@localhost:5433/load_calendar_test?sslmode=disable" \
	SKIP_TESTCONTAINERS=true \
	go test -tags=e2e -v ./e2e/tests/...
	$(MAKE) e2e-docker-down

# ============================================================================
# E2E Development Helpers
# ============================================================================

# Build the E2E test binary for debugging
e2e-build-test:
	go test -tags=e2e -c -o e2e/tests/e2e.test ./e2e/tests/

# Clean E2E test artifacts
e2e-clean:
	rm -f e2e/coverage.out e2e/coverage.html
	rm -f e2e/tests/*.test
	rm -rf tmp/e2e-server

# Check if Docker is available
e2e-check-docker:
	@docker info > /dev/null 2>&1 && echo "Docker is available" || echo "Docker is NOT available"

# ============================================================================
# Code Quality
# ============================================================================

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./internal/... ./cmd/...; \
	else \
		echo "golangci-lint not installed"; \
	fi

# ============================================================================
# Docker (Development)
# ============================================================================

# Start PostgreSQL with Docker
docker-up:
	docker run --name heatmap-postgres -d \
		-e POSTGRES_USER=heatmap \
		-e POSTGRES_PASSWORD=heatmap \
		-e POSTGRES_DB=heatmap \
		-p 5432:5432 \
		postgres:15-alpine

# Stop PostgreSQL Docker container
docker-down:
	docker stop heatmap-postgres || true
	docker rm heatmap-postgres || true

# ============================================================================
# Setup
# ============================================================================

# Create .env from example
env:
	cp -n .env.example .env || true

# Initialize everything for first run
init: env docker-up
	@echo "Waiting for PostgreSQL to start..."
	@sleep 3
	@echo "Done! Run 'make run' to start the server."

# ============================================================================
# Help
# ============================================================================

help:
	@echo "Build & Run:"
	@echo "  make build              - Build the application"
	@echo "  make run                - Build and run the application"
	@echo "  make dev                - Run with hot reload (requires air)"
	@echo "  make docs               - Generate Swagger docs"
	@echo ""
	@echo "Unit Tests:"
	@echo "  make test               - Run unit tests"
	@echo ""
	@echo "E2E Tests:"
	@echo "  make prepare-e2e        - Check/install E2E test dependencies"
	@echo "  make test-e2e           - Run all E2E tests (requires Docker)"
	@echo "  make test-e2e-verbose   - Run E2E with verbose output and 10m timeout"
	@echo "  make test-e2e-smoke     - Run smoke test only"
	@echo "  make test-e2e-api       - Run API tests only (no browser)"
	@echo "  make test-e2e-race      - Run E2E with race detector"
	@echo "  make test-e2e-coverage  - Run E2E with coverage report"
	@echo ""
	@echo "E2E Docker Compose Fallback:"
	@echo "  make e2e-docker-up      - Start test PostgreSQL via docker-compose"
	@echo "  make e2e-docker-down    - Stop test PostgreSQL"
	@echo "  make test-e2e-external-db - Run tests using docker-compose database"
	@echo ""
	@echo "E2E Development:"
	@echo "  make e2e-build-test     - Build E2E test binary for debugging"
	@echo "  make e2e-clean          - Clean E2E test artifacts"
	@echo "  make e2e-check-docker   - Check if Docker is available"
	@echo ""
	@echo "Code Quality:"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Run linter"
	@echo ""
	@echo "Docker (Development):"
	@echo "  make docker-up          - Start PostgreSQL in Docker"
	@echo "  make docker-down        - Stop PostgreSQL Docker container"
	@echo ""
	@echo "Setup:"
	@echo "  make env                - Create .env from example"
	@echo "  make init               - Initialize for first run"
