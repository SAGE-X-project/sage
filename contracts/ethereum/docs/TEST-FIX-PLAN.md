# Test Fix Implementation Plan

**Date:** 2025-10-07
**Status:** In Progress
**Branch:** security/phase1-critical-fixes

---

## Overview

After implementing Phase 1-4 security fixes, 14 tests are failing due to:
1. Migration to custom errors (Phase 4)
2. Deadline validation changes (Phase 3)
3. New test setup issues

This document tracks the fix progress.

---

## Progress Summary

**Status:** 1/14 tests fixed (7%)
- ✅ **Fixed:** 1 test
- ⏳ **In Progress:** 0 tests
- ❌ **Pending:** 13 tests

**Estimated Total Time:** 2-3 hours
**Time Spent:** 15 minutes
**Time Remaining:** ~2.5 hours

---

## Fix Categories

### Category 1: Custom Error Migration ✅ 1/1 Complete

**Status:** COMPLETE
**Time:** 15 minutes

| Test | File | Line | Status | Notes |
|------|------|------|--------|-------|
| Should reject validation request with insufficient stake | erc-8004.test.js | 430 | ✅ Fixed | Changed to revertedWithCustomError |

**Changes Made:**
```javascript
// Before:
).to.be.revertedWith("Insufficient stake");

// After:
).to.be.revertedWithCustomError(validationRegistry, "InsufficientStake");

// Also fixed deadline:
const deadline = currentBlock.timestamp + 3600 + 60; // 1 hour + buffer
```

---

### Category 2: Pull Payment Tests ⏳ 0/5 Complete

**Status:** PENDING
**Estimated Time:** 30 minutes

| Test | File | Line | Status | Fix Required |
|------|------|------|--------|--------------|
| Should allow users to withdraw | security-pull-payment.test.js | 154 | ❌ Pending | Update deadline |
| Should revert with zero balance | security-pull-payment.test.js | ~210 | ❌ Pending | Update deadline |
| Should handle multiple validators | security-pull-payment.test.js | ~260 | ❌ Pending | Update deadline |
| Should emit WithdrawalProcessed | security-pull-payment.test.js | ~290 | ❌ Pending | Update deadline |
| Should not send funds directly | security-pull-payment.test.js | ~340 | ❌ Pending | Update deadline |

**Fix Pattern:**
```javascript
// Find all instances of:
const deadline = currentBlock.timestamp + 7200;

// Replace with:
const deadline = currentBlock.timestamp + 3600 + 60; // MIN_DEADLINE_DURATION + buffer
```

**Command to Fix:**
```bash
# Edit file: test/security-pull-payment.test.js
# Search for: currentBlock.timestamp + 7200
# Replace with: currentBlock.timestamp + 3600 + 60
# Count: ~5 occurrences
```

---

### Category 3: Reentrancy Tests ⏳ 0/6 Complete

**Status:** PENDING
**Estimated Time:** 30 minutes

| Test | File | Line | Status | Fix Required |
|------|------|------|--------|--------------|
| Should prevent reentrancy during request | security-reentrancy.test.js | ~180 | ❌ Pending | Update deadline |
| Should prevent reentrancy during submission | security-reentrancy.test.js | ~230 | ❌ Pending | Update deadline |
| Should allow normal stake validation | security-reentrancy.test.js | ~250 | ❌ Pending | Update deadline |
| Should prevent multiple submissions | security-reentrancy.test.js | ~265 | ❌ Pending | Update deadline |
| Should handle complete validation flow | security-reentrancy.test.js | ~300 | ❌ Pending | Update deadline |
| Should measure gas cost increase | security-reentrancy.test.js | ~350 | ❌ Pending | Update deadline |

**Fix Pattern:** Same as Category 2

---

### Category 4: New Integration Tests ⏳ 0/2 Complete

**Status:** PENDING
**Estimated Time:** 45 minutes each = 1.5 hours

#### Test 4.1: Multi-Sig Governance Test

**File:** `test/multisig-governance.test.js`
**Status:** ❌ Fails in `beforeEach` hook
**Issue:** Contract deployment or interface mismatch

**Investigation Needed:**
1. Check which contract fails to deploy
2. Verify TimelockController is available
3. Verify SimpleMultiSig compiles
4. Update interface calls if changed

**Estimated Time:** 45 minutes

#### Test 4.2: Security Features Test

