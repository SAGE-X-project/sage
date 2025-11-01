# SAGE Cryptographic Operations

The `crypto` package provides comprehensive cryptographic key management for SAGE's secure agent communication infrastructure. It offers multi-algorithm support, flexible key storage backends, and blockchain-specific providers for decentralized identity management.

## Overview

SAGE requires robust cryptographic operations for agent authentication, message signing, and secure communication. The crypto package abstracts these operations behind clean interfaces, supporting multiple algorithms and storage backends while maintaining security best practices.

### Key Benefits

- **Multi-Algorithm Support**: Ed25519, Secp256k1 (Ethereum), X25519 (HPKE), RS256
- **Flexible Storage**: Memory, file-based, and OS keychain integration via Vault
- **Blockchain Ready**: Native Ethereum and Solana provider support
- **Format Agnostic**: Import/export keys in JWK and PEM formats
- **Production Secure**: File permissions (0600), secure key rotation, hardware-backed storage

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  SAGE Components (handshake, session, did)              │
│  - Agent authentication                                  │
│  - Message signing                                       │
│  - DID operations                                        │
└────────────────────┬────────────────────────────────────┘
                     │ uses
                     ▼
┌─────────────────────────────────────────────────────────┐
│  crypto.Manager (centralized key management)            │
│  - GenerateKeyPair()                                     │
│  - StoreKeyPair() / LoadKeyPair()                       │
│  - ExportKeyPair() / ImportKeyPair()                    │
└────────────────────┬────────────────────────────────────┘
                     │ manages
          ┌──────────┴──────────┬──────────────────┐
          ▼                     ▼                  ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐
│ KeyPair          │  │ KeyStorage       │  │ Key Formats  │
│ Interface        │  │ Interface        │  │              │
├──────────────────┤  ├──────────────────┤  ├──────────────┤
│ • Ed25519        │  │ • Memory         │  │ • JWK        │
│ • Secp256k1      │  │ • File           │  │ • PEM        │
│ • X25519         │  │ • Vault (OS)     │  │              │
│ • RS256          │  │                  │  │              │
└──────────────────┘  └──────────────────┘  └──────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────┐
│  Chain Providers (blockchain-specific operations)       │
│  ├─ Ethereum Provider (ECDSA/Secp256k1)                 │
│  └─ Solana Provider (Ed25519)                           │
└─────────────────────────────────────────────────────────┘
```

## Supported Algorithms

### Ed25519 (Edwards-curve Digital Signature Algorithm)

**Use Cases:**
- DID document signing
- Agent authentication
- Solana blockchain operations

**Properties:**
- **Key Size**: 32 bytes (private), 32 bytes (public)
- **Signature Size**: 64 bytes
- **Performance**: ~40-80 μs signing, ~100-200 μs verification
- **Security**: 128-bit security level
- **RFC 9421**: Supported (`ed25519` algorithm)

**Standards:**
- RFC 8032 (EdDSA: Ed25519 and Ed448)
- RFC 8037 (JWK for CFRG curves)

### Secp256k1 (ECDSA with Bitcoin/Ethereum curve)

**Use Cases:**
- Ethereum DID registration
- Ethereum smart contract interaction
- Blockchain transaction signing

**Properties:**
- **Key Size**: 32 bytes (private), 33 bytes (compressed) or 65 bytes (uncompressed)
- **Signature Size**: 64-65 bytes (with recovery ID for Ethereum)
- **Performance**: ~60-120 μs signing, ~150-300 μs verification
- **Security**: 128-bit security level
- **RFC 9421**: Supported (`es256k` algorithm)

**Standards:**
- SEC 2 v2.0 (Secp256k1 curve)
- EIP-191 (Ethereum Signed Message)

### X25519 (Curve25519 for ECDH)

**Use Cases:**
- HPKE key agreement (RFC 9180)
- Secure session establishment
- Forward secrecy

**Properties:**
- **Key Size**: 32 bytes (private), 32 bytes (public)
- **Performance**: ~30-60 μs key generation, ~60-80 μs shared secret derivation
- **Security**: 128-bit security level
- **Note**: Encryption only, no signing

**Standards:**
- RFC 7748 (Curve25519 and Curve448)
- RFC 9180 (HPKE - Hybrid Public Key Encryption)

### RS256 (RSA-SHA256)

**Use Cases:**
- Legacy system integration
- JWT token signing

**Properties:**
- **Key Size**: 2048-4096 bits
- **Performance**: Slower than elliptic curve algorithms
- **Security**: Depends on key size
- **RFC 9421**: Supported (`rsa-v1_5-sha256` algorithm)

**Note**: Ed25519 or Secp256k1 recommended for new implementations.

## Core Components

### Manager

Centralized cryptographic operations manager:

```go
type Manager struct {
    storage KeyStorage
}

