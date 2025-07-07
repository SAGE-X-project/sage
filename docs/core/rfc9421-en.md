# RFC-9421 HTTP Message Signatures

Implementation of RFC-9421 HTTP Message Signatures for the SAGE (Secure Agent Guarantee Engine) project, providing secure HTTP request signing and verification for AI agent communications.

## Overview

RFC-9421 defines a mechanism for creating, encoding, and verifying HTTP message signatures. This implementation enables AI agents to sign their HTTP requests, ensuring message integrity, authenticity, and preventing replay attacks.

## Key Features

- **HTTP Request Signing**: Sign HTTP requests with various signature algorithms
- **Signature Verification**: Verify signatures on incoming HTTP requests
- **Selective Field Signing**: Choose which HTTP components to include in signatures
- **Multiple Algorithm Support**: Ed25519, ECDSA P-256, RSA (planned)
- **Query Parameter Protection**: Selective signing of query parameters
- **Timestamp Validation**: Protection against replay attacks
- **Metadata Integration**: Integration with DID agent metadata

## Architecture

### Package Structure

```
core/rfc9421/
├── types.go              # Core type definitions
├── message.go            # Message structure and builder
├── parser.go             # Signature-Input and Signature header parsers
├── canonicalizer.go      # HTTP message canonicalization
├── verifier.go           # Message signature verification
└── verifier_http.go      # HTTP-specific verification
```

### Core Components

#### 1. Parser (`parser.go`)
Parses RFC-9421 signature headers according to RFC-8941 structured fields:
- `ParseSignatureInput`: Parses Signature-Input headers
- `ParseSignature`: Parses Signature headers with base64-encoded signatures

#### 2. Canonicalizer (`canonicalizer.go`)
Creates signature base strings from HTTP requests:
- Supports HTTP signature components: `@method`, `@target-uri`, `@authority`, `@scheme`, `@request-target`, `@path`, `@query`
- Handles regular HTTP headers with proper canonicalization
- Implements `@query-param` for selective query parameter signing

#### 3. HTTP Verifier (`verifier_http.go`)
Provides HTTP request signing and verification:
- `SignRequest`: Signs HTTP requests with private keys
- `VerifyRequest`: Verifies HTTP request signatures

## Usage Examples

### Signing an HTTP Request

```go
package main

import (
    "crypto/ed25519"
    "crypto/rand"
    "net/http"
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
    
    // Define signature parameters
    params := &rfc9421.SignatureInputParams{
        CoveredComponents: []string{
            `"@method"`,
            `"@path"`,
            `"content-type"`,
            `"date"`,
        },
        KeyID:     "agent-key-1",
        Algorithm: "ed25519",
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
- `@status`: Response status (responses only)

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
4. **Algorithm Selection**: Use Ed25519 for new implementations
5. **Component Selection**: Include critical components in signatures

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

## Standards Compliance

This implementation follows:
- [RFC-9421](https://datatracker.ietf.org/doc/rfc9421/): HTTP Message Signatures
- [RFC-8941](https://datatracker.ietf.org/doc/rfc8941/): Structured Field Values for HTTP
- [RFC-9110](https://datatracker.ietf.org/doc/rfc9110/): HTTP Semantics

## Future Enhancements

- [ ] Response signature support
- [ ] RSA-PSS and RSA-PKCS#1 v1.5 support
- [ ] Signature negotiation
- [ ] Performance optimizations
- [ ] Caching for signature verification