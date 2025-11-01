# Sage Project Makefile

# Variables
CRYPTO_BINARY=sage-crypto
DID_BINARY=sage-did
RANDOM_TEST_BINARY=random-test
VERIFY_BINARY=deployment-verify
TEST_CLIENT_BINARY=test-client
TEST_SERVER_BINARY=test-server
BUILD_DIR=build
BIN_DIR=$(BUILD_DIR)/bin
LIB_DIR=$(BUILD_DIR)/lib
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
BUILD_DATE?=$(shell date -u '+%Y-%m-%d %H:%M:%S UTC')

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
LIB_NAME=libsage.a
LIB_SO_NAME=libsage.so
LIB_DYLIB_NAME=libsage.dylib
LIB_DLL_NAME=libsage.dll

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

# Build libraries
.PHONY: build-lib
build-lib: build-lib-static build-lib-shared

# Build static library (.a) for current platform
.PHONY: build-lib-static
build-lib-static: $(LIB_DIR)/$(LIB_NAME)

$(LIB_DIR)/$(LIB_NAME):
	@echo "Building static library $(LIB_NAME) for current platform..."
	@mkdir -p $(LIB_DIR)
	$(GO) build -buildmode=c-archive -o $(LIB_DIR)/$(LIB_NAME) ./lib
	@echo "Build complete: $(LIB_DIR)/$(LIB_NAME)"

# Build shared library for current platform
.PHONY: build-lib-shared
build-lib-shared:
	@echo "Building shared library for current platform..."
	@mkdir -p $(LIB_DIR)
ifeq ($(UNAME_S),Darwin)
	@echo "Building macOS dylib..."
	$(GO) build -buildmode=c-shared -o $(LIB_DIR)/$(LIB_DYLIB_NAME) ./lib
	@echo "Build complete: $(LIB_DIR)/$(LIB_DYLIB_NAME)"
else ifeq ($(UNAME_S),Linux)
	@echo "Building Linux shared library..."
	$(GO) build -buildmode=c-shared -o $(LIB_DIR)/$(LIB_SO_NAME) ./lib
	@echo "Build complete: $(LIB_DIR)/$(LIB_SO_NAME)"
else
	@echo "Windows DLL build not supported directly from Makefile. Use build-lib-all-platforms instead."
endif

# Build libraries for all platforms and architectures
.PHONY: build-lib-all
build-lib-all:
	@echo "Building libraries for all platforms and architectures..."
	@echo "Note: Cross-platform library builds require platform-specific C toolchains."
	@echo "Some builds may fail if cross-compilation toolchains are not installed."
	@echo ""
	@$(MAKE) build-lib-linux-amd64 || echo "Warning: Linux amd64 build failed (may need cross-compiler)"
	@$(MAKE) build-lib-linux-arm64 || echo "Warning: Linux arm64 build failed (may need cross-compiler)"
	@$(MAKE) build-lib-darwin-amd64 || echo "Warning: macOS amd64 build failed (may need cross-compiler)"
	@$(MAKE) build-lib-darwin-arm64 || echo "Warning: macOS arm64 build failed (may need cross-compiler)"
	@$(MAKE) build-lib-windows-amd64 || echo "Warning: Windows amd64 build failed (may need cross-compiler)"
	@echo ""
	@echo "Library builds complete! (check for warnings above)"

# Build Linux static library (amd64)
.PHONY: build-lib-linux-amd64
build-lib-linux-amd64:
	@echo "Building Linux amd64 static library..."
	@mkdir -p $(LIB_DIR)/linux-amd64
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -buildmode=c-archive \
		-o $(LIB_DIR)/linux-amd64/libsage.a ./lib
	@echo "Build complete: $(LIB_DIR)/linux-amd64/libsage.a"

# Build Linux static library (arm64)
.PHONY: build-lib-linux-arm64
build-lib-linux-arm64:
	@echo "Building Linux arm64 static library..."
	@mkdir -p $(LIB_DIR)/linux-arm64
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(GO) build -buildmode=c-archive \
		-o $(LIB_DIR)/linux-arm64/libsage.a ./lib
	@echo "Build complete: $(LIB_DIR)/linux-arm64/libsage.a"

# Build macOS static library (amd64)
.PHONY: build-lib-darwin-amd64
build-lib-darwin-amd64:
	@echo "Building macOS amd64 static library..."
	@mkdir -p $(LIB_DIR)/darwin-amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 $(GO) build -buildmode=c-archive \
		-o $(LIB_DIR)/darwin-amd64/libsage.a ./lib
	@echo "Build complete: $(LIB_DIR)/darwin-amd64/libsage.a"

# Build macOS static library (arm64/Apple Silicon)
.PHONY: build-lib-darwin-arm64
build-lib-darwin-arm64:
	@echo "Building macOS arm64 static library..."
	@mkdir -p $(LIB_DIR)/darwin-arm64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 $(GO) build -buildmode=c-archive \
		-o $(LIB_DIR)/darwin-arm64/libsage.a ./lib
	@echo "Build complete: $(LIB_DIR)/darwin-arm64/libsage.a"

