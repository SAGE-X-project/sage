# SAGE Architecture Overview

**Version**: 1.0.0
**Date**: October 2025
**Purpose**: Security Audit Reference

## Table of Contents

1. [System Architecture](#1-system-architecture)
2. [Component Interactions](#2-component-interactions)
3. [Data Flow](#3-data-flow)
4. [Security Boundaries](#4-security-boundaries)
5. [Threat Model](#5-threat-model)

---

## 1. System Architecture

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         AI Agent Layer                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ Agent A  │  │ Agent B  │  │ Agent C  │  │ Agent D  │       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
└───────┼─────────────┼─────────────┼─────────────┼──────────────┘
        │             │             │             │
        └─────────────┴─────────────┴─────────────┘
                      │
        ┌─────────────▼─────────────┐
        │   SAGE Security Layer     │
        │  ┌─────────────────────┐  │
        │  │  Handshake Protocol │  │
        │  │  ┌───────────────┐  │  │
        │  │  │ 1. Invitation │  │  │
        │  │  │ 2. Request    │  │  │
        │  │  │ 3. Response   │  │  │
        │  │  │ 4. Complete   │  │  │
        │  │  └───────────────┘  │  │
        │  └─────────────────────┘  │
        │  ┌─────────────────────┐  │
        │  │  Session Management │  │
        │  │  - AEAD Encryption  │  │
        │  │  - Replay Protection│  │
        │  └─────────────────────┘  │
        │  ┌─────────────────────┐  │
        │  │  RFC 9421 Signatures│  │
        │  │  - Message Signing  │  │
        │  │  - Verification     │  │
        │  └─────────────────────┘  │
        └────────────┬──────────────┘
                     │
        ┌────────────▼──────────────┐
        │  DID Registry (Blockchain)│
        │  ┌─────────────────────┐  │
        │  │  SageRegistryV2     │  │
        │  │  - Agent Registry   │  │
        │  │  - Key Management   │  │
        │  │  - Access Control   │  │
        │  └─────────────────────┘  │
        │  ┌─────────────────────┐  │
        │  │  ERC8004 Validation │  │
        │  │  - Identity Verify  │  │
        │  └─────────────────────┘  │
        └───────────────────────────┘
```

### 1.2 Component Layers

#### Layer 1: AI Agent Layer
- External AI agents (ChatGPT, Claude, custom agents)
- Communicates via HTTP/HTTPS
- Uses SAGE SDK for integration

#### Layer 2: SAGE Security Layer
- **Handshake Protocol**: Secure session establishment
- **Session Management**: Encrypted communication channels
- **RFC 9421**: HTTP message signatures
- **HPKE**: Hybrid Public Key Encryption

#### Layer 3: DID Registry (Blockchain)
- **SageRegistryV2**: Decentralized agent registry
- **ERC8004**: Identity validation standard
- **Ethereum Network**: Trust anchor

---

## 2. Component Interactions

### 2.1 Agent Registration Flow

```
┌─────────┐              ┌──────────────┐              ┌─────────────┐
│ Agent A │              │ SAGE Backend │              │  Ethereum   │
└────┬────┘              └──────┬───────┘              └──────┬──────┘
     │                          │                             │
     │ 1. Generate Key Pair     │                             │
     │─────────────────────────>│                             │
     │                          │                             │
     │ 2. Register Agent        │                             │
     │─────────────────────────>│                             │
     │                          │ 3. Create Transaction       │
     │                          │────────────────────────────>│
     │                          │                             │
     │                          │  4. Execute registerAgent() │
     │                          │<────────────────────────────│
     │                          │  - Validate public key      │
     │                          │  - Generate DID             │
     │                          │  - Store metadata           │
     │                          │                             │
     │ 5. Return DID            │                             │
     │<─────────────────────────│                             │
     │                          │                             │
     │ did:sage:ethereum:0x...  │                             │
     │                          │                             │
```

**Steps**:
1. Agent generates Ed25519 or Secp256k1 key pair
2. Agent calls SAGE backend with registration data
3. Backend creates Ethereum transaction
4. Smart contract validates and registers agent
5. DID returned to agent

**Security Points**:
- Public key must be 32 bytes (Ed25519) or 33 bytes (Secp256k1)
- Non-zero key validation
- Ownership proof required for updates
- Only owner can modify agent data

### 2.2 Handshake Protocol Flow

```
Agent A (Client)                    Agent B (Server)
┌──────────┐                        ┌──────────┐
│          │                        │          │
│ Step 1: Invitation                │          │
│──────────────────────────────────>│          │
│ {                                 │ Verify   │
│   "context_id": "abc123",         │ DID      │
│   "agent_did": "did:sage:eth:A",  │ Signature│
│   "signature": "..."              │          │
│ }                                 │          │
│                                   │          │
│                                   │<─────────│
│ Step 2: Request (with ephemeral)  │ Response │
│──────────────────────────────────>│          │
│ {                                 │ Decrypt  │
│   "ephemeral_key": "X25519_A",    │ with     │
│   "encrypted": true               │ B's key  │
│ }                                 │          │
│                                   │          │
│                                   │<─────────│
│ Step 3: Response (with ephemeral) │          │
│<──────────────────────────────────│          │
│ {                                 │ Encrypt  │
│   "ephemeral_key": "X25519_B",    │ with     │
│   "encrypted": true               │ A's key  │
│ }                                 │          │
│                                   │          │
│ Decrypt B's ephemeral             │          │
│                                   │          │
│ Step 4: Complete                  │          │
│──────────────────────────────────>│          │
│ {                                 │ Derive   │
│   "session_established": true     │ Session  │
│ }                                 │ Keys     │
│                                   │          │
│ Both derive session keys via HKDF │          │
│ SessionID = HKDF(sharedSecret)    │          │
│ Key_A→B = HKDF(sessionID, "c2s")  │          │
│ Key_B→A = HKDF(sessionID, "s2c")  │          │
│                                   │          │
└──────────┘                        └──────────┘
```

**Security Features**:
1. **DID Authentication**: Every message signed with agent's DID
2. **HPKE Encryption**: Ephemeral keys encrypted with peer's public key
3. **Forward Secrecy**: Ephemeral X25519 keys (not stored)
4. **Session Keys**: Derived from HKDF (SHA-256)
5. **Directional Keys**: Separate keys for each direction

### 2.3 Message Signing & Verification (RFC 9421)

```
┌─────────┐                          ┌─────────┐
│ Sender  │                          │Receiver │
└────┬────┘                          └────┬────┘
     │                                    │
     │ 1. Create HTTP Message             │
     │    Method: POST                    │
     │    Path: /api/chat                 │
     │    Body: {"msg": "hello"}          │
     │                                    │
     │ 2. Canonicalize Message            │
     │    "@method": "POST"               │
     │    "@path": "/api/chat"            │
     │    "content-digest": "sha-256=..." │
     │                                    │
     │ 3. Sign with Session Key           │
     │    HMAC-SHA256(canonicalized)      │
     │                                    │
     │ 4. Send Message + Signature        │
     │───────────────────────────────────>│
     │                                    │
     │                                    │ 5. Verify Signature
     │                                    │    - Lookup Session by KeyID
     │                                    │    - Canonicalize Message
     │                                    │    - HMAC-SHA256(canonicalized)
     │                                    │    - Compare signatures
     │                                    │
     │                                    │ 6. Check Nonce (Replay)
     │                                    │    - Is nonce fresh?
     │                                    │    - Add to cache
     │                                    │
     │ 7. Response (signed)               │
     │<───────────────────────────────────│
     │                                    │
```

---

## 3. Data Flow

### 3.1 Session Establishment Data Flow

```
Input:
  - Agent A DID: did:sage:ethereum:0xAAAA...
  - Agent B DID: did:sage:ethereum:0xBBBB...
  - Agent A ephemeral key: X25519_A (32 bytes)
  - Agent B ephemeral key: X25519_B (32 bytes)

Process:
  1. HPKE Key Agreement
     sharedSecret = X25519(X25519_A_private, X25519_B_public)

  2. Session ID Derivation
     sessionID = HKDF-SHA256(
       ikm: sharedSecret,
       salt: "SAGE-Session-v1",
       info: concat(DID_A, DID_B),
       length: 32
     )

  3. Directional Key Derivation
     Key_A→B = HKDF-SHA256(
       ikm: sessionID,
       salt: "client-to-server",
       info: concat(DID_A, DID_B),
       length: 32
     )

     Key_B→A = HKDF-SHA256(
       ikm: sessionID,
       salt: "server-to-client",
       info: concat(DID_B, DID_A),
       length: 32
     )

  4. Key ID Assignment
     KeyID_A→B = random(16 bytes) // Opaque identifier
     KeyID_B→A = random(16 bytes)

     Mapping:
       KeyID_A→B → SessionID
       KeyID_B→A → SessionID

Output:
  - SessionID: 32 bytes (deterministic)
  - Key_A→B: 32 bytes (for A→B messages)
  - Key_B→A: 32 bytes (for B→A messages)
  - KeyID_A→B: 16 bytes (opaque)
  - KeyID_B→A: 16 bytes (opaque)
```

### 3.2 Message Encryption Data Flow

```
Input:
  - Plaintext: "Transfer $100 to user X"
  - Session Key: Key_A→B (32 bytes)
  - Nonce: 12 bytes (random)

Process:
  1. Prepare Associated Data (AD)
     AD = concat(
       "SAGE-v1",
       timestamp,
       sender_DID,
       receiver_DID
     )

  2. ChaCha20-Poly1305 Encryption
     ciphertext, tag = ChaCha20Poly1305.Encrypt(
       key: Key_A→B,
       nonce: nonce,
       plaintext: plaintext,
       ad: AD
     )

  3. Package Message
     message = {
       "ciphertext": base64(ciphertext),
       "tag": base64(tag),
       "nonce": base64(nonce),
       "key_id": KeyID_A→B,
       "timestamp": timestamp
     }

Output:
  - Encrypted message package
  - Tag for authentication (16 bytes)
  - Nonce for replay prevention
```

---

## 4. Security Boundaries

### 4.1 Trust Boundaries

```
┌────────────────────────────────────────────────────────┐
│ Untrusted Zone (Internet)                              │
│                                                         │
│  AI Agents, External Services, Attackers               │
│                                                         │
└─────────────────┬──────────────────────────────────────┘
                  │ TLS/HTTPS
                  │ DID Authentication
                  │ Message Signatures
         ┌────────▼────────┐
         │ SAGE Backend    │ ← Trusted if properly configured
         │                 │
         │ - Session Mgmt  │
         │ - Crypto Ops    │
         │ - DID Resolver  │
         └────────┬────────┘
                  │ JSON-RPC
                  │ Transaction Signing
         ┌────────▼────────────┐
         │ Blockchain Network  │ ← Trusted (Consensus)
         │                     │
         │ - Smart Contracts   │
         │ - Agent Registry    │
         │ - State Storage     │
         └─────────────────────┘
```

**Trust Levels**:
1. **Blockchain**: Highest trust (decentralized consensus)
2. **SAGE Backend**: Trusted if keys are secure
3. **AI Agents**: Untrusted (must authenticate)
4. **Network**: Untrusted (TLS required)

### 4.2 Attack Surfaces

```
Attack Surface                   Mitigation
──────────────────────────────────────────────────────────
1. Network (MitM)                TLS, DID signatures
2. Agent Impersonation           DID registry, key verification
3. Replay Attacks                Nonce cache, timestamps
4. Session Hijacking             Session keys, Key IDs
5. Smart Contract Exploits       Access control, validation
6. Key Compromise                Key rotation, revocation
7. DoS Attacks                   Rate limiting, gas limits
8. Front-running                 Commit-reveal (future)
```

---

## 5. Threat Model

### 5.1 Threat Actors

#### T1: External Attacker (Network-level)
- **Capabilities**: Eavesdrop, intercept, modify network traffic
- **Goal**: Steal data, impersonate agents, disrupt communication
- **Mitigations**:
  - TLS encryption
  - DID-based authentication
  - Message signatures

#### T2: Malicious Agent
- **Capabilities**: Registered agent with valid DID
- **Goal**: Impersonate other agents, send malicious messages
- **Mitigations**:
  - Per-message signatures
  - Session key isolation
  - Nonce-based replay prevention

#### T3: Compromised Backend
- **Capabilities**: Full access to SAGE backend
- **Goal**: Access session keys, modify data
- **Mitigations**:
  - No private keys stored in backend
  - Session keys derived (not stored long-term)
  - Audit logging

#### T4: Smart Contract Exploiter
- **Capabilities**: Interact with smart contracts
- **Goal**: Bypass validation, register fake agents
- **Mitigations**:
  - 5-step key validation
  - Challenge-response ownership proof
  - Access control modifiers

### 5.2 Attack Scenarios

#### Scenario 1: Man-in-the-Middle Attack
```
Attacker intercepts handshake between Agent A and Agent B

Mitigation:
1. Ephemeral keys encrypted with HPKE (peer's public key)
2. DID signatures on all messages
3. Cannot derive session keys without private ephemeral keys
```

#### Scenario 2: Replay Attack
```
Attacker captures and replays valid message

Mitigation:
1. Nonce included in every message
2. Nonce cache prevents duplicate nonces
3. Timestamp validation (± 5 minutes tolerance)
4. Message rejected if nonce seen before
```

#### Scenario 3: Key Compromise
```
Agent's private key stolen by attacker

Mitigation:
1. Key revocation via smart contract
2. Agents can rotate keys with ownership proof
3. Compromised key marked as revoked on-chain
4. Sessions using revoked keys invalidated
```

#### Scenario 4: Session Hijacking
```
Attacker tries to use stolen Key ID

Mitigation:
1. Key IDs are opaque (random 16 bytes)
2. Session keys never transmitted
3. Cannot derive session key from Key ID alone
4. Sessions expire automatically
```

### 5.3 Security Properties

#### Property 1: Agent Authentication
```
Invariant: Only agent with private key can sign messages as that DID

Verification:
- DID → Public Key mapping on-chain
- Message signature verified against public key
- Cannot forge signature without private key
```

#### Property 2: Message Integrity
```
Invariant: Messages cannot be modified in transit

Verification:
- HMAC-SHA256 over canonicalized message
- Poly1305 authentication tag (AEAD)
- Any modification detected during verification
```

#### Property 3: Forward Secrecy
```
Invariant: Past sessions remain secure even if long-term key compromised

Verification:
- Ephemeral X25519 keys used per session
- Ephemeral keys deleted after handshake
- Session keys derived, not stored
- Compromise of long-term key doesn't reveal past session keys
```

#### Property 4: Replay Prevention
```
Invariant: Same message cannot be processed twice

Verification:
- Nonce must be unique per message
- Nonce cache stores used nonces
- Duplicate nonce → reject message
- Timestamp prevents infinite nonce storage
```

---

## 6. Cryptographic Primitives

### 6.1 Algorithm Selection

| Purpose                | Algorithm         | Key Size | Notes                    |
|------------------------|-------------------|----------|--------------------------|
| Agent Identity         | Ed25519           | 256-bit  | DID signatures           |
| Ethereum Signing       | Secp256k1         | 256-bit  | Blockchain transactions  |
| Key Agreement          | X25519            | 256-bit  | HPKE ephemeral keys      |
| RSA Signing            | RS256             | 2048-bit | RSA-PSS-SHA256           |
| Session Encryption     | ChaCha20-Poly1305 | 256-bit  | AEAD for messages        |
| Message Authentication | HMAC-SHA256       | 256-bit  | RFC 9421 signatures      |
| Key Derivation         | HKDF-SHA256       | 256-bit  | Session key derivation   |
| Vault Encryption       | AES-256-GCM       | 256-bit  | File-based key storage   |
| Vault Key Derivation   | PBKDF2-SHA256     | 256-bit  | 100K iterations          |

### 6.2 Cryptographic Flow

```
1. Key Generation (Agent Setup)
   ┌─────────────────────────────────────┐
   │ Random Source (crypto/rand)         │
   │ 32 bytes entropy                    │
   └──────────────┬──────────────────────┘
                  │
   ┌──────────────▼──────────────────────┐
   │ Ed25519.GenerateKey()               │
   │ - Private: 32 bytes                 │
   │ - Public: 32 bytes                  │
   └──────────────┬──────────────────────┘
                  │
   ┌──────────────▼──────────────────────┐
   │ Register Public Key on Blockchain   │
   │ DID = "did:sage:eth:" + addr        │
   └─────────────────────────────────────┘

2. Session Establishment (HPKE)
   ┌─────────────────────────────────────┐
   │ X25519.GenerateKey() (ephemeral)    │
   │ - Private_A: 32 bytes               │
   │ - Public_A: 32 bytes                │
   └──────────────┬──────────────────────┘
                  │
   ┌──────────────▼──────────────────────┐
   │ HPKE.Encap(Public_B, Public_A)      │
   │ - Shared Secret: 32 bytes           │
   │ - Encapsulated Key: 32 bytes        │
   └──────────────┬──────────────────────┘
                  │
   ┌──────────────▼──────────────────────┐
   │ HKDF-SHA256(sharedSecret)           │
   │ - Session ID: 32 bytes              │
   │ - Key_A→B: 32 bytes                 │
   │ - Key_B→A: 32 bytes                 │
   └─────────────────────────────────────┘

3. Message Protection (AEAD)
   ┌─────────────────────────────────────┐
   │ Message: "Transfer $100"            │
   │ Nonce: 12 bytes (random)            │
   │ AD: "SAGE-v1||timestamp||DIDs"      │
   └──────────────┬──────────────────────┘
                  │
   ┌──────────────▼──────────────────────┐
   │ ChaCha20-Poly1305.Encrypt(          │
   │   key: Key_A→B,                     │
   │   nonce: nonce,                     │
   │   plaintext: message,               │
   │   ad: AD                            │
   │ )                                   │
   └──────────────┬──────────────────────┘
                  │
   ┌──────────────▼──────────────────────┐
   │ Output:                             │
   │ - Ciphertext: var length            │
   │ - Tag: 16 bytes (authentication)    │
   └─────────────────────────────────────┘
```

---

## 7. Smart Contract Architecture

### 7.1 Contract Hierarchy

```
┌────────────────────────────────────────┐
│  Proxy (UUPS Upgradeable)              │
│  - Delegatecall to Implementation      │
│  - Upgrade controlled by admin         │
└──────────────┬─────────────────────────┘
               │
┌──────────────▼─────────────────────────┐
│  SageRegistryV2 (Implementation)       │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │ State Variables                  │ │
│  │ - agents mapping                 │ │
│  │ - agentsByOwner mapping          │ │
│  │ - didToAddress mapping           │ │
│  └──────────────────────────────────┘ │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │ Modifiers                        │ │
│  │ - onlyAgentOwner                 │ │
│  │ - whenNotPaused                  │ │
│  └──────────────────────────────────┘ │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │ External Functions               │ │
│  │ - registerAgent()                │ │
│  │ - updatePublicKey()              │ │
│  │ - revokeKey()                    │ │
│  │ - deactivateAgent()              │ │
│  └──────────────────────────────────┘ │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │ Hooks                            │ │
│  │ - beforeRegister()               │ │
│  │ - beforeUpdate()                 │ │
│  └──────────────────────────────────┘ │
└────────────────────────────────────────┘
```

### 7.2 Access Control Matrix

| Function            | Caller       | Checks                                    |
|---------------------|--------------|-------------------------------------------|
| registerAgent       | Anyone       | Public key validation, ERC8004 check      |
| updatePublicKey     | Owner only   | Ownership proof, not revoked              |
| revokeKey           | Owner only   | Is owner of agent                         |
| deactivateAgent     | Owner only   | Is owner of agent                         |
| setHook             | Admin only   | onlyRole(DEFAULT_ADMIN_ROLE)              |
| pause/unpause       | Admin only   | onlyRole(DEFAULT_ADMIN_ROLE)              |
| upgrade             | Admin only   | UUPS authorization check                  |

---

## 8. Operational Security

### 8.1 Key Management Lifecycle

```
1. Generation
   ├─ Use crypto/rand (Go) or equivalent
   ├─ 32 bytes minimum entropy
   └─ No predictable seeds

2. Storage
   ├─ AES-256-GCM encrypted files (crypto/vault)
   ├─ PBKDF2 passphrase-based encryption (100K iterations)
   ├─ Never in plaintext on disk
   └─ File permissions: 0600

3. Usage
   ├─ Loaded into memory only when needed
   ├─ Memory zeroed after use
   └─ No key logging

4. Rotation
   ├─ Generate new key pair
   ├─ Submit updatePublicKey() with proof
   ├─ Old key revoked automatically
   └─ New sessions use new key

5. Revocation
   ├─ Call revokeKey() on-chain
   ├─ Key marked as revoked
   ├─ Existing sessions invalidated
   └─ Cannot be un-revoked
```

### 8.2 Monitoring & Logging

**Logged Events**:
- Agent registration
- Key updates
- Key revocations
- Session establishment
- Signature verification failures
- Nonce cache hits (replay attempts)

**Metrics**:
- Active sessions count
- Messages per second
- Signature verification latency
- Handshake success rate
- DID resolution time

---

## 9. Deployment Architecture

```
Production Environment (Ethereum Mainnet)

┌────────────────────────────────────────────────┐
│  Load Balancer (CloudFlare / AWS ELB)          │
└──────────────┬─────────────────────────────────┘
               │
    ┌──────────┴──────────┐
    │                     │
┌───▼────┐          ┌─────▼───┐
│ SAGE   │          │ SAGE    │
│ Node 1 │          │ Node 2  │
│        │          │         │
│ - API  │          │ - API   │
│ - DID  │          │ - DID   │
└───┬────┘          └─────┬───┘
    │                     │
    └──────────┬──────────┘
               │
    ┌──────────▼──────────────────┐
    │ Shared Redis (Session Cache)│
    └──────────┬──────────────────┘
               │
    ┌──────────▼──────────────────┐
    │ Ethereum Node (Alchemy/Infura)│
    │                             │
    │ - SageRegistryV2            │
    │ - ERC8004ValidationRegistry │
    └─────────────────────────────┘
```

---

**Document Version**: 1.0
**Last Updated**: October 2025
**Status**: Ready for Audit
