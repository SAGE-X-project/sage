# Cryptoinit Package

## Overview

The `cryptoinit` package provides initialization logic for the SAGE cryptographic subsystem. It registers key generators, storage implementations, and format constructors with the `pkg/agent/crypto` package during application startup.

This package uses Go's `init()` function to perform one-time registration of cryptographic components, making them available throughout the SAGE application.

## Purpose

- **Centralized Initialization**: Single point for all crypto subsystem initialization
- **Dependency Registration**: Registers key generators, storage backends, and key format handlers
- **Early Setup**: Runs before `main()` to ensure crypto components are ready
- **Decoupling**: Separates initialization logic from core crypto implementations

## Architecture

```
┌─────────────────────────────────────────────┐
│         Application Startup                 │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│    internal/cryptoinit/init.go              │
│    - Register key generators                │
│    - Register storage constructors          │
│    - Register format constructors           │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│    pkg/agent/crypto                         │
│    - Crypto registry ready                  │
│    - All components available               │
└─────────────────────────────────────────────┘
```

## Registered Components

### Key Generators

The package registers three key generation algorithms:

1. **Ed25519** - EdDSA signatures (RFC 8032)
   - Public key: 32 bytes
   - Private key: 32 bytes
   - Signature: 64 bytes
   - Use case: DID signatures, general-purpose signing

2. **Secp256k1** - ECDSA signatures (Bitcoin/Ethereum)
   - Public key: 65 bytes (uncompressed) or 33 bytes (compressed)
   - Private key: 32 bytes
   - Use case: Blockchain integration, Ethereum compatibility

3. **P-256** - NIST P-256 ECDSA signatures
   - Public key: 65 bytes (uncompressed) or 33 bytes (compressed)
   - Private key: 32 bytes
   - Use case: FIPS compliance, enterprise requirements

### Storage Constructors

1. **MemoryKeyStorage** - In-memory key storage
   - Fast, non-persistent storage
   - Use case: Testing, temporary keys

### Format Constructors

The package registers importers and exporters for key serialization:

1. **JWK (JSON Web Key)** - RFC 7517
   - Exporter: `formats.NewJWKExporter()`
   - Importer: `formats.NewJWKImporter()`
   - Use case: Web APIs, interoperability

2. **PEM (Privacy-Enhanced Mail)** - RFC 7468
   - Exporter: `formats.NewPEMExporter()`
   - Importer: `formats.NewPEMImporter()`
   - Use case: Traditional key files, compatibility

## Usage

### Automatic Initialization

Simply import the package to trigger initialization:

```go
import (
    _ "github.com/sage-x-project/sage/internal/cryptoinit"
)
```

The `init()` function runs automatically and registers all components.

### Verification

To verify that initialization succeeded:

```go
package main

import (
    "fmt"
    "log"

    _ "github.com/sage-x-project/sage/internal/cryptoinit"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
)

func main() {
    // Try generating a key pair
    kp, err := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)
    if err != nil {
        log.Fatal("Crypto not initialized:", err)
    }

    fmt.Printf("Successfully generated %s key pair\n", kp.Type())
}
```

### Advanced: Custom Registration

If you need to register additional components at runtime:

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

