# Sage Project Makefile

# Variables
CRYPTO_BINARY=sage-crypto
DID_BINARY=sage-did
VERIFY_BINARY=sage-verify
TEST_CLIENT_BINARY=test-client
TEST_SERVER_BINARY=test-server
BUILD_DIR=build
BIN_DIR=$(BUILD_DIR)/bin
CMD_DIR=cmd
EXAMPLES_DIR=examples
REPORTS_DIR=reports

# Go build variables
GO=go
GOFLAGS=-v
LDFLAGS=-w -s
GOTOOLCHAIN?=auto

# Version information
VERSION?=$(shell cat VERSION 2>/dev/null || echo "0.1.0")
GIT_COMMIT?=$(shell git rev-parse HEAD 2>/dev/null || echo "")
GIT_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
# Commit timestamp, not wall-clock time, so two builds of the same commit are identical
BUILD_DATE?=$(shell git log -1 --format=%cI 2>/dev/null || date -u '+%Y-%m-%dT%H:%M:%SZ')

# Build flags for version injection
VERSION_PKG=github.com/sage-x-project/sage/pkg/version
BUILD_LDFLAGS=-X '$(VERSION_PKG).Version=$(VERSION)' \
	-X '$(VERSION_PKG).GitCommit=$(GIT_COMMIT)' \
	-X '$(VERSION_PKG).GitBranch=$(GIT_BRANCH)' \
	-X '$(VERSION_PKG).BuildDate=$(BUILD_DATE)'

# Legacy support for main package version
MAIN_BUILD_LDFLAGS=$(BUILD_LDFLAGS) \
	-X 'main.Version=$(VERSION)' \
	-X 'main.Commit=$(GIT_COMMIT)' \
	-X 'main.BuildTime=$(BUILD_DATE)'

# Library build variables

# Platform detection
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Cross-compilation targets
PLATFORMS=linux darwin windows
ARCHITECTURES=amd64 arm64
DIST_DIR=$(BUILD_DIR)/dist

# Example binaries
EXAMPLE_BASIC_DEMO=basic-demo
EXAMPLE_BASIC_TOOL=basic-tool
EXAMPLE_CLIENT=sage-client
EXAMPLE_SIMPLE=simple-standalone
EXAMPLE_SECURE_CHAT=secure-chat
EXAMPLE_VULNERABLE_CHAT=vulnerable-chat
EXAMPLE_ATTACKER=attacker

# Default target
.PHONY: all
all: build

# Build all binaries
.PHONY: build
build: build-binaries build-examples

# Build core binaries
.PHONY: build-binaries
build-binaries: build-crypto build-did build-verify

# Build all binaries for all platforms
.PHONY: build-all-platforms
build-all-platforms:
	@echo "Building all binaries for all platforms and architectures..."
	@$(MAKE) build-binaries-all-platforms
	@echo "All platform builds complete!"

# Build core binaries for all platforms
.PHONY: build-binaries-all-platforms
build-binaries-all-platforms:
	@for platform in $(PLATFORMS); do \
		for arch in $(ARCHITECTURES); do \
			echo "Building for $$platform/$$arch..."; \
			$(MAKE) build-platform GOOS=$$platform GOARCH=$$arch || true; \
		done; \
	done

# Build for specific platform (called by build-binaries-all-platforms)
.PHONY: build-platform
build-platform:
	@echo "Building binaries for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(DIST_DIR)/$(GOOS)-$(GOARCH)
	@GOTOOLCHAIN=$(GOTOOLCHAIN) CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) -trimpath \
		-ldflags "$(LDFLAGS) $(BUILD_LDFLAGS)" \
		-o $(DIST_DIR)/$(GOOS)-$(GOARCH)/$(CRYPTO_BINARY)$(if $(filter windows,$(GOOS)),.exe,) \
		./$(CMD_DIR)/$(CRYPTO_BINARY)
	@GOTOOLCHAIN=$(GOTOOLCHAIN) CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) -trimpath \
		-ldflags "$(LDFLAGS) $(BUILD_LDFLAGS)" \
		-o $(DIST_DIR)/$(GOOS)-$(GOARCH)/$(DID_BINARY)$(if $(filter windows,$(GOOS)),.exe,) \
		./$(CMD_DIR)/$(DID_BINARY)
	@GOTOOLCHAIN=$(GOTOOLCHAIN) CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) -trimpath \
		-ldflags "$(LDFLAGS) $(BUILD_LDFLAGS)" \
		-o $(DIST_DIR)/$(GOOS)-$(GOARCH)/$(VERIFY_BINARY)$(if $(filter windows,$(GOOS)),.exe,) \
		./$(CMD_DIR)/$(VERIFY_BINARY)
	@echo "Build complete: $(DIST_DIR)/$(GOOS)-$(GOARCH)/"