# Build Windows static library (amd64)
.PHONY: build-lib-windows-amd64
build-lib-windows-amd64:
	@echo "Building Windows amd64 static library..."
	@mkdir -p $(LIB_DIR)/windows-amd64
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GO) build -buildmode=c-archive \
		-o $(LIB_DIR)/windows-amd64/libsage.a ./lib
	@echo "Build complete: $(LIB_DIR)/windows-amd64/libsage.a"

# Build Linux shared library (amd64)
.PHONY: build-lib-linux-amd64-shared
build-lib-linux-amd64-shared:
	@echo "Building Linux amd64 shared library..."
	@mkdir -p $(LIB_DIR)/linux-amd64
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -buildmode=c-shared \
		-o $(LIB_DIR)/linux-amd64/libsage.so ./lib
	@echo "Build complete: $(LIB_DIR)/linux-amd64/libsage.so"

# Build Linux shared library (arm64)
.PHONY: build-lib-linux-arm64-shared
build-lib-linux-arm64-shared:
	@echo "Building Linux arm64 shared library..."
	@mkdir -p $(LIB_DIR)/linux-arm64
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(GO) build -buildmode=c-shared \
		-o $(LIB_DIR)/linux-arm64/libsage.so ./lib
	@echo "Build complete: $(LIB_DIR)/linux-arm64/libsage.so"

# Build macOS shared library (amd64)
.PHONY: build-lib-darwin-amd64-shared
build-lib-darwin-amd64-shared:
	@echo "Building macOS amd64 shared library..."
	@mkdir -p $(LIB_DIR)/darwin-amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 $(GO) build -buildmode=c-shared \
		-o $(LIB_DIR)/darwin-amd64/libsage.dylib ./lib
	@echo "Build complete: $(LIB_DIR)/darwin-amd64/libsage.dylib"

# Build macOS shared library (arm64/Apple Silicon)
.PHONY: build-lib-darwin-arm64-shared
build-lib-darwin-arm64-shared:
	@echo "Building macOS arm64 shared library..."
	@mkdir -p $(LIB_DIR)/darwin-arm64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 $(GO) build -buildmode=c-shared \
		-o $(LIB_DIR)/darwin-arm64/libsage.dylib ./lib
	@echo "Build complete: $(LIB_DIR)/darwin-arm64/libsage.dylib"

# Build Windows shared library (amd64)
.PHONY: build-lib-windows-amd64-shared
build-lib-windows-amd64-shared:
	@echo "Building Windows amd64 DLL..."
	@mkdir -p $(LIB_DIR)/windows-amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc $(GO) build -buildmode=c-shared \
		-o $(LIB_DIR)/windows-amd64/libsage.dll ./lib
	@echo "Build complete: $(LIB_DIR)/windows-amd64/libsage.dll"

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
	@GOTOOLCHAIN=$(GOTOOLCHAIN) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) \
		-ldflags "$(LDFLAGS) $(BUILD_LDFLAGS)" \
		-o $(DIST_DIR)/$(GOOS)-$(GOARCH)/$(CRYPTO_BINARY)$(if $(filter windows,$(GOOS)),.exe,) \
		./$(CMD_DIR)/$(CRYPTO_BINARY)
	@GOTOOLCHAIN=$(GOTOOLCHAIN) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) \
		-ldflags "$(LDFLAGS) $(BUILD_LDFLAGS)" \
		-o $(DIST_DIR)/$(GOOS)-$(GOARCH)/$(DID_BINARY)$(if $(filter windows,$(GOOS)),.exe,) \
		./$(CMD_DIR)/$(DID_BINARY)
	@GOTOOLCHAIN=$(GOTOOLCHAIN) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) \
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
	@rm -f sage-crypto sage-did deployment-verify random-test
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
	@rm -rf contracts/solana/target/
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
	@rm -rf random/
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

# Build random-test binary
.PHONY: build-random-test
build-random-test: $(BIN_DIR)/$(RANDOM_TEST_BINARY)

$(BIN_DIR)/$(RANDOM_TEST_BINARY):
	@echo "Building $(RANDOM_TEST_BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(RANDOM_TEST_BINARY) ./$(CMD_DIR)/$(RANDOM_TEST_BINARY)
	@echo "Build complete: $(BIN_DIR)/$(RANDOM_TEST_BINARY)"

# Run random tests with default settings (100 iterations)
.PHONY: random-test
random-test: build-random-test
	@echo "Running random tests (100 iterations)..."
	@mkdir -p $(REPORTS_DIR)
	$(BIN_DIR)/$(RANDOM_TEST_BINARY) -iterations=100 -parallel=4

# Run quick random tests (10 iterations for validation)
.PHONY: random-test-quick
random-test-quick: build-random-test
	@echo "Running quick random tests (10 iterations)..."
	@mkdir -p $(REPORTS_DIR)
	$(BIN_DIR)/$(RANDOM_TEST_BINARY) -iterations=10 -parallel=2 -verbose

