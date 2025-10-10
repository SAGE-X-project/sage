# SAGE Architecture

This document provides a comprehensive overview of the SAGE (Secure Agent Guarantee Engine) architecture, component design, and data flow patterns.

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Principles](#architecture-principles)
3. [Component Architecture](#component-architecture)
4. [Data Flow](#data-flow)
5. [Security Architecture](#security-architecture)
6. [Smart Contract Architecture](#smart-contract-architecture)
7. [Deployment Architecture](#deployment-architecture)

## System Overview

SAGE is a blockchain-based security framework designed for secure AI agent communication. It combines cryptographic protocols (HPKE, RFC 9421), decentralized identity (DID), and smart contract registries to provide end-to-end encrypted, authenticated messaging.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     SAGE System Architecture                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────┐         ┌────────────┐         ┌────────────┐  │
│  │  Agent A   │ ◄─────► │    SAGE    │ ◄─────► │  Agent B   │  │
│  │            │  HPKE   │   Core     │  HPKE   │            │  │
│  └────────────┘         └─────┬──────┘         └────────────┘  │
│                               │                                 │
│                               ▼                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │           Blockchain DID Registry (Multi-Chain)          │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐         │  │
│  │  │  Ethereum  │  │   Solana   │  │    Kaia    │         │  │
│  │  │  Registry  │  │  Registry  │  │  Registry  │         │  │
│  │  └────────────┘  └────────────┘  └────────────┘         │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

- **SAGE Core**: Go-based backend with cryptographic operations, DID management, and session handling
- **Smart Contracts**: Solidity contracts for decentralized agent registry (Ethereum, Kaia)
- **Solana Program**: Rust-based on-chain registry for Solana network
- **CLI Tools**: Command-line interfaces for crypto operations, DID management, and verification

## Architecture Principles

### 1. Modularity
Each component is self-contained with clear interfaces, enabling independent development and testing.

### 2. Security by Design
- Defense in depth with multiple security layers
- Type-safe cryptographic operations
- Secure session management with automatic expiration

### 3. Multi-Chain Support
Abstracted blockchain interface allows seamless integration with multiple networks.

### 4. Standards Compliance
- **HPKE**: RFC 9180 (Hybrid Public Key Encryption)
- **HTTP Signatures**: RFC 9421 (HTTP Message Signatures)
- **Cryptography**: NIST-approved algorithms (Ed25519, X25519, ChaCha20-Poly1305)

## Component Architecture

### Core Backend (`pkg/agent/`)

#### 1. Cryptography Module (`crypto/`)

```
crypto/
├── keys/              # Key pair implementations
│   ├── ed25519.go     # Ed25519 signing (RFC 8032)
│   ├── secp256k1.go   # Secp256k1 for Ethereum compatibility
│   └── x25519.go      # X25519 key exchange (RFC 7748)
├── formats/           # Key serialization
│   ├── jwk.go         # JSON Web Key (RFC 7517)
│   └── pem.go         # PEM encoding (RFC 7468)
├── storage/           # Key storage backends
│   ├── file.go        # File-based encrypted storage
│   └── memory.go      # In-memory storage for testing
└── vault/             # OS keychain integration
    ├── darwin.go      # macOS Keychain
    ├── linux.go       # Linux Secret Service
    └── windows.go     # Windows Credential Manager
```

**Design Patterns:**
- Factory pattern for key generation
- Strategy pattern for storage backends
- Interface abstraction for algorithm flexibility

#### 2. DID Module (`did/`)

```
did/
├── manager.go         # Multi-chain DID coordinator
├── resolver.go        # DID document resolution with caching
├── ethereum/          # Ethereum DID client
│   ├── client.go      # Smart contract interaction
│   └── provider.go    # Enhanced RPC provider with retry logic
├── solana/            # Solana DID client
│   ├── client.go      # Solana RPC interaction
│   └── program.go     # On-chain program interface
└── types.go           # Shared DID types (AgentMetadata, etc.)
```

**Key Features:**
- Multi-chain resolver with automatic routing
- LRU caching for DID document lookups
- Retry logic with exponential backoff
- Parallel resolution for multiple chains

#### 3. HPKE Module (`hpke/`)

```
hpke/
├── client.go          # HPKE sender (encapsulation)
├── server.go          # HPKE receiver (decapsulation)
├── types.go           # HPKE context and info builders
└── utils.go           # Shared HPKE utilities
```

**HPKE Flow:**
```
Initiator                                    Responder
─────────                                    ─────────
1. Generate ephemeral X25519 key pair
2. Encapsulate with responder's public key
3. Derive shared secret (HKDF-SHA256)
4. Export session key
                    ──[enc, ciphertext]──►
                                            5. Decapsulate with private key
                                            6. Derive same shared secret
                                            7. Decrypt and verify
```

**Security Properties:**
- Forward secrecy with ephemeral keys
- Non-interactive key agreement
- Authenticated encryption (AEAD)

#### 4. Handshake Module (`handshake/`)

```
handshake/
├── client.go          # Handshake initiator
├── server.go          # Handshake responder
├── types.go           # Message types (Invitation, Request, Response, Complete)
└── utils.go           # Peer caching and lifecycle management
```

**Handshake Protocol Phases:**
```
Phase 1: Invitation
  - Server advertises service endpoint
  - Includes DID and public key info

Phase 2: Request
  - Client sends DID-signed request
  - Includes ephemeral public key for HPKE

Phase 3: Response
  - Server responds with signed acknowledgment
  - Includes server's ephemeral key

Phase 4: Complete
  - Both parties derive session key
  - Session is established and cached
```

#### 5. Session Module (`session/`)

```
session/
├── manager.go         # Session lifecycle management
├── session.go         # Secure session with AEAD encryption
├── nonce.go           # Replay protection with nonce cache
├── metadata.go        # Session state tracking
└── types.go           # Session configuration
```

**Session Management:**
- **Creation**: Derive session keys from HPKE shared secret
- **Encryption**: ChaCha20-Poly1305 AEAD for message confidentiality
- **Nonce Tracking**: 64-bit nonces with LRU cache for replay protection
- **Expiration**: Automatic cleanup of expired sessions
- **Key Rotation**: Time-based and message-count-based rotation

#### 6. RFC 9421 Module (`core/rfc9421/`)

```
core/rfc9421/
├── canonicalize.go    # HTTP message canonicalization
├── sign.go            # Signature generation
├── verify.go          # Signature verification
└── components.go      # Signature component extraction
```

**Supported Components:**
- `@method`, `@path`, `@query`, `@status`
- HTTP headers (case-insensitive)
- Request/response bodies
- Signature parameters (created, expires, nonce)

#### 7. Transport Layer (`transport/`)

The transport layer provides a protocol-agnostic abstraction for secure message transmission, decoupling security protocols (HPKE, handshake) from wire protocols (gRPC, HTTP, WebSocket).

```
transport/
├── interface.go       # MessageTransport interface definition
├── mock.go            # MockTransport for testing
├── selector.go        # Automatic transport selection by URL scheme
├── a2a/               # A2A/gRPC adapter (optional with build tag)
│   ├── client.go      # A2A client transport
│   ├── server.go      # A2A server adapter
│   └── register.go    # Auto-registration with selector
├── http/              # HTTP/REST transport
│   ├── client.go      # HTTP client transport
│   ├── server.go      # HTTP server adapter
│   ├── register.go    # Auto-registration with selector
│   └── README.md      # HTTP transport documentation
└── websocket/         # WebSocket transport
    ├── client.go      # WebSocket client transport
    ├── server.go      # WebSocket server adapter
    ├── register.go    # Auto-registration with selector
    └── README.md      # WebSocket transport documentation
```

**Core Interface:**

```go
type MessageTransport interface {
    Send(ctx context.Context, msg *SecureMessage) (*Response, error)
    Close() error
}

type SecureMessage struct {
    ID        string
    ContextID string
    TaskID    string
    Payload   []byte
    DID       string
    Signature []byte
    Metadata  map[string]string
    Role      string
}

type Response struct {
    Success   bool
    MessageID string
    TaskID    string
    Data      []byte
    Error     string
}
```

**Available Transports:**

1. **A2A/gRPC Transport** (`transport/a2a/`)
   - Wire protocol: gRPC with Protocol Buffers
   - Build tag: `//go:build a2a` (optional dependency)
   - Features: Bidirectional streaming, efficient binary protocol
   - Use case: High-performance agent-to-agent communication
   - Auto-registration: Registers for `grpc://` and `a2a://` URL schemes

2. **HTTP/REST Transport** (`transport/http/`)
   - Wire protocol: HTTP/1.1 or HTTP/2 with JSON
   - Features: Firewall-friendly, REST-compatible, simple integration
   - Use case: Web-based integrations, public APIs
   - Auto-registration: Registers for `http://` and `https://` URL schemes
   - Endpoint: `POST /messages` for message transmission

3. **WebSocket Transport** (`transport/websocket/`)
   - Wire protocol: WebSocket (RFC 6455) with JSON frames
   - Features: Persistent bidirectional connections, real-time communication
   - Use case: Interactive agents, streaming data, browser clients
   - Auto-registration: Registers for `ws://` and `wss://` URL schemes
   - Lifecycle: Automatic reconnection, heartbeat/ping-pong

4. **MockTransport** (`transport/mock.go`)
   - In-memory transport for unit testing
   - Thread-safe with mutex protection
   - Captures sent messages for verification
   - No network dependencies

**Transport Selector:**

Automatic transport selection based on URL scheme:

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/transport"
    _ "github.com/sage-x-project/sage/pkg/agent/transport/http"    // Auto-register HTTP
    _ "github.com/sage-x-project/sage/pkg/agent/transport/websocket" // Auto-register WS
)

// Automatic selection by URL
transport, err := transport.SelectByURL("https://agent.example.com")

// Manual selection
transport, err := transport.Select(transport.TransportHTTPS, "https://agent.example.com")

// Check available transports
types := transport.DefaultSelector.AvailableTransports()
```

**Design Principles:**

1. **Protocol Independence**: Security layers (HPKE, handshake, session) are decoupled from transport protocols
2. **Pluggable Architecture**: New transports can be added without modifying security code
3. **Auto-Registration**: Import-triggered registration using `init()` functions
4. **Build Tags**: Optional dependencies (A2A) use build tags to avoid forcing requirements
5. **Testability**: MockTransport enables fast, deterministic unit testing

**Transport Selection Guide:**

| Transport | Best For | Latency | Throughput | Firewall-Friendly | Complexity |
|-----------|----------|---------|------------|-------------------|------------|
| **A2A/gRPC** | Agent-to-agent, high performance | Low | High | Moderate | High |
| **HTTP** | REST APIs, web integrations | Moderate | Moderate | High | Low |
| **WebSocket** | Real-time, browser clients | Low | Moderate | High | Moderate |
| **Mock** | Unit testing, development | N/A | N/A | N/A | Very Low |

**Security Considerations:**

- All transports carry pre-encrypted payloads (HPKE + AEAD)
- Transport security (TLS) is independent of SAGE encryption
- DID signatures are verified at the handshake layer, not transport layer
- Each transport supports custom metadata via headers/fields

### Smart Contract Architecture

#### Ethereum Contracts (`contracts/ethereum/`)

```
contracts/ethereum/contracts/
├── SageRegistryV2.sol             # Main agent registry
├── ERC8004IdentityRegistry.sol    # Standalone identity registry
├── ERC8004ValidationRegistry.sol  # Validation contract
└── libraries/
    ├── SignatureValidator.sol     # Signature verification
    └── PublicKeyValidator.sol     # Key format validation
```

**Registry Contract Features:**
- Agent registration with Ed25519/Secp256k1 public keys
- Ownership verification with signature challenges
- Key revocation and rotation
- Event-driven architecture for off-chain indexing

**Security Enhancements (V2):**
- Reentrancy guards on state-changing functions
- Access control with owner-only operations
- Input validation for all public methods
- Gas-optimized storage patterns

#### Solana Program (`contracts/solana/`)

```
contracts/solana/programs/sage-registry/src/
├── lib.rs             # Program entry point
├── instruction.rs     # Instruction definitions
├── processor.rs       # Business logic
└── state.rs           # Account state structures
```

**Solana-Specific Considerations:**
- Account-based model with PDAs (Program Derived Addresses)
- Rent-exempt account sizing
- Cross-program invocation for composability

## Data Flow

### Complete HPKE Handshake Flow

```
┌──────────┐                                        ┌──────────┐
│ Agent A  │                                        │ Agent B  │
│(Initiator)│                                       │(Responder)│
└────┬─────┘                                        └────┬─────┘
     │                                                    │
     │ 1. Resolve Agent B's DID                          │
     ├─────────────► Blockchain Registry                 │
     │               (Get public key)                     │
     │◄────────────── Return: PublicKey_B                │
     │                                                    │
     │ 2. Generate ephemeral X25519 key pair             │
     │    (ephA_priv, ephA_pub)                          │
     │                                                    │
     │ 3. HPKE Encapsulate                               │
     │    shared_secret = HPKE-Encap(PublicKey_B,        │
     │                               ephA_pub, info)     │
     │                                                    │
     │ 4. Derive session key                             │
     │    session_key = HKDF(shared_secret, context)     │
     │                                                    │
     │ 5. Sign handshake request                         │
     │    signature = Sign_A(request || nonce)           │
     │                                                    │
     │ 6. Send handshake request                         │
     ├──────────────────────────────────────────────────►│
     │    {enc, ephA_pub, signature_A, DID_A}            │
     │                                                    │
     │                                   7. Resolve DID_A │
     │                          Blockchain ◄──────────────┤
     │                          (Verify signature)        │
     │                                                    │
     │                                8. HPKE Decapsulate │
     │                    shared_secret = HPKE-Decap(    │
     │                        PrivateKey_B, enc, info)   │
     │                                                    │
     │                                9. Derive session key│
     │                    (Same session_key as Agent A)  │
     │                                                    │
     │                              10. Generate response │
     │                    signature_B = Sign_B(response) │
     │                                                    │
     │ 11. Receive handshake response                    │
     │◄──────────────────────────────────────────────────┤
     │    {kid, ephB_pub, signature_B, status}           │
     │                                                    │
     │ 12. Verify signature_B                            │
     │                                                    │
     │ 13. Session established                           │
     │    Both parties have session_key                  │
     │                                                    │
     │ 14. Encrypted communication                       │
     │◄──────────────────────────────────────────────────►│
     │    Encrypt(message, session_key)                  │
     │                                                    │
```

### Message Encryption Flow

```
Sender                                             Receiver
──────                                             ────────
1. Retrieve session by key ID
2. Generate 64-bit nonce
3. Encrypt with ChaCha20-Poly1305
   ciphertext = Encrypt(plaintext, session_key, nonce, AAD)
4. Increment message counter
                  ──[ciphertext || nonce]──►
                                            5. Retrieve session
                                            6. Check nonce (replay protection)
                                            7. Decrypt with ChaCha20-Poly1305
                                            8. Verify AAD (additional auth data)
                                            9. Update nonce cache
```

## Security Architecture

### Threat Model

**Assumptions:**
- Blockchain registries are trusted (Byzantine fault tolerance)
- Cryptographic primitives are secure (no breaks in Ed25519, X25519, etc.)
- Agent endpoints are authenticated via TLS

**Threats Mitigated:**
1. **Man-in-the-Middle (MITM)**: HPKE forward secrecy prevents key compromise
2. **Replay Attacks**: Nonce-based deduplication and timestamp validation
3. **Impersonation**: DID-based identity with on-chain verification
4. **Key Compromise**: Ephemeral keys limit blast radius
5. **Message Tampering**: AEAD encryption provides integrity

**Threats NOT Mitigated:**
- Endpoint compromise (malicious agent software)
- Side-channel attacks (requires additional hardening)
- Denial of Service (application-layer DoS protection needed)

### Security Layers

```
┌─────────────────────────────────────────────────────────┐
│ Layer 7: Application Security                           │
│  - Input validation                                     │
│  - Rate limiting                                        │
│  - Access control                                       │
├─────────────────────────────────────────────────────────┤
│ Layer 6: Session Security                               │
│  - ChaCha20-Poly1305 AEAD encryption                    │
│  - Replay protection (nonce cache)                      │
│  - Automatic session expiration                         │
├─────────────────────────────────────────────────────────┤
│ Layer 5: Handshake Security                             │
│  - HPKE forward secrecy                                 │
│  - DID-based authentication                             │
│  - Ephemeral key exchange                               │
├─────────────────────────────────────────────────────────┤
│ Layer 4: Cryptographic Security                         │
│  - Ed25519 signatures (RFC 8032)                        │
│  - X25519 key agreement (RFC 7748)                      │
│  - HKDF key derivation (RFC 5869)                       │
├─────────────────────────────────────────────────────────┤
│ Layer 3: Identity Security                              │
│  - Blockchain-based DID registry                        │
│  - Public key ownership verification                    │
│  - Key revocation mechanism                             │
├─────────────────────────────────────────────────────────┤
│ Layer 2: Transport Security                             │
│  - TLS 1.3 for endpoint connections                     │
│  - Certificate validation                               │
├─────────────────────────────────────────────────────────┤
│ Layer 1: Network Security                               │
│  - Firewall rules                                       │
│  - Network segmentation                                 │
└─────────────────────────────────────────────────────────┘
```

## Deployment Architecture

### Single-Region Deployment

```
┌────────────────────────────────────────────────────────────┐
│                      Production Region                      │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │               Load Balancer (HTTPS)                  │  │
│  └────────────┬──────────────────────┬──────────────────┘  │
│               │                      │                      │
│  ┌────────────▼──────────┐  ┌───────▼──────────┐          │
│  │  SAGE Backend (Pod 1) │  │ SAGE Backend (2) │          │
│  │  - Go Application     │  │ - Go Application │ ...      │
│  │  - Session Manager    │  │ - Session Manager│          │
│  └───────────┬───────────┘  └──────────┬───────┘          │
│              │                          │                   │
│  ┌───────────▼──────────────────────────▼────────────┐    │
│  │              Redis (Session Cache)                 │    │
│  └────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │         PostgreSQL (Configuration & Logs)           │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└────────────────────────────────────────────────────────────┘
                           │
                           │ RPC Connection
                           ▼
┌────────────────────────────────────────────────────────────┐
│                   Blockchain Networks                       │
├────────────────────────────────────────────────────────────┤
│  ┌────────────┐    ┌────────────┐    ┌────────────┐       │
│  │  Ethereum  │    │   Solana   │    │    Kaia    │       │
│  │  Mainnet   │    │  Mainnet   │    │  Mainnet   │       │
│  └────────────┘    └────────────┘    └────────────┘       │
└────────────────────────────────────────────────────────────┘
```

### Multi-Region Deployment

```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   Region: US    │         │  Region: EU     │         │  Region: ASIA   │
├─────────────────┤         ├─────────────────┤         ├─────────────────┤
│ SAGE Backend    │         │ SAGE Backend    │         │ SAGE Backend    │
│ (3 replicas)    │         │ (3 replicas)    │         │ (3 replicas)    │
└────────┬────────┘         └────────┬────────┘         └────────┬────────┘
         │                           │                           │
         └───────────────────────────┼───────────────────────────┘
                                     │
                        ┌────────────▼────────────┐
                        │ Shared Redis Cluster    │
                        │ (Cross-Region Replication)│
                        └─────────────────────────┘
```

### Docker Deployment

```bash
# Build production image
docker build -t sage-backend:latest -f Dockerfile .

# Run with environment variables
docker run -d \
  --name sage-backend \
  -p 8080:8080 \
  -e ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_KEY \
  -e SOLANA_RPC_URL=https://api.mainnet-beta.solana.com \
  -e REDIS_URL=redis://redis:6379 \
  sage-backend:latest
```

## Performance Considerations

### Benchmarks

(See `tools/benchmark/README.md` for detailed benchmarks)

**Key Operations:**
- HPKE Derive: ~60-80 μs
- HPKE Roundtrip: ~120-160 μs
- Ed25519 Keygen: ~17-25 μs
- Ed25519 Sign: ~20-25 μs
- Ed25519 Verify: ~50 μs
- Session Encrypt (1KB): ~5-8 μs
- Session Decrypt (1KB): ~5-8 μs

### Scalability

**Horizontal Scaling:**
- Stateless backend design allows unlimited horizontal scaling
- Session state stored in Redis for cross-instance sharing
- Load balancer distributes requests across instances

**Vertical Scaling:**
- CPU-bound: Cryptographic operations (signing, encryption)
- Memory-bound: Session cache (configurable limits)
- I/O-bound: Blockchain RPC calls (caching recommended)

### Optimization Strategies

1. **DID Caching**: LRU cache with 1-hour TTL reduces blockchain queries
2. **Session Pooling**: Reuse session objects to minimize allocations
3. **Batch Operations**: Bundle multiple signatures for verification
4. **Async Processing**: Non-blocking blockchain interactions

## Monitoring and Observability

### Health Checks

```
GET /health
  - Component: Blockchain connectivity
  - Component: Redis availability
  - Component: Session manager status
```

### Metrics

- **Handshake Metrics**: Success rate, latency percentiles (p50, p95, p99)
- **Session Metrics**: Active sessions, encryption throughput, nonce cache hit rate
- **Blockchain Metrics**: RPC latency, retry count, error rate
- **Crypto Metrics**: Signature verification time, key generation time

### Logging

Structured logging with contextual fields:
```json
{
  "level": "info",
  "timestamp": "2025-10-10T12:00:00Z",
  "component": "handshake.server",
  "agent_did": "did:sage:0x1234...",
  "session_id": "sess-abc123",
  "message": "Handshake completed successfully",
  "duration_ms": 125
}
```

## Future Enhancements

1. **Multi-Party Sessions**: Support for group messaging with shared session keys
2. **Key Rotation Protocol**: Automatic rotation without session interruption
3. **Quantum-Resistant Cryptography**: Post-quantum key exchange algorithms
4. **Zero-Knowledge Proofs**: Privacy-preserving identity verification
5. **Cross-Chain Bridges**: Unified DID resolution across L2 solutions

## Related Documentation

- [Development Guide](BUILD.md) - Building and testing SAGE
- [Testing Guide](TESTING.md) - Integration and E2E testing
- [Benchmark Guide](../tools/benchmark/README.md) - Performance benchmarks
- [Coding Guidelines](CODING_GUIDELINES.md) - Code quality standards
- [CI/CD Documentation](CI-CD.md) - Continuous integration workflows
- [API Documentation](API.md) - HTTP and gRPC API reference

## References

- [RFC 9180: HPKE](https://www.rfc-editor.org/rfc/rfc9180.html)
- [RFC 9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [RFC 8032: Ed25519](https://www.rfc-editor.org/rfc/rfc8032.html)
- [RFC 7748: X25519](https://www.rfc-editor.org/rfc/rfc7748.html)
- [W3C DID Core](https://www.w3.org/TR/did-core/)
