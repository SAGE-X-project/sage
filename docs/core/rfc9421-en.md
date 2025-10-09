# RFC-9421 HTTP Message Signatures

Implementation of RFC-9421 HTTP Message Signatures for the SAGE (Secure Agent Guarantee Engine) project, providing secure HTTP request signing and verification for AI agent communications.

## Overview

RFC-9421 defines a mechanism for creating, encoding, and verifying HTTP message signatures. This implementation enables AI agents to sign their HTTP requests, ensuring message integrity, authenticity, and preventing replay attacks.

## Key Features

- **HTTP Request Signing**: Sign HTTP requests with various signature algorithms
- **Signature Verification**: Verify signatures on incoming HTTP requests
- **Selective Field Signing**: Choose which HTTP components to include in signatures
- **Multiple Algorithm Support**: Ed25519, ES256K (Secp256k1), RSA-PSS-SHA256
- **Dynamic Algorithm Registry**: Centralized crypto algorithm management
- **Query Parameter Protection**: Selective signing of query parameters
- **Timestamp Validation**: Protection against replay attacks
- **Metadata Integration**: Integration with DID agent metadata
- **Message Builder**: Fluent API for message construction

## Architecture

### Package Structure

```
core/rfc9421/
├── types.go              # Core type definitions
├── message_builder.go    # Message builder with fluent API
├── parser.go             # Signature-Input and Signature header parsers
├── canonicalizer.go      # HTTP message canonicalization
├── verifier.go           # Message signature verification
└── verifier_http.go      # HTTP-specific verification
```

### Algorithm Registry Integration

The RFC-9421 implementation integrates with SAGE's centralized cryptographic algorithm registry (`crypto` package). Supported algorithms are dynamically registered and validated:

```go
// Get list of supported algorithms
algorithms := rfc9421.GetSupportedAlgorithms()
// Returns: ["ed25519", "es256k", "rsa-pss-sha256"]

// Check if algorithm is supported
if rfc9421.IsAlgorithmSupported("ed25519") {
    // Algorithm is supported
}
```

**Currently Supported Algorithms**:
- **ed25519**: Edwards-curve Digital Signature Algorithm
- **es256k**: ECDSA with secp256k1 curve (Ethereum-compatible)
- **rsa-pss-sha256**: RSA with PSS padding and SHA-256

**Note**: ECDSA P-256 cryptographic operations are fully functional and tested, but the algorithm is not yet registered as a distinct RFC-9421 algorithm identifier. Currently, all ECDSA operations are mapped to `es256k` (secp256k1) in the algorithm registry. See `crypto/keys/algorithms.go` for implementation status.

### Core Components

#### 1. Parser (`parser.go`)
Parses RFC-9421 signature headers according to RFC-8941 structured fields:
- `ParseSignatureInput`: Parses Signature-Input headers
- `ParseSignature`: Parses Signature headers with base64-encoded signatures
- Error handling for malformed headers and invalid Base64 encoding

#### 2. Canonicalizer (`canonicalizer.go`)
Creates signature base strings from HTTP requests:
- Supports HTTP signature components: `@method`, `@target-uri`, `@authority`, `@scheme`, `@request-target`, `@path`, `@query`
- Handles regular HTTP headers with proper canonicalization
- Implements `@query-param` for selective query parameter signing
- Component normalization and ordering

#### 3. HTTP Verifier (`verifier_http.go`)
Provides HTTP request signing and verification:
- `SignRequest`: Signs HTTP requests with private keys
- `VerifyRequest`: Verifies HTTP request signatures with algorithm validation
- Integration with centralized algorithm registry for validation

#### 4. Message Builder (`message_builder.go`)
Provides fluent API for constructing RFC-9421 messages:
- `NewMessageBuilder()`: Creates new message builder
- Builder methods: `WithAgentDID()`, `WithMessageID()`, `WithTimestamp()`, etc.
- `Build()`: Constructs final message with default signed fields
- `ParseMessageFromHeaders()`: Parses messages from HTTP-style headers

