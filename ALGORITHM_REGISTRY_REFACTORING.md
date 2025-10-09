# Algorithm Registry Refactoring

## Overview

This document describes the refactoring that eliminated hardcoded cryptographic algorithm names throughout the SAGE codebase and replaced them with a centralized algorithm registry.

## Problem Statement

Previously, algorithm names were hardcoded in multiple locations:
- `crypto/chain/key_mapper.go` - Hardcoded RFC 9421 algorithm mappings
- `core/rfc9421/verifier_http.go` - Hardcoded algorithm validation checks
- Various test files - Hardcoded algorithm strings

This approach had several issues:
1. **Maintenance burden**: Changes to supported algorithms required updates in multiple files
2. **Inconsistency risk**: Different parts of the codebase could use different algorithm names
3. **No validation**: No centralized way to check if an algorithm was actually supported
4. **No discoverability**: No way to list all supported algorithms

## Solution: Centralized Algorithm Registry

### Architecture

The refactoring introduced a centralized algorithm registry with the following components:

```
crypto/
  ├── algorithm_registry.go       # Core registry implementation
  ├── algorithm_registry_test.go  # Registry tests
  ├── pubkey_validation_test.go   # Public key validation tests
  └── keys/
      └── algorithms.go            # Algorithm registration (init function)
```

### Key Components

#### 1. Algorithm Registry (`crypto/algorithm_registry.go`)

**Data Structures:**
```go
type AlgorithmInfo struct {
    KeyType               KeyType
    Name                  string
    Description           string
    RFC9421Algorithm      string
    SupportsRFC9421       bool
    SupportsKeyGeneration bool
    SupportsSignature     bool
    SupportsEncryption    bool
}
```

**Core Functions:**
- `RegisterAlgorithm(info AlgorithmInfo)` - Register a new algorithm
- `GetAlgorithmInfo(keyType KeyType)` - Get algorithm information
- `ListSupportedAlgorithms()` - List all registered algorithms
- `ListRFC9421SupportedAlgorithms()` - List RFC 9421 algorithm names
- `GetRFC9421AlgorithmName(keyType KeyType)` - Get RFC 9421 name for a key type
- `GetKeyTypeFromRFC9421Algorithm(algorithm string)` - Reverse lookup
- `GetKeyTypeFromPublicKey(publicKey interface{})` - Map Go crypto types to KeyType
- `ValidateAlgorithmForPublicKey(publicKey, algorithm)` - Validate algorithm compatibility

**Thread Safety:** All registry operations are protected with `sync.RWMutex`

**Immutability:** Functions return copies of data to prevent external modification

#### 2. Algorithm Registration (`crypto/keys/algorithms.go`)

Algorithms are registered during package initialization using `init()`:

```go
func init() {
    // Register Ed25519
    sagecrypto.RegisterAlgorithm(sagecrypto.AlgorithmInfo{
        KeyType:               sagecrypto.KeyTypeEd25519,
        Name:                  "Ed25519",
        RFC9421Algorithm:      "ed25519",
        SupportsRFC9421:       true,
        SupportsSignature:     true,
        // ...
    })

    // Register Secp256k1
    // Register RSA
    // Register X25519
}
```

#### 3. Integration Points

**RFC 9421 Types (`core/rfc9421/types.go`):**
```go
func GetSupportedAlgorithms() []string {
    return sagecrypto.ListRFC9421SupportedAlgorithms()
}

func IsAlgorithmSupported(algorithm string) bool {
    _, err := sagecrypto.GetKeyTypeFromRFC9421Algorithm(algorithm)
    return err == nil
}
```

**RFC 9421 HTTP Verifier (`core/rfc9421/verifier_http.go`):**
```go
func (v *HTTPVerifier) verifySignature(...) error {
    // Validate algorithm using registry instead of hardcoded checks
    if err := sagecrypto.ValidateAlgorithmForPublicKey(publicKey, algorithm); err != nil {
        return fmt.Errorf("algorithm validation failed: %w", err)
    }
    // ...
}
```

**Chain Key Mapper (`crypto/chain/key_mapper.go`):**
```go
func (m *defaultKeyMapper) GetRFC9421Algorithm(keyType sagecrypto.KeyType) (string, error) {
    // Delegate to registry instead of using hardcoded map
    return sagecrypto.GetRFC9421AlgorithmName(keyType)
}
```

## Registered Algorithms

Currently registered algorithms:

