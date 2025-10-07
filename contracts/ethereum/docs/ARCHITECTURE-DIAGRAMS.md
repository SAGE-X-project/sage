# SAGE Smart Contract Architecture Diagrams

**Version**: 1.0
**Date**: 2025-10-07
**Status**: Complete

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
│                        SAGE Ecosystem                           │
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
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │  │
│  │  │   Identity   │  │  Validation  │  │  Reputation  │ │  │
│  │  │   Registry   │  │   Registry   │  │   Registry   │ │  │
│  │  │ (SageV3/ERC) │  │   (ERC8004)  │  │   (ERC8004)  │ │  │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘ │  │
│  │         │                  │                  │         │  │
│  │         └──────────────────┼──────────────────┘         │  │
│  │                            │                            │  │
│  │                   ┌────────▼────────┐                  │  │
│  │                   │  TEE Key        │                  │  │
│  │                   │  Governance     │                  │  │
│  │                   └─────────────────┘                  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Purpose | Dependencies |
|-----------|---------|--------------|
| **SageRegistryV3** | Agent identity & DID management | None |
| **ERC8004ValidationRegistry** | Task validation coordination | Identity, Reputation |
| **ERC8004ReputationRegistry** | Agent reputation tracking | Identity, Validation |
| **TEEKeyRegistry** | Decentralized TEE key approval | None |

---

## Component Architecture

### Detailed Component Interaction

```
┌─────────────────────────────────────────────────────────────────┐
│                    Contract Dependencies                        │
│                                                                 │
│                   ┌─────────────────────┐                      │
│                   │  SageRegistryV3     │                      │
│                   │  (Identity Core)    │                      │
│                   └──────────┬──────────┘                      │
│                              │                                  │
│                              │ resolveAgent()                   │
│                              │ isAgentActive()                  │
│                              │                                  │
│         ┌────────────────────┼────────────────────┐           │
│         │                    │                    │           │
│         │                    │                    │           │
│  ┌──────▼──────┐      ┌──────▼──────┐     ┌──────▼──────┐   │
│  │  ERC8004    │      │  ERC8004    │     │  ERC8004    │   │
│  │  Identity   │◄─────┤  Validation │────►│  Reputation │   │
│  │  Registry   │      │  Registry   │     │  Registry   │   │
│  │  (Adapter)  │      │             │     │             │   │
│  └─────────────┘      └──────┬──────┘     └──────┬──────┘   │
│                              │                    │           │
│                              │                    │           │
│                              │ isTrustedKey()     │           │
│                              │                    │           │
│                       ┌──────▼────────────────────▼──┐       │
│                       │   TEEKeyRegistry            │       │
│                       │   (Governance)              │       │
│                       └─────────────────────────────┘       │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

### Contract Interface Relationships

```
                  ┌──────────────────────────┐
                  │  ISageRegistry           │
                  │  (Interface)             │
                  └─────────────┬────────────┘
                                │
                                │ implements
                                │
                  ┌─────────────▼────────────┐
                  │  SageRegistryV3          │
                  │  (Implementation)        │
                  └──────────────────────────┘
                                │
                                │ wraps
                                │
                  ┌─────────────▼────────────┐
                  │  ERC8004IdentityRegistry │
                  │  (ERC-8004 Adapter)      │
                  └──────────────────────────┘
                                │
                                │ used by
                                │
         ┌──────────────────────┴──────────────────────┐
         │                                              │
