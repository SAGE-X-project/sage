# Example 02: A2A Agent Card Generation

This example demonstrates how to generate an A2A (Agent-to-Agent) Agent Card from a registered agent's on-chain identity.

## What This Example Does

1. **Resolves agent metadata** from the blockchain using the agent's DID
2. **Generates an A2A Agent Card** in the standard format
3. **Validates the card structure** to ensure compliance
4. **Exports the card to JSON** for sharing with other agents

## What is an A2A Agent Card?

An A2A Agent Card is a portable, standardized identity document that contains:

- **Agent identification** (DID)
- **Contact information** (service endpoints)
- **Public keys** for verification and encryption
- **Metadata** (name, description, capabilities)

It's similar to a business card, but for AI agents, enabling:
- Self-sovereign identity
- Decentralized discovery
- Trust establishment
- Secure communication setup

## Prerequisites

### 1. Registered Agent

You must have completed Example 01 to register an agent:

```bash
cd ../01-register-agent
go run main.go
```

Note the agent DID from the output.

### 2. Set Environment Variables

```bash
export AGENT_DID="did:sage:ethereum:example-agent-20250119123456"
export REGISTRY_ADDRESS="0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
export RPC_URL="http://localhost:8545"
```

### 3. Blockchain Running

Ensure Hardhat node is still running from Example 01.

## Running the Example

```bash
cd examples/a2a-integration/02-generate-card
go run main.go
```

## Expected Output

```
╔═══════════════════════════════════════════════════════════╗
║     SAGE Example 02: A2A Agent Card Generation           ║
╚═══════════════════════════════════════════════════════════╝

📋 Configuration
─────────────────────────────────────────────────────────
Registry Address: 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9
RPC URL:          http://localhost:8545
Agent DID:        did:sage:ethereum:example-agent-20250119123456

🔗 Step 1: Connecting to Blockchain
─────────────────────────────────────────────────────────
✓ Connected to Ethereum (localhost)

🔍 Step 2: Resolving Agent from Blockchain
─────────────────────────────────────────────────────────
Agent resolved successfully!
  Name:        Multi-Key Example Agent
  Endpoint:    https://agent.example.com
  Owner:       0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
  Active:      true

🎴 Step 3: Generating A2A Agent Card
─────────────────────────────────────────────────────────
✓ A2A Agent Card generated successfully!

Card Details:
  Type:         AgentCard
  Version:      1.0
  ID (DID):     did:sage:ethereum:example-agent-20250119123456
  Name:         Multi-Key Example Agent
  Description:  Example agent demonstrating multi-key registration...
  Service URL:  https://agent.example.com
  Public Keys:  3 keys

Public Keys:
  [1] did:sage:ethereum:example-agent-20250119123456#key-0
      Type:       EcdsaSecp256k1VerificationKey2019
      Controller: did:sage:ethereum:example-agent-20250119123456
      Key Length: 33 bytes

  [2] did:sage:ethereum:example-agent-20250119123456#key-1
      Type:       Ed25519VerificationKey2020
      Controller: did:sage:ethereum:example-agent-20250119123456
      Key Length: 32 bytes

  [3] did:sage:ethereum:example-agent-20250119123456#key-2
      Type:       X25519KeyAgreementKey2020
      Controller: did:sage:ethereum:example-agent-20250119123456
      Key Length: 32 bytes
      Purpose:    keyAgreement

✅ Step 4: Validating A2A Card
─────────────────────────────────────────────────────────
✓ Card validation passed!
  - Type and version are valid
  - DID format is correct
  - Required fields are present
  - Public keys are well-formed

💾 Step 5: Exporting Card to JSON
─────────────────────────────────────────────────────────
✓ Card exported to: agent-card.json
  File size: 1234 bytes

╔═══════════════════════════════════════════════════════════╗
║     Card Generation Complete!                             ║
╚═══════════════════════════════════════════════════════════╝

🎉 Success! Your A2A Agent Card is ready.
```

## Generated File

The example creates `agent-card.json`:

