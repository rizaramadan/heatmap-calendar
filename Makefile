.PHONY: build run dev test clean docker-up docker-down

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run the application
run: build
	./bin/server

# Run in development mode with hot reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Installing air..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Run tests
test:
	go test -v ./...

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
		golangci-lint run; \
	else \
		echo "golangci-lint not installed"; \
	fi

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

# Create .env from example
env:
	cp -n .env.example .env || true

# Initialize everything for first run
init: env docker-up
	@echo "Waiting for PostgreSQL to start..."
	@sleep 3
	@echo "Done! Run 'make run' to start the server."

# Help
help:
	@echo "Available commands:"
	@echo "  make build      - Build the application"
	@echo "  make run        - Build and run the application"
	@echo "  make dev        - Run with hot reload (requires air)"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make fmt        - Format code"
	@echo "  make lint       - Run linter"
	@echo "  make docker-up  - Start PostgreSQL in Docker"
	@echo "  make docker-down- Stop PostgreSQL Docker container"
	@echo "  make env        - Create .env from example"
	@echo "  make init       - Initialize for first run"
