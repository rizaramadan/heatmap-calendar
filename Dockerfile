# Build stage
FROM golang:1.25.1-alpine AS builder

# Install build dependencies and swag for generating swagger docs
RUN apk add --no-cache git

# Install swag for generating swagger documentation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate swagger documentation before building
RUN /go/bin/swag init -g cmd/server/main.go -o docs

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Set timezone to Asia/Singapore (matching deployment region)
ENV TZ=Asia/Singapore

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy templates and static files
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Expose port (Cloud Run uses PORT env variable)
EXPOSE 8080

# Run the application
CMD ["./main"]
