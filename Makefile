# Makefile for POMBO Email Client

# Variables
BINARY_NAME=pombo
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Go related variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

# Build target
BUILD_DIR=build
BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)

# Tools
GOLANGCI_LINT=golangci-lint
GOSEC=gosec

.PHONY: all setup build clean test test-coverage test-race lint fmt vet security install run dev help

# Default target
all: clean build

## setup: Install dependencies and development tools
setup:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "Installing development tools..."
	@which $(GOLANGCI_LINT) || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@which $(GOSEC) || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	@echo "Creating directories..."
	mkdir -p $(BUILD_DIR)
	mkdir -p bin
	@echo "Setup complete!"

## build: Build the application
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BINARY_PATH) ./cmd/pombo
	@echo "Built $(BINARY_PATH)"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf bin
	go clean
	@echo "Clean complete!"

## install: Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/pombo
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## run: Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_PATH)

## dev: Run in development mode with debug logging
dev: build
	@echo "Running $(BINARY_NAME) in development mode..."
	POMBO_LOG_LEVEL=debug ./$(BINARY_PATH)

## test: Run unit tests
test:
	@echo "Running tests..."
	go test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-race: Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -race -v ./...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -tags=integration -v ./tests/...

## lint: Run linter
lint:
	@echo "Running linter..."
	$(GOLANGCI_LINT) run

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## security: Run security scanning
security:
	@echo "Running security scan..."
	$(GOSEC) ./...

## check: Run all quality checks
check: fmt vet lint security test
	@echo "All quality checks passed!"

## tidy: Tidy up dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

## deps: Update dependencies
deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

## config-examples: Generate example configuration files
config-examples:
	@echo "Generating example configuration files..."
	mkdir -p configs
	@echo "# Example POMBO configuration" > configs/pombo.example.yaml
	@echo "# Copy this file to ~/.config/pombo/config.yaml and customize" >> configs/pombo.example.yaml
	@echo "" >> configs/pombo.example.yaml
	@echo "app:" >> configs/pombo.example.yaml
	@echo "  cache_dir: ~/.cache/pombo" >> configs/pombo.example.yaml
	@echo "  config_dir: ~/.config/pombo" >> configs/pombo.example.yaml
	@echo "" >> configs/pombo.example.yaml
	@echo "accounts:" >> configs/pombo.example.yaml
	@echo "  - name: \"work\"" >> configs/pombo.example.yaml
	@echo "    email: \"user@company.com\"" >> configs/pombo.example.yaml
	@echo "    provider: \"outlook\"" >> configs/pombo.example.yaml
	@echo "    oauth:" >> configs/pombo.example.yaml
	@echo "      client_id: \"your-client-id\"" >> configs/pombo.example.yaml
	@echo "      redirect_uri: \"http://localhost:8080/callback\"" >> configs/pombo.example.yaml
	@echo "" >> configs/pombo.example.yaml
	@echo "ui:" >> configs/pombo.example.yaml
	@echo "  theme: \"dark\"" >> configs/pombo.example.yaml
	@echo "  vim_keybindings: true" >> configs/pombo.example.yaml
	@echo "  show_line_numbers: false" >> configs/pombo.example.yaml
	@echo "" >> configs/pombo.example.yaml
	@echo "security:" >> configs/pombo.example.yaml
	@echo "  pgp:" >> configs/pombo.example.yaml
	@echo "    auto_encrypt: false" >> configs/pombo.example.yaml
	@echo "    auto_sign: false" >> configs/pombo.example.yaml
	@echo "    keyring_path: \"~/.gnupg\"" >> configs/pombo.example.yaml
	@echo "Example configuration generated: configs/pombo.example.yaml"

## build-cross: Build for multiple platforms
build-cross:
	@echo "Building for multiple platforms..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/pombo
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/pombo
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/pombo
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/pombo
	@echo "Cross-compilation complete!"

## release: Build release artifacts
release: clean test check build-cross
	@echo "Creating release artifacts..."
	cd $(BUILD_DIR) && \
	for binary in *; do \
		if [ "$$binary" != "*.exe" ]; then \
			tar -czf "$$binary.tar.gz" "$$binary"; \
		else \
			zip "$$binary.zip" "$$binary"; \
		fi; \
	done
	@echo "Release artifacts created in $(BUILD_DIR)/"

## docker: Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t pombo:$(VERSION) .
	@echo "Docker image built: pombo:$(VERSION)"

## help: Show this help message
help:
	@echo "POMBO Email Client - Build System"
	@echo ""
	@echo "Available targets:"
	@awk '/^##/{h=$$0; getline; print substr(h,4) " " $$0}' $(MAKEFILE_LIST) | column -t -s ' ' | sed 's/^/  /'
	@echo ""
	@echo "Examples:"
	@echo "  make setup     # First time setup"
	@echo "  make build     # Build the application"
	@echo "  make test      # Run tests"
	@echo "  make run       # Build and run"
	@echo "  make dev       # Run in development mode"
	@echo "  make check     # Run all quality checks"