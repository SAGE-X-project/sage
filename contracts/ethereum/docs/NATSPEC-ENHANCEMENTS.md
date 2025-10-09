# NatSpec Documentation Enhancements

**Version:** 1.0
**Date:** 2025-10-07
**Status:** Implementation Guide

---

## Overview

This document provides specific NatSpec enhancements for SAGE smart contracts. All contracts currently have basic documentation, but need comprehensive enhancement following the [NATSPEC-GUIDE.md](./NATSPEC-GUIDE.md) standards.

---

## Current Documentation Status

### ✅ Contracts with Basic Documentation

1. **SageRegistryV3.sol**
   - Has contract-level NatSpec
   - Basic function documentation
   - **Needs:** Enhanced parameter docs, error docs, examples

2. **ERC8004ValidationRegistry.sol**
   - Has contract-level NatSpec
   - Basic function documentation
   - **Needs:** Economic model docs, consensus docs, gas estimates

3. **ERC8004ReputationRegistryV2.sol**
   - Has contract-level NatSpec
   - Commit-reveal docs present
   - **Needs:** Task authorization flow docs, examples

4. **TEEKeyRegistry.sol**
   - Has contract-level NatSpec
   - Basic struct/enum docs
   - **Needs:** Governance process docs, voting mechanism docs

5. **SimpleMultiSig.sol**
   - Has contract-level NatSpec
   - Function documentation
   - **Needs:** Security warnings, production deployment notes

---

## Priority Enhancements

### P0: Critical Security Documentation

#### 1. ERC8004ValidationRegistry - Economic Model

**Add to contract-level documentation:**

```solidity
/**
 * Economic Model:
 * ═══════════════
 *
 * Validator Stakes:
 * - Minimum stake: 0.1 ETH (minValidatorStake)
 * - Stake locked until finalization
 * - Refunded if validator is correct
 * - Slashed if validator is incorrect
 *
 * Rewards Distribution:
 * - Correct validators share validatorRewardPercentage (10%) of requester stake
 * - Example: Requester stakes 1 ETH, 3 correct validators → each gets 0.033 ETH
 *
 * Slashing:
 * - Incorrect validators lose slashingPercentage (100%) of stake
 * - Slashed funds go to contract treasury
 * - Treasury managed by owner (should be governance contract)
 *
 * Consensus Algorithm:
 * - Minimum validators required: minValidatorsRequired (default: 1)
 * - Consensus threshold: consensusThreshold (default: 66%)
 * - Example: 5 validators, 4 agree (80%) → consensus reached
 *
 * Gas Costs (estimated):
 * - Request validation: ~150,000 gas
 * - Submit stake validation: ~200,000 gas
 * - Submit TEE attestation: ~180,000 gas
 * - Finalization (3 validators): ~300,000 gas
 * - Finalization (10 validators): ~800,000 gas
 * - Finalization (100 validators): ~5,000,000 gas (max)
 *
 * DoS Protection:
 * - Maximum validators per request: 100 (prevents unbounded loops)
 * - See ARRAY-BOUNDS-CHECKING.md for details
 */
```

#### 2. SageRegistryV3 - Front-Running Protection

**Add to commitRegistration() function:**

```solidity
/**
 * Timing Constraints:
 * ══════════════════
 *
 * MIN_COMMIT_REVEAL_DELAY = 1 minute
 * - Must wait at least 1 minute after commit
 * - Ensures commitment is mined before reveal
 * - Prevents instant reveal attacks
 *
 * MAX_COMMIT_REVEAL_DELAY = 1 hour
 * - Must reveal within 1 hour of commit
 * - Prevents indefinite commitment locking
 * - Old commitments automatically expire
 *
 * Security Model:
 * ═══════════════
 *
 * Attack Prevention:
 * 1. Front-running: Attacker cannot see DID until reveal
 * 2. Replay attacks: chainId included in commitment
 * 3. Salt stealing: Salt never appears on-chain until reveal
 * 4. Timing attacks: Minimum delay prevents instant reveal
 *
 * Example Attack Scenario (Prevented):
 * 1. Alice wants "did:sage:alice"
 * 2. Alice commits hash (attacker sees hash but not DID)
 * 3. Attacker cannot extract DID from hash (cryptographically secure)
 * 4. Alice waits 1 minute, then reveals
 * 5. Alice's transaction gets priority (already committed)
 * 6. Attacker's front-run attempt fails
 *
 * Gas Cost Analysis:
 * ═════════════════
 *
 * Commit transaction: ~45,000 gas (~$0.10 at 50 gwei)
 * Reveal transaction: ~30,000 gas extra (~$0.07 at 50 gwei)
 * Total overhead: ~$0.17 per registration
 *
 * Worth it? YES
 * - Prevents DID theft
 * - Valuable DIDs (e.g., "did:sage:oracle") worth protecting
 * - One-time cost for permanent DID
 */
```

