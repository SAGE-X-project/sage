# SAGE Crypto Package

A Go package providing cryptographic functionality for the SAGE (Secure Agent Guarantee Engine) project.

## Key Features

- **Key Pair Generation**: Support for Ed25519, Secp256k1, X25519, and RS256 algorithms
- **Key Export/Import**: Support for JWK (JSON Web Key) and PEM formats
- **Secure Key Storage**: Memory and file-based storage
- **Encrypted Key Vault**: AES-256-GCM encrypted storage via Vault
- **Key Rotation**: Automatic key rotation with history management
- **Message Signing and Verification**: Digital signature generation and verification
- **ECDH Key Exchange**: X25519-based key agreement and HPKE support
- **RFC 9421 Support**: Compliance with HTTP Message Signatures standard
- **Blockchain Integration**: Ethereum and Solana address generation and validation

## Installation

```bash
go get github.com/sage-x-project/sage/crypto
```

## Architecture

### Package Structure

```
crypto/
├── types.go              # Core interface definitions
├── crypto.go             # Common utility functions
├── manager.go            # Centralized key manager
├── wrappers.go           # Convenience wrapper functions
├── algorithm_registry.go # Algorithm registry (RFC 9421 support)
├── keys/                 # Key generation and management
│   ├── ed25519.go       # Ed25519 implementation (signing)
│   ├── secp256k1.go     # Secp256k1 implementation (Ethereum)
│   ├── x25519.go        # X25519 implementation (ECDH + HPKE)
│   ├── rs256.go         # RSA-PSS-SHA256 implementation
│   ├── algorithms.go    # Algorithm registration
│   └── constructors.go  # Key generation factory
├── formats/              # Key format conversion
│   ├── jwk.go           # JWK format
│   └── pem.go           # PEM format
├── storage/              # Key storage
│   ├── memory.go        # Memory storage
│   └── file.go          # File storage
├── vault/                # Encrypted key vault
│   └── secure_storage.go # AES-256-GCM encrypted storage
├── rotation/             # Key rotation
│   └── rotator.go       # Key rotation management
└── chain/               # Blockchain integration
    ├── types.go         # Chain Provider interface
    ├── registry.go      # Provider registry
    ├── key_mapper.go    # Key type mapping
    ├── utils.go         # Utility functions
    ├── ethereum/        # Ethereum support
    │   ├── provider.go
    │   └── enhanced_provider.go
    └── solana/          # Solana support
        └── provider.go
```

## Build Instructions

### Building the CLI Tool

```bash
# Run from project root
go build -o sage-crypto ./cmd/sage-crypto

# Or use go install
go install ./cmd/sage-crypto
```

### Running Tests

```bash
# Run all tests
go test ./crypto/...

# Run tests with verbose output
go test -v ./crypto/...

# Test specific packages
go test ./crypto/keys
go test ./crypto/formats
go test ./crypto/storage
go test ./crypto/rotation
go test ./crypto/chain
go test ./crypto/chain/ethereum
go test ./crypto/chain/solana
```

## Usage

### 1. Programmatic Usage

#### Key Pair Generation

```go
package main

import (
    "fmt"
    "github.com/sage-x-project/sage/crypto/keys"
)

func main() {
    // Generate Ed25519 key pair
    ed25519Key, err := keys.GenerateEd25519KeyPair()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Ed25519 Key ID: %s\n", ed25519Key.ID())

    // Generate Secp256k1 key pair
    secp256k1Key, err := keys.GenerateSecp256k1KeyPair()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Secp256k1 Key ID: %s\n", secp256k1Key.ID())
}
```

#### Key Export/Import

```go
import (
    "github.com/sage-x-project/sage/crypto"
    "github.com/sage-x-project/sage/crypto/formats"
)

// Export to JWK format
exporter := formats.NewJWKExporter()
jwkData, err := exporter.Export(keyPair, crypto.KeyFormatJWK)

// Import from JWK format
importer := formats.NewJWKImporter()
importedKey, err := importer.Import(jwkData, crypto.KeyFormatJWK)

// Export to PEM format
pemExporter := formats.NewPEMExporter()
pemData, err := pemExporter.Export(keyPair, crypto.KeyFormatPEM)
```

