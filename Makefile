# Sage Project Makefile

# Variables
CRYPTO_BINARY=sage-crypto
DID_BINARY=sage-did
BUILD_DIR=build/bin
CMD_DIR=cmd

# Go build variables
GO=go
GOFLAGS=-v
LDFLAGS=

# Default target
.PHONY: all
all: build

# Build all binaries
.PHONY: build
build: build-crypto build-did

# Build sage-crypto binary
.PHONY: build-crypto
build-crypto: $(BUILD_DIR)/$(CRYPTO_BINARY)

$(BUILD_DIR)/$(CRYPTO_BINARY):
	@echo "Building $(CRYPTO_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(CRYPTO_BINARY) ./$(CMD_DIR)/$(CRYPTO_BINARY)
	@echo "Build complete: $(BUILD_DIR)/$(CRYPTO_BINARY)"

# Build sage-did binary
.PHONY: build-did
build-did: $(BUILD_DIR)/$(DID_BINARY)

$(BUILD_DIR)/$(DID_BINARY):
	@echo "Building $(DID_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(DID_BINARY) ./$(CMD_DIR)/$(DID_BINARY)
	@echo "Build complete: $(BUILD_DIR)/$(DID_BINARY)"

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

# Install binaries to GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(CRYPTO_BINARY)..."
	$(GO) install ./$(CMD_DIR)/$(CRYPTO_BINARY)
	@echo "Installing $(DID_BINARY)..."
	$(GO) install ./$(CMD_DIR)/$(DID_BINARY)

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
	@echo "  make build       - Build all CLI binaries (sage-crypto and sage-did)"
	@echo "  make build-crypto- Build sage-crypto binary only"
	@echo "  make build-did   - Build sage-did binary only"
	@echo "  make test        - Run all tests"
	@echo "  make test-crypto - Run crypto package tests only"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make install     - Install binaries to GOPATH/bin"
	@echo "  make lint        - Run linter"
	@echo "  make fmt         - Format code"
	@echo "  make tidy        - Run go mod tidy"
	@echo "  make help        - Show this help message"