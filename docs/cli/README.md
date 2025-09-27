# SAGE CLI Tools Documentation

SAGE provides two command-line tools for cryptographic operations and DID management:

- **sage-crypto**: Key management and cryptographic operations
- **sage-did**: Decentralized Identifier (DID) management

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/sage-x-project/sage.git
cd sage

# Build and install
make build
make install  # Installs to $GOPATH/bin
```

### Using Pre-built Binaries

Download the latest release from the GitHub releases page.

## sage-crypto

The `sage-crypto` tool provides comprehensive key management and cryptographic operations.

### Commands

#### generate - Generate a new key pair

```bash
# Generate Ed25519 key as JWK
sage-crypto generate --type ed25519 --format jwk

# Generate Secp256k1 key and save to file
sage-crypto generate --type secp256k1 --format pem --output mykey.pem

# Generate and store in key storage
sage-crypto generate --type ed25519 --format storage --storage-dir ./keys --key-id mykey
```

**Options:**
- `--type, -t`: Key type (ed25519, secp256k1)
- `--format, -f`: Output format (jwk, pem, storage)
- `--output, -o`: Output file path
- `--storage-dir, -s`: Storage directory for storage format
- `--key-id, -k`: Key ID for storage format

#### sign - Sign a message

```bash
# Sign with JWK key file
sage-crypto sign --key mykey.jwk --message "Hello, World!"

# Sign file content
sage-crypto sign --key mykey.pem --format pem --message-file document.txt

# Sign using stored key
sage-crypto sign --storage-dir ./keys --key-id mykey --message "Test"

# Output base64 signature only
sage-crypto sign --key mykey.jwk --message "Hello" --base64
```

**Options:**
- `--key`: Key file path
- `--key-format`: Key file format (jwk, pem)
- `--storage-dir, -s`: Storage directory
- `--key-id, -k`: Key ID in storage
- `--message, -m`: Message to sign
- `--message-file`: File containing message
- `--output, -o`: Output file for signature
- `--base64`: Output signature as base64 only

#### verify - Verify a signature

```bash
# Verify with base64 signature
sage-crypto verify --key public.jwk --message "Hello, World!" --signature-b64 "base64sig..."

# Verify with signature file
sage-crypto verify --key mykey.pem --format pem --message-file document.txt --signature-file sig.json
```

**Options:**
- `--key`: Public key file (required)
- `--key-format`: Key format (jwk, pem)
- `--message, -m`: Message to verify
- `--message-file`: File containing message
- `--signature-b64`: Base64 encoded signature
- `--signature-file`: Signature file

#### list - List keys in storage

```bash
sage-crypto list --storage-dir ./keys
```

**Options:**
- `--storage-dir, -s`: Storage directory (required)

#### rotate - Rotate a key

```bash
sage-crypto rotate --storage-dir ./keys --key-id mykey
```

**Options:**
- `--storage-dir, -s`: Storage directory (required)
- `--key-id, -k`: Key ID to rotate (required)
- `--keep-old`: Keep old key instead of deleting

#### address - Blockchain address operations

##### address generate - Generate blockchain addresses

```bash
# Generate Ethereum address from key
sage-crypto address generate --key mykey.pem --key-format pem --chain ethereum

# Generate Solana address
sage-crypto address generate --storage-dir ./keys --key-id mykey --chain solana
```

**Options:**
- `--key`: Key file path
- `--key-format`: Key format (jwk, pem)
- `--storage-dir, -s`: Storage directory
- `--key-id, -k`: Key ID in storage
- `--chain, -c`: Blockchain (ethereum, solana)

##### address parse - Parse and validate blockchain addresses

```bash
# Parse Ethereum address
sage-crypto address parse 0x742d35Cc6634C0532925a3b844Bc9e7595f0b0Bb

# Parse Solana address
sage-crypto address parse 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM
```

### Examples

#### Complete Workflow Example

```bash
# 1. Generate a new Ed25519 key pair
sage-crypto generate --type ed25519 --format jwk --output alice.jwk