#### Using Key Storage

```go
import "github.com/sage-x-project/sage/crypto/storage"

// Create memory storage
memStorage := storage.NewMemoryKeyStorage()

// Create file storage
fileStorage, err := storage.NewFileKeyStorage("./keys")

// Store key
err = fileStorage.Store("my-key", keyPair)

// Load key
loadedKey, err := fileStorage.Load("my-key")

// List keys
keyIDs, err := fileStorage.List()
```

#### Message Signing and Verification

```go
// Sign message
message := []byte("Hello, SAGE!")
signature, err := keyPair.Sign(message)

// Verify signature
err = keyPair.Verify(message, signature)
if err == nil {
    fmt.Println("Signature verified!")
}
```

#### X25519 Key Exchange and Encryption

```go
import "github.com/sage-x-project/sage/crypto/keys"

// Generate X25519 keys for Alice and Bob
aliceKey, _ := keys.GenerateX25519KeyPair()
bobKey, _ := keys.GenerateX25519KeyPair()

// Alice derives shared secret with Bob's public key
sharedSecret, _ := aliceKey.(*keys.X25519KeyPair).DeriveSharedSecret(
    bobKey.PublicKey().(*ecdh.PublicKey).Bytes(),
)

// Encrypt message
plaintext := []byte("Secret message")
nonce, ciphertext, _ := aliceKey.(*keys.X25519KeyPair).Encrypt(
    bobKey.PublicKey().(*ecdh.PublicKey).Bytes(),
    plaintext,
)

// Bob decrypts
decrypted, _ := bobKey.(*keys.X25519KeyPair).DecryptWithX25519(
    aliceKey.PublicKey().(*ecdh.PublicKey).Bytes(),
    nonce,
    ciphertext,
)
```

#### Secure Key Exchange using HPKE

```go
import "github.com/sage-x-project/sage/crypto/keys"

// Sender: Derive shared secret
info := []byte("application context")
exportCtx := []byte("shared secret")
enc, sharedSecret, _ := keys.HPKEDeriveSharedSecretToPeer(
    recipientPublicKey,
    info,
    exportCtx,
    32, // Generate 32-byte secret
)

// Receiver: Recover same shared secret
recoveredSecret, _ := keys.HPKEOpenSharedSecretWithPriv(
    recipientPrivateKey,
    enc,
    info,
    exportCtx,
    32,
)
// sharedSecret == recoveredSecret
```

#### Encrypted Key Storage using Vault

```go
import "github.com/sage-x-project/sage/crypto/vault"

// Create vault
fileVault, _ := vault.NewFileVault("./secure-keys")

// Store encrypted key
passphrase := "strong-password"
keyData := []byte("sensitive key material")
fileVault.StoreEncrypted("my-secure-key", keyData, passphrase)

// Load and decrypt key
decryptedKey, _ := fileVault.LoadDecrypted("my-secure-key", passphrase)

// List stored keys
keys := fileVault.ListKeys()
```

### 2. CLI Tool Usage

#### Key Generation

```bash
# Generate Ed25519 key (JWK format output)
./sage-crypto generate --type ed25519 --format jwk

# Generate Secp256k1 key and save to file
./sage-crypto generate --type secp256k1 --format pem --output mykey.pem

# Generate key and store in storage
./sage-crypto generate --type ed25519 --format storage --storage-dir ./keys --key-id mykey
```

#### Message Signing

```bash
# Sign with JWK key file
./sage-crypto sign --key mykey.jwk --message "Hello, World!"

# Sign file with PEM key
./sage-crypto sign --key mykey.pem --format pem --message-file document.txt

# Sign with key from storage
./sage-crypto sign --storage-dir ./keys --key-id mykey --message "Test message"

# Read message from stdin and sign (base64 output)
echo "Message to sign" | ./sage-crypto sign --key mykey.jwk --base64
```

#### Signature Verification

```bash
# Verify with public key and base64 signature
./sage-crypto verify --key public.jwk --message "Hello, World!" --signature-b64 "base64sig..."

# Verify with signature file
./sage-crypto verify --key mykey.pem --format pem --message-file document.txt --signature-file sig.json
```

#### Key Rotation