# Build sage-crypto binary
.PHONY: build-crypto
build-crypto: $(BIN_DIR)/$(CRYPTO_BINARY)

$(BIN_DIR)/$(CRYPTO_BINARY):
	@echo "Building $(CRYPTO_BINARY)..."
	@echo "Version: $(VERSION) | Commit: $(GIT_COMMIT) | Branch: $(GIT_BRANCH)"
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) $(MAIN_BUILD_LDFLAGS)" -o $(BIN_DIR)/$(CRYPTO_BINARY) ./$(CMD_DIR)/$(CRYPTO_BINARY)
	@echo "Build complete: $(BIN_DIR)/$(CRYPTO_BINARY)"

# Protocol test vectors (published in the sage-spec repository)
VECTORS_DIR ?= ../sage-spec/vectors

.PHONY: vectors vectors-check
vectors:
	$(GO) run ./$(CMD_DIR)/sage-vectors gen -dir $(VECTORS_DIR)

vectors-check:
	$(GO) run ./$(CMD_DIR)/sage-vectors check -dir $(VECTORS_DIR)

# Build sage-did binary
.PHONY: build-did
build-did: $(BIN_DIR)/$(DID_BINARY)

$(BIN_DIR)/$(DID_BINARY):
	@echo "Building $(DID_BINARY)..."
	@echo "Version: $(VERSION) | Commit: $(GIT_COMMIT) | Branch: $(GIT_BRANCH)"
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) $(MAIN_BUILD_LDFLAGS)" -o $(BIN_DIR)/$(DID_BINARY) ./$(CMD_DIR)/$(DID_BINARY)
	@echo "Build complete: $(BIN_DIR)/$(DID_BINARY)"

# Build sage-verify binary
.PHONY: build-verify
build-verify: $(BIN_DIR)/$(VERIFY_BINARY)

$(BIN_DIR)/$(VERIFY_BINARY):
	@echo "Building $(VERIFY_BINARY)..."
	@echo "Version: $(VERSION) | Commit: $(GIT_COMMIT) | Branch: $(GIT_BRANCH)"
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS) $(MAIN_BUILD_LDFLAGS)" -o $(BIN_DIR)/$(VERIFY_BINARY) ./$(CMD_DIR)/$(VERIFY_BINARY)
	@echo "Build complete: $(BIN_DIR)/$(VERIFY_BINARY)"

# Build test utilities (deprecated - moved to tests/handshake/)
# .PHONY: build-test-utils
# build-test-utils: $(BIN_DIR)/$(TEST_CLIENT_BINARY) $(BIN_DIR)/$(TEST_SERVER_BINARY)

# $(BIN_DIR)/$(TEST_CLIENT_BINARY):
# 	@echo "Building $(TEST_CLIENT_BINARY)..."
# 	@mkdir -p $(BIN_DIR)
# 	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(TEST_CLIENT_BINARY) ./$(CMD_DIR)/$(TEST_CLIENT_BINARY)
# 	@echo "Build complete: $(BIN_DIR)/$(TEST_CLIENT_BINARY)"

# $(BIN_DIR)/$(TEST_SERVER_BINARY):
# 	@echo "Building $(TEST_SERVER_BINARY)..."
# 	@mkdir -p $(BIN_DIR)
# 	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(TEST_SERVER_BINARY) ./$(CMD_DIR)/$(TEST_SERVER_BINARY)
# 	@echo "Build complete: $(BIN_DIR)/$(TEST_SERVER_BINARY)"

