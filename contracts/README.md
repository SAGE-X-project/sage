# SAGE Smart Contracts

Smart contracts for SAGE AI Agent Registry on Ethereum and Solana blockchains.

## Overview

The SAGE contracts provide a decentralized registry for AI agents with the following features:

- **Secure Registration**: Public key verification ensures only the owner can register agents
- **Metadata Storage**: Store agent DID, name, description, endpoint, and capabilities
- **Update/Deactivation**: Owners can update metadata or deactivate agents
- **Hook System**: Extensible before/after registration hooks for additional verification
- **Multi-chain Support**: Implementations for both Ethereum and Solana

## Architecture

### Ethereum Implementation

- **SageRegistry.sol**: Main registry contract with signature verification
- **ISageRegistry.sol**: Interface defining the registry methods
- **IRegistryHook.sol**: Hook interface for extensibility
- **SageVerificationHook.sol**: Example hook with rate limiting and blacklist

### Solana Implementation

- **sage-registry**: Main Anchor program using PDAs for agent accounts
- **sage-verification-hook**: Verification hook with rate limiting

## Security Features

1. **Signature Verification**: All registrations require valid signatures
2. **Owner-only Operations**: Only agent owners can update/deactivate
3. **Rate Limiting**: Prevent spam registrations (configurable)
4. **Blacklist Support**: Block malicious actors
5. **Nonce Tracking**: Prevent replay attacks

## Git Submodule Setup

To use these contracts in a separate repository:

```bash
# From your main project root
git submodule add https://github.com/your-org/sage-contracts.git contracts
git submodule update --init --recursive

# To update the submodule later
cd contracts
git pull origin main
cd ..
git add contracts
git commit -m "Update contracts submodule"
```

## Deployment

### Ethereum

```bash
cd contracts/ethereum
npm install
npx hardhat compile

# Deploy to local network
npx hardhat run scripts/deploy.js --network localhost

# Deploy to testnet
npx hardhat run scripts/deploy.js --network sepolia

# Verify contracts
npx hardhat run scripts/verify.js --network sepolia
```

### Solana

```bash
cd contracts/solana
anchor build
./scripts/deploy.sh

# Or deploy manually
anchor deploy
anchor run initialize
```

## Usage Examples

### Ethereum Registration

```javascript
const registry = new ethers.Contract(registryAddress, registryABI, signer);

const agentData = {
  did: "did:sage:agent001",
  name: "My AI Agent",
  description: "An intelligent assistant",
  endpoint: "https://api.myagent.ai",
  publicKey: "0x...", // Secp256k1 public key
  capabilities: JSON.stringify({ skills: ["chat", "code"] })
};

// Sign the registration data
const signature = await signer.signMessage(messageHash);

// Register agent
const tx = await registry.registerAgent(
  agentData.did,
  agentData.name,
  agentData.description,
  agentData.endpoint,
  agentData.publicKey,
  agentData.capabilities,
  signature
);
```

### Solana Registration

```typescript
const program = anchor.workspace.SageRegistry;

// Sign with Ed25519
const signature = await ed.sign(messageBytes, wallet.secretKey);

// Register agent
await program.methods
  .registerAgent(
    did,
    name,
    description,
    endpoint,
    capabilities,
    Buffer.from(signature)
  )
  .accounts({
    agent: agentPDA,
    registry: registryPDA,
    owner: wallet.publicKey,
    systemProgram: SystemProgram.programId,
    ed25519Program: ED25519_PROGRAM_ID,
  })
  .rpc();
```

## Gas Optimization

### Ethereum
- Efficient storage packing in AgentMetadata struct
- Minimal storage writes
- Events for off-chain indexing
- Signature verification using ecrecover

### Solana
- PDA-based accounts minimize rent
- Efficient account sizing
- Native Ed25519 support
- Minimal compute units

## Testing

### Ethereum
```bash
cd contracts/ethereum
npx hardhat test
```

### Solana
```bash
cd contracts/solana
anchor test
```

## License

MIT