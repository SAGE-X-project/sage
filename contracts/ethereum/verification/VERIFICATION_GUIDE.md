# Klaytnscope Contract Verification Guide

## Contract Addresses (Kairos Testnet)
- SageRegistryV2: 0xF1f53bd1dEc3f91Ffba5E66f4397aF2ec3eAF7fE  
- SageVerificationHook: 0xDA0777245c125a7F8a733945d93bAe100F814093

## ⚠️ CRITICAL: Via IR Compilation Required

The error "Cannot generate bytecode and ABI" occurs because of special compiler settings. Our contracts use **Via IR** compilation mode.

## Method 1: Klaytnscope Web Interface (Corrected Settings)

### Step 1: Navigate to Contract
1. Go to: https://kairos.klaytnscope.com
2. Search: `0xF1f53bd1dEc3f91Ffba5E66f4397aF2ec3eAF7fE`
3. Click "Contract" tab → "Verify and Publish"

### Step 2: EXACT Verification Settings

**Basic Settings:**
- Contract Name: `SageRegistryV2`
- Compiler Type: `Solidity (Single file)`
- Compiler Version: `v0.8.19+commit.7dd6d404`
- Open Source License: `MIT`

**Optimization Settings:**
- Optimization: `Yes`
- Optimization Runs: `200`

**🔴 CRITICAL - Advanced Compiler Settings:**
- Click "Show Advanced Options" (if available)
- **Via IR**: `YES` ✅ (MUST BE ENABLED!)
- EVM Version: `paris` or `default`
- Enable Details: `Yes`

### Step 3: Source Code
Copy the ENTIRE content from:
```
sage/contracts/ethereum/flattened/SageRegistryV2_flat.sol
```

### Step 4: Constructor Arguments
Leave empty (no constructor parameters)

## Method 2: Using Remix IDE (Alternative)

Since Klaytnscope might not support Via IR properly, use Remix:

### Step 1: Setup Remix
1. Open https://remix.ethereum.org
2. Create new file: `SageRegistryV2.sol`
3. Paste content from `flattened/SageRegistryV2_flat.sol`

### Step 2: Compiler Settings in Remix
1. Go to "Solidity Compiler" tab
2. Compiler: `0.8.19+commit.7dd6d404`
3. Click "Advanced Configurations"
4. **Enable optimization**: `200` runs
5. **Via IR**: ✅ Check this box!
6. EVM Version: `paris`

### Step 3: Compile
1. Click "Compile SageRegistryV2.sol"
2. Ensure no errors

### Step 4: Get Verification Data
1. After compilation, go to "Solidity Compiler" tab
2. Click on contract name
3. Copy the "Bytecode" (not deployedBytecode)
4. Copy the "ABI"

### Step 5: Manual Verification on Klaytnscope
1. Go back to Klaytnscope
2. Use the bytecode and ABI from Remix
3. Or try "Import from Remix" option if available

## Method 3: Standard JSON Input

Create a file with these EXACT settings:

```json
{
  "language": "Solidity",
  "sources": {
    "SageRegistryV2.sol": {
      "content": "// Paste flattened source here"
    }
  },
  "settings": {
    "optimizer": {
      "enabled": true,
      "runs": 200,
      "details": {
        "yul": true,
        "yulDetails": {
          "stackAllocation": true,
          "optimizerSteps": "dhfoDgvulfnTUtnIf"
        }
      }
    },
    "viaIR": true,
    "outputSelection": {
      "*": {
        "*": ["abi", "evm.bytecode", "evm.deployedBytecode"]
      }
    },
    "evmVersion": "paris"
  }
}
```

## Why This Error Happens

The contract was compiled with:
1. **Via IR** (Intermediate Representation) - A new compilation pipeline
2. **Yul Optimizer** - Advanced optimization
3. **Custom Optimizer Steps** - Specific optimization sequence

Standard verification often doesn't support these advanced features.

## Quick Checklist

- [ ] Compiler: v0.8.19+commit.7dd6d404
- [ ] Optimization: Enabled (200 runs)
- [ ] **Via IR: ENABLED** ← Most important!
- [ ] Source: Use flattened file
- [ ] Constructor Args: None

## If All Else Fails

1. **Contact Klaytnscope Support**: Mention Via IR compilation
2. **Use Sourcify**: Alternative verification service
3. **Leave Unverified**: Contract still works, just not verified

## Verification Status Check

Visit:
- https://kairos.klaytnscope.com/account/0xF1f53bd1dEc3f91Ffba5E66f4397aF2ec3eAF7fE?tabId=contractCode
- https://kairos.klaytnscope.com/account/0xDA0777245c125a7F8a733945d93bAe100F814093?tabId=contractCode