# Build all examples
.PHONY: build-examples
build-examples: build-example-basic-demo build-example-basic-tool build-example-client \
	build-example-simple build-example-secure-chat build-example-vulnerable-chat build-example-attacker

# Build basic-demo example
.PHONY: build-example-basic-demo
build-example-basic-demo: $(BIN_DIR)/$(EXAMPLE_BASIC_DEMO)

$(BIN_DIR)/$(EXAMPLE_BASIC_DEMO):
	@echo "Building example: $(EXAMPLE_BASIC_DEMO)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(EXAMPLE_BASIC_DEMO) ./$(EXAMPLES_DIR)/mcp-integration/$(EXAMPLE_BASIC_DEMO)
	@echo "Build complete: $(BIN_DIR)/$(EXAMPLE_BASIC_DEMO)"

# Build basic-tool example
.PHONY: build-example-basic-tool
build-example-basic-tool: $(BIN_DIR)/$(EXAMPLE_BASIC_TOOL)

$(BIN_DIR)/$(EXAMPLE_BASIC_TOOL):
	@echo "Building example: $(EXAMPLE_BASIC_TOOL)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(EXAMPLE_BASIC_TOOL) ./$(EXAMPLES_DIR)/mcp-integration/$(EXAMPLE_BASIC_TOOL)
	@echo "Build complete: $(BIN_DIR)/$(EXAMPLE_BASIC_TOOL)"

# Build sage-client example
.PHONY: build-example-client
build-example-client: $(BIN_DIR)/$(EXAMPLE_CLIENT)

$(BIN_DIR)/$(EXAMPLE_CLIENT):
	@echo "Building example: $(EXAMPLE_CLIENT)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(EXAMPLE_CLIENT) ./$(EXAMPLES_DIR)/mcp-integration/client
	@echo "Build complete: $(BIN_DIR)/$(EXAMPLE_CLIENT)"

# Build simple-standalone example
.PHONY: build-example-simple
build-example-simple: $(BIN_DIR)/$(EXAMPLE_SIMPLE)

$(BIN_DIR)/$(EXAMPLE_SIMPLE):
	@echo "Building example: $(EXAMPLE_SIMPLE)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(EXAMPLE_SIMPLE) ./$(EXAMPLES_DIR)/mcp-integration/$(EXAMPLE_SIMPLE)
	@echo "Build complete: $(BIN_DIR)/$(EXAMPLE_SIMPLE)"

# Build secure-chat example
.PHONY: build-example-secure-chat
build-example-secure-chat: $(BIN_DIR)/$(EXAMPLE_SECURE_CHAT)

$(BIN_DIR)/$(EXAMPLE_SECURE_CHAT):
	@echo "Building example: $(EXAMPLE_SECURE_CHAT)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(EXAMPLE_SECURE_CHAT) ./$(EXAMPLES_DIR)/mcp-integration/vulnerable-vs-secure/$(EXAMPLE_SECURE_CHAT)
	@echo "Build complete: $(BIN_DIR)/$(EXAMPLE_SECURE_CHAT)"

# Build vulnerable-chat example
.PHONY: build-example-vulnerable-chat
build-example-vulnerable-chat: $(BIN_DIR)/$(EXAMPLE_VULNERABLE_CHAT)

$(BIN_DIR)/$(EXAMPLE_VULNERABLE_CHAT):
	@echo "Building example: $(EXAMPLE_VULNERABLE_CHAT)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(EXAMPLE_VULNERABLE_CHAT) ./$(EXAMPLES_DIR)/mcp-integration/vulnerable-vs-secure/$(EXAMPLE_VULNERABLE_CHAT)
	@echo "Build complete: $(BIN_DIR)/$(EXAMPLE_VULNERABLE_CHAT)"

# Build attacker example
.PHONY: build-example-attacker
build-example-attacker: $(BIN_DIR)/$(EXAMPLE_ATTACKER)

$(BIN_DIR)/$(EXAMPLE_ATTACKER):
	@echo "Building example: $(EXAMPLE_ATTACKER)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(EXAMPLE_ATTACKER) ./$(EXAMPLES_DIR)/mcp-integration/vulnerable-vs-secure/$(EXAMPLE_ATTACKER)
	@echo "Build complete: $(BIN_DIR)/$(EXAMPLE_ATTACKER)"

