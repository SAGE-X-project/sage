# SAGE Testing Guide

**Last Updated**: 2025-10-10
**Version**: 1.0

This document provides comprehensive testing guidelines for the SAGE project, including unit tests, integration tests, and end-to-end testing strategies.

---

## Table of Contents

1. [Test Environment Setup](#test-environment-setup)
2. [Running Tests](#running-tests)
3. [Test Categories](#test-categories)
4. [Writing Tests](#writing-tests)
5. [CI/CD Integration](#cicd-integration)
6. [Troubleshooting](#troubleshooting)

---

## Test Environment Setup

### Automated Setup

SAGE provides automated scripts to set up the complete test environment including Ethereum and Solana local nodes.

#### Quick Start

```bash
# Setup test environment (Ethereum + Solana + Redis)
./tools/scripts/setup_test_env.sh

# With PostgreSQL database
./tools/scripts/setup_test_env.sh --with-db

# Skip cleanup of existing environment
./tools/scripts/setup_test_env.sh --skip-cleanup
```

#### Cleanup

```bash
# Stop and remove containers
./tools/scripts/cleanup_test_env.sh

# Remove containers and data volumes
./tools/scripts/cleanup_test_env.sh --remove-volumes

# Force cleanup without confirmation
./tools/scripts/cleanup_test_env.sh -v -f
```

### Manual Setup

If you prefer manual setup:

```bash
# Start test environment
docker-compose -f deployments/docker/test-environment.yml up -d

# Check service health
docker-compose -f deployments/docker/test-environment.yml ps

# View logs
docker-compose -f deployments/docker/test-environment.yml logs -f

# Stop services
docker-compose -f deployments/docker/test-environment.yml down
```

### Test Services

Once started, the following services are available:

| Service | URL | Purpose |
|---------|-----|---------|
| Ethereum RPC | http://localhost:8545 | Local Hardhat node for Ethereum testing |
| Solana RPC | http://localhost:8899 | Local Solana test validator |
| Redis | localhost:6380 | Session cache for integration tests |
| PostgreSQL* | localhost:5433 | DID registry database (optional) |

*PostgreSQL requires `--with-db` flag

### Environment Variables

Test environment variables are automatically exported to `.env.test`:

```bash
# Source test environment
source .env.test

# Or use in your tests
export $(cat .env.test | xargs)
```

---

## Running Tests

### Unit Tests

```bash
# Run all Go unit tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test ./pkg/agent/hpke/... -v

# Run with race detector
go test -race ./...
```

### Integration Tests

```bash
# Setup test environment first
./tools/scripts/setup_test_env.sh

# Run integration tests
make test-integration

# Run specific integration test
go test ./tests/integration/tests/session/... -v
```

### Component Tests

```bash
# HPKE tests
make test-hpke

# Handshake tests
make test-handshake

# Crypto tests
make test-crypto

# DID tests
make test-did
```

### Smart Contract Tests

```bash
# Ethereum contracts
cd contracts/ethereum
npm test

# Solana contracts
cd contracts/solana
cargo test
```

### End-to-End Tests

```bash
# Full test suite
./tools/scripts/full-test.sh

# Quick test (unit + integration)
./tools/scripts/quick-test.sh
```

---

## Test Categories

### 1. Unit Tests

**Location**: `pkg/*/` (alongside source files)

**Pattern**: `*_test.go`

**Purpose**: Test individual functions and methods in isolation

**Example**:
```go
func TestHPKEEncryption(t *testing.T) {
    // Test HPKE encryption/decryption
}
```

### 2. Integration Tests

**Location**: `tests/integration/tests/`

**Purpose**: Test interaction between components

**Examples**:
- Session management with Redis
- HPKE handshake flow
- DID resolution with blockchain

### 3. Contract Tests

**Location**:
- `contracts/ethereum/test/`
- `contracts/solana/tests/`

**Purpose**: Test smart contract logic

### 4. E2E Tests

**Location**: `test/e2e/`

**Purpose**: Test complete user workflows

---

## Writing Tests

### Test Structure

Follow the Arrange-Act-Assert (AAA) pattern:

```go
func TestSessionCreation(t *testing.T) {
    // Arrange
    mgr := session.NewManager()
    exporter := make([]byte, 32)

    // Act
    sess, sid, keyID, err := mgr.EnsureSessionFromExporterWithRole(
        exporter,
        "test-context",
        true,
        nil,
    )

    // Assert
    require.NoError(t, err)
    require.NotNil(t, sess)
    require.NotEmpty(t, sid)
    require.NotEmpty(t, keyID)
}
```

### Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
func TestKeyTypeValidation(t *testing.T) {
    tests := []struct {
        name    string
        key     interface{}
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid ed25519 key",
            key:     ed25519.PublicKey(make([]byte, 32)),
            wantErr: false,
        },
        {
            name:    "invalid key type",
            key:     "not a key",
            wantErr: true,
            errMsg:  "expected ed25519.PublicKey",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateKey(tt.key)
            if tt.wantErr {
                require.Error(t, err)
                require.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Mock Dependencies

Use interfaces for dependency injection:

```go
type MockResolver struct {
    mock.Mock
}

func (m *MockResolver) ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error) {
    args := m.Called(ctx, did)
    return args.Get(0), args.Error(1)
}

func TestWithMockResolver(t *testing.T) {
    resolver := new(MockResolver)
    resolver.On("ResolvePublicKey", mock.Anything, mock.Anything).
        Return(ed25519.PublicKey(make([]byte, 32)), nil)

    // Use resolver in test
}
```

### Integration Test Setup

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Check test environment
    ethRPC := os.Getenv("ETHEREUM_RPC_URL")
    if ethRPC == "" {
        t.Fatal("ETHEREUM_RPC_URL not set. Run setup_test_env.sh")
    }

    // Run integration test
}
```

---

## Test Best Practices

### 1. Test Independence

- Each test should be independent
- No shared state between tests
- Use `t.Cleanup()` for teardown

```go
func TestWithCleanup(t *testing.T) {
    tempFile := createTempFile()
    t.Cleanup(func() {
        os.Remove(tempFile)
    })

    // Test using tempFile
}
```

### 2. Error Testing

Always test both success and failure cases:

```go
func TestErrorCases(t *testing.T) {
    // Test nil input
    _, err := ProcessData(nil)
    require.Error(t, err)

    // Test invalid format
    _, err = ProcessData([]byte("invalid"))
    require.Error(t, err)

    // Test valid input
    result, err := ProcessData(validData)
    require.NoError(t, err)
    require.NotNil(t, result)
}
```

### 3. Timeout Handling

Use context with timeout for async operations:

```go
func TestAsyncOperation(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := AsyncFunction(ctx)
    require.NoError(t, err)
    require.NotNil(t, result)
}
```

### 4. Race Condition Testing

```bash
# Run tests with race detector
go test -race ./...
```

---

## Coverage

### Generate Coverage Report

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Goals

- **Unit Tests**: 80%+ coverage
- **Integration Tests**: Cover critical paths
- **Contract Tests**: 100% coverage for security-critical code

---

## CI/CD Integration

### GitHub Actions

Tests run automatically on:
- Pull requests
- Push to main/dev branches
- Scheduled daily runs

### Test Matrix

```yaml
strategy:
  matrix:
    go-version: [1.21, 1.22]
    os: [ubuntu-latest, macos-latest]
```

### Required Checks

Before merge, all PRs must pass:
- [ ] Unit tests
- [ ] Integration tests
- [ ] Contract tests
- [ ] Linter checks
- [ ] Security scans

---

## Troubleshooting

### Common Issues

#### 1. Test Environment Not Ready

**Error**: `connection refused` or `service not available`

**Solution**:
```bash
# Verify services are running
docker-compose -f deployments/docker/test-environment.yml ps

# Check service logs
docker-compose -f deployments/docker/test-environment.yml logs ethereum-node
docker-compose -f deployments/docker/test-environment.yml logs solana-node

# Restart services
./tools/scripts/cleanup_test_env.sh
./tools/scripts/setup_test_env.sh
```

#### 2. Port Conflicts

**Error**: `port already in use`

**Solution**:
```bash
# Check what's using the port
lsof -i :8545

# Kill the process or use different port
# Edit deployments/docker/test-environment.yml
```

#### 3. Docker Volumes Full

**Error**: `no space left on device`

**Solution**:
```bash
# Clean up test volumes
./tools/scripts/cleanup_test_env.sh --remove-volumes

# Prune Docker system
docker system prune -a --volumes
```

#### 4. Test Flakiness

**Issue**: Tests pass/fail inconsistently

**Solutions**:
- Increase timeouts
- Add retry logic for network operations
- Check for race conditions
- Ensure proper cleanup between tests

#### 5. Contract Deployment Fails

**Error**: `insufficient funds` or `nonce too low`

**Solution**:
```bash
# Reset blockchain state
./tools/scripts/cleanup_test_env.sh -v
./tools/scripts/setup_test_env.sh

# Use fresh accounts
```

---

## Performance Testing

### Benchmarks

```bash
# Run benchmarks
make bench

# Run specific benchmark
go test -bench=BenchmarkHPKE ./pkg/agent/hpke/...

# With memory profiling
go test -bench=. -benchmem ./...
```

### Load Testing

```bash
# Run load tests
./tools/scripts/run-loadtest.sh

# Custom configuration
./tools/scripts/run-loadtest.sh --users 100 --duration 60s
```

---

## Security Testing

### Fuzzing

```bash
# Run fuzz tests
./tools/scripts/run-fuzz.sh

# Fuzz specific function
go test -fuzz=FuzzHPKE ./pkg/agent/hpke/...
```

### Static Analysis

```bash
# Run gosec
gosec ./...

# Run staticcheck
staticcheck ./...
```

---

## Test Data

### Fixtures

Test fixtures are located in:
- `test/fixtures/` - General test data
- `tests/integration/fixtures/` - Integration test data
- `pkg/*/testdata/` - Package-specific test data

### Generating Test Data

```bash
# Generate test keys
go run tools/keygen/main.go --output test/fixtures/keys

# Generate test DIDs
go run tools/didgen/main.go --count 10 --output test/fixtures/dids
```

---

## Documentation

### Test Documentation

Each test package should have:
- `README.md` - Test overview
- Inline comments explaining complex test logic
- Examples for common testing patterns

### Updating This Guide

When adding new test infrastructure:
1. Update this document
2. Add examples
3. Update CI/CD configuration
4. Notify team

---

## Quick Reference

### Essential Commands

```bash
# Setup
./tools/scripts/setup_test_env.sh

# Run all tests
make test-all

# Run with coverage
make test-coverage

# Cleanup
./tools/scripts/cleanup_test_env.sh -v

# Integration tests only
make test-integration

# Benchmarks
make bench
```

### Environment Variables

```bash
# Test mode
export SAGE_ENV=test

# Verbose logging
export LOG_LEVEL=debug

# Skip slow tests
export SKIP_SLOW_TESTS=1

# Custom RPC URLs
export ETHEREUM_RPC_URL=http://localhost:8545
export SOLANA_RPC_URL=http://localhost:8899
```

---

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Library](https://github.com/stretchr/testify)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Hardhat Testing Guide](https://hardhat.org/tutorial/testing-contracts)
- [Solana Testing Guide](https://docs.solana.com/developing/test-validator)

---

**Document History**:
- 2025-10-10: v1.0 - Initial version with automated test environment setup