┌────────▼──────────┐                      ┌───────────▼─────────┐
│ ERC8004Validation │                      │ ERC8004Reputation   │
│ Registry          │                      │ Registry            │
└───────────────────┘                      └─────────────────────┘
```

---

## Agent Registration Flow

### Commit-Reveal Registration Process

```
┌─────────┐                                    ┌──────────────┐
│  Client │                                    │ SageRegistryV3│
└────┬────┘                                    └──────┬───────┘
     │                                                │
     │ 1. Generate salt & compute hash                │
     │    hash = keccak256(did, pubKey,              │
     │           sender, salt, chainId)               │
     │                                                │
     │ 2. commitRegistration(hash)                    │
     ├───────────────────────────────────────────────►│
     │                                                │
     │                                          Store │
     │                                          ┌─────┤
     │                                          │     │
     │                                          │  { │
     │                                          │   commitHash,│
     │                                          │   timestamp, │
     │                                          │   revealed   │
     │                                          │  }           │
     │                                          └─────┤
     │                                                │
     │ ◄──emit RegistrationCommitted()───────────────┤
     │                                                │
     │                                                │
     │ 3. Wait 1 minute (MIN_DELAY)                   │
     │    ⏱️  [60 seconds]                            │
     │                                                │
     │                                                │
     │ 4. registerAgentWithReveal(                    │
     │      did, name, desc, endpoint,                │
     │      publicKey, capabilities,                  │
     │      signature, salt                           │
     │    )                                           │
     ├───────────────────────────────────────────────►│
     │                                                │
     │                                          Verify│
     │                                          ┌─────┤
     │                                          │     │
     │                                  ✓ Timing OK   │
     │                                  ✓ Hash match  │
     │                                  ✓ Signature   │
     │                                  ✓ DID unique  │
     │                                          │     │
     │                                          └─────┤
     │                                                │
     │                                       Register │
     │                                       agentId  │
     │                                                │
     │ ◄──emit AgentRegistered(agentId)──────────────┤
     │                                                │
     │ ◄──return agentId──────────────────────────────┤
     │                                                │
```

### Attack Prevention

```
WITHOUT COMMIT-REVEAL:
┌─────────┐                ┌──────────────┐
│  Alice  │                │   Attacker   │
└────┬────┘                └──────┬───────┘
     │                             │
     │ registerAgent("alice")      │
     ├─────────────►               │
     │              │               │
     │              │ Sees in       │
     │              │ mempool       │
     │              └──────────────►│
     │                             │
     │              ◄──────────────┤
     │              Front-runs with │
     │              higher gas!     │
     │                             │
     │              registerAgent("alice")
     │              (Attacker wins) ❌
     │