# Run examples
.PHONY: run-example-basic-demo
run-example-basic-demo: build-example-basic-demo
	@echo "Running $(EXAMPLE_BASIC_DEMO)..."
	$(BIN_DIR)/$(EXAMPLE_BASIC_DEMO)

.PHONY: run-example-basic-tool
run-example-basic-tool: build-example-basic-tool
	@echo "Running $(EXAMPLE_BASIC_TOOL)..."
	$(BIN_DIR)/$(EXAMPLE_BASIC_TOOL)

.PHONY: run-example-client
run-example-client: build-example-client
	@echo "Running $(EXAMPLE_CLIENT)..."
	$(BIN_DIR)/$(EXAMPLE_CLIENT)

.PHONY: run-example-simple
run-example-simple: build-example-simple
	@echo "Running $(EXAMPLE_SIMPLE)..."
	$(BIN_DIR)/$(EXAMPLE_SIMPLE)

.PHONY: run-example-secure-chat
run-example-secure-chat: build-example-secure-chat
	@echo "Running $(EXAMPLE_SECURE_CHAT)..."
	$(BIN_DIR)/$(EXAMPLE_SECURE_CHAT)

.PHONY: run-example-vulnerable-chat
run-example-vulnerable-chat: build-example-vulnerable-chat
	@echo "Running $(EXAMPLE_VULNERABLE_CHAT)..."
	$(BIN_DIR)/$(EXAMPLE_VULNERABLE_CHAT)

.PHONY: run-example-attacker
run-example-attacker: build-example-attacker
	@echo "Running $(EXAMPLE_ATTACKER)..."
	$(BIN_DIR)/$(EXAMPLE_ATTACKER)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run crypto package tests only
.PHONY: test-crypto
test-crypto:
	@echo "Running crypto package tests..."
	$(GO) test -v ./pkg/agent/crypto/...

# Run Phase 1 complete test suite
# NOTE: Original test_phase1.sh script has been deprecated.
# This target now runs standard tests for all agent packages.
.PHONY: test-phase1
test-phase1:
	@echo "Running Phase 1 complete test suite..."
	@echo "Note: test_phase1.sh script not found - running standard tests instead"
	$(GO) test -v ./pkg/agent/...

# Run quick tests for Phase 1 components
# NOTE: Original run_tests.sh script has been deprecated.
# This target now runs standard tests for core components.
.PHONY: test-quick
test-quick:
	@echo "Running quick tests for Phase 1 components..."
	@echo "Note: run_tests.sh script not found - running standard tests instead"
	$(GO) test -v ./pkg/agent/crypto/... ./pkg/agent/did/... ./pkg/agent/core/...

# Run enhanced provider tests
.PHONY: test-provider
test-provider:
	@echo "Testing Enhanced Provider..."
	$(GO) test -v ./pkg/agent/crypto/chain/ethereum -count=1

# Run vault tests
.PHONY: test-vault
test-vault:
	@echo "Testing SecureVault..."
	$(GO) test -v ./pkg/agent/crypto/vault -count=1

# Run logger tests
.PHONY: test-logger
test-logger:
	@echo "Testing Logger..."
	$(GO) test -v ./internal/logger -count=1

# Run health checker tests
.PHONY: test-health
test-health:
	@echo "Testing Health Checker..."
	$(GO) test -v ./pkg/health -count=1

# Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@echo "Starting test environment..."
	@bash ./tools/scripts/setup_test_env.sh start
	@echo "Running tests..."
	@set -e; \
	trap 'echo "Stopping test environment..."; bash ./tools/scripts/setup_test_env.sh stop' EXIT; \
	$(GO) test -v ./tests/integration/... -tags=integration -count=1

# Run integration tests without setup (assumes environment is ready)
.PHONY: test-integration-only
test-integration-only:
	@echo "Running integration tests (environment should be ready)..."
	$(GO) test -v ./tests/integration/... -tags=integration -count=1

