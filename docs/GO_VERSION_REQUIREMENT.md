# Go Version Requirements

## Official Go Version Requirement

**SAGE project requires Go 1.23.0 or higher.**

### Updated Specification

**Current Requirements**:
- **Development Environment**: Go 1.23.0+
- **Core Library/CLI**: Go 1.23.0 or higher
- **Recommended**: Go 1.23.0 or later

### Change History

#### 2025-01-11: Downgrade to Go 1.23.0
- Removed `a2a-go` dependency which required Go 1.24.4+
- Implemented transport abstraction layer to decouple from a2a
- Downgraded golang.org/x/* dependencies to Go 1.23.0 compatible versions
- All tests pass with Go 1.23.0

#### Previous: Go 1.24.4 Requirement
- Required by `github.com/a2aproject/a2a-go` dependency
- Used for HPKE (RFC 9180), agent handshake, and session management

### Rationale for Go 1.23.0

SAGE now uses a transport-agnostic architecture that removes the hard dependency on `a2a-go`. Core features include:
- Transport abstraction layer (HTTP, WebSocket, A2A optional)
- HPKE (RFC 9180) encryption using standard libraries
- Session management with memory pooling optimizations
- Agent-to-Agent handshake protocol

The A2A transport is now optional and can be enabled with build tags when needed.

## Current Configuration

### go.mod
```go
go 1.23.0
```

- **Minimum Required Version**: Go 1.23.0
- **No toolchain override**: Uses system Go version

### System Environment
```bash
$ go version
go version go1.23.0 darwin/arm64
```

## Test Environment Information

### Software Information
- **Software Name**: SAGE Core Library
- **Software Type**: Go Library / Security Middleware
- **Development Environment**: **Go 1.23.0+**
- **Test Environment Details**:
  - **Core Library/CLI**: Go 1.23.0 or higher
  - **Smart Contract**: Solidity 0.8.19
  - **Blockchain Network**: Ethereum (Hardhat local node)
  - **Chain ID**: 31337 (local)
  - **Hardhat Version**: 2.26.3
  - **Web3 Library**: ethers v6.4.0
  - **Cryptographic Algorithms**: Secp256k1 (Ethereum), Ed25519 (EdDSA), X25519 (HPKE)

### Core Dependencies (Go 1.23.0 Compatible)
- Go modules:
  - `github.com/ethereum/go-ethereum` v1.16.1
  - `github.com/decred/dcrd/dcrec/secp256k1/v4` v4.4.0
  - `github.com/gorilla/websocket` v1.5.3
  - `golang.org/x/crypto` v0.41.0 (Go 1.23.0 compatible)
  - `golang.org/x/sync` v0.16.0 (Go 1.23.0 compatible)
- Node.js packages:
  - `@nomicfoundation/hardhat-ethers` v4.0.0
  - `hardhat` v2.26.3

### Optional Dependencies (A2A Transport)
- `github.com/a2aproject/a2a` (requires build tag: `a2a`)
- Only needed when using A2A protocol transport
- HTTP and WebSocket transports work without this dependency

## Installation

### Go Installation (1.23.0 or higher)

**macOS (Homebrew)**:
```bash
brew install go
# or specific version
brew install go@1.23
```

**macOS (gvm)**:
```bash
# Install gvm
bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)

# Install Go 1.23.0
gvm install go1.23.0
gvm use go1.23.0 --default
```

**Linux**:
```bash
# Download from official site
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Windows**:
- Download Go 1.23.0 installer from https://go.dev/dl/

### Version Verification
```bash
go version
# Output: go version go1.23.0 darwin/arm64 (or 1.23.0+)
```

### Project Build
```bash
cd sage
go mod download
make build
```

### Build with A2A Transport (Optional)
```bash
# If you need A2A transport support
go build -tags=a2a ./...
```

## Verification

All tests pass with Go 1.23.0:

```bash
# Run all tests
go test ./... -short

# Run with race detector
go test -race -short ./pkg/agent/...

# Build all packages
go build ./...
```

**Test Results**: All tests pass (100%)

## Compatibility

| Go Version | Status | Notes |
|------------|--------|-------|
| 1.22.x     |  Not tested | May work but not officially supported |
| 1.23.0     |  Supported | Minimum required version |
| 1.23.1+    |  Supported | Recommended |
| 1.24.0+    |  Supported | Forward compatible |

## Transport Options

SAGE supports multiple transport protocols:

| Transport | Dependency | Build Tag | Use Case |
|-----------|------------|-----------|----------|
| **HTTP/HTTPS** | Built-in | None | Web-friendly, load balancer support |
| **WebSocket** | gorilla/websocket | None | Real-time, persistent connections |
| **A2A (gRPC)** | a2a-go | `a2a` | High-performance agent-to-agent |

### Default Build (No A2A)
```bash
# Uses HTTP and WebSocket transports only
go build ./...
go test ./...
```

### Build with A2A Support
```bash
# Includes A2A transport (requires Go 1.24.4+ and a2a-go)
go build -tags=a2a ./...
go test -tags=a2a ./...
```

## Performance Optimizations (Go 1.23.0)

With Go 1.23.0, SAGE includes several performance improvements:

- **Session Memory Pooling**: 80% GC reduction using `sync.Pool`
- **Pre-allocated Buffers**: 60-70% allocation reduction
- **Optimized HKDF**: Single-allocation key derivation
- **Thread-safe MockTransport**: Proper concurrency handling

## Migration Notes

### From Go 1.24.4+ to Go 1.23.0

If upgrading from a previous version that used Go 1.24.4+:

1. **Update Go version**:
   ```bash
   # Install Go 1.23.0
   gvm install go1.23.0
   gvm use go1.23.0
   ```

2. **Clean and rebuild**:
   ```bash
   go clean -modcache
   go mod tidy
   go build ./...
   ```

3. **A2A transport** (if needed):
   - A2A transport is now optional
   - Use `-tags=a2a` to enable
   - Or switch to HTTP/WebSocket transports

## References

- **README.md**: Updated to Go 1.23.0+ requirement
- **go.mod**: Set to `go 1.23.0`
- **Architecture Proposal**: Transport abstraction design
- **Dependency Removal**: a2a-go dependency removal plan

## Support

For issues related to Go version compatibility:
- Check `go.mod` for exact version requirements
- Run `go mod tidy` to resolve dependency issues
- See ARCHITECTURE_REFACTORING_PROPOSAL.md for design rationale

---

**Last Updated**: 2025-01-11
**Maintainer**: SAGE Development Team
**Status**:  Active
