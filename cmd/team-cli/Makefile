# Team CLI Makefile
# Build automation for the team management CLI tool

.PHONY: all build clean test install uninstall lint fmt deps check-backend

# Build variables
BINARY_NAME := team
BINARY_NAME_ALT := team-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt

# Installation directories
PREFIX ?= /usr/local
ifeq ($(strip $(PREFIX)),)
$(error PREFIX must not be empty)
endif
BINDIR := $(PREFIX)/bin
DATADIR := $(PREFIX)/share/agent-guardrails-template/team-cli
REPO_ROOT := $(abspath ../..)
TEAM_MANAGER_SOURCE := $(REPO_ROOT)/scripts

# Default target
all: deps build

# Download dependencies
deps:
	$(GOMOD) tidy
	$(GOMOD) download

# Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

# Build with optimizations for release
release:
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -ldflags "-s -w" -trimpath -o $(BINARY_NAME) .

# Build for multiple platforms
cross-compile:
	# Linux AMD64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -ldflags "-s -w" -trimpath -o dist/$(BINARY_NAME)-linux-amd64 .
	# Linux ARM64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -ldflags "-s -w" -trimpath -o dist/$(BINARY_NAME)-linux-arm64 .
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -ldflags "-s -w" -trimpath -o dist/$(BINARY_NAME)-darwin-amd64 .
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -ldflags "-s -w" -trimpath -o dist/$(BINARY_NAME)-darwin-arm64 .
	# Windows AMD64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -ldflags "-s -w" -trimpath -o dist/$(BINARY_NAME)-windows-amd64.exe .

# Run tests
test:
	$(GOTEST) -v -race ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	$(GOFMT) .

# Run linter (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Install binary to system
check-backend:
	@test -f "$(TEAM_MANAGER_SOURCE)/team_manager.py" -a -f "$(TEAM_MANAGER_SOURCE)/batch_operations.py" -a -f "$(TEAM_MANAGER_SOURCE)/encryption.py" -a -f "$(TEAM_MANAGER_SOURCE)/migrations/__init__.py" -a -f "$(TEAM_MANAGER_SOURCE)/migrations/v1_to_v2.py" || { echo "Python backend payload is incomplete: $(TEAM_MANAGER_SOURCE)" >&2; exit 1; }

install: check-backend build
	@mkdir -p "$(BINDIR)" "$(DATADIR)/scripts/migrations"
	@install -m 644 "$(TEAM_MANAGER_SOURCE)/team_manager.py" "$(DATADIR)/scripts/team_manager.py"
	@install -m 644 "$(TEAM_MANAGER_SOURCE)/batch_operations.py" "$(DATADIR)/scripts/batch_operations.py"
	@install -m 644 "$(TEAM_MANAGER_SOURCE)/encryption.py" "$(DATADIR)/scripts/encryption.py"
	@install -m 644 "$(TEAM_MANAGER_SOURCE)/migrations/__init__.py" "$(DATADIR)/scripts/migrations/__init__.py"
	@install -m 644 "$(TEAM_MANAGER_SOURCE)/migrations/v1_to_v2.py" "$(DATADIR)/scripts/migrations/v1_to_v2.py"
	@install -m 755 "$(BINARY_NAME)" "$(BINDIR)/$(BINARY_NAME)"
	@ln -sf "$(BINARY_NAME)" "$(BINDIR)/$(BINARY_NAME_ALT)"
	@echo "Installed $(BINARY_NAME) and Python backend to $(BINDIR) and $(DATADIR)"

# Uninstall the package-owned CLI payload
uninstall:
	@set -eu; for path in "$(PREFIX)" "$(BINDIR)" "$(PREFIX)/share" "$(PREFIX)/share/agent-guardrails-template" "$(DATADIR)" "$(DATADIR)/scripts" "$(DATADIR)/scripts/migrations"; do \
	  if [ -L "$$path" ]; then echo "refusing to uninstall through symlink: $$path" >&2; exit 1; fi; \
	done; \
	rm -f "$(BINDIR)/$(BINARY_NAME)" "$(BINDIR)/$(BINARY_NAME_ALT)"; \
	rm -f "$(DATADIR)/scripts/team_manager.py" "$(DATADIR)/scripts/batch_operations.py" "$(DATADIR)/scripts/encryption.py"; \
	rm -f "$(DATADIR)/scripts/migrations/__init__.py" "$(DATADIR)/scripts/migrations/v1_to_v2.py"; \
	rmdir "$(DATADIR)/scripts/migrations" 2>/dev/null || true; \
	rmdir "$(DATADIR)/scripts" 2>/dev/null || true; \
	rmdir "$(DATADIR)" 2>/dev/null || true
	@echo "Uninstalled $(BINARY_NAME) and its backend from $(BINDIR) and $(DATADIR)"

# Clean build artifacts
clean:
	$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME_ALT)
	@rm -rf dist/
	@rm -f coverage.out coverage.html

# Verify build works
verify: build
	./$(BINARY_NAME) --version
	./$(BINARY_NAME) --help

# Development mode with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "air not found, installing..." && go install github.com/cosmtrek/air@latest)
	air -c .air.toml

# Update dependencies
update-deps:
	$(GOMOD) tidy
	$(GOMOD) update all

# Print build info
info:
	@echo "Binary: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $(shell go version)"