**File:** `test/security-features.test.js`
**Status:** ❌ Fails in `beforeEach` hook
**Issue:** Similar to 4.1

**Investigation Needed:**
1. Check SageRegistryV3 deployment
2. Check ERC8004ReputationRegistryV2 deployment
3. Check TEEKeyRegistry deployment
4. Verify all interfaces match

**Estimated Time:** 45 minutes

---

## Implementation Steps

### Step 1: Fix Pull Payment Tests (30 min) ⏳

```bash
# 1. Open file
vim test/security-pull-payment.test.js

# 2. Find and replace deadlines
:%s/currentBlock.timestamp + 7200/currentBlock.timestamp + 3600 + 60/g

# 3. Save and test
:wq
npm test test/security-pull-payment.test.js
```

**Expected Result:** 5 tests pass

---

### Step 2: Fix Reentrancy Tests (30 min) ⏳

```bash
# 1. Open file
vim test/security-reentrancy.test.js

# 2. Find and replace deadlines
:%s/currentBlock.timestamp + 7200/currentBlock.timestamp + 3600 + 60/g

# 3. Also check for other deadline patterns
# Search for: + 3600  (might need to change to + 3660)
# Search for: + 1800  (might need to change to + 3660)

# 4. Save and test
:wq
npm test test/security-reentrancy.test.js
```

**Expected Result:** 6 tests pass

---

### Step 3: Fix Multi-Sig Test (45 min) ⏳

```bash
# 1. Run test to see exact error
npm test test/multisig-governance.test.js

# 2. Identify which contract fails
# Look for deployment errors in output

# 3. Possible issues:
#    - TimelockController not in artifacts
#    - SimpleMultiSig not compiled
#    - Interface mismatch

# 4. Fix based on error
#    - Check hardhat.config.js for OpenZeppelin contracts
#    - Verify contract imports
#    - Update test if interface changed

# 5. Test again
npm test test/multisig-governance.test.js
```

**Expected Result:** 10+ tests pass

---

### Step 4: Fix Security Features Test (45 min) ⏳

```bash
# 1. Run test to see exact error
npm test test/security-features.test.js

# 2. Check which contracts fail to deploy:
#    - SageRegistryV3
#    - ERC8004ReputationRegistryV2
#    - TEEKeyRegistry

# 3. Verify contracts are compiled
npx hardhat compile

# 4. Fix deployment issues
#    - Check constructor parameters
#    - Verify contract exists in artifacts
#    - Update test setup

# 5. Test again
npm test test/security-features.test.js
```

**Expected Result:** 15+ tests pass

---

## Quick Fix Script

Create a script to automate deadline fixes:

```javascript
// scripts/fix-test-deadlines.js
const fs = require('fs');

const files = [
    'test/security-pull-payment.test.js',
    'test/security-reentrancy.test.js'
];

files.forEach(file => {
    let content = fs.readFileSync(file, 'utf8');

    // Replace short deadlines
    content = content.replace(
        /currentBlock\.timestamp \+ 7200/g,
        'currentBlock.timestamp + 3600 + 60'
    );

    content = content.replace(
        /currentBlock\.timestamp \+ 3600([^\+])/g,
        'currentBlock.timestamp + 3600 + 60$1'
    );

    fs.writeFileSync(file, content, 'utf8');
    console.log(`✅ Fixed: ${file}`);
});
```

**Run:**
```bash
node scripts/fix-test-deadlines.js
npm test
```

---

## Verification Checklist

After all fixes:

- [ ] All 108 tests pass
- [ ] No warnings in test output
- [ ] Test coverage >= 86%
- [ ] All security features verified:
  - [ ] ReentrancyGuard working
  - [ ] Pull payment working
  - [ ] Hook gas limits working
  - [ ] Ownable2Step working
  - [ ] Custom errors working
  - [ ] Deadline validation working
  - [ ] Pause mechanism working

---

## Final Test Run

```bash
# Clean and recompile
npm run clean
npm run compile

# Run all tests
npm test

# Expected result:
#   108 passing
#   0 failing
#   0 pending
```

---

## Next Phase

After all tests pass:

1. ✅ Generate test coverage report
2. ✅ Verify all security fixes tested
3. ✅ Test emergency pause procedures
4. ✅ Proceed to Sepolia deployment

---

**Document Version:** 1.0
**Last Updated:** 2025-10-07
**Status:** 1/14 tests fixed, 13 remaining