#### 5. Verifier (`verifier.go`)
Core verification logic:
- `VerifyWithMetadata()`: Verifies signature with metadata constraints
- `ConstructSignatureBase()`: Builds signature base string for debugging
- `VerifyHTTPRequest()`: Wrapper for HTTP verification
- Support for multiple signature algorithms via registry

## Usage Examples

### Signing an HTTP Request

```go
package main

import (
    "crypto/ed25519"
    "crypto/rand"
    "net/http"
    "strings"
    "time"

    "github.com/sage-x-project/sage/core/rfc9421"
)

func main() {
    // Generate key pair
    publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

    // Create HTTP request
    req, _ := http.NewRequest("POST", "https://api.example.com/agent/action",
        strings.NewReader(`{"action": "process"}`))

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Date", time.Now().Format(http.TimeFormat))

    // Define signature parameters (use registry algorithm names)
    params := &rfc9421.SignatureInputParams{
        CoveredComponents: []string{
            `"@method"`,
            `"@path"`,
            `"content-type"`,
            `"date"`,
        },
        KeyID:     "agent-key-1",
        Algorithm: "ed25519",  // Use registry algorithm names
        Created:   time.Now().Unix(),
    }

    // Sign the request
    verifier := rfc9421.NewHTTPVerifier()
    err := verifier.SignRequest(req, "sig1", params, privateKey)
    if err != nil {
        panic(err)
    }

    // Request now has Signature-Input and Signature headers
    fmt.Println("Signature-Input:", req.Header.Get("Signature-Input"))
    fmt.Println("Signature:", req.Header.Get("Signature"))
}
```

### Verifying an HTTP Request

```go
func verifyRequest(req *http.Request, publicKey ed25519.PublicKey) error {
    verifier := rfc9421.NewHTTPVerifier()

    // Verify with default options (5 minute max age)
    err := verifier.VerifyRequest(req, publicKey, nil)
    if err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }

    return nil
}
```

### Selective Query Parameter Signing

```go
// Sign only specific query parameters
params := &rfc9421.SignatureInputParams{
    CoveredComponents: []string{
        `"@method"`,
        `"@path"`,
        `"@query-param";name="api_key"`,  // Only sign api_key parameter
        `"@query-param";name="action"`,   // Only sign action parameter
    },
    Created: time.Now().Unix(),
}

// Other query parameters can be modified without invalidating the signature
```

### Building Messages with MessageBuilder

```go
// Create a message using the builder
builder := rfc9421.NewMessageBuilder()
message := builder.
    WithAgentDID("did:sage:ethereum:0x123...").
    WithMessageID("msg-001").
    WithTimestamp(time.Now()).
    WithNonce("random-nonce-123").
    WithBody([]byte(`{"action": "process"}`)).
    WithAlgorithm(rfc9421.AlgorithmEdDSA).
    WithKeyID("agent-key-1").
    WithSignedFields("agent_did", "message_id", "timestamp", "nonce", "body").
    AddHeader("Content-Type", "application/json").
    AddMetadata("capability", "signing").
    Build()

// Parse message from HTTP headers
headers := map[string]string{
    "X-Agent-DID":            "did:sage:ethereum:0x123...",
    "X-Message-ID":           "msg-001",
    "X-Timestamp":            time.Now().Format(time.RFC3339),
    "X-Nonce":                "random-nonce-123",
    "X-Signature-Algorithm":  "ed25519",
    "X-Key-ID":               "agent-key-1",
    "X-Signed-Fields":        "agent_did,message_id,timestamp",
}
body := []byte(`{"action": "process"}`)
message, err := rfc9421.ParseMessageFromHeaders(headers, body)
if err != nil {
    panic(err)
}
```

### Integration with DID