# 2. Sign a message
MESSAGE="Hello, SAGE!"
SIGNATURE=$(sage-crypto sign --key alice.jwk --message "$MESSAGE" --base64)
echo "Signature: $SIGNATURE"

# 3. Verify the signature
sage-crypto verify --key alice.jwk --message "$MESSAGE" --signature-b64 "$SIGNATURE"
# Output: Signature verification PASSED

# 4. Try with wrong message (should fail)
sage-crypto verify --key alice.jwk --message "Wrong message" --signature-b64 "$SIGNATURE"
# Output: Signature verification FAILED
```

#### Key Storage Example

```bash
# Create storage directory
mkdir -p ~/.sage/keys

# Generate and store multiple keys
sage-crypto generate --type ed25519 --format storage \
  --storage-dir ~/.sage/keys --key-id signing-key

sage-crypto generate --type secp256k1 --format storage \
  --storage-dir ~/.sage/keys --key-id ethereum-key

# List all keys
sage-crypto list --storage-dir ~/.sage/keys

# Use stored key for signing
sage-crypto sign --storage-dir ~/.sage/keys --key-id signing-key \
  --message "Signed with stored key"
```

## sage-did

The `sage-did` tool manages Decentralized Identifiers for AI agents on blockchain.

### Commands

#### register - Register a new AI agent

```bash
# Register on Ethereum
sage-did register --chain ethereum --name "My AI Agent" \
  --endpoint "https://api.myagent.com" \
  --key ethereum-key.pem --format pem \
  --description "AI assistant for code review"

# Register on Solana with capabilities
sage-did register --chain solana --name "Trading Bot" \
  --endpoint "https://bot.example.com" \
  --storage-dir ~/.sage/keys --key-id bot-key \
  --capabilities '{"trading": true, "analysis": true}'
```

**Options:**
- `--chain, -c`: Blockchain (ethereum, solana) [required]
- `--name, -n`: Agent name [required]
- `--endpoint`: Agent API endpoint [required]
- `--description, -d`: Agent description
- `--capabilities`: Agent capabilities (JSON)
- `--key, -k`: Key file path
- `--key-format`: Key format (jwk, pem)
- `--storage-dir`: Key storage directory
- `--key-id`: Key ID in storage
- `--rpc`: Blockchain RPC endpoint
- `--contract`: Registry contract address
- `--private-key`: Transaction signer private key

#### resolve - Resolve an agent DID

```bash
# Resolve agent metadata
sage-did resolve did:sage:ethereum:agent_12345

# Save to file
sage-did resolve did:sage:solana:bot_abc --output agent-info.json

# Text format output
sage-did resolve did:sage:ethereum:agent_12345 --format text
```

**Options:**
- `--rpc`: Blockchain RPC endpoint
- `--contract`: Registry contract address
- `--output, -o`: Output file path
- `--format`: Output format (json, text)

#### list - List agents by owner

```bash
# List all agents owned by an address
sage-did list --chain ethereum --owner 0x742d35Cc6634C0532925a3b844Bc9e7595f7F1a

# With custom RPC
sage-did list --chain solana --owner AgentOwnerPubkey... \
  --rpc https://api.devnet.solana.com
```

**Options:**
- `--chain, -c`: Blockchain (ethereum, solana) [required]
- `--owner`: Owner address [required]
- `--rpc`: Blockchain RPC endpoint
- `--contract`: Registry contract address

#### update - Update agent metadata

```bash
# Update endpoint
sage-did update did:sage:ethereum:agent_12345 \
  --endpoint "https://new-api.myagent.com" \
  --key owner-key.pem --format pem

# Update capabilities
sage-did update did:sage:solana:bot_abc \
  --capabilities '{"trading": true, "analysis": true, "reporting": true}' \
  --storage-dir ~/.sage/keys --key-id owner-key
