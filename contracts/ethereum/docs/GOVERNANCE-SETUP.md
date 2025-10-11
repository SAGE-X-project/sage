# SAGE Governance Setup Guide

**Version:** 1.0
**Date:** 2025-10-07
**Status:** Implementation Ready

---

## Overview

This guide explains how to set up decentralized governance for SAGE smart contracts using a multi-signature wallet and timelock controller. This is a **CRITICAL** step before mainnet deployment to eliminate centralization risks.

### What We're Building

```
┌─────────────────┐
│  5 Signers      │
│  (Hardware      │
│   Wallets)      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     Require 3/5      ┌──────────────────┐
│   Multi-sig     │ ─────signatures────▶ │  Propose Action  │
│   Wallet        │                       │  to Timelock     │
└─────────────────┘                       └────────┬─────────┘
                                                   │
                                                   ▼
                                          ┌──────────────────┐
                                          │   48 Hour Wait   │
                                          │   (Community     │
                                          │    Review)       │
                                          └────────┬─────────┘
                                                   │
                                                   ▼
                                          ┌──────────────────┐
                                          │   Execute if     │
                                          │   No Issues      │
                                          └────────┬─────────┘
                                                   │
                                                   ▼
                                          ┌──────────────────┐
                                          │  SAGE Contracts  │
                                          │  - SageRegistry  │
                                          │  - Validation    │
                                          │  - Reputation    │
                                          └──────────────────┘
```

---

## Prerequisites

### 1. Hardware Wallets (REQUIRED for Mainnet)
- 5 hardware wallets (Ledger or Trezor recommended)
- Each signer must have their own device
- Test signatures on testnet first

### 2. Network Setup
- Sepolia testnet access (for testing)
- Mainnet RPC endpoint (for production)
- Sufficient ETH for gas costs

### 3. Software Requirements
```bash
npm install @openzeppelin/contracts
npm install @nomicfoundation/hardhat-toolbox
```

---

## Step-by-Step Deployment

### Step 1: Identify Signers

Select 5 trusted individuals with:
- Technical competence
- Geographic distribution
- Different organizations
- 24/7 availability
- Hardware wallet access

**Example Signer Roles:**
1. Project Lead
2. Technical Lead
3. Security Lead
4. Community Representative
5. Legal/Compliance Officer

### Step 2: Configure Environment

```bash
# Copy example environment file
cp .env.governance.example .env

# Edit .env and fill in:
nano .env
```

Required variables:
```bash
# Signer addresses (from hardware wallets)
SIGNER1_ADDRESS=0x...
SIGNER2_ADDRESS=0x...
SIGNER3_ADDRESS=0x...
SIGNER4_ADDRESS=0x...
SIGNER5_ADDRESS=0x...

# Network config
SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
SEPOLIA_PRIVATE_KEY=deployer_private_key
ETHERSCAN_API_KEY=your_api_key
```

### Step 3: Deploy Multi-sig and Timelock

```bash
# Deploy to Sepolia testnet
npx hardhat run scripts/deploy-multisig-governance.js --network sepolia

# Verify on Etherscan
npx hardhat verify --network sepolia MULTISIG_ADDRESS SIGNER1 SIGNER2 SIGNER3 SIGNER4 SIGNER5 3
npx hardhat verify --network sepolia TIMELOCK_ADDRESS
```

**Expected Output:**
```
✅ Multi-sig deployed at: 0x...
✅ Timelock deployed at: 0x...
   Threshold: 3 of 5
   Min Delay: 48 hours
```

**Save these addresses!**

### Step 4: Test Multi-sig Flow

Before transferring ownership, test the multi-sig:

```bash
# Run test script
npx hardhat run scripts/test-multisig-flow.js --network sepolia
```

Test checklist:
- [ ] Can propose transaction with 1 signature
- [ ] Cannot execute with < 3 signatures
- [ ] Can execute with >= 3 signatures
- [ ] All signers can access wallet
- [ ] Hardware wallets work correctly

### Step 5: Transfer SAGE Contract Ownership

⚠️ **CRITICAL: Test on testnet first!**

```bash
# Transfer ownership to Timelock
npx hardhat run scripts/transfer-ownership-to-timelock.js --network sepolia
```

