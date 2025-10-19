# Example 03: A2A Card Exchange and Verification

This example demonstrates the complete workflow for exchanging and verifying A2A Agent Cards between two agents.

## What This Example Does

1. **Registers two agents** (Agent A and Agent B) with multi-key support
2. **Generates A2A cards** for both agents
3. **Simulates card exchange** (Agent B receives Agent A's card)
4. **Validates card structure** using SAGE validation functions
5. **Verifies card against blockchain** to ensure authenticity
6. **Establishes mutual trust** between agents

## Why This Matters

Card exchange is the foundation of agent-to-agent trust:

- **Discovery**: Agents learn about each other's capabilities
- **Authentication**: Blockchain verification proves DID ownership
- **Trust**: Cross-checking prevents impersonation
- **Communication**: Exchanged keys enable secure messaging

## Prerequisites

- Local Hardhat node running
- SageRegistryV4 contract deployed
- Environment variables set (see main README)

## Running the Example

```bash
cd examples/a2a-integration/03-exchange-cards
go run main.go
```

## What Happens

### Phase 1: Agent Registration

Both agents are registered with:
- ECDSA key (Ethereum signing)
- Ed25519 key (message signing)
- X25519 key (encryption)

### Phase 2: Card Generation

Each agent generates an A2A card containing:
- DID and metadata
- Service endpoint
- Public keys (all 3 types)

Cards are saved to:
- `agent-a-card.json`
- `agent-b-card.json`

### Phase 3: Card Exchange

Agent B receives Agent A's card (simulated via file in this example; in production, this would be via HTTP/HTTPS).

### Phase 4: Validation

**Structural validation:**
- Card format is correct
- Required fields are present
- DID syntax is valid
- Public keys are well-formed

**Blockchain verification:**
- Resolve DID from blockchain
- Compare card data with on-chain data
- Verify agent is active
- Ensure no tampering

### Phase 5: Trust Establishment

Once verified, Agent B can trust Agent A and:
- Send encrypted messages
- Verify A's signatures
- Establish secure channels

## Output

```
╔═══════════════════════════════════════════════════════════╗
║     SAGE Example 03: A2A Card Exchange                   ║
╚═══════════════════════════════════════════════════════════╝

👤 AGENT A: Registering and Generating Card
═════════════════════════════════════════════════════════
✓ Agent A registered: did:sage:ethereum:Agent-A-123456
💾 Agent A's card saved to: agent-a-card.json

👤 AGENT B: Registering and Generating Card
═════════════════════════════════════════════════════════
✓ Agent B registered: did:sage:ethereum:Agent-B-123456
💾 Agent B's card saved to: agent-b-card.json

📨 Step 1: Agent B Receives Agent A's Card
═════════════════════════════════════════════════════════
✓ Agent B received Agent A's card
  From:         Agent-A
  DID:          did:sage:ethereum:Agent-A-123456
  Endpoint:     https://Agent-A.example.com
  Public Keys:  3

✅ Step 2: Validating Card Structure
═════════════════════════════════════════════════════════
✓ Card structure is valid
  - Correct type and version
  - Valid DID format
  - All required fields present
  - Public keys are well-formed

🔗 Step 3: Verifying Card Against Blockchain
═════════════════════════════════════════════════════════
✓ Agent resolved from blockchain
  Name:      Agent-A
  Endpoint:  https://Agent-A.example.com
  Owner:     0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
  Active:    true

🔍 Cross-checking card with blockchain data...
✓ Name matches
✓ Endpoint matches
✓ Agent is active

✅ Card verification successful!
   The card matches blockchain data.

🤝 Step 4: Establishing Trust
═════════════════════════════════════════════════════════
Agent B now trusts Agent A because:
  1. ✓ Card structure is valid (proper format)
  2. ✓ DID is registered on blockchain
  3. ✓ Card data matches blockchain data
  4. ✓ Agent is active and not revoked

╔═══════════════════════════════════════════════════════════╗
║     Card Exchange Complete!                               ║
╚═══════════════════════════════════════════════════════════╝

🎉 Success! Agents A and B have exchanged and verified cards.
```

## Security Checks Performed

| Check | Purpose | Attack Prevented |
|-------|---------|------------------|
| Card structure validation | Ensure proper format | Malformed data |
| DID syntax validation | Verify DID format | Invalid identifiers |
| Blockchain resolution | Confirm DID exists | Fake DIDs |
| Data cross-check | Match card vs chain | Tampering |
| Active status check | Verify not revoked | Deactivated agents |

## In Production

### Card Exchange via HTTP

```go
// Agent A serves its card
http.HandleFunc("/agent/card", func(w http.ResponseWriter, r *http.Request) {
    cardJSON, _ := json.Marshal(myCard)
    w.Header().Set("Content-Type", "application/json")
    w.Write(cardJSON)
})

// Agent B fetches Agent A's card
resp, _ := http.Get("https://agent-a.example.com/agent/card")
defer resp.Body.Close()
var card did.A2AAgentCard
json.NewDecoder(resp.Body).Decode(&card)
```

### Verification Workflow

```go
// 1. Receive card
var receivedCard did.A2AAgentCard
json.Unmarshal(cardData, &receivedCard)

// 2. Validate structure
if err := did.ValidateA2ACard(&receivedCard); err != nil {
    return fmt.Errorf("invalid card: %w", err)
}

// 3. Verify against blockchain
agent, err := manager.ResolveAgent(ctx, receivedCard.ID)
if err != nil {
    return fmt.Errorf("DID not found: %w", err)
}

// 4. Cross-check data
if receivedCard.Name != agent.Name {
    return fmt.Errorf("name mismatch")
}

// 5. Establish trust
// Now safe to use the card for secure communication
```

## Key Concepts

### Trust Model

```
┌─────────────┐         ┌──────────────┐
│   Agent A   │────────▶│   Blockchain │
│  Issues Card│         │  (Registry)  │
└─────────────┘         └──────┬───────┘
                               │
                               │ Verify
                               ▼
┌─────────────┐         ┌──────────────┐
│   Agent B   │────────▶│ Verification │
│Receives Card│         │    Result    │
└─────────────┘         └──────────────┘
```

Trust is established through:
1. **Blockchain anchoring**: DID is on-chain
2. **Data integrity**: Card matches blockchain
3. **Liveness check**: Agent is active

### Threat Model

| Threat | Mitigation |
|--------|------------|
| Fake DID | Blockchain resolution fails |
| Tampered card | Cross-check detects mismatch |
| Revoked agent | Active status check |
| Man-in-the-middle | Use HTTPS for transport |

## Next Steps

1. **Run Example 04**: Secure messaging with the exchanged cards
2. **Implement caching**: Store verified cards locally
3. **Add expiration**: Refresh cards periodically
4. **Build discovery**: Create agent directory services

## License

LGPL-v3 - See [LICENSE](../../../LICENSE)