```

**Options:**
- `--endpoint`: New endpoint URL
- `--description`: New description
- `--capabilities`: New capabilities (JSON)
- `--key`: Owner key file
- `--key-format`: Key format
- `--storage-dir`: Key storage directory
- `--key-id`: Key ID in storage
- `--rpc`: Blockchain RPC endpoint
- `--contract`: Registry contract address

#### deactivate - Deactivate an agent

```bash
sage-did deactivate did:sage:ethereum:agent_12345 \
  --key owner-key.pem --format pem
```

**Options:**
- `--key`: Owner key file [required]
- `--key-format`: Key format
- `--storage-dir`: Key storage directory
- `--key-id`: Key ID in storage
- `--rpc`: Blockchain RPC endpoint
- `--contract`: Registry contract address

#### verify - Verify agent metadata

```bash
# Verify agent is active and endpoint is reachable
sage-did verify did:sage:ethereum:agent_12345

# Skip endpoint check
sage-did verify did:sage:solana:bot_abc --skip-endpoint
```

**Options:**
- `--rpc`: Blockchain RPC endpoint
- `--contract`: Registry contract address
- `--skip-endpoint`: Skip endpoint reachability check

### Examples

#### Complete Agent Registration Workflow

```bash
# 1. Generate key for Ethereum
sage-crypto generate --type secp256k1 --format storage \
  --storage-dir ~/.sage/keys --key-id agent-key

# 2. Get Ethereum address
AGENT_ADDR=$(sage-crypto address --storage-dir ~/.sage/keys \
  --key-id agent-key --chain ethereum)
echo "Agent address: $AGENT_ADDR"

# 3. Register agent (after funding address)
sage-did register --chain ethereum \
  --name "Code Review Assistant" \
  --description "AI agent for automated code reviews" \
  --endpoint "https://api.codereview-bot.com" \
  --capabilities '{"review": true, "suggest": true, "lint": true}' \
  --storage-dir ~/.sage/keys --key-id agent-key \
  --private-key $DEPLOYER_PRIVATE_KEY

# 4. Resolve to verify registration
sage-did resolve did:sage:ethereum:agent_$AGENT_ADDR
```

## Environment Variables

Both tools support the following environment variables:

```bash
# Default storage directory
export SAGE_KEY_STORAGE="$HOME/.sage/keys"

# Default RPC endpoints
export SAGE_ETH_RPC="https://eth-mainnet.g.alchemy.com/v2/your-key"
export SAGE_SOL_RPC="https://api.mainnet-beta.solana.com"

# Default contract addresses
export SAGE_ETH_CONTRACT="0x..."
export SAGE_SOL_CONTRACT="..."
```

## Security Best Practices

1. **Key Storage**: Always store keys in encrypted storage or use hardware security modules in production
2. **Private Keys**: Never share or commit private keys
3. **RPC Endpoints**: Use authenticated RPC endpoints in production
4. **Permissions**: Set appropriate file permissions (0600) for key files
5. **Backups**: Regularly backup key storage directories

## Troubleshooting

### Common Issues

1. **"Key not found" error**
   - Check the key file path or storage directory
   - Verify the key ID matches

2. **"Invalid signature" error**
   - Ensure the message matches exactly (including whitespace)
   - Verify you're using the correct key

3. **"Connection refused" for blockchain operations**
   - Check RPC endpoint is accessible
   - Verify network connectivity

4. **"Insufficient funds" for DID registration**
   - Ensure the account has enough native tokens for gas fees

### Debug Mode

Enable debug output with the `SAGE_DEBUG` environment variable:

```bash
SAGE_DEBUG=1 sage-crypto sign --key mykey.jwk --message "test"
```

## Getting Help

```bash
# General help
sage-crypto --help
sage-did --help

# Command-specific help
sage-crypto sign --help
sage-did register --help
```

For issues and feature requests, visit: https://github.com/sage-x-project/sage/issues