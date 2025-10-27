# AgentCardRegistry Migration Guide

This guide helps you migrate from the legacy single-phase registration to the new three-phase AgentCardRegistry system.

## Table of Contents

- [Overview](#overview)
- [Breaking Changes](#breaking-changes)
- [Migration Steps](#migration-steps)
- [Code Examples](#code-examples)
- [CLI Commands](#cli-commands)
- [FAQ](#faq)

## Overview

### What Changed?

**Legacy System (SageRegistryV4):**
- Single-phase registration
- Immediate activation
- Vulnerable to front-running attacks
- Simple but less secure

**New System (AgentCardRegistry):**
- Three-phase registration (commit → register → activate)
- Anti-front-running protection via commit-reveal
- Time-locked activation (1+ hour delay)
- Economic security via stake (0.01 ETH)
- Enhanced operator management

### Why Migrate?

1. **Security**: Commit-reveal prevents front-running attacks
2. **Sybil Resistance**: Activation delay and stake requirement
3. **Better Multi-Key Support**: Unified handling of ECDSA, Ed25519, X25519
4. **Future-Proof**: Foundation for cross-chain expansion

## Breaking Changes

### 1. Registration Flow

**Before (Single-Phase):**
```go
// One-step registration
client := ethereum.NewEthereumClientV4(config)
result, err := client.RegisterAgent(ctx, params)
// Agent immediately active
```

**After (Three-Phase):**
```go
// Phase 1: Commit
client := ethereum.NewAgentCardClient(config)
status, err := client.CommitRegistration(ctx, params)

// Wait 1-60 minutes

// Phase 2: Register
status, err = client.RegisterAgent(ctx, status)

// Wait 1+ hour

// Phase 3: Activate
err = client.ActivateAgent(ctx, status)
```

### 2. CLI Commands

**Before:**
```bash
sage-did register \
  --chain ethereum \
  --name "My Agent" \
  --endpoint https://agent.example.com \
  --key agent.pem
```

**After:**
```bash
# Phase 1: Commit
sage-did commit \
  --name "My Agent" \
  --endpoint https://agent.example.com \
  --key agent.pem

# Wait 1-60 minutes

# Phase 2: Register
sage-did register <commit-hash>

# Wait 1+ hour

# Phase 3: Activate
sage-did activate <commit-hash>
```

### 3. Contract Interface

**Removed:**
- `ISageRegistryV4`
- `pkg/blockchain/ethereum/contracts/registryv4/`
- `pkg/agent/did/ethereum/clientv4.go`

**Added:**
- `AgentCardRegistry.sol`
- `pkg/blockchain/ethereum/contracts/agentcardregistry/`
- `pkg/agent/did/ethereum/agentcard_client.go`

### 4. KeyType Enum Values

**CRITICAL CHANGE** - Enum values were reordered to match Solidity:

```go
// BEFORE (WRONG):
const (
    KeyTypeEd25519 KeyType = 0
    KeyTypeECDSA   KeyType = 1
    KeyTypeX25519  KeyType = 2
)

// AFTER (CORRECT):
const (
    KeyTypeECDSA   KeyType = 0  // MUST be 0
    KeyTypeEd25519 KeyType = 1  // MUST be 1
    KeyTypeX25519  KeyType = 2  // MUST be 2
)
```

**Action Required:** If you stored KeyType values, you MUST update them!

## Migration Steps

### Step 1: Update Dependencies

```bash
# Pull latest code
git pull origin main

# Update contract bindings
cd contracts/ethereum
npm install
npm run compile

# Rebuild Go bindings
cd ../..
make generate-bindings
```

### Step 2: Update Import Paths

```go
// Remove old imports
// import "github.com/sage-x-project/sage/pkg/agent/did/ethereum"

// Add new imports
import (
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)
```

### Step 3: Replace Client Creation

**Before:**
```go
config := &did.RegistryConfig{
    RPCEndpoint:     "http://localhost:8545",
    ContractAddress: "0x...",
    PrivateKey:      "0x...",
}
client, err := ethereum.NewEthereumClientV4(config)
```

**After:**
```go
config := &did.RegistryConfig{
    RPCEndpoint:     "http://localhost:8545",
    ContractAddress: "0x...",  // New AgentCardRegistry address
    PrivateKey:      "0x...",
}
client, err := ethereum.NewAgentCardClient(config)
```

### Step 4: Implement Three-Phase Flow

See [Code Examples](#code-examples) below.

### Step 5: Test Migration

```bash
# Run tests
go test ./pkg/agent/did/ethereum/...

# Test with local blockchain
./scripts/test-local-registration.sh
```

## Code Examples

### Complete Three-Phase Registration

```go
package main

import (
    "context"
    "crypto/rand"
    "fmt"
    "time"

    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

func registerAgent() error {
    ctx := context.Background()

    // Create client
    config := &did.RegistryConfig{
        RPCEndpoint:     "http://localhost:8545",
        ContractAddress: "0x...",
        PrivateKey:      "0x...",
    }
    client, err := ethereum.NewAgentCardClient(config)
    if err != nil {
        return err
    }

    // Prepare registration parameters
    params := &did.RegistrationParams{
        DID:          "did:sage:ethereum:my-agent",
        Name:         "My Agent",
        Description:  "Production agent",
        Endpoint:     "https://my-agent.example.com",
        Capabilities: `{"features":["chat","vision"]}`,
        Keys:         [][]byte{publicKey1, publicKey2},
        KeyTypes:     []did.KeyType{did.KeyTypeECDSA, did.KeyTypeEd25519},
        Signatures:   [][]byte{signature1, signature2},
    }

    // Generate random salt
    if _, err := rand.Read(params.Salt[:]); err != nil {
        return err
    }

    // Phase 1: Commit
    fmt.Println("Phase 1: Committing registration...")
    status, err := client.CommitRegistration(ctx, params)
    if err != nil {
        return fmt.Errorf("commit failed: %w", err)
    }
    fmt.Printf("✓ Committed. Hash: %x\n", status.CommitHash)

    // Save commitment status
    if err := saveCommitmentStatus(status); err != nil {
        return err
    }

    // Wait for reveal window (1-60 minutes)
    fmt.Println("Waiting 2 minutes for reveal window...")
    time.Sleep(2 * time.Minute)

    // Phase 2: Register (reveal)
    fmt.Println("Phase 2: Revealing commitment...")
    status, err = client.RegisterAgent(ctx, status)
    if err != nil {
        return fmt.Errorf("registration failed: %w", err)
    }
    fmt.Printf("✓ Registered. Agent ID: %x\n", status.AgentID)

    // Update commitment status
    if err := saveCommitmentStatus(status); err != nil {
        return err
    }

    // Wait for activation delay (1+ hour)
    fmt.Println("Waiting 1 hour for activation delay...")
    time.Sleep(1 * time.Hour)

    // Phase 3: Activate
    fmt.Println("Phase 3: Activating agent...")
    if err := client.ActivateAgent(ctx, status); err != nil {
        return fmt.Errorf("activation failed: %w", err)
    }
    fmt.Println("✓ Agent activated successfully!")

    return nil
}

func saveCommitmentStatus(status *did.CommitmentStatus) error {
    // Implement your own persistence logic
    // See cmd/sage-did/commit.go for example
    return nil
}
```

### Agent Management Operations

```go
// Get agent metadata
agentID := [32]byte{...}
agent, err := client.GetAgent(ctx, agentID)
if err != nil {
    return err
}

fmt.Printf("Agent: %s\n", agent.Name)
fmt.Printf("Active: %v\n", agent.IsActive)
fmt.Printf("Keys: %d\n", len(agent.Keys))

// Update agent endpoint
err = client.UpdateAgent(ctx, agentID, "https://new-endpoint.com", "")
if err != nil {
    return err
}

// Set operator approval
operatorAddress := common.HexToAddress("0x...")
err = client.SetApprovalForAgent(ctx, agentID, operatorAddress, true)
if err != nil {
    return err
}

// Deactivate agent
err = client.DeactivateAgent(ctx, agentID)
if err != nil {
    return err
}
```

## CLI Commands

### Registration Workflow

```bash
# 1. Commit (Phase 1)
sage-did commit \
  --chain ethereum \
  --name "Production Agent" \
  --description "Main production agent" \
  --endpoint https://agent.example.com \
  --capabilities '{"features":["chat","vision"]}' \
  --key keys/agent.pem \
  --rpc http://localhost:8545 \
  --contract 0x... \
  --private-key 0x...

# Output:
# ✓ Commitment successful!
#   Commit Hash: abc123def456...
#   Timestamp: 2025-01-15T10:00:00Z
#
# NEXT STEPS:
#   1. Wait 1-60 minutes
#   2. Run: sage-did register --commit-hash abc123def456...

# 2. Wait 1-60 minutes (anti-front-running window)
sleep 120  # 2 minutes for testing

# 3. Register (Phase 2)
sage-did register abc123def456... \
  --rpc http://localhost:8545 \
  --contract 0x... \
  --private-key 0x...

# Output:
# ✓ Registration successful!
#   Agent ID: def456abc789...
#   Can activate at: 2025-01-15T11:05:00Z
#
# NEXT STEPS:
#   1. Wait at least 1 hour
#   2. Run: sage-did activate abc123def456...

# 4. Wait 1+ hour (Sybil resistance delay)
sleep 3600  # 1 hour

# 5. Activate (Phase 3)
sage-did activate abc123def456... \
  --rpc http://localhost:8545 \
  --contract 0x... \
  --private-key 0x...

# Output:
# ✓ Activation successful!
#   Agent ID: def456abc789...
#   Your 0.01 ETH stake will be refunded
#
# Your agent is now fully active and can be queried:
#   sage-did resolve did:sage:ethereum:my-agent
```

### Agent Management

```bash
# Resolve agent DID
sage-did resolve did:sage:ethereum:my-agent

# List all registered agents
sage-did list --chain ethereum

# Update agent endpoint
sage-did update did:sage:ethereum:my-agent \
  --endpoint https://new-endpoint.com

# Deactivate agent
sage-did deactivate did:sage:ethereum:my-agent --yes
```

## FAQ

### Q: Can I skip the waiting periods during testing?

**A:** For local testing, you can modify the contract's timing parameters, but on mainnet/testnet these are enforced:
- Reveal window: 1-60 minutes
- Activation delay: 1+ hour

### Q: What happens to my stake?

**A:** Your 0.01 ETH stake is:
1. Locked during commit phase
2. Held during registration phase
3. **Refunded** upon successful activation

### Q: Can I cancel a commitment?

**A:** Yes, but only after the 60-minute reveal window expires. The stake is forfeited if you don't complete registration within the window.

### Q: Do I need to migrate existing agents?

**A:** Existing agents registered via SageRegistryV4 will continue to work, but:
- New features (operators, enhanced security) require AgentCardRegistry
- We recommend migrating for better security
- Migration tool: `sage-did migrate <old-did>`

### Q: What if registration fails in Phase 2 or 3?

**A:**
- **Phase 2 failure**: Commitment remains valid, you can retry
- **Phase 3 failure**: Agent is registered but inactive, you can retry activation
- Always save the commitment status file for recovery

### Q: How do I handle the commitment status file?

**A:** The CLI saves to `~/.sage/commitments/<commit-hash>.json`. For production:
```bash
# Backup commitment
cp ~/.sage/commitments/*.json /secure/backup/

# Restore if needed
cp /secure/backup/*.json ~/.sage/commitments/
```

### Q: Can I automate the three-phase flow?

**A:** Yes, but **do not skip timing delays** on mainnet. Example automation:

```bash
#!/bin/bash
set -e

# Phase 1
HASH=$(sage-did commit --name "Auto Agent" --endpoint https://api.com --key agent.pem | grep "Commit Hash:" | awk '{print $3}')

# Wait 2 minutes
sleep 120

# Phase 2
sage-did register $HASH

# Wait 1 hour
sleep 3600

# Phase 3
sage-did activate $HASH
```

### Q: What about multi-key registration?

**A:** Fully supported! Provide multiple keys in params:

```go
params := &did.RegistrationParams{
    Keys: [][]byte{
        ecdsaPublicKey,    // For Ethereum
        ed25519PublicKey,  // For Solana
        x25519PublicKey,   // For HPKE
    },
    KeyTypes: []did.KeyType{
        did.KeyTypeECDSA,
        did.KeyTypeEd25519,
        did.KeyTypeX25519,
    },
    Signatures: [][]byte{
        ecdsaSignature,
        ed25519Signature,
        x25519Signature,
    },
}
```

### Q: Where can I get help?

- **GitHub Issues**: https://github.com/sage-x-project/sage/issues
- **Discussions**: https://github.com/sage-x-project/sage/discussions
- **Documentation**: https://github.com/sage-x-project/sage/tree/main/docs

## Troubleshooting

### Error: "commitment hash mismatch"

**Cause:** Go and Solidity hash calculation don't match.

**Solution:** Ensure you're using the latest version with fixed `computeCommitmentHash()`.

### Error: "must wait X minutes before registration"

**Cause:** Trying to register too soon after commit.

**Solution:** Wait for the 1-60 minute window. Check `status.CommitTimestamp`.

### Error: "commitment expired"

**Cause:** Waited more than 60 minutes to register.

**Solution:** Start over with a new commitment. The old stake is forfeited.

### Error: "agent not found"

**Cause:** Wrong contract address or agent not yet registered.

**Solution:** Verify contract address and complete Phase 2 first.

## Contract Addresses

### Mainnet
```
AgentCardRegistry: TBD (not yet deployed)
```

### Sepolia Testnet
```
AgentCardRegistry: TBD (not yet deployed)
```

### Local Development
```
Deploy your own:
cd contracts/ethereum
npx hardhat run scripts/deploy-agentcard.ts --network localhost
```

## Version Compatibility

| SAGE Version | Contract | Status |
|--------------|----------|--------|
| v1.0.0 - v1.4.0 | SageRegistryV4 | Legacy (deprecated) |
| v1.5.0+ | AgentCardRegistry | Current (recommended) |

## Support

If you encounter issues during migration:

1. Check this guide for common solutions
2. Review the [REFACTORING_PLAN_V4.md](./REFACTORING_PLAN_V4.md) for technical details
3. Open an issue with:
   - SAGE version
   - Error message
   - Steps to reproduce
   - Contract address used

---

**Last Updated:** 2025-01-15
**SAGE Version:** v1.5.0-alpha
