# Sage Project Makefile

# Variables
BINARY_NAME=sage-crypto
BUILD_DIR=build/bin
CMD_DIR=cmd

# Go build variables
GO=go
GOFLAGS=-v
LDFLAGS=

# Default target
.PHONY: all
all: build

# Build the sage-crypto binary
.PHONY: build
build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME):
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)/$(BINARY_NAME)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run crypto package tests only
.PHONY: test-crypto
test-crypto:
	@echo "Running crypto package tests..."
	$(GO) test -v ./crypto/...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

# Install binary to GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install ./$(CMD_DIR)/$(BINARY_NAME)

# Run linting
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Run go mod tidy
.PHONY: tidy
tidy:
	@echo "Running go mod tidy..."
	$(GO) mod tidy

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build       - Build the sage-crypto binary"
	@echo "  make test        - Run all tests"
	@echo "  make test-crypto - Run crypto package tests only"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make install     - Install binary to GOPATH/bin"
	@echo "  make lint        - Run linter"
	@echo "  make fmt         - Format code"
	@echo "  make tidy        - Run go mod tidy"
	@echo "  make help        - Show this help message"