// Core operations
func NewManager() *Manager
func (m *Manager) GenerateKeyPair(keyType KeyType) (KeyPair, error)
func (m *Manager) StoreKeyPair(keyPair KeyPair) error
func (m *Manager) LoadKeyPair(id string) (KeyPair, error)
func (m *Manager) ExportKeyPair(keyPair KeyPair, format KeyFormat) ([]byte, error)
func (m *Manager) ImportKeyPair(data []byte, format KeyFormat) (KeyPair, error)
```

**Features:**
- Pluggable storage backends
- Multi-algorithm support
- Format conversion (JWK, PEM)
- Key lifecycle management

### KeyPair Interface

Unified interface for all key types:

```go
type KeyPair interface {
    PublicKey() crypto.PublicKey
    PrivateKey() crypto.PrivateKey
    Type() KeyType
    Sign(message []byte) ([]byte, error)
    Verify(message, signature []byte) error
    ID() string
}
```

**Implementations:**
- `Ed25519KeyPair` - Edwards curve signing
- `Secp256k1KeyPair` - Ethereum-compatible ECDSA
- `X25519KeyPair` - ECDH key agreement
- `RSAKeyPair` - RSA signing (legacy)

### KeyStorage Interface

Pluggable storage backends:

```go
type KeyStorage interface {
    Store(id string, keyPair KeyPair) error
    Load(id string) (KeyPair, error)
    Delete(id string) error
    List() ([]string, error)
}
```

**Available Backends:**

#### 1. Memory Storage (Testing)
- **Package**: `github.com/sage-x-project/sage/pkg/agent/crypto/storage`
- **Use Case**: Unit tests, temporary keys
- **Persistence**: None (RAM only)

```go
import "github.com/sage-x-project/sage/pkg/agent/crypto/storage"

storage := storage.NewMemoryKeyStorage()
```

#### 2. File Storage (Development)
- **Package**: `github.com/sage-x-project/sage/pkg/agent/crypto/storage`
- **Use Case**: Development, local deployments
- **Persistence**: Encrypted files with 0600 permissions
- **Location**: Configurable directory

```go
import "github.com/sage-x-project/sage/pkg/agent/crypto/storage"

storage := storage.NewFileKeyStorage("/path/to/keys")
```

**Security:**
- Files stored with 0600 permissions (owner read/write only)
- Keys encrypted at rest
- Configurable directory location

#### 3. Vault Storage (Production)
- **Package**: `github.com/sage-x-project/sage/pkg/agent/crypto/vault`
- **Use Case**: Production deployments
- **Persistence**: OS keychain (Keychain on macOS, GNOME Keyring on Linux, Credential Manager on Windows)
- **Hardware**: Supports hardware-backed secure enclaves (e.g., Secure Enclave on macOS)

```go
import "github.com/sage-x-project/sage/pkg/agent/crypto/vault"