# Run full random tests (1000 iterations for evaluation)
.PHONY: random-test-full
random-test-full: build-random-test
	@echo "Running full random tests (1000 iterations)..."
	@mkdir -p $(REPORTS_DIR)
	$(BIN_DIR)/$(RANDOM_TEST_BINARY) -iterations=1000 -parallel=10 -report=$(REPORTS_DIR)/random-test-full.html

# Run evaluation random tests (10000 iterations for maximum score)
.PHONY: random-test-eval
random-test-eval: build-random-test
	@echo "Running evaluation random tests (10000 iterations)..."
	@mkdir -p $(REPORTS_DIR)
	$(BIN_DIR)/$(RANDOM_TEST_BINARY) -iterations=10000 -parallel=20 -report=$(REPORTS_DIR)/random-test-evaluation.html

# Run random tests for specific category
.PHONY: random-test-rfc9421
random-test-rfc9421: build-random-test
	@echo "Running RFC 9421 random tests..."
	@mkdir -p $(REPORTS_DIR)
	$(BIN_DIR)/$(RANDOM_TEST_BINARY) -iterations=500 -categories=rfc9421 -parallel=5

.PHONY: random-test-crypto
random-test-crypto: build-random-test
	@echo "Running crypto random tests..."
	@mkdir -p $(REPORTS_DIR)
	$(BIN_DIR)/$(RANDOM_TEST_BINARY) -iterations=500 -categories=crypto -parallel=5

.PHONY: random-test-did
random-test-did: build-random-test
	@echo "Running DID random tests..."
	@mkdir -p $(REPORTS_DIR)
	$(BIN_DIR)/$(RANDOM_TEST_BINARY) -iterations=500 -categories=did -parallel=5

# Clean test reports
.PHONY: clean-reports
clean-reports:
	@echo "Cleaning test reports..."
	rm -rf $(REPORTS_DIR)

# Create release packages for all platforms
.PHONY: package
package: build-all-platforms build-lib-all
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

# Full release build (binaries + libraries + packages + checksums)
.PHONY: release
release: clean build-all-platforms build-lib-all package checksums
	@echo "===================="
	@echo "Release build complete!"
	@echo "===================="
	@echo ""
	@echo "Binaries:"
	@find $(DIST_DIR) -type f \( -name "sage-*" -o -name "*.exe" \) -exec ls -lh {} \;
	@echo ""
	@echo "Libraries:"
	@find $(LIB_DIR) -type f \( -name "*.a" -o -name "*.so" -o -name "*.dylib" -o -name "*.dll" \) -exec ls -lh {} \;
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
	@echo "  make build-lib-all      - Build libraries for all platforms"
	@echo "  make release            - Full release build with packages"
	@echo ""
	@echo "Build targets:"
	@echo "  make build              - Build all binaries and examples"
	@echo "  make build-binaries     - Build all CLI binaries"
	@echo "  make build-crypto       - Build sage-crypto binary only"
	@echo "  make build-did          - Build sage-did binary only"
	@echo "  make build-verify       - Build deployment-verify binary only"
	@echo ""
	@echo "Cross-platform build targets:"
	@echo "  make build-all-platforms         - Build binaries for all platforms"
	@echo "  make build-platform GOOS=linux GOARCH=amd64  - Build for specific platform"
	@echo ""
	@echo "Library build targets:"
	@echo "  make build-lib                   - Build library for current platform"
	@echo "  make build-lib-static            - Build static library (.a)"
	@echo "  make build-lib-shared            - Build shared library (.so/.dylib)"
	@echo "  make build-lib-all               - Build libraries for all platforms"
	@echo ""
	@echo "Platform-specific library builds:"
	@echo "  make build-lib-linux-amd64       - Linux x86_64 static library"
	@echo "  make build-lib-linux-arm64       - Linux ARM64 static library"
	@echo "  make build-lib-darwin-amd64      - macOS Intel static library"
	@echo "  make build-lib-darwin-arm64      - macOS Apple Silicon static library"
	@echo "  make build-lib-windows-amd64     - Windows x86_64 static library"
	@echo "  make build-lib-linux-amd64-shared   - Linux x86_64 shared library (.so)"
	@echo "  make build-lib-linux-arm64-shared   - Linux ARM64 shared library (.so)"
	@echo "  make build-lib-darwin-amd64-shared  - macOS Intel shared library (.dylib)"
	@echo "  make build-lib-darwin-arm64-shared  - macOS Apple Silicon shared library (.dylib)"
	@echo "  make build-lib-windows-amd64-shared - Windows x86_64 DLL (requires MinGW)"
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
	@echo "  make random-test         - Run random tests (100 iterations)"
	@echo "  make random-test-quick   - Run quick validation (10 iterations)"
	@echo "  make random-test-full    - Run full tests (1000 iterations)"
	@echo "  make random-test-eval    - Run evaluation tests (10000 iterations)"
	@echo "  make random-test-rfc9421 - Test RFC 9421 only"
	@echo "  make random-test-crypto  - Test crypto only"
	@echo "  make random-test-did     - Test DID only"
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