```go
// Create verification service with DID resolver
verificationService := core.NewVerificationService(didManager)

// Verify agent message with metadata
message := &rfc9421.Message{
    AgentDID:  "did:sage:ethereum:0x123...",
    Body:      []byte("AI response"),
    Signature: signature,
    Algorithm: "ed25519",
}

result, err := verificationService.VerifyAgentMessage(
    ctx,
    message,
    &rfc9421.VerificationOptions{
        RequireActiveAgent: true,
        VerifyMetadata:     true,
    },
)

if result.Valid {
    fmt.Printf("Message verified from agent: %s\n", result.AgentName)
}
```

## Advanced Features

### Metadata Verification

The verifier supports advanced metadata validation:

```go
verifier := rfc9421.NewVerifier()

// Define expected metadata
expectedMetadata := map[string]interface{}{
    "version": "1.0",
    "environment": "production",
}

// Define required capabilities
requiredCapabilities := []string{"signing", "verification"}

// Verify with metadata
result, err := verifier.VerifyWithMetadata(
    publicKey,
    message,
    expectedMetadata,
    requiredCapabilities,
    &rfc9421.VerificationOptions{
        RequireActiveAgent: true,
        VerifyMetadata:     true,
    },
)

if result.Valid {
    fmt.Println("Message verified with metadata constraints")
}
```

### Signature Base Construction

For debugging or custom verification flows:

```go
verifier := rfc9421.NewVerifier()

// Get the signature base string
signatureBase := verifier.ConstructSignatureBase(message)
fmt.Println("Signature base:", signatureBase)

// Output format:
// agent_did: did:sage:ethereum:0x123...
// message_id: msg-001
// timestamp: 2025-01-15T10:30:00Z
// nonce: random-nonce-123
// body: {"action": "process"}
```

## Supported HTTP Components

### Special Components
- `@method`: HTTP method (GET, POST, etc.)
- `@target-uri`: Full target URI
- `@authority`: Host and port
- `@scheme`: URI scheme (http/https)
- `@request-target`: Method and path
- `@path`: URI path
- `@query`: Full query string
- `@query-param`: Selective query parameters
- `@status`: Response status code (detection implemented, response signing/verification not yet available)

### Header Components
Any HTTP header can be included by using its lowercase name:
- `date`
- `content-type`
- `content-length`
- `authorization`
- etc.

## Security Considerations

1. **Timestamp Validation**: Always verify `created` and `expires` timestamps
2. **Replay Protection**: Use nonces for critical operations
3. **Key Management**: Rotate keys regularly using the rotation package
4. **Algorithm Selection**: Use Ed25519 for new implementations, ES256K for Ethereum compatibility
5. **Component Selection**: Include critical components in signatures
6. **Algorithm Validation**: The implementation automatically validates algorithm compatibility with public keys

## Configuration Options

### Verification Options

```go
opts := &rfc9421.HTTPVerificationOptions{
    // Maximum age for signatures (default: 5 minutes)
    MaxAge: 10 * time.Minute,

    // Required signature name (if multiple signatures exist)
    SignatureName: "sig1",

    // Required components that must be in the signature
    RequiredComponents: []string{`"@method"`, `"@path"`},
}
```

### Message Verification Options

```go
opts := &rfc9421.VerificationOptions{
    // Maximum clock skew allowed
    MaxClockSkew: 5 * time.Minute,

    // Require agent to be active
    RequireActiveAgent: true,

    // Verify metadata fields
    VerifyMetadata: true,

    // Required capabilities
    RequiredCapabilities: []string{"signing", "verification"},
}
```

## Testing

The implementation includes comprehensive tests based on RFC-9421 test vectors:

```bash
# Run RFC-9421 tests
go test ./core/rfc9421/...

# Run with race detection
go test -race ./core/rfc9421/...

# Check coverage
go test -cover ./core/rfc9421/...
```

### Test Coverage

The implementation has **100% coverage** of the documented test plan in `rfc-9421-test.md`:

#### Unit Tests
- ✅ **Parser tests** (6/6 tests passing)
  - Basic parsing, multiple signatures, whitespace handling
  - Error cases: malformed headers, invalid Base64