# Run E2E tests (requires external services like Sepolia)
.PHONY: test-e2e
test-e2e:
	@echo "Running E2E tests..."
	@echo "Note: Requires SEPOLIA_RPC_URL and SEPOLIA_PRIVATE_KEY environment variables"
	$(GO) test -v -tags=e2e ./tests/integration/... -timeout 10m

# Run E2E tests on Sepolia testnet
.PHONY: test-e2e-sepolia
test-e2e-sepolia:
	@echo "Running Sepolia E2E tests..."
	$(GO) test -v -tags=e2e ./tests/integration/... -run Sepolia -timeout 10m

# Run E2E tests without external blockchain (local only)
.PHONY: test-e2e-local
test-e2e-local:
	@echo "Running local E2E tests (RFC 9421, key management, cross-chain)..."
	$(GO) test -v -tags=e2e ./tests/integration/... -run "RFC9421|KeyType|CrossChain|KeyRotation|MultiChain|Performance" -timeout 5m

# Run E2E tests with coverage
.PHONY: test-e2e-coverage
test-e2e-coverage:
	@echo "Running E2E tests with coverage..."
	@mkdir -p $(REPORTS_DIR)
	$(GO) test -v -tags=e2e -coverprofile=$(REPORTS_DIR)/e2e-coverage.out ./tests/integration/... -timeout 10m
	$(GO) tool cover -html=$(REPORTS_DIR)/e2e-coverage.out -o $(REPORTS_DIR)/e2e-coverage.html
	@echo "Coverage report: $(REPORTS_DIR)/e2e-coverage.html"

# DEPRECATED: Handshake and HPKE test scripts have been removed/integrated into standard tests
# Use 'make test-integration' or 'make test' instead
# .PHONY: test-handshake
# test-handshake:
# 	@echo "Running handshake scenario..."
# 	@bash ./tests/integration/session/handshake/run_handshake.sh
#
# .PHONY: test-hpke
# test-hpke:
# 	@echo "Running HPKE based handshake scenario..."
# 	@bash ./tests/integration/session/hpke/run_hpke_handshake.sh

# Start local blockchain for testing
.PHONY: blockchain-start
blockchain-start:
	@echo "Starting local blockchain..."
	@bash ./tools/scripts/setup_test_env.sh start

# Stop local blockchain
.PHONY: blockchain-stop
blockchain-stop:
	@echo "Stopping local blockchain..."
	@bash ./tools/scripts/setup_test_env.sh stop

# Check blockchain status
.PHONY: blockchain-status
blockchain-status:
	@bash ./tools/scripts/setup_test_env.sh status

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Run comprehensive benchmarks using script
.PHONY: bench-full
bench-full:
	@echo "Running comprehensive benchmarks..."
	@bash ./tools/scripts/run-benchmarks.sh

# Run integration benchmarks
.PHONY: bench-integration
bench-integration:
	@echo "Running integration benchmarks..."
	$(GO) test -bench=. -benchmem ./tests/integration/... -tags=integration

# Run fuzz tests
.PHONY: fuzz
fuzz:
	@echo "Running fuzz tests..."
	@bash ./tools/scripts/run-fuzz.sh

# Run load tests
.PHONY: loadtest
loadtest:
	@echo "Running load tests..."
	@bash ./tools/scripts/run-loadtest.sh

# Verify all features (comprehensive feature verification)
.PHONY: verify-features
verify-features:
	@echo "Running comprehensive feature verification..."
	@bash ./tools/scripts/verify_all_features.sh -v

# Run full test suite (all tests + verification)
.PHONY: test-full
test-full:
	@echo "Running full test suite..."
	@bash ./tools/scripts/full-test.sh

# Quick verification (fast feature check)
.PHONY: verify-quick
verify-quick:
	@echo "Running quick verification..."
	@bash ./tools/scripts/quick_verify.sh

# Additional verification targets
.PHONY: verify-makefile
verify-makefile:
	@echo "Verifying Makefile consistency..."
	@bash ./tools/scripts/verify_makefile.sh

.PHONY: verify-rfc9421-ed25519
verify-rfc9421-ed25519:
	@echo "Verifying RFC 9421 Ed25519 implementation..."
	@bash ./tools/scripts/verify_rfc9421_ed25519.sh