```bash
# Rotate key (delete old key)
./sage-crypto rotate --storage-dir ./keys --key-id mykey

# Rotate key (keep old key)
./sage-crypto rotate --storage-dir ./keys --key-id mykey --keep-old
```

#### List Keys

```bash
# List all keys in storage
./sage-crypto list --storage-dir ./keys
```

#### Blockchain Address Generation

```bash
# Generate Solana address with Ed25519 key
./sage-crypto generate --type ed25519 --format storage --storage-dir ./keys --key-id alice-sol
./sage-crypto address generate --storage-dir ./keys --key-id alice-sol --chain solana

# Generate Ethereum address with Secp256k1 key
./sage-crypto generate --type secp256k1 --format storage --storage-dir ./keys --key-id alice-eth
./sage-crypto address generate --storage-dir ./keys --key-id alice-eth --chain ethereum

# Generate all compatible blockchain addresses
./sage-crypto address generate --storage-dir ./keys --key-id alice-eth --all
```

#### Blockchain Address Parsing

```bash
# Parse and validate Ethereum address
./sage-crypto address parse 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80

# Parse and validate Solana address
./sage-crypto address parse 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM
```

## Blockchain Support

### Supported Blockchains

| Blockchain | Required Key Type | Address Format | Public Key Recovery | Networks |
|------------|------------------|----------------|---------------------|----------|
| Ethereum | Secp256k1 | 40-char hex starting with 0x | No | Mainnet, Sepolia, Holesky |
| Solana | Ed25519 | Base58 encoded (32 bytes) | Yes | Mainnet, Devnet, Testnet |

### Programmatic Blockchain Address Usage

```go
import (
    "github.com/sage-x-project/sage/crypto/chain"
    "github.com/sage-x-project/sage/crypto/keys"
)

// Generate Ethereum address
secp256k1Key, _ := keys.GenerateSecp256k1KeyPair()
ethProvider, _ := chain.GetProvider(chain.ChainTypeEthereum)
ethAddress, _ := ethProvider.GenerateAddress(
    secp256k1Key.PublicKey(), 
    chain.NetworkEthereumMainnet,
)

// Generate Solana address
ed25519Key, _ := keys.GenerateEd25519KeyPair()
solProvider, _ := chain.GetProvider(chain.ChainTypeSolana)
solAddress, _ := solProvider.GenerateAddress(
    ed25519Key.PublicKey(),
    chain.NetworkSolanaMainnet,
)

// Generate all compatible addresses from key
addresses, _ := chain.AddressFromKeyPair(secp256k1Key)
for chainType, address := range addresses {
    fmt.Printf("%s: %s\n", chainType, address.Value)
}

// Parse and validate address
parsedAddr, _ := chain.ParseAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80")
fmt.Printf("Chain: %s, Network: %s\n", parsedAddr.Chain, parsedAddr.Network)

// Recover public key from Solana address
solPubKey, _ := solProvider.GetPublicKeyFromAddress(
    ctx, 
    "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
    chain.NetworkSolanaMainnet,
)
```

### Adding a New Blockchain

To support a new blockchain, implement the `ChainProvider` interface:

```go
type MyChainProvider struct{}

func (p *MyChainProvider) ChainType() chain.ChainType {
    return "mychain"
}

func (p *MyChainProvider) GenerateAddress(publicKey crypto.PublicKey, network chain.Network) (*chain.Address, error) {
    // Implement address generation logic
}

// Implement other methods...

// Register provider
func init() {
    chain.RegisterProvider(&MyChainProvider{})
}
```

## Real-World Examples

### 1. Complete Workflow Example (with Blockchain)

```bash
# 1. Create key storage directory
mkdir -p ./my-keys

# 2. Generate Ed25519 key and store
./sage-crypto generate --type ed25519 --format storage \
    --storage-dir ./my-keys --key-id alice-key

# 3. List keys
./sage-crypto list --storage-dir ./my-keys

# 4. Sign message
echo "Important message from Alice" | ./sage-crypto sign \
    --storage-dir ./my-keys --key-id alice-key \
    --output alice-signature.json

# 5. Verify signature
./sage-crypto verify --storage-dir ./my-keys --key-id alice-key \
    --message "Important message from Alice" \
    --signature-file alice-signature.json

# 6. Generate blockchain addresses
./sage-crypto address generate --storage-dir ./my-keys --key-id alice-key --all

# 7. Rotate key
./sage-crypto rotate --storage-dir ./my-keys --key-id alice-key --keep-old
```