- ✅ **Canonicalizer tests** (10/10 tests passing)
  - HTTP components (`@method`, `@path`, `@query`, etc.)
  - Header normalization and whitespace handling
  - Query parameter protection (`@query-param`)
- ✅ **Message Builder tests** (3/3 tests passing)
  - Fluent API construction, header parsing

#### Integration Tests
- ✅ **End-to-end tests** (2/2 tests passing)
  - Ed25519 signing and verification
  - ECDSA P-256 signing and verification
- ✅ **Negative tests** (5/5 tests passing)
  - Signature tampering detection
  - Signed header modification detection
  - Unsigned header modification (should pass)
  - Expiry validation (`created` + `MaxAge`, `expires`)

#### Advanced Tests
- ✅ **Query parameter tests** (5/5 tests passing)
  - Selective parameter signing and protection
  - Parameter case sensitivity
  - Non-existent parameter handling
- ✅ **Edge case tests** (3/3 tests passing)
  - Empty paths, special characters, proxy requests

**Total: 26/26 tests passing (100% coverage)**

### Test Files
- `parser_test.go` - Header parsing and error handling (6 tests)
- `canonicalizer_test.go` - Signature base construction (10 tests)
- `verifier_test.go` - Signature verification logic (varies)
- `integration_test.go` - End-to-end and negative test cases (7 tests)
- `message_builder_test.go` - Message construction API (3 tests)

## Standards Compliance

This implementation follows:
- [RFC-9421](https://datatracker.ietf.org/doc/rfc9421/): HTTP Message Signatures
- [RFC-8941](https://datatracker.ietf.org/doc/rfc8941/): Structured Field Values for HTTP
- [RFC-9110](https://datatracker.ietf.org/doc/rfc9110/): HTTP Semantics

## Algorithm Support Status

| Algorithm | Status | RFC-9421 Name | Notes |
|-----------|--------|---------------|-------|
| Ed25519 | ✅ Fully Supported | `ed25519` | Recommended for new implementations |
| ES256K (Secp256k1) | ✅ Fully Supported | `es256k` | Ethereum-compatible |
| RSA-PSS-SHA256 | ✅ Fully Supported | `rsa-pss-sha256` | RSA with PSS padding |
| ECDSA P-256 | ⚠️ Crypto Only | N/A | Cryptographic operations work, not registered as distinct algorithm |
| RSA-PKCS#1 v1.5 | ❌ Not Supported | `rsa-v1_5-sha256` | Legacy RSA (planned) |

## Implementation Status & Roadmap

### Completed Features
- ✅ **RSA-PSS-SHA256 support** - Fully implemented and registered in algorithm registry
- ✅ **Core RFC-9421 compliance** - HTTP request signing with Ed25519, ES256K, RSA-PSS-SHA256
- ✅ **Comprehensive test coverage** - 100% coverage of documented test plan (26/26 tests passing)

### Partially Implemented
- ⚠️ **Response signature support** - `@status` component detection implemented, signing/verification methods pending
- ⚠️ **ECDSA P-256 support** - Cryptographic operations fully functional and tested, algorithm registration as distinct identifier pending

### Planned Enhancements
- **RSA-PKCS#1 v1.5 support** - Legacy RSA algorithm (`rsa-v1_5-sha256`)
- **Complete ECDSA P-256 registration** - Register as distinct algorithm (`ecdsa-p256-sha256`) separate from secp256k1
- **Response signing methods** - `SignResponse()` and `VerifyResponse()` for HTTP responses
- **Signature negotiation** - Accept-Signature header, algorithm capability advertisement
- **Performance optimizations** - Buffer pooling, goroutine pools, pre-allocation strategies
- **Caching layer** - Public key cache, DID resolution cache, parsed signature cache

### Technical Debt
- Complete ECDSA P-256 registration in algorithm registry (see `crypto/keys/algorithms.go:58-60`)
- Implement response canonicalization for `@status` component