#### 3. TEEKeyRegistry - Governance Process

**Add comprehensive governance documentation:**

```solidity
/**
 * Governance Process:
 * ══════════════════
 *
 * Step 1: Proposal
 * ────────────────
 * - Anyone can propose TEE key
 * - Must stake proposalStake (1 ETH)
 * - Provide attestation report URL
 * - Specify TEE type (SGX/SEV/TrustZone/Nitro)
 *
 * Step 2: Voting
 * ──────────────
 * - Voting period: votingPeriod (7 days)
 * - Registered voters can vote (weighted)
 * - One vote per address per proposal
 * - Votes are final (cannot be changed)
 *
 * Step 3: Execution
 * ─────────────────
 * - Anyone can call executeProposal() after voting period
 * - Approval threshold: approvalThreshold (66%)
 * - Minimum participation: minVoterParticipation (10%)
 *
 * Step 4: Outcome
 * ───────────────
 * If APPROVED:
 * - Key added to approvedTEEKeys mapping
 * - Proposer receives stake back
 * - Key can be used in ERC8004ValidationRegistry
 *
 * If REJECTED:
 * - Key not added
 * - Proposer loses slashingPercentage (50%) of stake
 * - Slashed funds go to treasury
 * - Remaining stake returned to proposer
 *
 * Economic Incentives:
 * ══════════════════
 *
 * For Proposers:
 * - Legitimate TEE keys: Stake returned, reputation gain
 * - Malicious keys: Lose 50% of stake (0.5 ETH)
 * - Prevents spam: 1 ETH stake requirement
 *
 * For Voters:
 * - Participate in governance
 * - Protect ecosystem from malicious TEE keys
 * - Future: Consider voter rewards
 *
 * Example Scenarios:
 * ════════════════
 *
 * Scenario 1: Legitimate Intel SGX Key
 * - Proposer stakes 1 ETH, provides SGX attestation
 * - Community reviews attestation (7 days)
 * - 75% vote in favor (exceeds 66% threshold)
 * - Key approved, proposer gets 1 ETH back
 * - Ecosystem can now use this SGX instance
 *
 * Scenario 2: Suspicious Key
 * - Proposer stakes 1 ETH, attestation looks fake
 * - Community investigates
 * - 80% vote against
 * - Key rejected, proposer loses 0.5 ETH
 * - 0.5 ETH returned, 0.5 ETH to treasury
 *
 * Scenario 3: Insufficient Participation
 * - Proposer stakes 1 ETH
 * - Only 5% of voters participate (below 10% minimum)
 * - Proposal fails automatically
 * - Proposer gets full stake back
 * - Can re-propose after community awareness increases
 *
 * Security Considerations:
 * ══════════════════════
 *
 * Voter Registration:
 * - Owner registers initial voters (bootstrap)
 * - Can add/remove voters (should be timelocked)
 * - Future: Token-weighted voting
 *
 * Emergency Controls:
 * - Owner can revoke malicious keys immediately
 * - Owner can pause proposals during attack
 * - Owner should be TimelockController
 *
 * Attack Vectors:
 * - Sybil attack: Prevented by voter registration
 * - Governance attack: Prevented by timelock
 * - Flash loan attack: N/A (voting requires registration)
 */
```

---

## Function-Level Documentation Templates

### Template: Payable Function

```solidity
/**
 * @notice [User-facing description]
 * @dev [Technical details]
 *
 * Process:
 * 1. [Step 1]
 * 2. [Step 2]
 * ...
 *
 * @param [paramName] [Description]
 * @return [returnName] [Description]
 *
 * @custom:throws [ErrorName] [When thrown]
 * @custom:security-warning This function handles funds. Protected by [Modifier].
 * @custom:gas-cost Approximately [X] gas
 */
```

### Template: View Function