### 2. JWK Format Example

```bash
# Generate JWK key
./sage-crypto generate --type ed25519 --format jwk --output alice.jwk

# Sign with JWK key
./sage-crypto sign --key alice.jwk --message "Test message" --output signature.json

# Verify signature
./sage-crypto verify --key alice.jwk --message "Test message" --signature-file signature.json
```

### 3. PEM Format Example

```bash
# Generate PEM key
./sage-crypto generate --type secp256k1 --format pem --output bob.pem

# Sign file with PEM key
echo "Document content" > document.txt
./sage-crypto sign --key bob.pem --format pem --message-file document.txt --base64

# Verify with base64 signature
./sage-crypto verify --key bob.pem --format pem --message-file document.txt \
    --signature-b64 "MEUCIQDx..."
```

### 4. Blockchain Integration Example

```bash
# Generate keys for Ethereum and Solana
./sage-crypto generate --type secp256k1 --format storage --storage-dir ./keys --key-id eth-key
./sage-crypto generate --type ed25519 --format storage --storage-dir ./keys --key-id sol-key

# Generate blockchain addresses for each key
./sage-crypto address generate --storage-dir ./keys --key-id eth-key --all
./sage-crypto address generate --storage-dir ./keys --key-id sol-key --all

# Generate specific chain addresses
./sage-crypto address generate --storage-dir ./keys --key-id eth-key --chain ethereum
./sage-crypto address generate --storage-dir ./keys --key-id sol-key --chain solana

# Validate addresses
./sage-crypto address parse 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80
./sage-crypto address parse 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM

# Export addresses to JSON
./sage-crypto address generate --storage-dir ./keys --key-id eth-key --all --output addresses.json
```

## Security Considerations

1. **Key File Permissions**: Generated key files are automatically set with `0600` permissions, allowing only the owner to read and write.

2. **Key Rotation**: Regular key rotation is recommended. Use the `--keep-old` option to retain previous keys.

3. **Storage Security**: When using file-based storage, ensure proper directory permissions.

## Supported Algorithms

| Algorithm | Type | Purpose | RFC 9421 | Signing | Encryption | Key Exchange |
|-----------|------|---------|----------|---------|------------|--------------|
| **Ed25519** | EdDSA | Digital signatures | ✅ ed25519 | ✅ | ❌ | ❌ |
| **Secp256k1** | ECDSA | Signing, blockchain | ✅ es256k | ✅ | ❌ | ❌ |
| **X25519** | ECDH | Key exchange, encryption | ❌ | ❌ | ✅ | ✅ |
| **RS256** | RSA | Signing, encryption | ✅ rsa-pss-sha256 | ✅ | ✅ | ❌ |

### Algorithm Features

- **Ed25519**: Fast and secure EdDSA signature algorithm (Curve25519-based)
- **Secp256k1**: ECDSA elliptic curve used in Bitcoin and Ethereum
- **X25519**: ECDH (Elliptic Curve Diffie-Hellman) key exchange with HPKE support
- **RS256**: RSA-PSS-SHA256 with 2048-bit key length

## Supported Formats

- **JWK (JSON Web Key)**: JSON-based standard key format
- **PEM (Privacy Enhanced Mail)**: Base64-encoded text format

## Troubleshooting

### Key Not Found
```
Error: key not found
```
Check the storage directory and key ID.

### Invalid Signature
```
Signature verification FAILED
```
Ensure you're using the correct key and message.

### Permission Error
```
failed to create key storage directory: permission denied
```
Check write permissions for the directory.

### Invalid Key Type
```
Error: invalid public key: Ethereum requires secp256k1 keys
```
Use the correct key type for the blockchain:
- Ethereum: Secp256k1
- Solana: Ed25519

### Unsupported Blockchain
```
Error: unsupported chain: bitcoin
```
Currently only Ethereum and Solana are supported. New blockchains can be added by implementing the ChainProvider interface.

## License

Provided as part of the SAGE project.