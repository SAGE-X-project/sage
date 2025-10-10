# SAGE API Documentation

**Version:** 1.0.0
**Last Updated:** 2025-10-10
**License:** LGPL-3.0

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Authentication](#authentication)
4. [Endpoints](#endpoints)
5. [Security](#security)
6. [Error Handling](#error-handling)
7. [Rate Limiting](#rate-limiting)
8. [Examples](#examples)
9. [SDK Reference](#sdk-reference)
10. [API Explorer](#api-explorer)

---

## Overview

The SAGE (Secure Agent Guarantee Engine) API provides secure, decentralized communication infrastructure for AI agents. Built on Web3 principles, SAGE ensures agent identity verification, message confidentiality, and replay attack prevention.

### Key Features

- **DID-Based Identity**: Decentralized identifiers anchored on blockchain
- **HPKE Encryption**: Hybrid Public Key Encryption (RFC 9180) for secure sessions
- **HTTP Signatures**: RFC 9421 compliant message authentication
- **Replay Protection**: Nonce-based prevention of replay attacks
- **Session Management**: Efficient stateful communication channels
- **Production-Ready**: PostgreSQL persistence, Prometheus metrics, Grafana dashboards

### Architecture

```
┌─────────────┐                                    ┌─────────────┐
│   Client    │                                    │   Server    │
│   Agent     │                                    │   Agent     │
└──────┬──────┘                                    └──────┬──────┘
       │                                                  │
       │  1. Get KEM Public Key                          │
       │ ─────────────────────────────────────────────> │
       │                                                  │
       │  2. HPKE Handshake (DID + Encrypted Payload)    │
       │ ─────────────────────────────────────────────> │
       │                                                  │
       │  3. Session Established (Session ID)            │
       │ <───────────────────────────────────────────── │
       │                                                  │
       │  4. Encrypted Message (Session ID)              │
       │ ─────────────────────────────────────────────> │
       │                                                  │
       │  5. Encrypted Response                          │
       │ <───────────────────────────────────────────── │
       │                                                  │
```

---

## Quick Start

### 1. Install Dependencies

```bash
# Clone repository
git clone https://github.com/sage-x-project/sage.git
cd sage

# Install Go dependencies
go mod download

# Build CLI tools
make build

# Start infrastructure
docker-compose up -d postgres redis blockchain
```

### 2. Generate Agent Credentials

```bash
# Generate client agent credentials
./build/bin/sage-crypto generate -o alice

# Generate server agent credentials
./build/bin/sage-crypto generate -o server

# Register DIDs (development only)
./build/bin/sage-did register \
  --key alice/private.key \
  --name "Alice Agent" \
  --chain-id 1337
```

### 3. Start Server

```bash
# Run SAGE server
go run tests/session/handshake/server/main.go
```

Server will start on `http://localhost:8080`

### 4. Send Test Message

```bash
# Run example client
go run tests/session/handshake/client/main.go
```

Expected output:
```
Handshake successful
Session ID: 550e8400-e29b-41d4-a716-446655440000
Message sent: Hello, Server!
Response: Hello, Client!
```

---

## Authentication

SAGE uses a multi-layered authentication approach:

### Layer 1: DID Resolution

Every agent must have a registered DID (Decentralized Identifier):

```
did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb
│    │    │        │
│    │    │        └─ Ethereum address (agent owner)
│    │    └─────────── Blockchain network
│    └──────────────── Method (sage)
└───────────────────── Scheme
```

**DID Resolution Flow:**
1. Extract Ethereum address from DID
2. Query blockchain smart contract
3. Retrieve public keys (Ed25519 for signing, X25519 for encryption)
4. Verify DID is active (not revoked)
5. Cache DID document locally

### Layer 2: HPKE Session Establishment

Initial message exchange uses HPKE (Hybrid Public Key Encryption):

```go
// Client side
hpkeContext, encapsulatedKey := hpke.SetupSender(serverKemPublicKey)
ciphertext := hpkeContext.Seal(plaintext, nil)
message := encapsulatedKey || ciphertext

// Server side
hpkeContext := hpke.SetupReceiver(encapsulatedKey, serverKemPrivateKey)
plaintext := hpkeContext.Open(ciphertext, nil)
```

**Benefits:**
- Forward secrecy (ephemeral keys)
- No shared secrets required
- Resistant to quantum attacks (with appropriate KEM choice)

### Layer 3: Message Signatures

Every message is signed with Ed25519:

```
signature = Ed25519.Sign(
    privateKey,
    sender_did || receiver_did || message || timestamp
)
```

**Verification:**
1. Resolve sender DID to get public key
2. Reconstruct signed data
3. Verify Ed25519 signature
4. Check timestamp (prevent replay)
5. Store signature nonce (prevent reuse)

### Layer 4: HTTP Signatures (Protected Endpoints)

Some endpoints require RFC 9421 HTTP Message Signatures:

```http
Signature-Input: sig1=("@method" "@path" "@authority"
  "content-type" "content-digest");created=1234567890;
  keyid="did:sage:ethereum:0x...";alg="ed25519"
Signature: sig1=:base64_signature:
```

See [HTTP Signatures Example](../api/examples/signatures.md) for details.

---

## Endpoints

### Base URL

- **Production**: `https://api.sage.example.com/v1`
- **Staging**: `https://staging-api.sage.example.com/v1`
- **Development**: `http://localhost:8080/v1`

### A2A Endpoints

#### POST `/v1/a2a:sendMessage`

Send encrypted agent-to-agent message.

**Request:**
```json
{
  "sender_did": "did:sage:ethereum:0xAlice",
  "receiver_did": "did:sage:ethereum:0xServer",
  "message": "AgECA...base64_encrypted_payload...",
  "timestamp": 1234567890,
  "signature": "bXlfc2lnbmF0dXJl...base64_signature..."
}
```

**Response (Handshake):**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "response": "AgECA...base64_encrypted_response..."
}
```

**Response (Subsequent Messages):**
```json
{
  "response": "AgECA...base64_encrypted_response..."
}
```

**Status Codes:**
- `200 OK`: Message processed successfully
- `400 Bad Request`: Invalid message format
- `401 Unauthorized`: Signature verification failed
- `500 Internal Server Error`: Server error

**See Also:** [Authentication Example](../api/examples/authentication.md)

---

### Protected Endpoints

#### POST `/protected`

Demonstration of RFC 9421 HTTP signature authentication.

**Headers:**
```http
Content-Type: application/json
Content-Digest: sha-256=:X48E9qO...:
X-Timestamp: 1234567890
Signature-Input: sig1=(...)
Signature: sig1=:base64_sig:
```

**Request:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "encrypted_data": "AgECA..."
}
```

**Response:**
```json
{
  "message": "Request authenticated successfully",
  "decrypted_data": {
    "example": "decrypted content"
  }
}
```

**See Also:** [HTTP Signatures Example](../api/examples/signatures.md)

---

### Debug Endpoints

#### GET `/debug/kem-pub`

Get server's KEM public key for HPKE.

**Response:**
```json
{
  "kem_public_key": "j8tTZ3xQ9K2nL4mP5rS6vW7xY8zA1bC2dE3fG4hI5jK="
}
```

**Note:** 32-byte X25519 public key, base64-encoded

---

#### GET `/debug/server-did`

Get server's DID.

**Response:**
```json
{
  "did": "did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"
}
```

---

#### POST `/debug/register-agent`

Register agent metadata (development only).

**Request:**
```json
{
  "did": "did:sage:ethereum:0xAlice",
  "name": "Alice Agent",
  "is_active": true,
  "public_key": "Y29ycmVjdF9lZDI1NTE5X3B1YmxpY19rZXk=",
  "public_kem_key": "Y29ycmVjdF94MjU1MTlfcHVibGljX2tleQ=="
}
```

**Response:**
```json
{
  "message": "agent registered"
}
```

**Production:** Agents are resolved from blockchain contracts, not manually registered.

---

#### GET `/debug/health`

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-10-10T12:00:00Z",
  "sessions": {
    "active": 42,
    "total": 150
  }
}
```

---

## Security

### Threat Model

SAGE protects against:

- **Man-in-the-Middle (MITM)**: HPKE encryption + DID authentication
- **Replay Attacks**: Nonce tracking + timestamp validation
- **Message Tampering**: Ed25519 signatures + content digests
- **Impersonation**: Blockchain-anchored DID verification
- **Session Hijacking**: Encrypted session keys + expiration

### Security Best Practices

#### 1. Key Management

```bash
# Generate keys securely
./sage-crypto generate -o agent_name

# Permissions
chmod 600 agent_name/private.key
chmod 644 agent_name/public.key

# Storage
#  DO: Use encrypted key stores (HashiCorp Vault, AWS KMS)
#  DON'T: Commit keys to version control
#  DON'T: Share keys between environments
```

#### 2. DID Registration

```bash
# Production DID registration
./sage-did register \
  --key /secure/path/private.key \
  --name "Production Agent" \
  --chain-id 1 \  # Mainnet
  --confirm

# Verify registration
./sage-did resolve did:sage:ethereum:0x...
```

#### 3. Session Configuration

```yaml
# config.yaml
session:
  max_age: "1h"           # Limit session lifetime
  idle_timeout: "10m"     # Auto-expire inactive sessions
  cleanup_interval: "30s" # Regular cleanup

security:
  nonce_ttl: "5m"         # Nonce expiration
  max_clock_skew: "5m"    # Timestamp tolerance
```

#### 4. TLS/HTTPS

```yaml
# Production configuration
server:
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    min_version: "TLS1.3"
```

**Note:** NEVER disable TLS in production.

#### 5. Database Security

```bash
# Use SSL for PostgreSQL
DB_SSLMODE=require
DB_SSLCERT=/path/to/client-cert.pem
DB_SSLKEY=/path/to/client-key.pem
DB_SSLROOTCERT=/path/to/ca-cert.pem

# Strong passwords
DB_PASSWORD=$(openssl rand -base64 32)

# Restricted access
# Only allow connections from application servers
```

---

## Error Handling

### Error Response Format

All errors follow this format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": {
    "additional": "context",
    "field": "value"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request body/headers |
| `INVALID_DID` | 400 | DID format incorrect |
| `INVALID_SIGNATURE` | 401 | Signature verification failed |
| `EXPIRED_SIGNATURE` | 401 | Signature timestamp too old |
| `REPLAY_ATTACK` | 401 | Nonce already used |
| `SESSION_EXPIRED` | 401 | Session not found or expired |
| `UNKNOWN_DID` | 404 | DID not registered/revoked |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

### Example Error Responses

**Invalid Signature:**
```json
{
  "error": "signature verification failed",
  "code": "INVALID_SIGNATURE",
  "details": {
    "sender_did": "did:sage:ethereum:0xAlice"
  }
}
```

**Session Expired:**
```json
{
  "error": "session not found or expired",
  "code": "SESSION_EXPIRED",
  "details": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "expired_at": "2025-10-10T13:00:00Z"
  }
}
```

**Rate Limit:**
```json
{
  "error": "rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED",
  "details": {
    "limit": 100,
    "window": "1m",
    "retry_after": 42
  }
}
```

---

## Rate Limiting

### Default Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| `/v1/a2a:sendMessage` | 100 req | 1 minute |
| `/protected` | 50 req | 1 minute |
| `/debug/*` | 200 req | 1 minute |

### Rate Limit Headers

Responses include rate limit information:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1234567890
```

### Handling Rate Limits

**Best Practices:**
1. Monitor `X-RateLimit-Remaining` header
2. Implement exponential backoff
3. Cache data to reduce requests
4. Use websockets for high-frequency communication (future)

**Example (Go):**
```go
func sendWithRetry(msg *Message) error {
    backoff := 1 * time.Second
    maxBackoff := 32 * time.Second

    for {
        resp, err := sendMessage(msg)
        if err == nil {
            return nil
        }

        if resp.StatusCode == 429 {
            // Rate limited, wait and retry
            time.Sleep(backoff)
            backoff = min(backoff*2, maxBackoff)
            continue
        }

        return err
    }
}
```

---

## Examples

### Complete Examples

1. **[Authentication](../api/examples/authentication.md)**
   - HPKE handshake flow
   - DID resolution
   - Message signing
   - Session establishment

2. **[Session Management](../api/examples/sessions.md)**
   - Session lifecycle
   - Expiration handling
   - Activity tracking
   - Monitoring queries

3. **[HTTP Signatures](../api/examples/signatures.md)**
   - RFC 9421 implementation
   - Signature creation
   - Signature verification
   - Content digests

### Code Samples

**Go:**
```go
// See tests/session/handshake/client/main.go
// Complete working example
```

**Python:**
```python
# Coming soon: Python SDK
# pip install sage-python-sdk
```

**Rust:**
```rust
// Coming soon: Rust SDK
// cargo add sage-sdk
```

**Java:**
```java
// Coming soon: Java SDK
// implementation 'com.sage:sage-sdk:1.0.0'
```

---

## SDK Reference

### Official SDKs

| Language | Status | Repository | Documentation |
|----------|--------|------------|---------------|
| Go |  Built-in | `pkg/` | [GoDoc](https://pkg.go.dev/github.com/sage-x-project/sage) |
| Python |  Planned | TBD | TBD |
| Rust |  Planned | TBD | TBD |
| Java |  Planned | TBD | TBD |

### Go SDK

**Installation:**
```bash
go get github.com/sage-x-project/sage
```

**Usage:**
```go
import (
    "github.com/sage-x-project/sage/pkg/crypto"
    "github.com/sage-x-project/sage/pkg/session"
    "github.com/sage-x-project/sage/pkg/signature"
)

// Generate keys
keyPair, err := crypto.GenerateKeyPair()

// Create session
sess, err := session.NewSession(clientDID, serverDID)

// Sign message
sig, err := signature.SignMessage(message, privateKey)
```

---

## API Explorer

### Swagger UI

Interactive API documentation available via Swagger UI:

```bash
# Start Swagger UI with docker-compose
docker-compose --profile docs up swagger-ui

# Access at http://localhost:8081
```

### OpenAPI Specification

Download the OpenAPI 3.0 specification:

```bash
# Raw YAML
curl http://localhost:8081/api/openapi.yaml

# JSON format
curl http://localhost:8081/api/openapi.json
```

**File Location:** `api/openapi.yaml`

### Code Generation

Generate client SDKs from OpenAPI spec:

```bash
# Install openapi-generator
npm install -g @openapitools/openapi-generator-cli

# Generate Python client
openapi-generator-cli generate \
  -i api/openapi.yaml \
  -g python \
  -o sdk/python

# Generate Java client
openapi-generator-cli generate \
  -i api/openapi.yaml \
  -g java \
  -o sdk/java
```

---

## Monitoring

### Prometheus Metrics

SAGE exposes metrics at `http://localhost:9090/metrics`:

```promql
# Request rate
rate(sage_api_requests_total[5m])

# Error rate
rate(sage_api_errors_total[5m])

# Response time (95th percentile)
histogram_quantile(0.95, rate(sage_api_duration_seconds_bucket[5m]))

# Active sessions
sage_sessions_active
```

### Grafana Dashboards

Pre-built dashboards available:

1. **API Overview**: Request rates, errors, latencies
2. **Sessions**: Active sessions, expiration rates
3. **Security**: Signature failures, replay attacks
4. **Database**: Query performance, connection pool

**Access:** `http://localhost:3000` (default: admin/admin)

---

## Versioning

### API Version

Current version: **v1**

**URL Pattern:** `/v1/{endpoint}`

### Deprecation Policy

1. **Announcement**: 6 months before deprecation
2. **Deprecation Headers**: Added to responses
3. **Sunset Date**: Communicated in headers
4. **Documentation**: Migration guides provided

**Example Deprecation Header:**
```http
Deprecation: Sun, 11 Apr 2026 00:00:00 GMT
Sunset: Sun, 11 Oct 2026 00:00:00 GMT
Link: <https://docs.sage.com/migration/v2>; rel="deprecation"
```

---

## Support

### Documentation

- **GitHub**: https://github.com/sage-x-project/sage
- **Docs**: https://docs.sage.com
- **API Reference**: https://api-docs.sage.com

### Community

- **Discord**: https://discord.gg/sage
- **GitHub Discussions**: https://github.com/sage-x-project/sage/discussions
- **Stack Overflow**: Tag `sage-api`

### Issues

Report bugs and issues on GitHub:
https://github.com/sage-x-project/sage/issues

**Include:**
1. API endpoint and method
2. Request/response examples
3. Error messages
4. SAGE version
5. Environment (OS, Go version, etc.)

---

## Changelog

### v1.0.0 (2025-10-10)

**Features:**
- Initial API release
- A2A messaging with HPKE encryption
- DID-based authentication
- RFC 9421 HTTP signatures
- Session management
- PostgreSQL persistence
- Prometheus metrics
- Swagger UI documentation

**Security:**
- Ed25519 signatures
- Nonce-based replay protection
- Timestamp validation
- TLS 1.3 support

---

## License

SAGE API is licensed under LGPL-3.0. See [LICENSE](../LICENSE) for details.

**Smart Contracts:** MIT License (see `contracts/LICENSE`)

---

**Last Updated:** 2025-10-10
**API Version:** 1.0.0
**Contact:** sage-x-project@example.com