```solidity
/**
 * @notice [User-facing description]
 * @dev [Technical details]
 *
 * @param [paramName] [Description]
 * @return [returnName] [Description]
 *
 * @custom:gas-optimization [If applicable]
 */
```

### Template: Admin Function

```solidity
/**
 * @notice [User-facing description]
 * @dev [Technical details]
 *
 * @param [paramName] [Description]
 *
 * @custom:throws [ErrorName] [When thrown]
 * @custom:governance This should be controlled by [TimelockController/MultiSig]
 * @custom:security-note [Why this parameter is important]
 */
```

---

## Error Documentation Templates

```solidity
/**
 * @notice [Short explanation of error condition]
 * @dev [Detailed explanation + how to avoid]
 * @param [paramName] [Description of error parameter]
 */
error MyError(uint256 paramName);
```

---

## Examples by Contract

### ERC8004ValidationRegistry Enhancements

#### submitStakeValidation()

```solidity
/**
 * @notice Submit stake-based validation for a request
 * @dev Validator re-executes task and stakes ETH on result
 *
 * Validation Process:
 * 1. Validator re-runs computation off-chain
 * 2. Computes hash of result
 * 3. Submits hash with stake (min 0.1 ETH)
 * 4. Contract compares with expected hash
 * 5. Response recorded for consensus
 * 6. Auto-finalizes if enough validators
 *
 * Economic Incentives:
 * - Correct validators earn 10% of requester stake (split)
 * - Incorrect validators lose 100% of their stake
 * - Encourages honest validation
 *
 * @param requestId Unique identifier for validation request
 * @param computedHash Validator's result hash (should match request.dataHash)
 *
 * @return success True if validation accepted
 *
 * @custom:throws RequestNotFound if requestId invalid
 * @custom:throws RequestNotPending if already finalized
 * @custom:throws "Request expired" if past deadline
 * @custom:throws ValidatorAlreadyResponded if already submitted
 * @custom:throws InsufficientValidatorStake if msg.value < minValidatorStake
 * @custom:throws ValidationTypeNotSupported if request doesn't allow STAKE
 * @custom:throws RequesterNotActive if validator's agent not active
 *
 * @custom:security-warning Handles funds. Protected by ReentrancyGuard + whenNotPaused.
 * @custom:gas-cost ~200,000 gas + finalization if triggered (~300k-5M depending on validators)
 *
 * Example:
 * ```solidity
 * bytes32 requestId = 0x123...;
 * bytes32 myHash = keccak256(abi.encodePacked(myResult));
 * validationRegistry.submitStakeValidation{value: 0.1 ether}(requestId, myHash);
 * ```
 */
```

#### _checkAndFinalizeValidation()

```solidity
/**
 * @notice Internal function to check if consensus reached and finalize
 * @dev Called after each validator response to check finalization criteria
 *
 * Consensus Algorithm:
 * 1. Count successful vs failed responses
 * 2. Calculate success percentage
 * 3. If >= consensusThreshold (66%), consensus reached
 * 4. Finalize and distribute rewards/slashing
 *
 * DoS Protection:
 * - Only processes first 100 responses (MAX_VALIDATORS_PER_REQUEST)
 * - Prevents gas limit DoS attack
 * - See ARRAY-BOUNDS-CHECKING.md for analysis
 *
 * Edge Cases:
 * - If exactly 50-50 split, follows majority rule
 * - If <minValidatorsRequired, waits for more
 * - If request expires before consensus, anyone can trigger finalization
 *
 * @param requestId The validation request identifier
 *
 * @custom:gas-cost Varies by validator count:
 *   - 1 validator: ~100k gas
 *   - 10 validators: ~800k gas
 *   - 100 validators: ~5M gas (max)
 */
```

---

## Struct Documentation

### AgentMetadata

