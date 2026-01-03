.PHONY: build run test test-cover lint clean deps

# Build variables
BINARY_NAME=unifiedui-agent-service
BUILD_DIR=./bin

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build the application
build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

# Run the application
run:
	@echo "Running..."
	$(GOCMD) run ./cmd/server

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with coverage percentage
test-cover-percent:
	@echo "Running tests with coverage percentage..."
	$(GOTEST) -v -cover ./... | grep -E "coverage:|PASS|FAIL"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Generate mocks (requires mockery)
mocks:
	@echo "Generating mocks..."
	mockery --all --dir=./internal/core --output=./tests/mocks --outpkg=mocks

# Run the application with hot reload (requires air)
dev:
	@echo "Running with hot reload..."
	air

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(BINARY_NAME):latest

# Help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application"
	@echo "  run             - Run the application"
	@echo "  test            - Run all tests"
	@echo "  test-cover      - Run tests with coverage report"
	@echo "  test-cover-percent - Run tests with coverage percentage"
	@echo "  lint            - Run linter"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Download dependencies"
	@echo "  mocks           - Generate mocks"
	@echo "  dev             - Run with hot reload"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run Docker container"