storage := vault.NewSecureStorage("sage-agent-keys")
```

**Security:**
- Hardware-backed encryption (when available)
- OS-level access control
- Tamper-resistant storage
- Automatic key escrow and recovery

### Key Formats

#### JWK (JSON Web Key) - RFC 7517

**Use Cases:**
- Web APIs and JWT tokens
- Cross-platform key exchange
- Standard-compliant key storage

**Example (Ed25519):**
```json
{
  "kty": "OKP",
  "crv": "Ed25519",
  "x": "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
  "d": "nWGxne_9WmC6hEr0kuwsxERJxWl7MmkZcDusAxyuf2A"
}
```

**Example (Secp256k1):**
```json
{
  "kty": "EC",
  "crv": "secp256k1",
  "x": "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
  "y": "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
  "d": "870MB6gfuTJ4HtUnUvYMyJpr5eUZNP4Bk43bVdj3eAE"
}
```

#### PEM (Privacy-Enhanced Mail)

**Use Cases:**
- OpenSSL compatibility
- Traditional Unix systems
- Certificate infrastructure

**Example (Ed25519 Private Key):**
```
-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEINVhsZ3v/VpguoRK9JLsrMREScVpezJpGXA7rAMcrn9g
-----END PRIVATE KEY-----
```

**Example (Ed25519 Public Key):**
```
-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEANVqAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURo=
-----END PUBLIC KEY-----
```

### Chain Providers

Blockchain-specific cryptographic operations:

#### Ethereum Provider

**Package**: `github.com/sage-x-project/sage/pkg/agent/crypto/chain/ethereum`

**Features:**
- Secp256k1 key generation
- Ethereum address derivation (Keccak256 hash)
- EIP-191 message signing
- Transaction signing
- Enhanced provider with gas estimation

**Usage:**
```go
import "github.com/sage-x-project/sage/pkg/agent/crypto/chain/ethereum"

provider := ethereum.NewProvider()
keyPair, _ := provider.GenerateKeyPair()
address := provider.DeriveAddress(keyPair.PublicKey())
// Result: "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
```

#### Solana Provider

**Package**: `github.com/sage-x-project/sage/pkg/agent/crypto/chain/solana`

**Features:**
- Ed25519 key generation
- Base58 address encoding
- Transaction signing

**Usage:**
```go
import "github.com/sage-x-project/sage/pkg/agent/crypto/chain/solana"

provider := solana.NewProvider()
keyPair, _ := provider.GenerateKeyPair()
address := provider.DeriveAddress(keyPair.PublicKey())
// Result: "CuieVDEDtLo7FypA9SbLM9saXFdb1dsshEkyErMqkRQq"
```

## Usage Examples

### Basic Key Generation

```go
import "github.com/sage-x-project/sage/pkg/agent/crypto"

// Create manager
manager := crypto.NewManager()

// Generate Ed25519 key pair
ed25519Key, err := manager.GenerateKeyPair(crypto.KeyTypeEd25519)
if err != nil {
    log.Fatal(err)
}

// Generate Secp256k1 key pair (Ethereum)
secp256k1Key, err := manager.GenerateKeyPair(crypto.KeyTypeSecp256k1)
if err != nil {
    log.Fatal(err)
}

// Generate X25519 key pair (HPKE)
import "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
x25519Key, err := keys.GenerateX25519KeyPair()
if err != nil {
    log.Fatal(err)
}
```

### Signing and Verification

```go
import "github.com/sage-x-project/sage/pkg/agent/crypto"

manager := crypto.NewManager()
keyPair, _ := manager.GenerateKeyPair(crypto.KeyTypeEd25519)

// Sign message
message := []byte("Hello, SAGE!")
signature, err := keyPair.Sign(message)
if err != nil {
    log.Fatal(err)
}

log.Printf("Signature: %x", signature)

// Verify signature
err = keyPair.Verify(message, signature)
if err != nil {
    log.Fatal("Signature verification failed:", err)
}

log.Println("Signature verified successfully!")
```

### Key Storage and Loading

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/crypto/storage"
)

// Create manager with file storage
manager := crypto.NewManager()
fileStorage := storage.NewFileKeyStorage("./keys")
manager.SetStorage(fileStorage)

// Generate and store key
keyPair, _ := manager.GenerateKeyPair(crypto.KeyTypeEd25519)
err := manager.StoreKeyPair(keyPair)
if err != nil {
    log.Fatal(err)
}

log.Printf("Stored key with ID: %s", keyPair.ID())

// Load key
loadedKey, err := manager.LoadKeyPair(keyPair.ID())
if err != nil {
    log.Fatal(err)
}

log.Println("Key loaded successfully!")
```

### Production: Vault Storage

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/crypto/vault"
)

// Create manager with vault storage (OS keychain)
manager := crypto.NewManager()
vaultStorage := vault.NewSecureStorage("sage-agent-keys")
manager.SetStorage(vaultStorage)

