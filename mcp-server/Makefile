.PHONY: all build test clean docker-build docker-run migrate-up migrate-down

# Variables
BINARY_NAME=guardrail-mcp
DOCKER_IMAGE=guardrail-mcp:latest
MIGRATIONS_DIR=internal/database/migrations

# Go build flags
LDFLAGS=-ldflags "-w -s"

all: build

# Build the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/server

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f $(BINARY_NAME)

# Build Docker image
docker-build:
	podman build -t $(DOCKER_IMAGE) -f deploy/Dockerfile .

# Run locally with docker-compose
docker-up:
	podman-compose -f deploy/podman-compose.yml up -d

# Stop docker-compose
docker-down:
	podman-compose -f deploy/podman-compose.yml down

# View logs
docker-logs:
	podman-compose -f deploy/podman-compose.yml logs -f

# Database migrations
migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down

# Development
dev:
	go run ./cmd/server

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Generate mocks (if needed)
generate:
	go generate ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Check for vulnerabilities
vuln:
	govulncheck ./...