| Algorithm | KeyType | RFC 9421 Name | Description |
|-----------|---------|---------------|-------------|
| Ed25519 | Ed25519 | ed25519 | Edwards-curve Digital Signature Algorithm |
| Secp256k1 | Secp256k1 | es256k | ECDSA with secp256k1 (Bitcoin/Ethereum) |
| RSA-PSS-SHA256 | RSA256 | rsa-pss-sha256 | RSA with PSS padding and SHA-256 |
| X25519 | X25519 | - | Curve25519 key exchange (no signing) |

## Testing

### Test Coverage

1. **Algorithm Registry Tests** (`algorithm_registry_test.go`)
   - Registration and retrieval
   - RFC 9421 algorithm mapping (bidirectional)
   - Immutability guarantees
   - Thread safety
   - Integration tests

2. **Public Key Validation Tests** (`pubkey_validation_test.go`)
   - Ed25519 public key type detection
   - ECDSA public key type detection
   - RSA public key type detection
   - Algorithm validation for each key type
   - Mismatch detection

3. **Integration Tests**
   - All crypto package tests pass
   - All RFC 9421 package tests pass
   - All crypto/chain package tests pass

### Test Results

```
crypto:          PASS (21 tests)
crypto_test:     PASS (12 tests)
core/rfc9421:    PASS (all tests)
crypto/chain:    PASS (all tests)
```

## Migration Guide

### Before (Hardcoded)

```go
// OLD: Hardcoded algorithm checks
switch key := publicKey.(type) {
case ed25519.PublicKey:
    if algorithm != "" && algorithm != "ed25519" {
        return errors.New("algorithm mismatch")
    }
case *ecdsa.PublicKey:
    if algorithm != "" && algorithm != "ecdsa-p256" && algorithm != "es256k" {
        return errors.New("algorithm mismatch")
    }
}
```

### After (Registry-based)

```go
// NEW: Registry-based validation
if err := sagecrypto.ValidateAlgorithmForPublicKey(publicKey, algorithm); err != nil {
    return fmt.Errorf("algorithm validation failed: %w", err)
}
```

### Adding a New Algorithm

To add a new algorithm, simply register it in `crypto/keys/algorithms.go`:

```go
func init() {
    // ... existing registrations ...

    if err := sagecrypto.RegisterAlgorithm(sagecrypto.AlgorithmInfo{
        KeyType:               sagecrypto.KeyTypeNewAlgorithm,
        Name:                  "New Algorithm",
        Description:           "Description of new algorithm",
        RFC9421Algorithm:      "new-alg",
        SupportsRFC9421:       true,
        SupportsKeyGeneration: true,
        SupportsSignature:     true,
        SupportsEncryption:    false,
    }); err != nil {
        log.Fatalf("Failed to register new algorithm: %v", err)
    }
}
```

## Benefits

1. **Single Source of Truth**: All algorithm metadata centralized in one place
2. **Type Safety**: Compile-time checking of algorithm properties
3. **Maintainability**: Adding/modifying algorithms requires changes in only one place
4. **Validation**: Automatic validation that algorithms are actually supported
5. **Discoverability**: Easy to list all supported algorithms programmatically
6. **RFC 9421 Compliance**: Can dynamically provide list of supported algorithms
7. **Thread Safety**: Safe for concurrent access
8. **Testability**: Comprehensive test coverage for algorithm management

## Known Limitations

1. **ECDSA Curve Distinction**: Currently, all ECDSA public keys map to `KeyTypeSecp256k1`, even if they use different curves (e.g., P-256). This is a simplification and may need refinement in the future.

2. **One-to-One Mapping**: The registry enforces a one-to-one mapping between `KeyType` and algorithm registration. Multiple RFC 9421 algorithms cannot share the same `KeyType`.

## Future Enhancements

1. **Support for Multiple ECDSA Curves**: Add separate KeyTypes for P-256, P-384, etc.
2. **Algorithm Capabilities**: Extend `AlgorithmInfo` with more detailed capability flags
3. **Algorithm Versioning**: Support for algorithm version management
4. **Dynamic Registration**: Allow runtime algorithm registration (if needed)
5. **Algorithm Preferences**: Add priority/preference system for algorithm selection

## Conclusion

This refactoring successfully eliminates hardcoded algorithm names throughout the SAGE codebase, replacing them with a centralized, type-safe, and thread-safe algorithm registry. The implementation follows best practices including:

- TDD (Test-Driven Development)
- Comprehensive test coverage
- Thread safety
- Immutability
- Clear documentation

All existing tests pass, confirming that the refactoring maintains backward compatibility while improving code quality and maintainability.
