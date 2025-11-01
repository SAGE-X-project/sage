# Quick Start Guide: PR #118 Security Enhancements

This guide demonstrates how to use the new security features introduced in PR #118.

## Table of Contents
- [RFC9421 Body Integrity Validation](#1-rfc9421-body-integrity-validation)
- [HPKE ECDSA Signature Support](#2-hpke-ecdsa-signature-support)
- [Enhanced HPKE Error Handling](#3-enhanced-hpke-error-handling)
- [DID X25519 Key Support](#4-did-x25519-key-support)

---

## 1. RFC9421 Body Integrity Validation

Prevents HTTP request body tampering by validating Content-Digest headers.

### Automatic Validation (Recommended)

```go
package main

import (
    "net/http"
    "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
)

func main() {
    // Create HTTP request
    body := []byte(`{"message":"hello world"}`)
    req, _ := http.NewRequest("POST", "https://api.example.com/endpoint", bytes.NewReader(body))

    // Compute and set Content-Digest header
    digest := rfc9421.ComputeContentDigest(body)
    req.Header.Set("Content-Digest", digest)
    // Result: "sha-256=:X48E9qOokqqrvdts8nOJRJN3OWDUoyWxBf7kbu9DBPE=:"

    // Sign the request (Content-Digest will be covered by signature)
    // ... signing logic ...

    // Verify request (body integrity automatically validated)
    verifier := rfc9421.NewHTTPVerifier()
    err := verifier.VerifyHTTPRequest(req, publicKey)
    if err != nil {
        // Body tampering detected or signature invalid
        panic(err)
    }
}
```

### Manual Validation (Advanced)

```go
package main

import (
    "net/http"
    "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
)

func main() {
    // Create body integrity validator
    validator := rfc9421.NewBodyIntegrityValidator()

    // Define which components are covered by the signature
    coveredComponents := []string{"content-digest", "@method", "@path"}

    // Validate Content-Digest matches actual body
    err := validator.ValidateContentDigest(req, coveredComponents)
    if err != nil {
        // Body has been tampered with
        panic(err)
    }

    // Check if a specific component is covered
    if rfc9421.IsComponentCovered(coveredComponents, "content-digest") {
        // Content-Digest is protected by signature
    }
}
```

### Security Benefits

-  **Prevents body tampering**: Attackers cannot modify request body without detection
-  **Cryptographic guarantee**: SHA-256 hash ensures body integrity
-  **RFC 9421 compliant**: Standard HTTP message signatures

---

## 2. HPKE ECDSA Signature Support

Enables Ethereum-compatible agent communication using Secp256k1 keys.

### Automatic Usage (Recommended)

```go
package main

import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/hpke"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

func main() {
    // Option 1: Use Ed25519 key (original support)
    ed25519Key, _ := crypto.GenerateEd25519KeyPair()

    // Option 2: Use ECDSA Secp256k1 key (NEW - Ethereum compatible)
    secp256k1Key, _ := crypto.GenerateSecp256k1KeyPair()

    // Create HPKE client with either key type
    client := hpke.NewClient(
        secp256k1Key,  // Works with both Ed25519 and Secp256k1
        transport,
        resolver,
        &hpke.ClientOpts{
            KEM: kemKey,
        },
    )

    // Complete handshake - signature verification is automatic
    ctx := context.Background()
    resp, err := client.CompleteHandshake(ctx,
        "did:sage:ethereum:0xYourAddress",
        "did:sage:ethereum:0xRecipientAddress",
    )
    if err != nil {
        panic(err)
    }

    // Handshake successful - ECDSA signature verified automatically
}
```

### Manual Signature Verification (Advanced)

```go
package main

import (
    "github.com/sage-x-project/sage/pkg/agent/hpke"
)

func main() {
    // Verify ECDSA signature directly
    ecdsaVerifier := hpke.NewECDSAVerifier()

    payload := []byte("message to verify")
    signature := []byte{...} // 64-byte or 65-byte Ethereum signature

    err := ecdsaVerifier.Verify(payload, signature, ecdsaPublicKey)
    if err != nil {
        // Signature invalid
        panic(err)
    }

    // Verify Ed25519 signature directly
    ed25519Verifier := hpke.NewEd25519Verifier()
    err = ed25519Verifier.Verify(payload, signature, ed25519PublicKey)

    // Auto-select verifier based on key type (Composite Pattern)
    compositeVerifier := hpke.NewCompositeVerifier()
    err = compositeVerifier.Verify(payload, signature, anyPublicKey)

    // Check if a verifier supports a specific key type
    if ecdsaVerifier.Supports(publicKey) {
        // This is an ECDSA key
    }
}
```

### Ethereum Integration Example

```go
package main

import (
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/hpke"
)

func main() {
    // Generate Ethereum-compatible key
    keyPair, _ := crypto.GenerateSecp256k1KeyPair()

    // Derive Ethereum address
    ethAddress, _ := did.DeriveEthereumAddress(keyPair)
    // Example: "0x742d35cc6634c0532925a3b844bc9e7595f0beef"

    // Create DID with Ethereum address
    agentDID := did.GenerateAgentDIDWithAddress(did.ChainEthereum, ethAddress)
    // Result: "did:sage:ethereum:0x742d35cc6634c0532925a3b844bc9e7595f0beef"

    // Use in HPKE handshake
    client := hpke.NewClient(keyPair, transport, resolver, nil)
    resp, err := client.CompleteHandshake(ctx, agentDID, responderDID)
    if err != nil {
        panic(err)
    }

    // ECDSA signature automatically verified with Ethereum-style Keccak256 hash
}
```

### Security Benefits

-  **Ethereum compatibility**: Use Secp256k1 keys for blockchain integration
-  **Flexible architecture**: Strategy Pattern allows easy algorithm extension
-  **Multiple formats**: Supports 64-byte raw, 65-byte Ethereum, and DER-encoded signatures
-  **Backward compatible**: Existing Ed25519 code works unchanged

---

## 3. Enhanced HPKE Error Handling

Better error messages for easier debugging of handshake failures.

### Example Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/sage-x-project/sage/pkg/agent/hpke"
)

func main() {
    client := hpke.NewClient(keyPair, transport, resolver, nil)

    resp, err := client.CompleteHandshake(ctx, initDID, respDID)
    if err != nil {
        // Enhanced error messages help identify the problem quickly
        fmt.Printf("Handshake failed: %v\n", err)

        // Possible error messages:
        //  "transport send: network timeout"
        //  "nil response from server"
        //  "handshake failed: authentication failed"
        //  "handshake failed: Invalid signature"
        //  "handshake failed: no error details provided"
        //  "empty response data"

        return
    }

    // Success
    fmt.Println("Handshake completed successfully")
}
```

### Error Types and Debugging

```go
package main

import (
    "errors"
    "strings"
)

func handleHandshakeError(err error) {
    if err == nil {
        return
    }

    errMsg := err.Error()

    switch {
    case strings.Contains(errMsg, "transport send"):
        // Network issue - check connectivity
        fmt.Println("Network error: check firewall, DNS, or endpoint URL")

    case strings.Contains(errMsg, "authentication failed"):
        // DID or signature issue - check keys and DID resolution
        fmt.Println("Auth error: verify DID registration and key pairs")

    case strings.Contains(errMsg, "Invalid signature"):
        // Signature verification failed - check key matching
        fmt.Println("Signature error: ensure public key matches DID")

    case strings.Contains(errMsg, "empty response"):
        // Server returned empty data - check server implementation
        fmt.Println("Empty response: server may have crashed or timed out")

    default:
        fmt.Printf("Unexpected error: %v\n", err)
    }
}
```

### Benefits

-  **Clear diagnostics**: Know exactly what went wrong
-  **Faster debugging**: Specific error messages guide troubleshooting
-  **Production-ready**: Better logging for monitoring and alerting

---

## 4. DID X25519 Key Support

Support for X25519 elliptic curve cryptography in DID key management.

### Basic Usage

```go
package main

import (
    "crypto/ecdh"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

func main() {
    // X25519 key bytes (32 bytes)
    x25519KeyBytes := []byte{
        0x50, 0x4a, 0x36, 0x99, 0x9f, 0x48, 0x9c, 0xd2,
        0xfd, 0xbc, 0x08, 0xba, 0xff, 0x3d, 0x88, 0xfa,
        0x00, 0x56, 0x9d, 0x94, 0x71, 0x66, 0x3f, 0x27,
        0x98, 0x39, 0xd9, 0x71, 0xd4, 0x87, 0x23, 0x6d,
    }

    // Unmarshal X25519 key
    publicKey, err := did.UnmarshalPublicKey(x25519KeyBytes, "x25519")
    if err != nil {
        // Enhanced error message shows actual size
        // "invalid X25519 public key size: expected 32 bytes, got 16"
        panic(err)
    }

    // Use with HPKE (convert to ecdh.PublicKey)
    keyBytes := publicKey.([]byte)
    x25519 := ecdh.X25519()
    hpkeKey, err := x25519.NewPublicKey(keyBytes)
    if err != nil {
        panic(err)
    }

    // Use hpkeKey for encryption
}
```

### Integration with DID Registry

```go
package main

import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

func main() {
    // Resolve agent's X25519 key from blockchain
    client, _ := ethereum.NewAgentCardRegistryClient(rpcURL, contractAddr, privateKey)

    agentDID := did.AgentDID("did:sage:ethereum:0xAddress")

    // Get X25519 public key from registry
    keyBytes, keyType, err := client.GetPublicKey(context.Background(), agentDID, "x25519")
    if err != nil {
        panic(err)
    }

    // Unmarshal the key
    x25519Key, err := did.UnmarshalPublicKey(keyBytes, keyType)
    if err != nil {
        panic(err)
    }

    // Use for HPKE encryption
    // ... HPKE operations ...
}
```

### Error Handling

```go
package main

import (
    "fmt"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

func validateX25519Key(keyBytes []byte) error {
    _, err := did.UnmarshalPublicKey(keyBytes, "x25519")
    if err != nil {
        // Error messages are descriptive
        fmt.Printf("Key validation failed: %v\n", err)

        // Example errors:
        //  "invalid X25519 public key size: expected 32 bytes, got 0"
        //  "invalid X25519 public key size: expected 32 bytes, got 64"

        return err
    }

    return nil
}
```

### Benefits

-  **HPKE support**: Use X25519 keys for hybrid encryption
-  **Memory safety**: Returns copy of key bytes, not reference
-  **Better errors**: Actual key size shown in error messages
-  **Registry integration**: Seamless blockchain key resolution

---

## Complete Example: Secure Agent-to-Agent Communication

```go
package main

import (
    "bytes"
    "context"
    "crypto/ecdh"
    "fmt"
    "net/http"

    "github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
    "github.com/sage-x-project/sage/pkg/agent/hpke"
    "github.com/sage-x-project/sage/pkg/agent/transport"
)

func main() {
    // 1. Generate Ethereum-compatible keys
    signingKey, _ := crypto.GenerateSecp256k1KeyPair()

    // 2. Generate X25519 KEM key for HPKE
    x25519Curve := ecdh.X25519()
    kemPrivate, _ := x25519Curve.GenerateKey(rand.Reader)
    kemKey := crypto.NewKeyPairFromECDH(kemPrivate)

    // 3. Derive Ethereum address and create DID
    ethAddress, _ := did.DeriveEthereumAddress(signingKey)
    agentDID := did.GenerateAgentDIDWithAddress(did.ChainEthereum, ethAddress)
    // Result: "did:sage:ethereum:0x742d35cc6634c0532925a3b844bc9e7595f0beef"

    // 4. Register agent on blockchain (with both keys)
    client, _ := ethereum.NewAgentCardRegistryClient(rpcURL, contractAddr, privateKey)

    signingKeyBytes, _ := did.MarshalPublicKey(signingKey.PublicKey())
    kemKeyBytes := kemPrivate.PublicKey().Bytes()

    err := client.Register(context.Background(), agentDID, map[string][]byte{
        "secp256k1": signingKeyBytes,
        "x25519":    kemKeyBytes,
    })
    if err != nil {
        panic(err)
    }

    // 5. Create HTTP request with body integrity
    body := []byte(`{"action":"transfer","amount":100}`)
    req, _ := http.NewRequest("POST", "https://agent.example.com/api", bytes.NewReader(body))

    // Compute and set Content-Digest (prevents body tampering)
    digest := rfc9421.ComputeContentDigest(body)
    req.Header.Set("Content-Digest", digest)

    // 6. Sign the request (RFC 9421)
    signer := rfc9421.NewHTTPSigner()
    params := &rfc9421.SignatureInputParams{
        KeyID:              string(agentDID),
        Algorithm:          rfc9421.AlgorithmECDSASecp256k1,
        CoveredComponents:  []string{"content-digest", "@method", "@authority", "@path"},
        Created:            time.Now(),
    }

    err = signer.SignRequest(req, "sig1", params, signingKey)
    if err != nil {
        panic(err)
    }

    // 7. Setup HPKE client for handshake
    resolver, _ := ethereum.NewEthereumResolver(rpcURL, contractAddr)

    transport := transport.NewHTTPTransport("https://agent.example.com")

    hpkeClient := hpke.NewClient(
        signingKey,  // ECDSA Secp256k1 signing key
        transport,
        resolver,
        &hpke.ClientOpts{
            KEM: kemKey,  // X25519 KEM key
        },
    )

    // 8. Complete HPKE handshake with recipient
    recipientDID := did.AgentDID("did:sage:ethereum:0xRecipientAddress")

    resp, err := hpkeClient.CompleteHandshake(context.Background(), agentDID, recipientDID)
    if err != nil {
        // Enhanced error messages help debug
        fmt.Printf("Handshake failed: %v\n", err)
        return
    }

    fmt.Println(" Secure communication established!")
    fmt.Printf("Session ID: %s\n", resp.SessionID)
    fmt.Printf("Key ID: %s\n", resp.KeyID)

    // 9. Send encrypted message using established session
    // ... encryption logic ...
}
```

---

## Testing Your Implementation

Run the comprehensive test suite to verify functionality:

```bash
# Test RFC9421 body integrity
go test ./pkg/agent/core/rfc9421 -v -run TestBodyIntegrity

# Test HPKE ECDSA signatures
go test ./pkg/agent/hpke -v -run TestSignatureVerifier

# Test X25519 key support
go test ./pkg/agent/did -v -run TestUnmarshalPublicKey_X25519

# Run all security tests
go test ./pkg/agent/core/rfc9421 -v -run EdgeCases
go test ./pkg/agent/core/rfc9421 -v -run SecurityCases
go test ./pkg/agent/hpke -v -run ErrorHandling

# Check test coverage
go test ./pkg/agent/core/rfc9421 -coverprofile=coverage.out
go tool cover -func=coverage.out
```

---

## Architecture Benefits

### SOLID Principles
-  **Single Responsibility**: Each component has one clear purpose
-  **Open/Closed**: New algorithms can be added without modifying existing code
-  **Liskov Substitution**: SignatureVerifier implementations are interchangeable
-  **Interface Segregation**: Small, focused interfaces
-  **Dependency Inversion**: Depend on abstractions, not concrete implementations

### Design Patterns
-  **Strategy Pattern**: Algorithm selection (BodyIntegrityValidator, SignatureVerifier)
-  **Composite Pattern**: Multiple verifiers in CompositeVerifier
-  **Factory Pattern**: NewBodyIntegrityValidator(), NewCompositeVerifier()

---

## Migration from PR #118

If you're migrating from the original PR #118 implementation, **no code changes are required**. The current implementation is 100% backward compatible while offering additional features:

### What Stays the Same
-  All public APIs unchanged
-  Same function signatures
-  Same error messages
-  Same behavior

### What's Better
-  Independent module usage (optional)
-  Better error messages with actual sizes
-  Strategy Pattern for extensibility
-  100+ edge case tests
-  Security validation (timing attacks, replay attacks)

---

## Additional Resources

- **RFC 9421**: [HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- **HPKE**: [Hybrid Public Key Encryption](https://www.rfc-editor.org/rfc/rfc9180.html)
- **X25519**: [Curve25519](https://www.rfc-editor.org/rfc/rfc7748.html)
- **Ethereum Keys**: [Secp256k1 Signatures](https://ethereum.org/en/developers/docs/accounts/)

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Documentation: https://docs.sage-x.org