// Generate and securely store key
keyPair, _ := manager.GenerateKeyPair(crypto.KeyTypeSecp256k1)
err := manager.StoreKeyPair(keyPair)
if err != nil {
    log.Fatal(err)
}

// Key is now stored in OS keychain (hardware-backed if available)
log.Println("Key securely stored in OS keychain")
```

### Key Format Conversion

#### Export to JWK

```go
import "github.com/sage-x-project/sage/pkg/agent/crypto"

manager := crypto.NewManager()
keyPair, _ := manager.GenerateKeyPair(crypto.KeyTypeEd25519)

// Export to JWK
jwkData, err := manager.ExportKeyPair(keyPair, crypto.KeyFormatJWK)
if err != nil {
    log.Fatal(err)
}

log.Printf("JWK:\n%s", string(jwkData))
```

#### Export to PEM

```go
import "github.com/sage-x-project/sage/pkg/agent/crypto"

manager := crypto.NewManager()
keyPair, _ := manager.GenerateKeyPair(crypto.KeyTypeSecp256k1)

// Export to PEM
pemData, err := manager.ExportKeyPair(keyPair, crypto.KeyFormatPEM)
if err != nil {
    log.Fatal(err)
}

log.Printf("PEM:\n%s", string(pemData))
```

#### Import from JWK

```go
import "github.com/sage-x-project/sage/pkg/agent/crypto"

manager := crypto.NewManager()

jwkData := []byte(`{
  "kty": "OKP",
  "crv": "Ed25519",
  "x": "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
  "d": "nWGxne_9WmC6hEr0kuwsxERJxWl7MmkZcDusAxyuf2A"
}`)

keyPair, err := manager.ImportKeyPair(jwkData, crypto.KeyFormatJWK)
if err != nil {
    log.Fatal(err)
}

log.Printf("Imported %s key pair", keyPair.Type())
```

### Blockchain Provider Usage

#### Ethereum

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto/chain/ethereum"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Generate Ethereum-compatible key
provider := ethereum.NewProvider()
keyPair, _ := provider.GenerateKeyPair()

// Derive Ethereum address
address := provider.DeriveAddress(keyPair.PublicKey())
log.Printf("Ethereum address: %s", address)

// Generate DID
agentDID := did.GenerateAgentDIDWithAddress(did.ChainEthereum, address)
log.Printf("Agent DID: %s", agentDID)
// Result: "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"

// Sign Ethereum message (EIP-191)
message := []byte("Hello, Ethereum!")
signature, _ := keyPair.Sign(message)
log.Printf("Signature: %x", signature)
```

#### Solana

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto/chain/solana"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Generate Solana-compatible key
provider := solana.NewProvider()
keyPair, _ := provider.GenerateKeyPair()

// Derive Solana address
address := provider.DeriveAddress(keyPair.PublicKey())
log.Printf("Solana address: %s", address)

// Generate DID
agentDID := did.GenerateAgentDIDWithAddress(did.ChainSolana, address)
log.Printf("Agent DID: %s", agentDID)
// Result: "did:sage:solana:CuieVDEDtLo7FypA9SbLM9saXFdb1dsshEkyErMqkRQq"
```

### Key Rotation

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/crypto/rotation"
)

manager := crypto.NewManager()

// Create rotator
rotator := rotation.NewKeyRotator(manager)

// Generate initial key
oldKey, _ := manager.GenerateKeyPair(crypto.KeyTypeEd25519)
manager.StoreKeyPair(oldKey)

// Rotate key
newKey, err := rotator.RotateKey(oldKey.ID(), crypto.KeyTypeEd25519)
if err != nil {
    log.Fatal(err)
}

log.Printf("Rotated from %s to %s", oldKey.ID(), newKey.ID())

// Old key is automatically deleted
// New key is stored and ready to use
```

## Directory Structure

