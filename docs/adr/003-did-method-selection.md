# ADR-003: DID Method Selection for Agent Identity

**Status:** Accepted

**Date:** 2024-10-26

**Decision Makers:** SAGE Core Team

**Technical Story:** [DID Implementation](https://github.com/sage-x-project/sage/tree/main/pkg/agent/did)

---

## Context

SAGE (Secure Agent Guarantee Engine) enables secure communication between AI agents. For agents to establish trust and authenticate each other, they need verifiable digital identities. This identity system must support:

1. **Decentralization**: No single point of failure or central authority
2. **Verifiability**: Anyone can verify an agent's identity and keys
3. **Persistence**: Identities remain valid across sessions and deployments
4. **Ownership**: Agents control their own identities
5. **Interoperability**: Cross-platform and cross-implementation compatibility
6. **Auditability**: Identity operations are transparent and traceable

### Problem Statement

Traditional identity systems present several challenges for AI agent communication:

1. **Centralization Risks**
   - X.509 certificates require trusted Certificate Authorities (CAs)
   - OAuth/OIDC depend on central identity providers
   - Single points of failure
   - Censorship and access control by authorities

2. **Trust Establishment**
   - How does Agent A verify Agent B's public key?
   - How to prevent man-in-the-middle attacks?
   - How to ensure keys haven't been compromised?

3. **Agent Mobility**
   - Agents may migrate between servers
   - Identity must persist across deployments
   - No reliance on DNS or fixed IP addresses

4. **Scalability**
   - Support millions of agents
   - Fast identity resolution (<100ms)
   - Reasonable operational costs

5. **Standards Compliance**
   - Interoperability with other agent platforms
   - Future-proof identity format
   - Well-defined specifications

6. **Multi-Chain Support**
   - Different applications prefer different blockchains
   - Ethereum for EVM compatibility
   - Solana for performance
   - Potential future chains (Cosmos, Polkadot, etc.)

### Requirements

**Must Have:**
- Decentralized identity (no central authority)
- Blockchain-anchored for immutability
- Public key binding to identity
- W3C DID specification compliance
- Multi-chain support (Ethereum + Solana minimum)

**Should Have:**
- Fast resolution (<100ms typical)
- Low operational cost
- Support for key rotation
- Metadata extensibility
- Privacy-preserving options

**Nice to Have:**
- DIDs readable by humans
- Compatibility with existing DID infrastructure
- Support for agent capabilities/schemas
- Integration with A2A (Agent-to-Agent) protocol

---

## Decision

We decided to implement a **custom DID method called `did:sage`** with blockchain-based registries on Ethereum and Solana.

### DID Format

#### Ethereum DIDs

```
did:sage:ethereum:<ethereum-address>
```

**Example:**
```
did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266
```

**Components:**
- `did:` - W3C DID scheme
- `sage` - Our custom DID method name
- `ethereum` - Blockchain network identifier
- `0xf39fd...` - Ethereum address (20 bytes, hex-encoded)

#### Solana DIDs

```
did:sage:solana:<base58-public-key>
```

**Example:**
```
did:sage:solana:CuieVDEDtLo7FypA9SbLM9saXFdb1dsshEkyErMqkRQq
```

**Components:**
- `did:` - W3C DID scheme
- `sage` - Our custom DID method name
- `solana` - Blockchain network identifier
- `CuieV...` - Base58-encoded Ed25519 public key (32 bytes)

### Architecture

```
┌───────────────────────────────────────────────────────┐
│  SAGE Agent A                                         │
│  Identity: did:sage:ethereum:0xAlice                  │
└───────────────────┬───────────────────────────────────┘
                    │
                    │ 1. Resolve Agent B's DID
                    ▼
┌───────────────────────────────────────────────────────┐
│  SAGE DID Resolver                                    │
│  ├─ Parse DID (method, chain, identifier)            │
│  ├─ Select blockchain client                         │
│  └─ Query on-chain registry                          │
└───────────────────┬───────────────────────────────────┘
                    │
                    │ 2. Query blockchain
                    ▼
┌───────────────────────────────────────────────────────┐
│  Blockchain Registries                                │
│  ┌─────────────────────────────────────────────────┐ │
│  │ Ethereum: SageRegistryV4                        │ │
│  │ Contract Address: 0x...                         │ │
│  │ Data: {                                         │ │
│  │   owner: 0xBob,                                 │ │
│  │   publicKeys: [key1, key2, ...],               │ │
│  │   metadata: {...},                              │ │
│  │   isActive: true                                │ │
│  │ }                                               │ │
│  └─────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────┐ │
│  │ Solana: Agent Registry Program                  │ │
│  │ Account: AgentData {                            │ │
│  │   owner: Bob's pubkey,                          │ │
│  │   public_key: Ed25519 key,                      │ │
│  │   metadata: {...},                              │ │
│  │   is_active: true                               │ │
│  │ }                                               │ │
│  └─────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────┘
                    │
                    │ 3. Return DID Document
                    ▼
┌───────────────────────────────────────────────────────┐
│  DID Document (JSON)                                  │
│  {                                                    │
│    "id": "did:sage:ethereum:0xBob",                   │
│    "publicKey": [{                                    │
│      "id": "#key-1",                                  │
│      "type": "Ed25519VerificationKey2020",            │
│      "publicKeyMultibase": "z6Mk..."                  │
│    }],                                                │
│    "authentication": ["#key-1"],                      │
│    "service": [{                                      │
│      "type": "AgentService",                          │
│      "serviceEndpoint": "https://agent-b.example.com" │
│    }]                                                 │
│  }                                                    │
└───────────────────────────────────────────────────────┘
```

### Implementation Details

#### On-Chain Storage

**Ethereum (SageRegistryV4):**
```solidity
struct AgentData {
    address owner;           // Agent owner address
    bytes[] publicKeys;      // Up to 10 keys
    string metadata;         // JSON metadata
    bool isActive;           // Agent status
    uint256 updatedAt;       // Last update timestamp
}

mapping(address => AgentData) public agents;
```

**Solana (Rust Program):**
```rust
pub struct AgentAccount {
    pub owner: Pubkey,           // Owner public key
    pub public_key: [u8; 32],    // Ed25519 public key
    pub metadata: String,         // JSON metadata
    pub is_active: bool,          // Agent status
    pub updated_at: i64,          // Unix timestamp
}
```

#### DID Resolution Process

1. **Parse DID**: Extract method, chain, and identifier
   ```
   did:sage:ethereum:0xf39fd... → {
     method: "sage",
     chain: "ethereum",
     address: "0xf39fd..."
   }
   ```

2. **Select Blockchain Client**: Based on chain identifier
   - `ethereum` → Ethereum client (go-ethereum)
   - `solana` → Solana client (solana-go-sdk)

3. **Query Registry**: Call smart contract/program
   - Ethereum: `registry.getAgent(address)`
   - Solana: Fetch agent account data

4. **Construct DID Document**: Transform on-chain data to W3C format
   ```json
   {
     "id": "did:sage:ethereum:0x...",
     "publicKey": [...],
     "authentication": [...],
     "service": [...]
   }
   ```

5. **Return + Cache**: Cache for performance (5-minute TTL)

### Multi-Chain Strategy

**Why Multiple Chains?**
- **Ethereum**: EVM compatibility, large ecosystem, tooling
- **Solana**: High performance, low fees, Ed25519 native

**Chain Selection Guide:**
- **High-Value Agents**: Ethereum (more security, higher fees)
- **High-Throughput**: Solana (lower fees, faster finality)
- **EVM Compatibility**: Ethereum (many tools, auditors)
- **Ed25519 Native**: Solana (simpler key management)

---

## Consequences

### Positive

1. **Decentralization**
   - No central identity authority
   - Censorship-resistant
   - No single point of failure
   - Agent owns their identity (private key controls DID)

2. **Verifiability**
   - Anyone can verify agent identity on blockchain
   - Public keys cryptographically bound to DID
   - Immutable registration history (blockchain audit trail)
   - On-chain verification of ownership

3. **W3C Standards Compliance**
   - Interoperable with other DID systems
   - Well-defined specification
   - Tooling compatibility (DID resolvers, validators)
   - Future-proof (standards evolve gradually)

4. **Multi-Chain Flexibility**
   - Choose blockchain based on requirements
   - Not locked into single ecosystem
   - Risk diversification (if one chain fails, others continue)
   - Leverage unique chain features (Ethereum security vs. Solana speed)

5. **Persistence**
   - Identity persists across agent deployments
   - No dependence on DNS or IP addresses
   - Survives server migrations
   - Long-term stability (blockchain immutability)

6. **Metadata Extensibility**
   - Store arbitrary metadata on-chain
   - Support agent capabilities, schemas, endpoints
   - Versioning and updates possible
   - A2A Agent Card compatibility

### Negative

1. **Blockchain Costs**
   - **Ethereum**: ~$5-50 per registration (depending on gas price)
   - **Solana**: ~$0.0001 per registration (much cheaper)
   - Key updates cost additional gas/SOL
   - May be expensive for large-scale agent deployments
   - **Mitigation**: Use Solana for cost-sensitive apps, batch operations

2. **Resolution Latency**
   - Blockchain queries add latency (~50-200ms)
   - Cache misses hurt performance
   - Real-time applications may notice delay
   - **Mitigation**: Aggressive caching (5-minute TTL), local resolvers

3. **Blockchain Dependency**
   - Requires blockchain node access (Infura, Alchemy, etc.)
   - Vulnerable to blockchain downtime/congestion
   - Must handle chain reorganizations
   - **Mitigation**: Fallback providers, local caching, multi-provider redundancy

4. **Privacy Concerns**
   - All registrations are public
   - Agent metadata visible on-chain
   - Activity can be traced (registration, updates)
   - **Mitigation**: Separate identities for different contexts, metadata minimization

5. **Key Rotation Complexity**
   - Updating keys requires blockchain transaction
   - Costs gas fees
   - Takes time to finalize (block confirmation)
   - **Mitigation**: Design for infrequent key rotation, use subkeys

6. **Operational Complexity**
   - Must manage blockchain accounts
   - Private key security critical
   - Need ETH/SOL for transactions
   - **Mitigation**: Automated key management, hardware wallets, gas relayers

### Trade-offs Accepted

- **Cost vs. Decentralization**: We accept blockchain costs for decentralization benefits
- **Latency vs. Verifiability**: We accept query latency for cryptographic verifiability
- **Privacy vs. Transparency**: We accept public registration for transparent verification
- **Complexity vs. Trust**: We accept operational complexity to eliminate central trust

---

## Alternatives Considered

### Alternative 1: X.509 Certificates (PKI)

**Approach:** Use traditional X.509 certificates issued by Certificate Authorities.

**Pros:**
- Extremely well-established (HTTPS, email, code signing)
- Mature tooling and infrastructure
- Hardware support (HSMs, TPMs)
- FIPS validated implementations

**Cons:**
- **Centralized**: Requires trusted Certificate Authorities
  - Single point of failure
  - CA compromise affects all certificates
  - CAs can revoke certificates arbitrarily
- **Cost**: Commercial CAs charge fees
- **Complexity**: Certificate lifecycle (issuance, renewal, revocation)
- **Not Agent-Friendly**: Designed for human-operated systems
- **DNS Dependency**: Many use cases require DNS validation

**Why Rejected:** Too centralized. AI agents should not depend on human-operated CAs. Doesn't align with SAGE's decentralization goals.

---

### Alternative 2: OAuth 2.0 / OpenID Connect

**Approach:** Use OAuth for agent authentication.

**Pros:**
- Industry standard for API authentication
- Many libraries and implementations
- Token-based (no certificates)
- Well-understood security model

**Cons:**
- **Centralized Identity Provider**: Requires OAuth server
  - Single point of failure
  - All agents must trust one provider
  - Provider controls access
- **Not Suitable for P2P**: Designed for client-server
- **Short-Lived Tokens**: Constant renewal overhead
- **Privacy**: IdP knows all authentications

**Why Rejected:** OAuth requires central identity provider. Doesn't support decentralized peer-to-peer agent communication.

---

### Alternative 3: did:key Method

**Approach:** Use `did:key` method (keys encoded directly in DID).

**Pros:**
- **Simplest DID method**: No blockchain, no registration
- **Offline**: Works without network
- **Free**: No transaction costs
- **Fast**: Instant resolution (parse DID to extract key)

**Example:**
```
did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK
```

**Cons:**
- **No Blockchain Anchor**: Can't verify registration
- **No Key Rotation**: DID is tied to key (can't update)
- **No Metadata**: Can't store agent capabilities, endpoints
- **No Revocation**: Can't deactivate compromised keys
- **No Ownership Proof**: Anyone can generate `did:key`

**Why Rejected:** Lacks blockchain anchoring for trust. Can't rotate keys or store metadata. Doesn't meet our verifiability requirements.

---

### Alternative 4: did:web Method

**Approach:** Use `did:web` method (DIDs backed by HTTPS/DNS).

**Pros:**
- **Web-Native**: Uses existing web infrastructure
- **No Blockchain**: No transaction costs
- **Fast Resolution**: HTTPS is fast
- **Mutable**: Can update DID documents easily

**Example:**
```
did:web:agent.example.com
→ Resolves to https://agent.example.com/.well-known/did.json
```

**Cons:**
- **DNS Dependency**: Relies on centralized DNS
  - DNS hijacking risk
  - Domain expiration problem
  - Censorship via DNS
- **HTTPS Dependency**: Requires TLS certificates (back to PKI)
- **Domain Costs**: Must maintain domain registration
- **Not Persistent**: DIDs die if domain expires

**Why Rejected:** DNS and HTTPS are centralized. Doesn't provide decentralization benefits we need.

---

### Alternative 5: did:ethr Method

**Approach:** Use existing `did:ethr` method (Ethereum-based DIDs).

**Pros:**
- **Ethereum-Based**: Similar to our approach
- **Existing Infrastructure**: ERC-1056 registry
- **Tooling**: Supported by uPort, Veramo, etc.
- **Standards**: Well-specified

**Example:**
```
did:ethr:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266
```

**Cons:**
- **Ethereum-Only**: No Solana support
- **Legacy Registry**: ERC-1056 predates modern patterns
- **Limited Metadata**: Not optimized for agents
- **No Multi-Key Support**: Single key per DID
- **Not Agent-Focused**: Designed for human identity

**Why Rejected:** We need multi-chain support (Solana) and agent-specific features (multiple keys, metadata). `did:ethr` is human-centric.

---

### Alternative 6: Self-Signed Certificates

**Approach:** Agents generate and sign their own certificates.

**Pros:**
- **No CA Required**: Self-sufficient
- **Free**: No costs
- **Fast**: Generate instantly
- **Simple**: Standard X.509 tools work

**Cons:**
- **No Trust Chain**: How to verify first time?
- **No PKI**: Can't revoke or validate
- **TOFU Problem**: Trust-On-First-Use is vulnerable
- **No Discovery**: Can't find agents

**Why Rejected:** No trust establishment mechanism. Vulnerable to man-in-the-middle on first contact.

---

### Alternative 7: IPFS-Based Identity (IPNS)

**Approach:** Use IPFS/IPNS for identity documents.

**Pros:**
- **Decentralized Storage**: No central server
- **Content-Addressed**: Immutable by default
- **Mutable Names (IPNS)**: Can update identity

**Cons:**
- **No Canonical Registry**: Hard to enumerate agents
- **Availability**: Requires IPFS nodes online
- **No Consensus**: No agreed-upon state
- **Not Standardized**: No W3C DID spec

**Why Rejected:** IPFS provides storage, not identity. Doesn't solve trust establishment. Not a W3C DID method.

---

## Related Decisions

- **Multi-Chain Support**: Decided in ADR-003 (this document)
- **SageRegistryV4**: See [V4 Update Deployment Guide](../V4_UPDATE_DEPLOYMENT_GUIDE.md)
- **DID Document Format**: W3C DID Core 1.0 compliant

---

## Future Considerations

### Additional Blockchains

**Potential Future Support:**
- **Cosmos/IBC**: Interchain communication
- **Polkadot**: Parachain ecosystem
- **Cardano**: UTXO model
- **Near**: Fast finality

**Decision Process:**
1. Evaluate blockchain adoption in target markets
2. Assess cost and performance
3. Check native key support (Ed25519, Secp256k1)
4. Implement blockchain client
5. Deploy registry contract/program

### Privacy Enhancements

**Private DIDs (Future Work):**
- **Zero-Knowledge Proofs**: Prove identity without revealing DID
- **Stealth Addresses**: One-time use DIDs
- **Encrypted Metadata**: On-chain encrypted, off-chain decryption key

**Trade-off:** Privacy vs. Simplicity

### Cross-Chain Identity

**Potential Future:**
- **Bridged Identities**: Same agent on multiple chains
- **Universal DID**: Single DID resolves on any chain
- **Interchain Queries**: Query Ethereum DID from Solana

**Challenge:** Consensus across chains

---

## Implementation Notes

### Registering an Agent (Ethereum)

```go
import "github.com/sage-x-project/sage/pkg/agent/did"

// Initialize Ethereum client
client, _ := did.NewEthereumClient("https://eth-sepolia.g.alchemy.com/v2/...")

// Register agent
tx, err := client.RegisterAgent(
    did.AgentData{
        PublicKeys: [][]byte{ed25519PublicKey, x25519PublicKey},
        Metadata:   `{"name": "Agent A", "capabilities": ["chat", "search"]}`,
        IsActive:   true,
    },
    privateKey, // Ethereum private key
)
```

### Resolving a DID

```go
import "github.com/sage-x-project/sage/pkg/agent/did"

// Initialize DID Manager
manager := did.NewManager()
manager.AddChain("ethereum", ethereumClient)
manager.AddChain("solana", solanaClient)

// Resolve DID
didDoc, err := manager.Resolve("did:sage:ethereum:0xf39fd...")

// Access public key
publicKey := didDoc.PublicKeys[0]
```

### DID Document Structure

```json
{
  "@context": "https://www.w3.org/ns/did/v1",
  "id": "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
  "publicKey": [
    {
      "id": "did:sage:ethereum:0xf39fd...#key-1",
      "type": "Ed25519VerificationKey2020",
      "controller": "did:sage:ethereum:0xf39fd...",
      "publicKeyMultibase": "z6MkhaXgBZDvotDkL..."
    }
  ],
  "authentication": [
    "did:sage:ethereum:0xf39fd...#key-1"
  ],
  "service": [
    {
      "id": "did:sage:ethereum:0xf39fd...#agent-endpoint",
      "type": "AgentService",
      "serviceEndpoint": "https://agent-a.example.com"
    }
  ]
}
```

---

## Related Documents

- [DID Implementation](../../pkg/agent/did/README.md)
- [SageRegistryV4 Smart Contract](../../contracts/ethereum/contracts/SageRegistryV4.sol)
- [Solana Agent Program](../../contracts/solana/programs/agent_registry/)
- [W3C DID Core Specification](https://www.w3.org/TR/did-core/)
- [A2A Integration Guide](../SAGE_A2A_INTEGRATION_GUIDE.md)

---

## References

- [W3C DID Core 1.0](https://www.w3.org/TR/did-core/)
- [W3C DID Specification Registries](https://www.w3.org/TR/did-spec-registries/)
- [DID Method Rubric](https://www.w3.org/TR/did-rubric/)
- [Ethereum DID Registry (ERC-1056)](https://github.com/decentralized-identity/ethr-did-resolver)
- [Solana Account Model](https://docs.solana.com/developing/programming-model/accounts)

---

## Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2024-10-26 | 1.0 | SAGE Team | Initial ADR |

---

## Approval

This ADR has been reviewed and accepted by the SAGE core team. DID implementation is complete and in production use.

**Acceptance Criteria Met:**
-  `did:sage` method specified and implemented
-  Ethereum and Solana blockchain support
-  W3C DID Core 1.0 compliant
-  SageRegistryV4 smart contract deployed
-  Solana agent registry program deployed
-  DID resolution working (<100ms cached)
-  Multi-key support (up to 10 keys)
-  Comprehensive test coverage