# Cleanup test environment
.PHONY: test-cleanup
test-cleanup:
	@echo "Cleaning up test environment..."
	@bash ./tools/scripts/cleanup_test_env.sh

# Database management targets
.PHONY: db-backup
db-backup:
	@echo "Backing up database..."
	@bash ./tools/scripts/backup-db.sh

.PHONY: db-restore
db-restore:
	@echo "Restoring database..."
	@bash ./tools/scripts/restore-db.sh

.PHONY: db-seed
db-seed:
	@echo "Seeding database with test data..."
	@bash ./tools/scripts/seed-db.sh

.PHONY: db-migrate-up
db-migrate-up:
	@echo "Running database migrations (up)..."
	@bash ./tools/scripts/migrate-up.sh

.PHONY: db-migrate-down
db-migrate-down:
	@echo "Rolling back database migrations..."
	@bash ./tools/scripts/migrate-down.sh

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@bash ./tools/scripts/docker-build.sh

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	@bash ./tools/scripts/docker-run.sh

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
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
	@echo "Cleaning Rust build artifacts..."
	@rm -rf target/
	@echo "Cleaning SDK artifacts..."
	@find sdk/ -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	@find sdk/ -type d -name ".pytest_cache" -exec rm -rf {} + 2>/dev/null || true
	@find sdk/ -type d -name "*.egg-info" -exec rm -rf {} + 2>/dev/null || true
	@find sdk/ -type d -name "target" -exec rm -rf {} + 2>/dev/null || true
	@find sdk/ -type f -name "Cargo.lock" -delete 2>/dev/null || true
	@echo "Cleaning test artifacts..."
	@rm -rf integration/tests/integration/
	@rm -rf integration/tests/session/
	@rm -rf testdata/
	@echo "Cleaning loadtest results..."
	@rm -rf tools/loadtest/analysis/*
	@rm -rf tools/loadtest/reports/*
	@echo "Cleaning user data and reports..."
	@rm -rf keys/
	@rm -rf logs/
	@rm -rf reports/
	@rm -rf testutil/
	@rm -rf handshake/
	@rm -rf integration/
	@echo "Clean complete"

# Clean everything including reports
.PHONY: clean-all
clean-all: clean
	@echo "Full clean complete"

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

# Smart contracts live in github.com/SAGE-X-project/sage-contracts. The Go
# bindings are generated from the ABI files of the commit recorded in
# .contracts-version, checked out into .sage-contracts (override CONTRACTS_DIR
# to use another checkout, for example ../sage-contracts while developing).
CONTRACTS_REPO ?= https://github.com/SAGE-X-project/sage-contracts.git
CONTRACTS_DIR ?= $(CURDIR)/.sage-contracts
CONTRACTS_REF := $(shell cat .contracts-version)

.PHONY: contracts-checkout
contracts-checkout:
	@if [ ! -d "$(CONTRACTS_DIR)/.git" ]; then git clone -q $(CONTRACTS_REPO) "$(CONTRACTS_DIR)"; fi
	@git -C "$(CONTRACTS_DIR)" fetch -q origin $(CONTRACTS_REF) && git -C "$(CONTRACTS_DIR)" checkout -q $(CONTRACTS_REF)
	@echo "sage-contracts at $(CONTRACTS_REF) in $(CONTRACTS_DIR)"

# Regenerate the Go contract bindings from the sage-contracts ABIs
.PHONY: bindings
bindings:
	@echo "Generating Go bindings from $(CONTRACTS_DIR)/abi..."
	@CONTRACTS_DIR="$(CONTRACTS_DIR)" GOTOOLCHAIN=$(GOTOOLCHAIN) tools/scripts/gen-bindings.sh

# Fail when the committed bindings differ from freshly generated ones
.PHONY: bindings-check
bindings-check:
	@echo "Checking Go contract bindings for drift..."
	@rm -rf $(REPORTS_DIR)/bindings && CONTRACTS_DIR="$(CONTRACTS_DIR)" GOTOOLCHAIN=$(GOTOOLCHAIN) tools/scripts/gen-bindings.sh $(REPORTS_DIR)/bindings >/dev/null
	@diff -r $(REPORTS_DIR)/bindings pkg/blockchain/ethereum/contracts/agentcardregistry \
		|| { echo "Go bindings are out of date: run 'make bindings' and commit the result"; exit 1; }
	@echo "Bindings are up to date"

# Regenerate docs/INDEX.md from the Markdown files in the repository
.PHONY: docs-index
docs-index:
	@python3 tools/scripts/gen-docs-index.py

# Fail when docs/INDEX.md differs from the generated index
.PHONY: docs-index-check
docs-index-check:
	@python3 tools/scripts/gen-docs-index.py --check

# Fail when Dockerfiles or workflows build with a Go version other than go.mod's toolchain
.PHONY: check-go-version
check-go-version:
	@bash tools/scripts/check-go-version.sh

# Fail if a library package registers itself in init()
.PHONY: check-no-init
check-no-init:
	@bash tools/scripts/check-no-init.sh

# Regenerate the AST-based code graph (docs/refactoring/graph)
.PHONY: codegraph
codegraph:
	@echo "Building code graph..."
	@cd tools/codegraph && GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) run . -dir ../.. -out ../../docs/refactoring/graph

# Fail on layer violations that are not in tools/codegraph/layer-baseline.txt
.PHONY: codegraph-check
codegraph-check:
	@echo "Checking layer boundaries..."
	@cd tools/codegraph && GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) run . -dir ../.. -out $(REPORTS_DIR)/codegraph -layer-baseline layer-baseline.txt
	@bash tools/scripts/check-no-init.sh

# Run CI lint checks (same as GitHub Actions)
.PHONY: lint-ci
lint-ci:
	@echo "Running CI lint checks (same as GitHub Actions)..."
	@bash ./tools/scripts/lint-ci.sh

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

# Version management
.PHONY: update-version
update-version:
	@echo "Updating project version..."
	@bash ./tools/scripts/update-version.sh

# Clean test reports
.PHONY: clean-reports
clean-reports:
	@echo "Cleaning test reports..."
	rm -rf $(REPORTS_DIR)

# Create release packages for all platforms
.PHONY: package
package: build-all-platforms
	@echo "Creating release packages..."
	@mkdir -p $(DIST_DIR)/packages
	@for platform in $(PLATFORMS); do \
		for arch in $(ARCHITECTURES); do \
			if [ -d "$(DIST_DIR)/$$platform-$$arch" ]; then \
				echo "Packaging $$platform-$$arch..."; \
				cd $(DIST_DIR)/$$platform-$$arch && \
				tar czf ../packages/sage-$$platform-$$arch.tar.gz * && \
				cd ../..; \
			fi; \
		done; \
	done
	@echo "Package creation complete!"
	@echo "Packages available in: $(DIST_DIR)/packages/"
	@ls -lh $(DIST_DIR)/packages/

# Create checksums for release packages
.PHONY: checksums
checksums:
	@echo "Generating checksums..."
	@cd $(DIST_DIR)/packages && sha256sum *.tar.gz > SHA256SUMS
	@echo "Checksums generated: $(DIST_DIR)/packages/SHA256SUMS"
	@cat $(DIST_DIR)/packages/SHA256SUMS

# Full release build (binaries + packages + checksums)
.PHONY: release
release: clean build-all-platforms package checksums
	@echo "===================="
	@echo "Release build complete!"
	@echo "===================="
	@echo ""
	@echo "Binaries:"
	@find $(DIST_DIR) -type f \( -name "sage-*" -o -name "*.exe" \) -exec ls -lh {} \;
	@echo ""
	@echo "Libraries:"
	@echo ""
	@echo "Packages:"
	@ls -lh $(DIST_DIR)/packages/

# Help
.PHONY: help
help:
	@echo "========================================"
	@echo "SAGE Build System"
	@echo "========================================"
	@echo ""
	@echo "Quick Start:"
	@echo "  make                    - Build all binaries and examples (default)"
	@echo "  make build-all-platforms - Build for Linux, macOS, Windows (x86/ARM)"
	@echo "  make release            - Full release build with packages"
	@echo ""
	@echo "Build targets:"
	@echo "  make build              - Build all binaries and examples"
	@echo "  make build-binaries     - Build all CLI binaries"
	@echo "  make build-crypto       - Build sage-crypto binary only"
	@echo "  make build-did          - Build sage-did binary only"
	@echo "  make build-verify       - Build sage-verify binary only"
	@echo ""
	@echo "Cross-platform build targets:"
	@echo "  make build-all-platforms         - Build binaries for all platforms"
	@echo "  make build-platform GOOS=linux GOARCH=amd64  - Build for specific platform"
	@echo ""
	@echo "Library build targets:"
	@echo ""
	@echo ""
	@echo "Release targets:"
	@echo "  make package            - Create release packages (tar.gz)"
	@echo "  make checksums          - Generate SHA256 checksums"
	@echo "  make release            - Full release build (all platforms + packages)"
	@echo ""
	@echo "Example build targets:"
	@echo "  make build-examples              - Build all examples"
	@echo "  make build-example-basic-demo    - Build basic-demo example"
	@echo "  make build-example-basic-tool    - Build basic-tool example"
	@echo "  make build-example-client        - Build sage-client example"
	@echo "  make build-example-simple        - Build simple-standalone example"
	@echo "  make build-example-secure-chat   - Build secure-chat example"
	@echo "  make build-example-vulnerable-chat - Build vulnerable-chat example"
	@echo "  make build-example-attacker      - Build attacker example"
	@echo ""
	@echo "Run example targets:"
	@echo "  make run-example-basic-demo      - Run basic-demo example"
	@echo "  make run-example-basic-tool      - Run basic-tool example"
	@echo "  make run-example-client          - Run sage-client example"
	@echo "  make run-example-simple          - Run simple-standalone example"
	@echo "  make run-example-secure-chat     - Run secure-chat example"
	@echo "  make run-example-vulnerable-chat - Run vulnerable-chat example"
	@echo "  make run-example-attacker        - Run attacker example"
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
	@echo "E2E test targets:"
	@echo "  make test-e2e              - Run all E2E tests"
	@echo "  make test-e2e-sepolia      - Run Sepolia E2E tests only"
	@echo "  make test-e2e-local        - Run local E2E tests (no blockchain)"
	@echo "  make test-e2e-coverage     - Run E2E tests with coverage report"
	@echo ""
	@echo "Benchmark targets:"
	@echo "  make bench            - Run all benchmarks"
	@echo "  make bench-integration - Run integration benchmarks"
	@echo ""
	@echo "Random Test targets:"
	@echo ""
	@echo "Verification targets:"
	@echo "  make verify-features         - Run comprehensive feature verification"
	@echo "  make verify-quick            - Run quick feature verification"
	@echo "  make test-full               - Run full test suite with all checks"
	@echo "  make verify-makefile         - Verify Makefile consistency"
	@echo "  make verify-rfc9421-ed25519  - Verify RFC 9421 Ed25519 implementation"
	@echo ""
	@echo "Advanced test targets:"
	@echo "  make fuzz         - Run fuzz tests"
	@echo "  make loadtest     - Run load tests"
	@echo "  make bench-full   - Run comprehensive benchmarks"
	@echo ""
	@echo "Database management targets:"
	@echo "  make db-backup       - Backup database"
	@echo "  make db-restore      - Restore database from backup"
	@echo "  make db-seed         - Seed database with test data"
	@echo "  make db-migrate-up   - Run database migrations (up)"
	@echo "  make db-migrate-down - Roll back database migrations"
	@echo ""
	@echo "Docker targets:"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run Docker container"
	@echo ""
	@echo "Utility targets:"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make clean-all      - Remove all build artifacts and reports"
	@echo "  make clean-reports  - Remove test reports only"
	@echo "  make test-cleanup   - Cleanup test environment"
	@echo "  make install        - Install binaries to GOPATH/bin"
	@echo "  make lint           - Run linter"
	@echo "  make lint-ci        - Run CI lint checks (same as GitHub Actions)"
	@echo "  make fmt            - Format code"
	@echo "  make tidy           - Run go mod tidy"
	@echo "  make update-version - Update project version"
	@echo "  make help           - Show this help message"