```
pkg/agent/crypto/
├── README.md                    # This file
├── manager.go                   # Crypto manager implementation
├── types.go                     # Core interfaces (KeyPair, KeyStorage, etc.)
├── crypto.go                    # Helper functions and wrappers
├── wrappers.go                  # Convenience wrappers
├── algorithm_registry.go        # Algorithm registration and discovery
│
├── keys/                        # Key implementations
│   ├── algorithms.go            # Algorithm registration (init)
│   ├── constructors.go          # Key pair constructors
│   ├── ed25519.go               # Ed25519 implementation
│   ├── ed25519_test.go          # Ed25519 tests
│   ├── secp256k1.go             # Secp256k1 implementation
│   ├── secp256k1_test.go        # Secp256k1 tests
│   ├── x25519.go                # X25519 implementation
│   ├── x25519_test.go           # X25519 tests
│   ├── rs256.go                 # RSA-SHA256 implementation
│   └── rs256_test.go            # RS256 tests
│
├── storage/                     # Storage backends
│   ├── memory.go                # In-memory storage (testing)
│   ├── memory_test.go           # Memory storage tests
│   ├── file.go                  # File-based storage
│   └── file_test.go             # File storage tests
│
├── vault/                       # Secure storage (OS keychain)
│   ├── secure_storage.go        # Vault implementation
│   └── secure_storage_test.go   # Vault tests
│
├── formats/                     # Key format converters
│   ├── jwk.go                   # JWK import/export
│   ├── jwk_test.go              # JWK tests
│   ├── pem.go                   # PEM import/export
│   └── pem_test.go              # PEM tests
│
├── chain/                       # Blockchain providers
│   ├── types.go                 # Provider interface
│   ├── registry.go              # Provider registry
│   ├── key_mapper.go            # Algorithm to chain mapping
│   ├── ethereum/                # Ethereum provider
│   │   ├── provider.go          # Basic provider
│   │   ├── enhanced_provider.go # Enhanced provider (gas, nonce)
│   │   └── provider_test.go     # Provider tests
│   └── solana/                  # Solana provider
│       ├── provider.go          # Solana provider
│       └── provider_test.go     # Provider tests
│
└── rotation/                    # Key rotation
    ├── rotator.go               # Key rotation logic
    └── rotator_test.go          # Rotation tests
```

## Testing

### Unit Tests

```bash
# Run all crypto tests
go test ./pkg/agent/crypto/...

# Run with coverage
go test -cover ./pkg/agent/crypto/...

# Run specific algorithm tests
go test ./pkg/agent/crypto/keys -run TestEd25519
go test ./pkg/agent/crypto/keys -run TestSecp256k1
```

### Fuzz Testing

```bash
# Run fuzzing (10 seconds)
./tools/scripts/run-fuzz.sh --time 10s --type go

# Run specific fuzzer
go test -fuzz=FuzzSignVerify -fuzztime=1m ./pkg/agent/crypto
```

### Benchmark Tests

```bash
# Run crypto benchmarks
go test -bench=. -benchmem ./tools/benchmark -run=^$ -bench="Key|Sign|Verif"

# Expected results:
# - Ed25519 key generation: ~50-100 μs
# - Ed25519 signing: ~40-80 μs
# - Ed25519 verification: ~100-200 μs
# - Secp256k1 key generation: ~100-200 μs
# - Secp256k1 signing: ~60-120 μs
# - Secp256k1 verification: ~150-300 μs
```

## Security Considerations

### Key Storage

**Development:**
-  Use file storage with 0600 permissions
-  Encrypt keys at rest
-  Never commit keys to version control

**Production:**
-  Use vault storage (OS keychain)
-  Enable hardware-backed encryption
-  Implement key rotation policy
-  Monitor key access logs

### Algorithm Selection

**For DID Signing:**
-  Ed25519 (recommended for new implementations)
-  Secp256k1 (required for Ethereum)
-  RS256 (legacy only, avoid for new systems)

**For Key Agreement:**
-  X25519 (HPKE, RFC 9180)
-  Never use signing keys for encryption

### Best Practices

1. **Key Generation**
   - Use cryptographically secure random number generator
   - Never reuse keys across different purposes
   - Generate new keys for each agent

2. **Signature Verification**
   - Always verify signatures before trusting messages
   - Check message replay (nonce, timestamp)
   - Validate DID ownership

3. **Key Rotation**
   - Rotate keys periodically (e.g., every 90 days)
   - Use atomic rotation to prevent inconsistent states
   - Keep old keys for signature verification during transition

