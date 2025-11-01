# Example 01: Multi-Key Agent Registration

This example demonstrates how to register an AI agent with multiple cryptographic keys on the Ethereum blockchain using SAGE.

## What This Example Does

1. **Generates three types of cryptographic keys:**
   - ECDSA (secp256k1) - For Ethereum compatibility and transaction signing
   - Ed25519 - For high-performance message signing
   - X25519 - For key agreement and encryption (HPKE)

2. **Registers the agent with all keys in a single transaction**

3. **Verifies the registration** by resolving the agent from the blockchain

4. **Shows the Ed25519 approval workflow** (required for Ethereum)

## Key Concepts

### Multi-Key Support

Traditional blockchain identity systems support only one key per agent. SAGE's SageRegistryV4 contract supports multiple keys with different cryptographic algorithms:

- **ECDSA (secp256k1)**: Native Ethereum signing, automatically verified
- **Ed25519**: High-performance signing, requires off-chain approval
- **X25519**: Key encapsulation for HPKE encryption

### Key Verification

- **ECDSA keys**: Automatically verified on-chain via signature check
- **Ed25519 keys**: Require contract owner approval (prevents key pollution)
- **X25519 keys**: No verification needed (used for encryption, not authentication)

### Single Transaction

All keys are registered atomically in one transaction, ensuring:
- Consistent DID state
- Lower gas costs
- Atomic success/failure

## Prerequisites

### 1. Start Local Blockchain

```bash
cd contracts/ethereum
npx hardhat node
```

### 2. Deploy SageRegistryV4

```bash
npx hardhat run scripts/deploy-v4-local.js --network localhost
```

Note the contract address from the output.

### 3. Set Environment Variables

```bash
export REGISTRY_ADDRESS="0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
export RPC_URL="http://localhost:8545"
export PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
```

## Running the Example

```bash
cd examples/a2a-integration/01-register-agent
go run main.go
```

## Expected Output

```
╔═══════════════════════════════════════════════════════════╗
║     SAGE Example 01: Multi-Key Agent Registration        ║
╚═══════════════════════════════════════════════════════════╝

 Configuration
─────────────────────────────────────────────────────────
Registry Address: 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9
RPC URL:          http://localhost:8545

 Step 1: Generating Cryptographic Keys
─────────────────────────────────────────────────────────
Generating ECDSA (secp256k1) key...
 ECDSA key generated (33 bytes)
Generating Ed25519 key...
 Ed25519 key generated (32 bytes)
Generating X25519 key...
 X25519 key generated (32 bytes)

 Step 2: Connecting to Blockchain
─────────────────────────────────────────────────────────
 Connected to Ethereum (localhost)

 Step 3: Preparing Registration Request
─────────────────────────────────────────────────────────
Agent DID:    did:sage:ethereum:example-agent-20250119123456
Agent Name:   Multi-Key Example Agent
Endpoint:     https://agent.example.com
Keys:         3 (ECDSA + Ed25519 + X25519)

 Step 4: Registering Agent on Blockchain
─────────────────────────────────────────────────────────
 Submitting transaction...

 Agent Registered Successfully!
─────────────────────────────────────────────────────────
Transaction Hash: 0x1234...
Block Number:     42
Gas Used:         524288

 Step 5: Verifying Registration
─────────────────────────────────────────────────────────
Agent DID:        did:sage:ethereum:example-agent-20250119123456
Agent Name:       Multi-Key Example Agent
Agent Endpoint:   https://agent.example.com
Owner Address:    0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
Active:           true

  Step 6: Ed25519 Key Approval Required
─────────────────────────────────────────────────────────
Ed25519 keys require approval by the registry contract owner.
Until approved, the Ed25519 key will be marked as 'unverified'.

To approve (as contract owner):
  ./build/bin/sage-did key approve <keyhash> \
    --chain ethereum \
    --contract-address 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9 \
    --rpc-url http://localhost:8545 \
    --private-key $OWNER_PRIVATE_KEY

╔═══════════════════════════════════════════════════════════╗
║     Registration Complete!                                ║
╚═══════════════════════════════════════════════════════════╝

 Success! Your multi-key agent is now registered on-chain.

Next steps:
  1. Approve the Ed25519 key (see command above)
  2. Run example 02 to generate an A2A Agent Card
  3. Use the agent for secure communication

Agent DID: did:sage:ethereum:example-agent-20250119123456
```

## Code Walkthrough

### Step 1: Generate Keys

```go
// Generate ECDSA key (primary)
ecdsaKeyPair, err := crypto.GenerateSecp256k1KeyPair()

// Generate Ed25519 key (signing)
ed25519KeyPair, err := crypto.GenerateEd25519KeyPair()

// Generate X25519 key (encryption)
x25519KeyPair, err := crypto.GenerateX25519KeyPair()
```

### Step 2: Configure Manager

```go
manager := did.NewManager()
config := &did.RegistryConfig{
    ContractAddress: "0x...",
    RPCEndpoint:     "http://localhost:8545",
    PrivateKey:      "0x...",
}
manager.Configure(did.ChainEthereum, config)
```

### Step 3: Register Agent

```go
req := &did.RegistrationRequest{
    DID:         agentDID,
    Name:        "Multi-Key Example Agent",
    Description: "Example agent...",
    Endpoint:    "https://agent.example.com",
    KeyPair:     ecdsaKeyPair, // Primary key
    Keys: []did.AgentKey{
        {Type: did.KeyTypeEd25519, KeyData: ed25519PubKey},
        {Type: did.KeyTypeX25519, KeyData: x25519PubKey},
    },
}

result, err := manager.RegisterAgent(ctx, did.ChainEthereum, req)
```

## Gas Costs

Typical gas usage for multi-key registration:

| Keys | Gas Used | Cost (@ 50 Gwei) |
|------|----------|------------------|
| 1    | ~200k    | ~0.01 ETH        |
| 3    | ~500k    | ~0.025 ETH       |
| 5    | ~800k    | ~0.04 ETH        |

*Note: Actual costs vary based on network congestion and key data size*

## Next Steps

1. **Approve Ed25519 Key**: Use the `sage-did key approve` command
2. **Generate A2A Card**: Run example 02
3. **Exchange Cards**: Run example 03
4. **Secure Messaging**: Run example 04

## Troubleshooting

### "Connection refused"
- Ensure Hardhat node is running: `npx hardhat node`

### "Contract not found"
- Deploy the contract: `npx hardhat run scripts/deploy-v4-local.js --network localhost`

### "Insufficient funds"
- Use the default Hardhat account with pre-funded ETH

## Related Documentation

- [SAGE Multi-Key Design](../../../contracts/MULTI_KEY_DESIGN.md)
- [DID Specification](../../../docs/DID_SPECIFICATION.md)
- [SageRegistryV4 Contract](../../../contracts/ethereum/contracts/SageRegistryV4.sol)

## License

LGPL-v3 - See [LICENSE](../../../LICENSE)