This will:
1. Call `transferOwnership(timelock)` on each contract
2. Set timelock as pending owner
3. Generate acceptance transactions

### Step 6: Accept Ownership via Timelock

Multi-sig must propose and execute `acceptOwnership()` through Timelock:

```bash
# Helper script to create proposal
npx hardhat run scripts/accept-ownership.js --network sepolia
```

**Process:**
1. **Day 0:** Multi-sig proposes acceptOwnership() to Timelock
2. **Day 0:** Signers 1-3 approve the proposal (need 3/5)
3. **Day 2:** After 48-hour delay, execute transaction
4. **Day 2:** Ownership transfer complete

### Step 7: Verify Governance Setup

```bash
# Check ownership
npx hardhat run scripts/verify-governance.js --network sepolia
```

Expected results:
- [ ] SageRegistryV2 owned by Timelock
- [ ] ValidationRegistry owned by Timelock
- [ ] ReputationRegistry owned by Timelock
- [ ] IdentityRegistry owned by Timelock
- [ ] No EOA has admin access
- [ ] Multi-sig controls Timelock

---

## Using the Governance System

### Making Parameter Changes

Example: Change minimum stake in ValidationRegistry

#### 1. Create Proposal

```javascript
// Encode function call
const validationRegistry = await ethers.getContractAt(
    "ERC8004ValidationRegistry",
    VALIDATION_REGISTRY_ADDRESS
);

const newMinStake = ethers.parseEther("0.02");
const calldata = validationRegistry.interface.encodeFunctionData(
    "setMinStake",
    [newMinStake]
);
```

#### 2. Submit to Timelock via Multi-sig

```javascript
// Multi-sig proposes to Timelock
const timelock = await ethers.getContractAt("TimelockController", TIMELOCK_ADDRESS);

const tx = await timelock.schedule(
    VALIDATION_REGISTRY_ADDRESS,  // target
    0,                             // value
    calldata,                      // data
    ethers.ZeroHash,               // predecessor
    ethers.ZeroHash,               // salt
    48 * 60 * 60                   // 48 hour delay
);
```

#### 3. Get 3/5 Signatures

- Signer 1 approves
- Signer 2 approves
- Signer 3 approves
- ✅ Threshold reached

#### 4. Wait 48 Hours

Community can:
- Review the proposal
- Discuss on governance forum
- Raise concerns if malicious
- Trigger emergency pause if needed

#### 5. Execute After Delay

```javascript
await timelock.execute(
    VALIDATION_REGISTRY_ADDRESS,
    0,
    calldata,
    ethers.ZeroHash,
    ethers.ZeroHash
);
```

✅ Parameter changed!

---

## Emergency Procedures

### Emergency Pause (24-hour delay)

If critical bug discovered:

```javascript
// 1. Multi-sig proposes pause
const calldata = validationRegistry.interface.encodeFunctionData("pause", []);

await timelock.schedule(
    VALIDATION_REGISTRY_ADDRESS,
    0,
    calldata,
    ethers.ZeroHash,
    ethers.ZeroHash,
    24 * 60 * 60  // Emergency: 24 hours instead of 48
);

// 2. Get 3/5 signatures ASAP

// 3. Wait 24 hours

// 4. Execute pause
await timelock.execute(...);
```

### Contract Upgrade

For major upgrades:

1. Deploy new contract version
2. Test thoroughly on testnet
3. External audit of changes
4. Multi-sig proposes migration
5. 48-hour community review
6. Execute migration
7. Verify new contract
8. Update documentation

---

## Security Best Practices

### Hardware Wallet Management

✅ **DO:**
- Use hardware wallets for all signers
- Store seed phrases in multiple secure locations
- Test signatures on testnet first
- Keep firmware updated
- Use strong PINs
- Geographic distribution of signers

❌ **DON'T:**
- Store seed phrases digitally
- Use same device for multiple signers
- Share seed phrases
- Use hot wallets for signers
- Skip testing

### Operational Security

✅ **DO:**
- Verify all addresses before signing
- Double-check transaction data
- Review proposals on Etherscan
- Use secure communication channels
- Keep contact info updated
- Have backup signers identified