```json
{
  "@context": "https://w3id.org/a2a/v1",
  "type": "AgentCard",
  "version": "1.0",
  "id": "did:sage:ethereum:example-agent-20250119123456",
  "name": "Multi-Key Example Agent",
  "description": "Example agent demonstrating multi-key registration...",
  "serviceEndpoint": "https://agent.example.com",
  "publicKeys": [
    {
      "id": "did:sage:ethereum:example-agent-20250119123456#key-0",
      "type": "EcdsaSecp256k1VerificationKey2019",
      "controller": "did:sage:ethereum:example-agent-20250119123456",
      "publicKeyMultibase": "z..."
    },
    {
      "id": "did:sage:ethereum:example-agent-20250119123456#key-1",
      "type": "Ed25519VerificationKey2020",
      "controller": "did:sage:ethereum:example-agent-20250119123456",
      "publicKeyMultibase": "z..."
    },
    {
      "id": "did:sage:ethereum:example-agent-20250119123456#key-2",
      "type": "X25519KeyAgreementKey2020",
      "controller": "did:sage:ethereum:example-agent-20250119123456",
      "publicKeyMultibase": "z...",
      "purpose": "keyAgreement"
    }
  ]
}
```

## Key Concepts

### A2A Card Structure

The card follows the A2A protocol specification:

- **@context**: JSON-LD context for semantic interoperability
- **type**: "AgentCard" identifier
- **version**: Protocol version (1.0)
- **id**: The agent's DID
- **publicKeys**: Array of cryptographic keys

### Public Key Encoding

Keys are encoded in **Multibase** format:
- Prefix indicates encoding type (e.g., 'z' for base58btc)
- Self-describing format
- Interoperable across systems

### Key Types

| Key Type | Purpose | Usage |
|----------|---------|-------|
| EcdsaSecp256k1VerificationKey2019 | Ethereum signing | Transaction signatures |
| Ed25519VerificationKey2020 | Message signing | High-performance signatures |
| X25519KeyAgreementKey2020 | Key agreement | ECDH for encryption |

## Code Walkthrough

### Resolve Agent

```go
manager := did.NewManager()
manager.Configure(did.ChainEthereum, config)
agent, err := manager.ResolveAgent(ctx, agentDID)
```

### Generate Card

```go
metadataV4 := did.FromAgentMetadata(agent)
card, err := did.GenerateA2ACard(metadataV4)
```

### Validate Card

```go
err := did.ValidateA2ACard(card)
```

### Export to JSON

```go
cardJSON, err := json.MarshalIndent(card, "", "  ")
os.WriteFile("agent-card.json", cardJSON, 0644)
```

## Using the Card

The generated card can be:

### 1. Shared Over HTTP

```http
GET /agent/card HTTP/1.1
Host: agent.example.com

HTTP/1.1 200 OK
Content-Type: application/json

{
  "@context": "https://w3id.org/a2a/v1",
  "type": "AgentCard",
  ...
}
```

### 2. Imported by Other Agents

```go
cardData, _ := os.ReadFile("agent-card.json")
var card did.A2AAgentCard
json.Unmarshal(cardData, &card)
```

### 3. Verified Against Blockchain

```go
// Verify card matches on-chain data
agent, _ := manager.ResolveAgent(ctx, card.ID)
// Compare publicKeys, endpoint, etc.
```

## Next Steps

1. **Run Example 03**: Exchange cards with another agent
2. **Run Example 04**: Use the cards for secure messaging
3. **Deploy to production**: Share cards via HTTPS endpoints

## Troubleshooting

### "Agent DID not found"
- Ensure you ran Example 01 first
- Check that the DID is correct
- Verify the blockchain node is running

### "Failed to generate card"
- Agent must have at least one key registered
- Check that agent metadata is complete

### "Card validation failed"
- This indicates a bug in card generation
- Check that all required fields are present

## Related Documentation

- [A2A Protocol Specification](https://github.com/a2aproject/a2a)
- [DID Specification](../../../docs/DID_SPECIFICATION.md)
- [Multibase Encoding](https://github.com/multiformats/multibase)

## License

LGPL-v3 - See [LICENSE](../../../LICENSE)