WITH COMMIT-REVEAL:
┌─────────┐                ┌──────────────┐
│  Alice  │                │   Attacker   │
└────┬────┘                └──────┬───────┘
     │                             │
     │ commit(hash)                │
     ├─────────────►               │
     │              │               │
     │              │ Sees hash     │
     │              │ (can't decode)│
     │              └──────────────►│
     │              ◄──────────────┤
     │              Can't predict   │
     │              DID without salt│
     │                             │
     │ Wait 1 min                   │
     │                             │
     │ reveal("alice", salt)        │
     ├─────────────►               │
     │ (Alice wins) ✅              │
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
    │          │ 9. updateReputation(        │                        │
    │          │      serverAgent,           │                        │
    │          │      result                 │                        │
    │          │    )                        │                        │
    │          └───────────────────────────────────────────────────►  │
    │                           │                                      │
    │                           │                               Update │
    │                           │                               score  │
    │                           │                                      │
    │ ◄──emit ValidationFinalized(requestId, result)──────────────────┤
    │                           │                                      │
```

### Consensus Calculation

```
┌──────────────────────────────────────────────────────────┐
│  Consensus Algorithm (Byzantine Fault Tolerant)          │
│                                                          │
│  Input: N validator responses                           │
│         - Each response has: success (bool) + stake     │
│                                                          │
│  Step 1: Count votes                                    │
│    successVotes = Σ(stake where success = true)        │
│    failVotes = Σ(stake where success = false)          │
│    totalVotes = successVotes + failVotes                │
│                                                          │
│  Step 2: Calculate rate                                 │
│    successRate = (successVotes / totalVotes) × 100      │
│                                                          │
│  Step 3: Determine outcome                              │
│    if successRate ≥ 66%:                                │
│      → VALIDATED (task correct)                         │
│      → Reward SUCCESS voters                            │
│      → Slash FAIL voters                                │
│    else if successRate ≤ 33%:                           │
│      → FAILED (task incorrect)                          │
│      → Reward FAIL voters                               │
│      → Slash SUCCESS voters                             │
│    else:                                                │
│      → DISPUTED (no consensus)                          │
│      → Return all stakes                                │
│      → No rewards, no slashing                          │
│                                                          │
│  Example:                                               │
│    10 validators, 0.1 ETH each                          │
│    7 vote SUCCESS, 3 vote FAIL                          │
│    successRate = 70% ≥ 66% → VALIDATED ✓                │
│    7 validators: 0.1 + (1.0 × 0.1 / 7) ≈ 0.114 ETH     │
│    3 validators: 0 ETH (slashed)                        │
│                                                          │
└──────────────────────────────────────────────────────────┘
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
    │    ⏱️  [MIN_DELAY]                 │
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

### Reputation Update Flow

```
┌─────────────┐          ┌────────────┐          ┌─────────────┐
│ Validation  │          │ Reputation │          │  Agent      │
│ Registry    │          │ Registry   │          │  (on-chain) │
└──────┬──────┘          └─────┬──────┘          └──────┬──────┘
       │                       │                        │
       │ 1. Validation complete│                        │
       │    (consensus reached)│                        │
       │                       │                        │
       │ 2. submitFeedback(    │                        │
       │      taskId,          │                        │
       │      serverAgent,     │                        │
       │      rating,          │                        │
       │      success          │                        │
       │    )                  │                        │
       ├──────────────────────►│                        │
       │                       │                        │
       │                 Store │                        │
       │                 feedback                       │
       │                       │                        │
       │                       │ 3. Query reputation    │
       │                       │◄───────────────────────┤
       │                       │                        │
       │                       │ 4. getAgentReputation( │
       │                       │      agent             │
       │                       │    )                   │
       │                       │                        │
       │                Calculate                       │
       │                average                         │
       │                rating                          │
       │                       │                        │
       │                       │ return reputation      │
       │                       ├───────────────────────►│
       │                       │                        │
```

---

## TEE Key Governance Flow

### Proposal → Vote → Execute

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
      │            │  (Repeat for multiple voters) │
      │            │               │               │
      │            │               │               │
      │ 5. Wait 7 days             │               │
      │    ⏱️                       │               │
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

### Voting Weight Distribution

```
┌─────────────────────────────────────────────────────────┐
│  Voter Registration & Weight Assignment                 │
│                                                         │
│  Owner (initially) can register voters:                │
│                                                         │
│  ┌──────────┐                                          │
│  │  Owner   │                                          │
│  └────┬─────┘                                          │
│       │                                                │
│       │ registerVoter(address, weight)                │
│       │                                                │
│       ├──────────────┐                                 │
│       │              │                                 │
│       ▼              ▼                                 │
│  ┌────────┐    ┌────────┐                             │
│  │Voter 1 │    │Voter 2 │    ...                      │
│  │Weight: │    │Weight: │                             │
│  │  100   │    │  50    │                             │
│  └────────┘    └────────┘                             │
│                                                        │
│  Total Weight = Σ(all voter weights)                  │
│                                                        │
│  Participation = (Σ votes cast / Total Weight) × 100  │
│  Approval = (Σ FOR votes / Σ all votes) × 100         │
│                                                        │
│  Example:                                             │
│    Total Weight: 300 (3 voters: 100, 100, 100)       │
│    Votes Cast: 200 (2 voters voted)                  │
│    Participation: 66.7% ≥ 10% ✓                       │
│    FOR votes: 150                                     │
│    Approval: 75% ≥ 66% ✓ → APPROVED                   │
│                                                        │
└─────────────────────────────────────────────────────────┘
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
│  │ registerWithReveal│ ~250k gas (first)                 │
│  │                   │ ~150k gas (subsequent)             │
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
│  │ • DID-based authentication                         │ │
│  │ • Public key signature verification               │ │
│  │ • Active agent status checks                      │ │
│  │ • Owner/governance role separation                │ │
│  └────────────────────────────────────────────────────┘ │
│                           ▼                              │
│  Layer 2: Transaction Security                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │ • Commit-reveal pattern (front-running)           │ │
│  │ • ChainId inclusion (cross-chain replay)          │ │
│  │ • Nonce tracking (replay attacks)                 │ │
│  │ • Timing constraints (MEV protection)             │ │
│  └────────────────────────────────────────────────────┘ │
│                           ▼                              │
│  Layer 3: Economic Security                             │
│  ┌────────────────────────────────────────────────────┐ │
│  │ • Stake requirements (Sybil resistance)           │ │
│  │ • Slashing mechanisms (misbehavior penalty)       │ │
│  │ • Reward distribution (honest incentives)         │ │
│  │ • Pull payment pattern (griefing prevention)      │ │
│  └────────────────────────────────────────────────────┘ │
│                           ▼                              │
│  Layer 4: DoS Protection                                │
│  ┌────────────────────────────────────────────────────┐ │
│  │ • Array bounds checking (gas limits)              │ │
│  │ • Maximum validators per request (100)            │ │
│  │ • Deadline enforcement (resource locking)         │ │
│  │ • Gas limit controls on hooks                     │ │
│  └────────────────────────────────────────────────────┘ │
│                           ▼                              │
│  Layer 5: Smart Contract Security                       │
│  ┌────────────────────────────────────────────────────┐ │
│  │ • Reentrancy guards (ReentrancyGuard)            │ │
│  │ • Pausable contracts (emergency stops)            │ │
│  │ • Two-step ownership transfer (Ownable2Step)      │ │
│  │ • Custom errors (gas optimization)                │ │
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
│  Attacker sees transaction  →   Commit-reveal pattern    │
│  and submits with higher gas    Hides intent until reveal│
│                                                           │
│  Cross-Chain Replay                                      │
│  Attacker replays signature →   ChainId in all hashes    │
│  on different network           Network-specific commits │
│                                                           │
│  Sybil Attack                                            │
│  Attacker creates many      →   Stake requirements       │
│  identities to manipulate       Economic cost per vote   │
│  validation                                              │
│                                                           │
│  DoS via Gas                                             │
│  Attacker submits 1000+     →   Max 100 validators       │
│  validators to exceed gas       Array bounds checking    │
│  limit                          ~5.2M gas max            │
│                                                           │
│  Reentrancy                                              │
│  Attacker calls back into   →   ReentrancyGuard on all   │
│  contract during execution      payable functions        │
│                                                           │
│  MEV Exploitation                                        │
│  Sandwich attacks on        →   Timing constraints       │
│  validator submissions          Commit delays (30s-1h)   │
│                                                           │
│  Centralization Risk                                     │
│  Owner controls trusted     →   Community governance     │
│  TEE keys                       Proposal→Vote→Execute    │
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
│  SageRegistryV3.sol:             ~15 KB                 │
│  ERC8004ValidationRegistry.sol:  ~20 KB                 │
│  ERC8004ReputationRegistryV2.sol: ~12 KB                │
│  TEEKeyRegistry.sol:             ~18 KB                 │
│                                                         │
│  Total System Size:              ~65 KB                 │
│                                                         │
│  Optimization Techniques:                               │
│  • Custom errors instead of strings (-30% gas)          │
│  • Packed structs for storage efficiency               │
│  • View functions for off-chain queries                │
│  • Events for historical data tracking                 │
│  • Pull payment pattern for withdrawals                │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Deployment Sequence

```
1. Deploy SageRegistryV3
   └─ No dependencies

2. Deploy ERC8004IdentityRegistry(SageRegistryV3)
   └─ Depends on: SageRegistryV3

3. Deploy ERC8004ReputationRegistryV2(IdentityRegistry)
   └─ Depends on: IdentityRegistry

4. Deploy ERC8004ValidationRegistry(IdentityRegistry, ReputationRegistry)
   └─ Depends on: IdentityRegistry, ReputationRegistry

5. Link ReputationRegistry.setValidationRegistry(ValidationRegistry)
   └─ Enable ValidationRegistry to submit feedback

6. Deploy TEEKeyRegistry
   └─ No dependencies (standalone governance)

7. (Optional) Register initial TEE keys via governance
```

---

## Summary

This architecture provides:

✅ **Decentralized Identity**: DID-based agent registration with ownership proofs
✅ **Trustless Validation**: Crypto-economic incentives for honest verification
✅ **Byzantine Fault Tolerance**: 66% consensus threshold
✅ **Front-Running Protection**: Commit-reveal pattern on sensitive operations
✅ **DoS Resistance**: Array bounds and gas limit controls
✅ **Community Governance**: Decentralized TEE key approval
✅ **Modular Design**: Independent contracts with clear interfaces
✅ **Audit Ready**: Comprehensive documentation and test coverage

**Next Steps**: Integration guide and deployment procedures

---

**Document Version**: 1.0
**Last Updated**: 2025-10-07
**Status**: ✅ Complete