```solidity
/**
 * @notice Complete agent information
 * @dev Core data structure for registered agents
 *
 * Field Descriptions:
 * ══════════════════
 *
 * agentId:
 * - Unique identifier (keccak256 of DID)
 * - Immutable after registration
 * - Used as primary key in storage
 *
 * did:
 * - Decentralized Identifier (e.g., "did:sage:alice")
 * - Must be unique across registry
 * - Used for human-readable agent discovery
 *
 * owner:
 * - Address that controls this agent
 * - Can update agent metadata
 * - Can revoke agent keys
 * - Cannot be zero address
 *
 * publicKey:
 * - Cryptographic public key for agent
 * - Length: MIN_PUBLIC_KEY_LENGTH to MAX_PUBLIC_KEY_LENGTH (32-65 bytes)
 * - Used for authentication and signature verification
 * - Can be revoked but not changed
 *
 * active:
 * - Whether agent is currently active
 * - Set to false when key is revoked
 * - Inactive agents cannot perform validation
 *
 * Invariants:
 * ══════════
 * 1. agentId == keccak256(bytes(did))
 * 2. owner != address(0)
 * 3. MIN_PUBLIC_KEY_LENGTH <= publicKey.length <= MAX_PUBLIC_KEY_LENGTH
 * 4. registeredAt <= lastUpdated
 * 5. If !active, agent is revoked (permanent)
 */
```

---

## Constant Documentation

### Economic Parameters

```solidity
/**
 * @notice Minimum stake required from validator
 * @dev Economic parameter - changes affect security model
 *
 * Rationale:
 * - 0.1 ETH provides meaningful economic stake
 * - High enough to prevent spam
 * - Low enough to allow small validators
 * - Can be updated by owner (should be governance)
 *
 * Security Impact:
 * - Higher stake: More security, fewer validators
 * - Lower stake: More validators, less security per validator
 *
 * @custom:governance This parameter should be controlled by TimelockController
 */
uint256 public minValidatorStake = 0.1 ether;

/**
 * @notice Percentage of requester stake given as reward
 * @dev Shared among all correct validators
 *
 * Example:
 * - Requester stakes 1 ETH
 * - 10% reward percentage
 * - 5 correct validators
 * - Each validator gets: 1 ETH * 10% / 5 = 0.02 ETH
 *
 * Rationale:
 * - 10% provides meaningful incentive
 * - Not too high (requester doesn't overpay)
 * - Not too low (validators have incentive)
 *
 * @custom:range 0-100 (percentage)
 * @custom:governance Should be adjusted based on market conditions
 */
uint256 public validatorRewardPercentage = 10;
```

---

## Implementation Checklist

### Per Contract

- [ ] Enhanced contract-level NatSpec
- [ ] Economic model documented (if applicable)
- [ ] Security features listed
- [ ] Invariants documented
- [ ] Gas cost estimates for key functions
- [ ] All public/external functions enhanced
- [ ] All custom errors documented
- [ ] All events documented with indexed params explanation
- [ ] All structs with field-by-field docs
- [ ] All constants with rationale

### Before Audit

- [ ] All P0 contracts documented
- [ ] Security warnings on critical functions
- [ ] Governance controls documented
- [ ] Economic parameters explained
- [ ] DoS prevention measures documented
- [ ] Gas cost analysis complete
- [ ] Example usage provided for complex functions
- [ ] Cross-references to other docs (FRONT-RUNNING-PROTECTION.md, etc.)

---

## Tools for Validation

### Generate Documentation Site

```bash
# Using Hardhat
npx hardhat docgen

# Using Foundry
forge doc --out docs/generated
```

### Check Completeness

```bash
# Count functions without NatSpec
grep -r "function " contracts/ | grep -v "/**" | wc -l

# List functions missing @param
grep -A 5 "function " contracts/ | grep -v "@param" | grep "function"
```

---

## Next Steps

1. ✅ Created NATSPEC-GUIDE.md with standards
2. ✅ Created NATSPEC-ENHANCEMENTS.md with specific improvements
3. ⏳ **Next:** Apply enhancements to P0 contracts:
   - ERC8004ValidationRegistry.sol
   - SageRegistryV3.sol
   - ERC8004ReputationRegistryV2.sol
   - TEEKeyRegistry.sol
4. ⏳ Generate documentation site
5. ⏳ Review before audit submission

---

## Summary

**Current Status:** All contracts have basic NatSpec
**Target Status:** Audit-ready comprehensive documentation
**Estimated Effort:** 4-6 hours for P0 contracts
**Priority:** HIGH (required for audit)

**Key Improvements:**
- Economic model documentation
- Gas cost estimates
- Security warnings
- Governance considerations
- Example usage
- Error condition documentation
- Cross-references to guides

---

**Document Version:** 1.0
**Last Updated:** 2025-10-07
**Status:** Implementation Guide Ready

