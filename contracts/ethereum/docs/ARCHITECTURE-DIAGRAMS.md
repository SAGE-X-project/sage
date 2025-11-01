# SAGE Smart Contract Architecture Diagrams

**Version**: 2.0
**Date**: 2025-11-01
**Status**: Current - AgentCard Architecture

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Component Architecture](#component-architecture)
3. [Agent Registration Flow](#agent-registration-flow)
4. [Validation Flow](#validation-flow)
5. [Reputation Flow](#reputation-flow)
6. [TEE Key Governance Flow](#tee-key-governance-flow)
7. [Data Flow Diagrams](#data-flow-diagrams)
8. [Security Architecture](#security-architecture)

---

## System Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     SAGE AgentCard Ecosystem                    │
│                                                                 │
│  ┌───────────────┐  ┌───────────────┐  ┌──────────────────┐  │
│  │   Clients     │  │  AI Agents    │  │   Validators     │  │
│  │  (Users/Dapps)│  │  (Services)   │  │  (Verifiers)     │  │
│  └───────┬───────┘  └───────┬───────┘  └────────┬─────────┘  │
│          │                  │                     │            │
│          └──────────────────┼─────────────────────┘            │
│                             │                                  │
│  ┌──────────────────────────▼──────────────────────────────┐  │
│  │           Smart Contract Layer (Ethereum)               │  │
│  │                                                          │  │
│  │  ┌──────────────────┐  ┌────────────┐  ┌──────────┐   │  │
│  │  │  AgentCard       │  │ Validation │  │ Reputation│   │  │
│  │  │  Registry        │  │ Registry   │  │ Registry  │   │  │
│  │  │ (Native ERC-8004)│  │ (ERC-8004) │  │(ERC-8004) │   │  │
│  │  └────────┬─────────┘  └─────┬──────┘  └─────┬─────┘   │  │
│  │           │                   │                │         │  │
│  │           │  ┌────────────────▼────────────────▼───┐    │  │
│  │           │  │     AgentCardVerifyHook            │    │  │
│  │           │  │  (DID validation, Rate limiting)   │    │  │
│  │           │  └────────────────────────────────────┘    │  │
│  │           │                                             │  │
│  │           │  ┌────────────────┐                        │  │
│  │           └─►│ AgentCard      │                        │  │
│  │              │ Storage        │                        │  │
│  │              └────────────────┘                        │  │
│  │                                                          │  │
│  │  ┌──────────────────────────────────────────────────┐  │  │
│  │  │            Governance Layer                       │  │
│  │  │  ┌──────────────┐  ┌───────────┐  ┌──────────┐ │  │
│  │  │  │ TEEKey       │  │ MultiSig  │  │ Timelock │ │  │
│  │  │  │ Registry     │  │           │  │          │ │  │
│  │  │  └──────────────┘  └───────────┘  └──────────┘ │  │
│  │  └──────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Purpose | Type |
|-----------|---------|------|
| **AgentCardRegistry** | Main agent identity & multi-key management | Core Contract |
| **AgentCardStorage** | Isolated storage layer for upgradability | Abstract Contract |
| **AgentCardVerifyHook** | Pre-registration validation & security | Hook Contract |
| **ERC8004ValidationRegistry** | Task validation coordination | Standalone |
| **ERC8004ReputationRegistry** | Agent reputation tracking | Standalone |
| **TEEKeyRegistry** | Decentralized TEE key approval | Governance |
| **SimpleMultiSig** | Multi-signature wallet | Governance |
| **TimelockController** | Time-delayed execution | Governance |

---

## Component Architecture

### Detailed Component Interaction

```
┌──────────────────────────────────────────────────────────────┐
│                  AgentCard Architecture                      │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │          AgentCardRegistry.sol                     │    │
│  │  (Main Logic + Native ERC-8004 Implementation)     │    │
│  │                                                     │    │
│  │  • Multi-key support (ECDSA, Ed25519, X25519)     │    │
│  │  • Commit-reveal pattern                           │    │
│  │  • Stake & time-lock activation                    │    │
│  │  • KME public key storage (X25519)                 │    │
│  │  • ERC-8004 compliant interface                    │    │
│  └──────────────┬─────────────────────────────────────┘    │
│                 │ inherits                                  │
│  ┌──────────────▼─────────────────────────────────────┐    │
│  │          AgentCardStorage.sol                      │    │
│  │  (Isolated Storage Layer)                          │    │
│  │                                                     │    │
│  │  • Agent metadata mapping                          │    │
│  │  • Multi-key storage                               │    │
│  │  • Commit-reveal data                              │    │
│  │  • Nonce & rate limiting                           │    │
│  │  • Public key reuse prevention                     │    │
│  └──────────────┬─────────────────────────────────────┘    │
│                 │ validates with                            │
│  ┌──────────────▼─────────────────────────────────────┐    │
│  │        AgentCardVerifyHook.sol                     │    │
│  │  (Pre-registration Validation)                     │    │
│  │                                                     │    │
│  │  • DID format validation                           │    │
│  │  • Rate limiting (24/day)                          │    │
│  │  • Blacklist/whitelist                             │    │
│  │  • Public key reuse check                          │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│                   ERC-8004 Ecosystem                         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │     ERC8004ValidationRegistry.sol                  │    │
│  │  (Standalone - Task Validation)                    │    │
│  │                                                     │    │
│  │  • Stake-based validation                          │    │
│  │  • TEE attestation support                         │    │
│  │  • Consensus mechanism (66% threshold)             │    │
│  │  • Automatic finalization                          │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │     ERC8004ReputationRegistry.sol                  │    │
│  │  (Standalone - Reputation Management)              │    │
│  │                                                     │    │
│  │  • Task authorization (commit-reveal)              │    │
│  │  • Feedback submission                             │    │
│  │  • Off-chain aggregation support                   │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │     ERC8004IdentityRegistry.sol                    │    │
│  │  (Standalone - Alternative Implementation)         │    │
│  │                                                     │    │
│  │  Note: AgentCardRegistry provides native ERC-8004  │    │
│  │  This standalone version is for reference only     │    │
│  └────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
```

### Contract Interface Relationships

```
                  ┌──────────────────────────┐
                  │  IERC8004IdentityRegistry│
                  │  (ERC-8004 Interface)    │
                  └─────────────┬────────────┘
                                │
                                │ implements (native)
                                │
                  ┌─────────────▼────────────┐
                  │  AgentCardRegistry       │
                  │  (Main Implementation)   │
                  │                          │
                  │  • Multi-key support     │
                  │  • Commit-reveal         │
                  │  • KME key storage       │
                  │  • Native ERC-8004       │
                  └────────┬─────────────────┘
                           │
                           │ uses
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
         │                 │                 │
┌────────▼──────┐  ┌───────▼────────┐  ┌────▼──────────┐
│ Validation    │  │  Reputation    │  │  VerifyHook   │
│ Registry      │  │  Registry      │  │  (Validation) │
│ (Standalone)  │  │  (Standalone)  │  │               │
└───────────────┘  └────────────────┘  └───────────────┘
```

---

## Agent Registration Flow

### Commit-Reveal Registration Process

```
┌─────────┐                                    ┌──────────────────┐
│  Client │                                    │ AgentCardRegistry│
└────┬────┘                                    └──────┬───────────┘
     │                                                │
     │ 1. Generate salt & compute hash                │
     │    hash = keccak256(did, keys,                 │
     │           owner, salt, chainId)                │
     │                                                │
     │ 2. commitRegistration(hash) + 0.01 ETH         │
     ├───────────────────────────────────────────────►│
     │                                                │
     │                                          Store │
     │                                          ┌─────┤
     │                                          │     │
     │                                    commitHash, │
     │                                    timestamp,  │
     │                                    revealed=false
     │                                          └─────┤
     │                                                │
     │ ◄──emit RegistrationCommitted()───────────────┤
     │                                                │
     │                                                │
     │ 3. Wait 1 minute (MIN_DELAY)                   │
     │    [60 seconds]                                 │
     │                                                │
     │                                                │
     │ 4. registerAgent(                              │
     │      RegistrationParams {                      │
     │        did, name, description,                 │
     │        endpoint, keys[], keyTypes[],           │
     │        signatures[], capabilities,             │
     │        kmePublicKey (optional)                 │
     │      },                                        │
     │      salt                                      │
     │    )                                           │
     ├───────────────────────────────────────────────►│
     │                                                │
     │                                          Verify│
     │                                          ┌─────┤
     │                                          │     │
     │                                   Hash match  │
     │                                   Timing OK   │
     │                                   Call hook   │
     │                                   Verify keys │
     │                                   Store data  │
     │                                          │     │
     │                                          └─────┤
     │                                                │
     │                                       Register │
     │                                       agentId  │
     │                                       Set activation
     │                                       time = now + 1h
     │                                                │
     │ ◄──emit AgentRegistered(agentId)──────────────┤
     │                                                │
     │ ◄──return agentId──────────────────────────────┤
     │                                                │
     │                                                │
     │ 5. Wait 1 hour for activation                  │
     │    [activation delay]                           │
     │                                                │
     │ 6. activateAgent(agentId)                      │
     ├───────────────────────────────────────────────►│
     │                                                │
     │                                    Set active=true
     │                                                │
     │ ◄──emit AgentActivated(agentId)───────────────┤
     │                                                │
```

### Multi-Key Support

```
┌────────────────────────────────────────────────────────┐
│  Agent with Multiple Keys (up to 10)                   │
│                                                        │
│  Agent ID: 0x5c7c...                                  │
│  DID: did:sage:ethereum:0x123...                      │
│  Owner: 0x123...                                      │
│                                                        │
│  ┌──────────────────────────────────────────────────┐ │
│  │ Key 1: ECDSA (secp256k1)                         │ │
│  │  • Public Key: 0x04... (65 bytes uncompressed)   │ │
│  │  • Signature: 0x...                              │ │
│  │  • Verified: Yes (on-chain ecrecover)            │ │
│  │  • Use: Ethereum transactions                    │ │
│  └──────────────────────────────────────────────────┘ │
│                                                        │
│  ┌──────────────────────────────────────────────────┐ │
│  │ Key 2: Ed25519                                    │ │
│  │  • Public Key: 0x... (32 bytes)                  │ │
│  │  • Signature: 0x... (64 bytes)                   │ │
│  │  • Verified: Yes (owner pre-approved)            │ │
│  │  • Use: High-performance signing (did:key)       │ │
│  └──────────────────────────────────────────────────┘ │
│                                                        │
│  ┌──────────────────────────────────────────────────┐ │
│  │ Key 3: X25519 (KME Public Key)                   │ │
│  │  • Public Key: 0x... (32 bytes)                  │ │
│  │  • Signature: N/A (encryption key)               │ │
│  │  • Verified: Yes (no signature needed)           │ │
│  │  • Use: HPKE encryption, ECDH key exchange       │ │
│  │  • Stored in: agent.kmePublicKey field           │ │
│  └──────────────────────────────────────────────────┘ │
│                                                        │
│  Total Keys: 3/10                                     │
│  Storage: 3 x keyHash in keyHashes[]                  │
│           + 1 kmePublicKey in metadata                │
└────────────────────────────────────────────────────────┘
```

---

## Validation Flow

### Complete Validation Process

```
┌────────┐  ┌────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Client │  │ Server │  │  Validation  │  │ Validators   │  │  Reputation  │
│        │  │ Agent  │  │  Registry    │  │   (×N)       │  │  Registry    │
└───┬────┘  └───┬────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
    │           │               │                  │                  │
    │ 1. Assign task           │                  │                  │
    ├──────────►│               │                  │                  │
    │           │               │                  │                  │
    │           │ 2. Execute task                  │                  │
    │           │               │                  │                  │
    │           │ 3. Return result                 │                  │
    │◄──────────┤               │                  │                  │
    │           │               │                  │                  │
    │ 4. requestValidation(     │                  │                  │
    │      taskId,              │                  │                  │
    │      serverAgent,         │                  │                  │
    │      dataHash,            │                  │                  │
    │      validationType,      │                  │                  │
    │      deadline             │                  │                  │
    │    ) + stake              │                  │                  │
    ├──────────────────────────►│                  │                  │
    │                           │                  │                  │
    │                     Create│                  │                  │
    │                     request                  │                  │
    │                           │                  │                  │
    │ ◄──emit ValidationRequested(requestId)───────┤                  │
    │                           │                  │                  │
    │                           │                  │                  │
    │                           │ 5. Listen for requests              │
    │                           │◄─────────────────┤                  │
    │                           │                  │                  │
    │                           │                  │ 6. Re-execute    │
    │                           │                  │    task          │
    │                           │                  │                  │
    │                           │ 7. submitStakeValidation(           │
    │                           │      requestId,                     │
    │                           │      computedHash                   │
    │                           │    ) + stake                        │
    │                           │◄─────────────────┤                  │
    │                           │                  │                  │
    │                           │    (Repeat for N validators)        │
    │                           │◄─────────────────┤                  │
    │                           │◄─────────────────┤                  │
    │                           │◄─────────────────┤                  │
    │                           │                  │                  │
    │                     Check │                  │                  │
    │                     if min│                  │                  │
    │                     validators               │                  │
    │                     reached                  │                  │
    │                           │                  │                  │
    │                    Calculate                 │                  │
    │                    consensus                 │                  │
    │                           │                  │                  │
    │                           │ 8. Auto-finalize if consensus       │
    │                           │                  │                  │
    │                    ┌──────┴──────┐           │                  │
    │                    │ SUCCESS?    │           │                  │
    │                    └──────┬──────┘           │                  │
    │                           │                  │                  │
    │              ┌────────────┼────────────┐     │                  │
    │              │            │            │     │                  │
    │         YES  │            │            │ NO  │                  │
    │              │            │            │     │                  │
    │    ┌─────────▼──┐    ┌────▼─────┐    ┌▼────────────┐          │
    │    │ Distribute │    │ Disputed │    │ Return all  │          │
    │    │ rewards to │    │ Everyone │    │ stakes      │          │
    │    │ majority   │    │ gets stake     │             │          │
    │    │ Slash      │    │ back (no │    │             │          │
    │    │ minority   │    │ rewards) │    │             │          │
    │    └─────┬──────┘    └────┬─────┘    └┬────────────┘          │
    │          │                │            │                        │
    │          │                │            │                        │
    │          │ 9. submitFeedback(          │                        │
    │          │      taskId,                │                        │
    │          │      serverAgent,           │                        │
    │          │      rating,                │                        │
    │          │      success                │                        │
    │          │    )                        │                        │
    │          └───────────────────────────────────────────────────►  │
    │                           │                                      │
    │                           │                               Update │
    │                           │                               score  │
    │                           │                                      │
    │ ◄──emit ValidationFinalized(requestId, result)──────────────────┤
    │                           │                                      │
```

---

## Reputation Flow

### Task Authorization with Commit-Reveal

```
┌────────┐                      ┌──────────────────┐
│ Client │                      │  ReputationV2    │
└───┬────┘                      └────────┬─────────┘
    │                                    │
    │ 1. Compute commit hash             │
    │    hash = keccak256(               │
    │      taskId,                       │
    │      serverAgent,                  │
    │      deadline,                     │
    │      salt,                         │
    │      chainId                       │
    │    )                               │
    │                                    │
    │ 2. commitTaskAuthorization(hash)   │
    ├───────────────────────────────────►│
    │                                    │
    │                              Store │
    │                              commit│
    │                                    │
    │ ◄─emit AuthorizationCommitted()────┤
    │                                    │
    │                                    │
    │ 3. Wait 30 seconds                 │
    │    [MIN_DELAY]                      │
    │                                    │
    │                                    │
    │ 4. authorizeTaskWithReveal(        │
    │      taskId,                       │
    │      serverAgent,                  │
    │      deadline,                     │
    │      salt                          │
    │    )                               │
    ├───────────────────────────────────►│
    │                                    │
    │                              Verify│
    │                              hash  │
    │                              timing│
    │                                    │
    │                            Authorize
    │                            task    │
    │                                    │
    │ ◄─emit AuthorizationRevealed()─────┤
    │                                    │
    │ ◄─return success───────────────────┤
    │                                    │
```

---

## TEE Key Governance Flow

### Proposal -> Vote -> Execute

```
┌──────────┐  ┌─────────┐  ┌──────────────┐  ┌──────────┐
│ Proposer │  │ Voters  │  │ TEERegistry  │  │ Anyone   │
└─────┬────┘  └────┬────┘  └──────┬───────┘  └────┬─────┘
      │            │               │               │
      │ 1. Prepare attestation     │               │
      │    documentation           │               │
      │                            │               │
      │ 2. proposeTEEKey(          │               │
      │      keyHash,              │               │
      │      attestationURL,       │               │
      │      teeType               │               │
      │    ) + 1 ETH               │               │
      ├───────────────────────────►│               │
      │                            │               │
      │                      Create│               │
      │                      proposal              │
      │                      7-day voting          │
      │                            │               │
      │ ◄──emit TEEKeyProposed()───┤               │
      │            │               │               │
      │            │               │               │
      │            │ 3. Review     │               │
      │            │    attestation│               │
      │            │               │               │
      │            │ 4. vote(      │               │
      │            │      proposalId,              │
      │            │      support  │               │
      │            │    )          │               │
      │            ├──────────────►│               │
      │            │               │               │
      │            │         Record│               │
      │            │         vote  │               │
      │            │         with  │               │
      │            │         weight│               │
      │            │               │               │
      │            │ ◄──emit VoteCast()────────────┤
      │            │               │               │
      │            │               │               │
      │ 5. Wait 7 days             │               │
      │    [wait]                   │               │
      │                            │               │
      │                            │ 6. executeProposal(
      │                            │      proposalId
      │                            │    )           │
      │                            │◄───────────────┤
      │                            │               │
      │                      Check │               │
      │                      ≥10% participation    │
      │                      ≥66% approval         │
      │                            │               │
      │              ┌─────────────┴──────────┐   │
      │              │ APPROVED?              │   │
      │              └─────────────┬──────────┘   │
      │                            │               │
      │                  YES       │       NO      │
      │                            │               │
      │    ┌───────────────────────┼──────────────────────┐
      │    │                       │                      │
      │    │                       │                      │
      │  ┌─▼──────────────┐  ┌─────▼────────────┐       │
      │  │ Add to trusted │  │ Slash 50% stake  │       │
      │  │ TEE keys       │  │ Return 50%       │       │
      │  │ Return 100%    │  │                  │       │
      │  │ stake          │  │                  │       │
      │  └─┬──────────────┘  └─────┬────────────┘       │
      │    │                       │                      │
      │ ◄──┴───emit ProposalExecuted()──┴───────────────┤
      │                            │                      │
```

---

## Data Flow Diagrams

### End-to-End: Client Request to Reputation Update

```
    Client                Server Agent           Blockchain
      │                        │                      │
      │  1. Request Service    │                      │
      ├───────────────────────►│                      │
      │                        │                      │
      │                        │  2. Execute Task     │
      │                        │                      │
      │  3. Receive Result     │                      │
      │◄───────────────────────┤                      │
      │                        │                      │
      │  4. Request Validation │                      │
      │    (taskId, dataHash)  │                      │
      ├────────────────────────┼─────────────────────►│
      │                        │                   Store
      │                        │                   Request
      │                        │                      │
      │  5. Listen for ValidationRequested event     │
      │◄───────────────────────┼──────────────────────┤
      │                        │                      │


   Validators (×N)                              Blockchain
      │                                             │
      │  6. Detect Validation Request              │
      │◄────────────────────────────────────────────┤
      │                                             │
      │  7. Re-execute Task Independently          │
      │                                             │
      │  8. Submit Validation                       │
      │    (requestId, computedHash) + stake       │
      ├────────────────────────────────────────────►│
      │                                          Store
      │                                          Response
      │                                             │
      │  (Repeat for all N validators)              │
      │                                             │
      │                                        Calculate
      │                                        Consensus
      │                                             │
      │  9. Auto-finalize when                     │
      │     minValidators reached                  │
      │                                        ┌────┤
      │                                        │    │
      │                              IF ≥66% consensus
      │                                        │    │
      │                                Distribute   │
      │                                rewards +    │
      │                                slash       │
      │                                        │    │
      │                                        └────┤
      │                                             │
      │  10. Update Reputation                     │
      │                                             │
      │                                      submitFeedback()
      │                                             │
      │◄──emit ValidationFinalized()────────────────┤
      │                                             │
```

### Gas Flow Diagram

```
┌────────────────────────────────────────────────────────────┐
│  Gas Costs Across Different Operations                    │
│                                                            │
│  Registration Flow:                                        │
│  ┌───────────────────┐                                    │
│  │ commitRegistration│ ~50k gas                           │
│  └─────────┬─────────┘                                    │
│            │                                               │
│            ▼                                               │
│  ┌───────────────────┐                                    │
│  │ registerAgent     │ ~650k gas (multi-key)              │
│  │                   │ ~450k gas (single key)             │
│  └─────────┬─────────┘                                    │
│            │                                               │
│            ▼                                               │
│  ┌───────────────────┐                                    │
│  │ activateAgent     │ ~45k gas                           │
│  └───────────────────┘                                    │
│                                                            │
│  Key Management:                                           │
│  ┌───────────────────┐                                    │
│  │ addKey            │ ~100k gas                          │
│  └─────────┬─────────┘                                    │
│            │                                               │
│            ▼                                               │
│  ┌───────────────────┐                                    │
│  │ revokeKey         │ ~70k gas                           │
│  └───────────────────┘                                    │
│                                                            │
│  Validation Flow:                                         │
│  ┌────────────────────┐                                   │
│  │ requestValidation  │ ~180k gas                         │
│  └─────────┬──────────┘                                   │
│            │                                               │
│            ▼                                               │
│  ┌────────────────────┐                                   │
│  │ submitValidation   │ ~120k gas × N validators          │
│  │ (per validator)    │                                   │
│  └─────────┬──────────┘                                   │
│            │                                               │
│            ▼                                               │
│  ┌────────────────────┐                                   │
│  │ finalizeValidation │ ~250k + (50k × N) gas             │
│  │                    │ Max: 5.2M with 100 validators     │
│  └────────────────────┘                                   │
│                                                            │
│  Governance Flow:                                         │
│  ┌────────────────────┐                                   │
│  │ proposeTEEKey      │ ~180k gas                         │
│  └─────────┬──────────┘                                   │
│            │                                               │
│            ▼                                               │
│  ┌────────────────────┐                                   │
│  │ vote               │ ~85k gas × voters                 │
│  └─────────┬──────────┘                                   │
│            │                                               │
│            ▼                                               │
│  ┌────────────────────┐                                   │
│  │ executeProposal    │ ~150k + (5k × voters)             │
│  └────────────────────┘                                   │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

---

## Security Architecture

### Layered Security Model

```
┌──────────────────────────────────────────────────────────┐
│                 Security Layer Overview                  │
│                                                          │
│  Layer 1: Identity & Access Control                     │
│  ┌────────────────────────────────────────────────────┐ │
│  │ • DID-based authentication (W3C compliant)         │ │
│  │ • Multi-key signature verification               │ │
│  │ • Active agent status checks                      │ │
│  │ • Owner/operator role separation                  │ │
│  │ • KME public key verification (X25519)            │ │
│  └────────────────────────────────────────────────────┘ │
│                           ▼                              │
│  Layer 2: Transaction Security                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │ • Commit-reveal pattern (front-running)           │ │
│  │ • ChainId inclusion (cross-chain replay)          │ │
│  │ • Nonce tracking (replay attacks)                 │ │
│  │ • Timing constraints (MEV protection)             │ │
│  │ • Stake requirement (0.01 ETH)                    │ │
│  └────────────────────────────────────────────────────┘ │
│                           ▼                              │
│  Layer 3: Economic Security                             │
│  ┌────────────────────────────────────────────────────┐ │
│  │ • Registration stake (Sybil resistance)           │ │
│  │ • Time-locked activation (1 hour delay)           │ │
│  │ • Slashing mechanisms (misbehavior penalty)       │ │
│  │ • Reward distribution (honest incentives)         │ │
│  │ • Pull payment pattern (griefing prevention)      │ │
│  └────────────────────────────────────────────────────┘ │
│                           ▼                              │
│  Layer 4: DoS Protection                                │
│  ┌────────────────────────────────────────────────────┐ │
│  │ • Array bounds checking (gas limits)              │ │
│  │ • Maximum validators per request (100)            │ │
│  │ • Rate limiting (24 registrations/day)            │ │
│  │ • Maximum keys per agent (10)                     │ │
│  │ • Deadline enforcement (resource locking)         │ │
│  └────────────────────────────────────────────────────┘ │
│                           ▼                              │
│  Layer 5: Smart Contract Security                       │
│  ┌────────────────────────────────────────────────────┐ │
│  │ • Reentrancy guards (ReentrancyGuard)            │ │
│  │ • Pausable contracts (emergency stops)            │ │
│  │ • Two-step ownership transfer (Ownable2Step)      │ │
│  │ • Custom errors (gas optimization)                │ │
│  │ • Public key reuse prevention                     │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### Attack Surface & Mitigations

```
┌──────────────────────────────────────────────────────────┐
│  Attack Vector                  Mitigation               │
├──────────────────────────────────────────────────────────┤
│  Front-Running                                           │
│  Attacker sees transaction  ->   Commit-reveal pattern    │
│  and submits with higher gas    Hides intent until reveal│
│                                                           │
│  Cross-Chain Replay                                      │
│  Attacker replays signature ->   ChainId in all hashes    │
│  on different network           Network-specific commits │
│                                                           │
│  Sybil Attack                                            │
│  Attacker creates many      ->   Stake requirement        │
│  identities to manipulate       Rate limiting (24/day)   │
│  validation                     Economic cost per agent  │
│                                                           │
│  DoS via Gas                                             │
│  Attacker submits 1000+     ->   Max 10 keys per agent    │
│  keys to exceed gas limit       Max 100 validators       │
│                                 Array bounds checking    │
│                                                           │
│  Reentrancy                                              │
│  Attacker calls back into   ->   ReentrancyGuard on all   │
│  contract during execution      payable functions        │
│                                                           │
│  MEV Exploitation                                        │
│  Sandwich attacks on        ->   Commit-reveal delays     │
│  validator submissions          (1 min - 1 hour window)  │
│                                                           │
│  Key Reuse Attack                                        │
│  Attacker reuses public key ->   Key hash tracking        │
│  across multiple agents         Prevent reuse globally   │
│                                                           │
│  Time-based Attack                                       │
│  Instant malicious agent    ->   1-hour activation delay  │
│  registration & usage           Community review period  │
│                                                           │
└──────────────────────────────────────────────────────────┘
```

---

## Implementation Notes

### Contract Size & Optimization

```
┌─────────────────────────────────────────────────────────┐
│  Contract Sizes (Approximate)                           │
│                                                         │
│  AgentCardRegistry.sol:             ~18 KB              │
│  AgentCardStorage.sol:              ~8 KB               │
│  AgentCardVerifyHook.sol:           ~6 KB               │
│  ERC8004ValidationRegistry.sol:     ~20 KB              │
│  ERC8004ReputationRegistry.sol:     ~12 KB              │
│  TEEKeyRegistry.sol:                ~18 KB              │
│  SimpleMultiSig.sol:                ~8 KB               │
│  TimelockController.sol:            ~12 KB              │
│                                                         │
│  Total System Size:                 ~102 KB             │
│                                                         │
│  Optimization Techniques:                               │
│  • Custom errors instead of strings (-30% gas)          │
│  • Packed structs for storage efficiency               │
│  • View functions for off-chain queries                │
│  • Events for historical data tracking                 │
│  • Pull payment pattern for withdrawals                │
│  • Struct parameters to avoid stack too deep           │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Deployment Sequence

```
1. Deploy AgentCardVerifyHook
   └─ No dependencies

2. Deploy AgentCardRegistry(verifyHookAddress)
   └─ Depends on: AgentCardVerifyHook

3. Deploy ERC8004ReputationRegistry(identityRegistry)
   └─ Depends on: AgentCardRegistry (as identity registry)

4. Deploy ERC8004ValidationRegistry(identityRegistry, reputationRegistry)
   └─ Depends on: AgentCardRegistry, ReputationRegistry

5. Link ReputationRegistry.setValidationRegistry(ValidationRegistry)
   └─ Enable ValidationRegistry to submit feedback

6. Deploy TEEKeyRegistry
   └─ No dependencies (standalone governance)

7. (Optional) Deploy SimpleMultiSig
   └─ Multi-signature wallet for governance

8. (Optional) Deploy TimelockController
   └─ Time-delayed execution for critical operations

9. (Optional) Transfer AgentCardRegistry ownership to governance
   └─ Multi-sig or timelock for decentralized control
```

---

## Summary

This architecture provides:

- **Multi-Key Support**: ECDSA, Ed25519, X25519 in a single agent
- **Native ERC-8004**: Direct implementation, no adapter needed
- **KME Integration**: X25519 public key storage for HPKE
- **Front-Running Protection**: Commit-reveal pattern
- **Sybil Resistance**: Stake + rate limiting + time-lock
- **DoS Resistance**: Array bounds and gas limit controls
- **Community Governance**: Decentralized TEE key approval
- **Modular Design**: Independent contracts with clear interfaces
- **Upgradable**: Separate storage layer enables future upgrades
- **Audit Ready**: Comprehensive documentation and test coverage

**Key Improvements over V2/V3/V4**:
- Native ERC-8004 implementation (no adapter)
- Multi-key support with key lifecycle management
- KME public key storage for HPKE encryption
- Time-locked activation for community review
- Public key reuse prevention
- Enhanced rate limiting and anti-Sybil measures

---

**Document Version**: 2.0
**Last Updated**: 2025-11-01
**Status**: Current - Reflects AgentCard Architecture
