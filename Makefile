# Sage Project Makefile

# Variables
CRYPTO_BINARY=sage-crypto
DID_BINARY=sage-did
BUILD_DIR=build
BIN_DIR=$(BUILD_DIR)/bin
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
build-crypto: $(BIN_DIR)/$(CRYPTO_BINARY)

$(BIN_DIR)/$(CRYPTO_BINARY):
	@echo "Building $(CRYPTO_BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(CRYPTO_BINARY) ./$(CMD_DIR)/$(CRYPTO_BINARY)
	@echo "Build complete: $(BIN_DIR)/$(CRYPTO_BINARY)"

# Build sage-did binary
.PHONY: build-did
build-did: $(BIN_DIR)/$(DID_BINARY)

$(BIN_DIR)/$(DID_BINARY):
	@echo "Building $(DID_BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(DID_BINARY) ./$(CMD_DIR)/$(DID_BINARY)
	@echo "Build complete: $(BIN_DIR)/$(DID_BINARY)"

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

# Run Phase 1 complete test suite
.PHONY: test-phase1
test-phase1:
	@echo "Running Phase 1 complete test suite..."
	@bash ./test_phase1.sh

# Run quick tests for Phase 1 components
.PHONY: test-quick
test-quick:
	@echo "Running quick tests for Phase 1 components..."
	@bash ./run_tests.sh

# Run enhanced provider tests
.PHONY: test-provider
test-provider:
	@echo "Testing Enhanced Provider..."
	$(GO) test -v ./crypto/chain/ethereum -count=1

# Run vault tests
.PHONY: test-vault
test-vault:
	@echo "Testing SecureVault..."
	$(GO) test -v ./crypto/vault -count=1

# Run logger tests
.PHONY: test-logger
test-logger:
	@echo "Testing Logger..."
	$(GO) test -v ./internal/logger -count=1

# Run health checker tests
.PHONY: test-health
test-health:
	@echo "Testing Health Checker..."
	$(GO) test -v ./health -count=1

# Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@echo "Starting test environment..."
	@bash ./tests/integration/setup_test_env.sh start
	@echo "Running tests..."
	$(GO) test -v ./tests/integration/... -tags=integration -count=1
	@echo "Stopping test environment..."
	@bash ./tests/integration/setup_test_env.sh stop

# Run integration tests without setup (assumes environment is ready)
.PHONY: test-integration-only
test-integration-only:
	@echo "Running integration tests (environment should be ready)..."
	$(GO) test -v ./tests/integration/... -tags=integration -count=1

.PHONY: test-handshake
test-handshake:
	@echo "Running handshake scenario..."
	@bash ./tests/handshake/run_handshake.sh

# Start local blockchain for testing
.PHONY: blockchain-start
blockchain-start:
	@echo "Starting local blockchain..."
	@bash ./tests/integration/setup_test_env.sh start

# Stop local blockchain
.PHONY: blockchain-stop
blockchain-stop:
	@echo "Stopping local blockchain..."
	@bash ./tests/integration/setup_test_env.sh stop

# Check blockchain status
.PHONY: blockchain-status
blockchain-status:
	@bash ./tests/integration/setup_test_env.sh status

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Run integration benchmarks
.PHONY: bench-integration
bench-integration:
	@echo "Running integration benchmarks..."
	$(GO) test -bench=. -benchmem ./tests/integration/... -tags=integration

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f sage-crypto sage-did sage-verify
	@rm -f test_output.tmp
	@rm -f coverage.out coverage.html
	@rm -f *.test
	@rm -rf test-storage
	@rm -f test-*.jwk test-*.pem test-message.txt
	@rm -f test_accounts.json
	@rm -f .blockchain.pid
	@find . -name "*.test" -type f -delete
	@find . -name "*.out" -type f -delete
	@find . -name "*.log" -type f -delete
	@find . -type d -name "__debug_bin*" -exec rm -rf {} + 2>/dev/null || true
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
	@echo ""
	@echo "Build targets:"
	@echo "  make build         - Build all CLI binaries (sage-crypto and sage-did)"
	@echo "  make build-crypto  - Build sage-crypto binary only"
	@echo "  make build-did     - Build sage-did binary only"
	@echo ""
	@echo "Test targets:"
	@echo "  make test          - Run all tests"
	@echo "  make test-crypto   - Run crypto package tests only"
	@echo "  make test-phase1   - Run Phase 1 complete test suite"
	@echo "  make test-quick    - Run quick tests for Phase 1 components"
	@echo "  make test-provider - Run enhanced provider tests"
	@echo "  make test-vault    - Run SecureVault tests"
	@echo "  make test-logger   - Run logger tests"
	@echo "  make test-health   - Run health checker tests"
	@echo ""
	@echo "Integration test targets:"
	@echo "  make test-integration      - Run integration tests with setup"
	@echo "  make test-integration-only - Run integration tests (no setup)"
	@echo "  make blockchain-start      - Start local blockchain"
	@echo "  make blockchain-stop       - Stop local blockchain"
	@echo "  make blockchain-status     - Check blockchain status"
	@echo ""
	@echo "Benchmark targets:"
	@echo "  make bench            - Run all benchmarks"
	@echo "  make bench-integration - Run integration benchmarks"
	@echo ""
	@echo "Utility targets:"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make install       - Install binaries to GOPATH/bin"
	@echo "  make lint          - Run linter"
	@echo "  make fmt           - Format code"
	@echo "  make tidy          - Run go mod tidy"
	@echo "  make help          - Show this help message"