4. **Storage**
   - Encrypt keys at rest
   - Use OS keychain for production
   - Limit key access to necessary processes

## Performance

### Benchmark Results (Apple M1, Go 1.24)

| Operation | Algorithm | Time | Memory |
|-----------|-----------|------|--------|
| Key Generation | Ed25519 | ~50-100 μs | 1-2 KB |
| Key Generation | Secp256k1 | ~100-200 μs | 2-3 KB |
| Key Generation | X25519 | ~30-60 μs | 1 KB |
| Signing | Ed25519 | ~40-80 μs | <1 KB |
| Signing | Secp256k1 | ~60-120 μs | 1-2 KB |
| Verification | Ed25519 | ~100-200 μs | <1 KB |
| Verification | Secp256k1 | ~150-300 μs | 1-2 KB |
| JWK Export | All | ~10-20 μs | 1-2 KB |
| PEM Export | All | ~10-20 μs | 1-2 KB |

### Optimization Tips

1. **Reuse KeyPair objects**: Key pair objects can be reused for multiple operations
2. **Cache public keys**: Public key parsing/validation is expensive
3. **Batch verification**: Verify multiple signatures in parallel
4. **Use X25519 for encryption**: Much faster than RSA for key agreement

## FAQ

### Q: Which algorithm should I use for DID signing?

A: For new implementations, use **Ed25519**:
- Faster than Secp256k1
- Smaller keys and signatures
- Constant-time operations (side-channel resistant)

Use **Secp256k1** only when Ethereum compatibility is required.

### Q: Can I use the same key for signing and encryption?

A: **No**. Use different keys for different purposes:
- **Ed25519/Secp256k1**: Signing and verification
- **X25519**: Key agreement (HPKE)

Never use signing keys for encryption operations.

### Q: How do I securely store keys in production?

A: Use **Vault storage** with OS keychain:
```go
import "github.com/sage-x-project/sage/pkg/agent/crypto/vault"

vaultStorage := vault.NewSecureStorage("sage-agent-keys")
manager.SetStorage(vaultStorage)
```

This provides:
- Hardware-backed encryption (when available)
- OS-level access control
- Tamper-resistant storage

### Q: How often should I rotate keys?

A: **Every 90 days** for production agents:
```go
import "github.com/sage-x-project/sage/pkg/agent/crypto/rotation"

rotator := rotation.NewKeyRotator(manager)
newKey, _ := rotator.RotateKey(oldKeyID, crypto.KeyTypeEd25519)
```

Rotate immediately if:
- Key compromise suspected
- Employee departure
- Security audit recommendation

### Q: What's the difference between JWK and PEM?

**JWK (JSON Web Key):**
- Modern, JSON-based format
- Web API friendly
- Standard for OAuth/JWT
- Example: `{"kty":"OKP","crv":"Ed25519",...}`

**PEM (Privacy-Enhanced Mail):**
- Traditional, base64-encoded format
- OpenSSL compatible
- Unix systems standard
- Example: `-----BEGIN PRIVATE KEY-----`

Use **JWK** for web APIs and **PEM** for traditional systems.

## See Also

- [DID Management](../did/README.md) - Decentralized identity
- [Session Management](../session/README.md) - Secure sessions
- [HPKE Documentation](../hpke/) - Key agreement
- [RFC 9421 Implementation](../core/rfc9421/) - HTTP signatures
- [Transport Layer](../transport/README.md) - Protocol abstraction

## References

- [RFC 8032](https://www.rfc-editor.org/rfc/rfc8032) - EdDSA: Ed25519 and Ed448
- [RFC 7748](https://www.rfc-editor.org/rfc/rfc7748) - Curve25519 and Curve448
- [RFC 7517](https://www.rfc-editor.org/rfc/rfc7517) - JSON Web Key (JWK)
- [RFC 9180](https://www.rfc-editor.org/rfc/rfc9180) - HPKE
- [SEC 2 v2.0](https://www.secg.org/sec2-v2.pdf) - Secp256k1 curve
- [EIP-191](https://eips.ethereum.org/EIPS/eip-191) - Ethereum Signed Message

## License

LGPL-3.0 - See LICENSE file for details.