❌ **DON'T:**
- Sign without verifying
- Use public WiFi for signing
- Share private keys
- Rush critical transactions
- Skip the delay period

### Communication Protocol

**For Normal Operations:**
- Proposals discussed in governance forum
- 48-hour minimum discussion period
- Public voting/temperature checks
- Formal proposal with rationale

**For Emergencies:**
- Immediate notification via Telegram
- Emergency meeting within 1 hour
- Document incident
- Public disclosure after resolution

---

## Monitoring and Alerts

### Set Up Alerts

```javascript
// Use Tenderly/Defender for real-time alerts

// Alert on:
- Any ownership transfer attempt
- Timelock proposal submissions
- Timelock executions
- Pause/unpause events
- Large fund movements
- Multi-sig transactions
```

### Regular Health Checks

Weekly:
- [ ] Verify all signers have access
- [ ] Check pending proposals
- [ ] Review recent transactions
- [ ] Test emergency contact methods

Monthly:
- [ ] Full security audit
- [ ] Review access logs
- [ ] Update documentation
- [ ] Practice emergency procedures

---

## Troubleshooting

### Issue: Signer Lost Hardware Wallet

**Solution:**
1. Don't panic - need 3/5 signatures
2. Signer recovers with seed phrase
3. Update multi-sig if needed
4. Document incident

### Issue: Transaction Stuck in Timelock

**Solution:**
1. Check if delay period has passed
2. Verify transaction data
3. Check gas prices
4. Re-execute if needed

### Issue: Accidental Proposal

**Solution:**
1. Do NOT sign the proposal
2. Let it expire (after 7 days inactive)
3. Document what happened
4. Improve proposal process

---

## Cost Estimates

### Deployment Costs (Sepolia)

- Multi-sig deployment: ~0.01 ETH
- Timelock deployment: ~0.02 ETH
- Ownership transfers (4 contracts): ~0.005 ETH each
- **Total:** ~0.05 ETH ($150 at $3000/ETH)

### Ongoing Costs

- Parameter change: ~0.002 ETH per transaction
- Emergency pause: ~0.001 ETH
- Monthly operations: ~0.01 ETH

---

## Mainnet Checklist

Before deploying to mainnet:

### Pre-Deployment
- [ ] All 5 signers identified and trained
- [ ] Hardware wallets purchased and set up
- [ ] All signers tested on Sepolia
- [ ] Emergency procedures documented
- [ ] Monitoring system set up
- [ ] Communication channels established

### Deployment
- [ ] Multi-sig deployed to mainnet
- [ ] Timelock deployed to mainnet
- [ ] Contracts verified on Etherscan
- [ ] Test transaction executed successfully
- [ ] Ownership transfer complete
- [ ] All signers confirmed access

### Post-Deployment
- [ ] Announcement to community
- [ ] Documentation published
- [ ] Monitoring alerts active
- [ ] First governance proposal tested
- [ ] Emergency procedures tested

---

## Resources

### Documentation
- OpenZeppelin Timelock: https://docs.openzeppelin.com/contracts/api/governance#TimelockController
- Gnosis Safe: https://docs.safe.global/
- SAGE Governance: /docs/GOVERNANCE.md

### Tools
- Gnosis Safe UI: https://app.safe.global/
- Tenderly: https://tenderly.co/
- OpenZeppelin Defender: https://defender.openzeppelin.com/

### Support
- GitHub Issues: https://github.com/SAGE-X-project/sage/issues
- Discord: #governance channel
- Email: governance@sage-project.io

---

## Conclusion

Decentralized governance is **CRITICAL** for mainnet security. The multi-sig + timelock setup:

✅ Eliminates single point of failure
✅ Requires 3/5 agreement for changes
✅ Gives community 48-hour review period
✅ Prevents rushed malicious changes
✅ Allows emergency response (24h pause)

**Timeline:**
- Day 1: Deploy multi-sig and timelock
- Day 2: Test multi-sig flow
- Day 3: Transfer ownership
- Day 5: Accept ownership (after 48h delay)
- Day 6: Verify setup complete

**Total time: ~1 week for testnet, 2 weeks for mainnet**

---

**Document Version:** 1.0
**Last Updated:** 2025-10-07
**Next Review:** After testnet deployment