func init() {
    // Register custom key generator
    crypto.RegisterKeyGenerator(
        "my-algorithm",
        func() (crypto.KeyPair, error) {
            return keys.GenerateCustomKeyPair()
        },
    )
}
```

## Design Decisions

### Why `init()` Function?

- **Early Initialization**: Runs before `main()`, ensuring components are ready
- **Idempotent**: Safe to import multiple times
- **Declarative**: Clear dependency declaration through imports

### Why Separate Package?

- **Separation of Concerns**: Keeps initialization logic separate from crypto implementations
- **Internal Package**: Not exposed to external users
- **Testability**: Can be tested independently
- **Circular Dependency Prevention**: Breaks potential import cycles

### Registration Pattern

The package uses a registration pattern (similar to `database/sql` drivers):

1. Core package (`pkg/agent/crypto`) defines interfaces
2. Implementations (`pkg/agent/crypto/keys`, `pkg/agent/crypto/storage`) provide concrete types
3. Init package (`internal/cryptoinit`) registers implementations

Benefits:
- **Loose Coupling**: Core doesn't depend on implementations
- **Extensibility**: Easy to add new algorithms
- **Testing**: Can swap implementations in tests

## File Structure

```
internal/cryptoinit/
├── README.md       # This file
└── init.go         # Registration logic
```

## Dependencies

### Direct Dependencies
- `github.com/sage-x-project/sage/pkg/agent/crypto` - Core crypto interfaces
- `github.com/sage-x-project/sage/pkg/agent/crypto/keys` - Key generation implementations
- `github.com/sage-x-project/sage/pkg/agent/crypto/storage` - Key storage implementations
- `github.com/sage-x-project/sage/pkg/agent/crypto/formats` - Key format implementations

### No External Dependencies
This package intentionally has no external dependencies beyond the standard library and SAGE packages.

## Testing

The cryptoinit package is tested through integration tests:

```bash
# Run tests that verify initialization
go test github.com/sage-x-project/sage/pkg/agent/crypto/...

# Verify all key types can be generated
go test -v -run TestKeyGeneration
```

## Common Issues

### Import Not Recognized

If crypto components aren't available:

```go
//  Wrong - package not imported
package main

import "github.com/sage-x-project/sage/pkg/agent/crypto"

func main() {
    crypto.GenerateKeyPair(crypto.KeyTypeEd25519) // May fail
}

//  Correct - explicit import
package main

import (
    _ "github.com/sage-x-project/sage/internal/cryptoinit"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
)

func main() {
    crypto.GenerateKeyPair(crypto.KeyTypeEd25519) // Works
}
```

### Multiple Registrations

The package is safe to import multiple times (Go only runs `init()` once per package):

```go
// Safe - init() only runs once
import (
    _ "github.com/sage-x-project/sage/internal/cryptoinit"
    "github.com/sage-x-project/sage/cmd/sage-crypto" // Also imports cryptoinit
)
```

## Best Practices

### 1. Always Use Blank Import

```go
//  Correct - blank import for side effects
import _ "github.com/sage-x-project/sage/internal/cryptoinit"

//  Wrong - no need for named import
import cryptoinit "github.com/sage-x-project/sage/internal/cryptoinit"
```

### 2. Import Early

Import in your application's entry point (e.g., `cmd/sage-server/main.go`):

```go
package main

import (
    _ "github.com/sage-x-project/sage/internal/cryptoinit" // First
    "github.com/sage-x-project/sage/pkg/agent"
    // ... other imports
)
```

### 3. Don't Call Functions Directly

This package has no public API - all work is done in `init()`:

```go
//  Wrong - package has no public functions
cryptoinit.Initialize()

//  Correct - just import
import _ "github.com/sage-x-project/sage/internal/cryptoinit"
```

## Integration Points

### Command-Line Tools

All SAGE CLI tools import this package:

- `cmd/sage-crypto/main.go`
- `cmd/sage-did/main.go`
- `cmd/sage-server/main.go`

### Tests

Test files that use crypto should import:

```go
package mypackage_test

import (
    "testing"

    _ "github.com/sage-x-project/sage/internal/cryptoinit"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
)

func TestCryptoOperation(t *testing.T) {
    kp, err := crypto.GenerateKeyPair(crypto.KeyTypeEd25519)
    // ...
}
```

## Related Packages

- `pkg/agent/crypto` - Core crypto interfaces and registry
- `pkg/agent/crypto/keys` - Key generation implementations
- `pkg/agent/crypto/storage` - Key storage backends
- `pkg/agent/crypto/formats` - Key serialization formats

## References

- [RFC 8032 - Edwards-Curve Digital Signature Algorithm (EdDSA)](https://tools.ietf.org/html/rfc8032)
- [RFC 7517 - JSON Web Key (JWK)](https://tools.ietf.org/html/rfc7517)
- [RFC 7468 - Textual Encodings of PKIX, PKCS, and CMS Structures](https://tools.ietf.org/html/rfc7468)
- [Go init() function documentation](https://go.dev/doc/effective_go#init)
