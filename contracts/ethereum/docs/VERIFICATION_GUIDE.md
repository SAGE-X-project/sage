# Contract Verification Guide for Kaia Network

## Overview
This guide explains how to verify smart contracts on Kaia (mainnet) and Kairos (testnet) networks using Klaytnscope explorer.

## Important Notes
- **Kaia networks (Kairos/Kaia) do NOT require API keys** for contract verification
- Klaytnscope is the official explorer for both Kaia mainnet and Kairos testnet
- Verification can be done automatically via Hardhat or manually through the web interface

## Automatic Verification

### 1. Deploy Your Contracts First
```bash
# For Kairos testnet
npm run deploy:kairos

# For Kaia mainnet
npm run deploy:kaia
```

### 2. Run Verification Script
```bash
# Verify on Kairos testnet
npm run verify:kairos

# Verify on Kaia mainnet
npm run verify:kaia
```

The verification script will:
- Load deployment information
- Attempt automatic verification via hardhat-verify
- Provide manual verification instructions if automatic verification fails

## Manual Verification (Recommended)

### Method 1: Via Klaytnscope Web Interface

#### For Kairos Testnet:
1. Navigate to your contract on Kairos explorer:
   ```
   https://kairos.klaytnscope.com/account/[CONTRACT_ADDRESS]
   ```

2. Click on the **"Contract"** tab

3. Click **"Verify and Publish"** button

4. Fill in the verification form:
   - **Contract Address**: Auto-filled
   - **Contract Name**: `SageRegistry` or `SageVerificationHook`
   - **Compiler Version**: `v0.8.19+commit.7dd6d404`
   - **Optimization**: `Yes`
   - **Optimization Runs**: `200`
   - **EVM Version**: `london` (or latest)

5. Paste your contract source code:
   - Copy the entire content from `contracts/SageRegistry.sol` or `contracts/SageVerificationHook.sol`
   - Include all imports and interfaces

6. If your contract has constructor arguments:
   - Encode them using the ABI encoder
   - Paste the encoded arguments

7. Complete the CAPTCHA and submit

#### For Kaia Mainnet:
Follow the same steps but use:
```
https://klaytnscope.com/account/[CONTRACT_ADDRESS]
```

### Method 2: Using Hardhat Verify Plugin

The hardhat-verify plugin is configured to work with Klaytnscope:

```bash
# Single contract verification
npx hardhat verify --network kairos [CONTRACT_ADDRESS]

# With constructor arguments
npx hardhat verify --network kairos [CONTRACT_ADDRESS] "arg1" "arg2"

# Specify contract file if multiple contracts exist
npx hardhat verify --network kairos [CONTRACT_ADDRESS] --contract contracts/SageRegistry.sol:SageRegistry
```

## Verification Status

### How to Check Verification Status:
1. Go to the contract address on Klaytnscope
2. Look for the green checkmark ✓ next to "Contract"
3. Verified contracts will show:
   - Source code in the "Contract" tab
   - Read/Write functions interface
   - Contract ABI

### Verification Benefits:
- **Transparency**: Users can read and verify the source code
- **Trust**: Shows the contract is legitimate and matches deployed bytecode
- **Interaction**: Direct contract interaction through Klaytnscope interface
- **Integration**: Other dApps can easily integrate with verified contracts

## Troubleshooting

### Common Issues and Solutions:

#### 1. "Contract source code not verified"
**Solution**: 
- Ensure you're using the exact compiler version (0.8.19)
- Check optimization settings match (Enabled, 200 runs)
- Verify all imported files are included

#### 2. "Constructor arguments mismatch"
**Solution**:
- Encode constructor arguments correctly
- Use `npx hardhat verify` with correct arguments
- Check deployment transaction for actual arguments used

#### 3. "Multiple contracts with same bytecode"
**Solution**:
- Specify the exact contract path and name
- Use `--contract` flag with full path

#### 4. "Verification pending for too long"
**Solution**:
- Klaytnscope verification is usually instant
- Try manual verification through web interface
- Check if contract is already verified

## Alternative: Sourcify Verification

Kaia networks also support Sourcify verification:

```javascript
// In hardhat.config.js
sourcify: {
  enabled: true
}
```

Then verify using:
```bash
npx hardhat verify --network kairos [CONTRACT_ADDRESS]
```

## Quick Commands Reference

```bash
# Deploy to Kairos testnet
npm run deploy:kairos

# Automatic verification on Kairos
npm run verify:kairos

# Manual verification (opens browser)
npm run verify:kairos:manual

# Deploy to Kaia mainnet
npm run deploy:kaia

# Automatic verification on Kaia
npm run verify:kaia

# Manual verification (opens browser)
npm run verify:kaia:manual

# Direct hardhat verify command
npx hardhat verify --network kairos [ADDRESS]
```

## Verification Checklist

- [ ] Contract deployed successfully
- [ ] Deployment information saved in `deployments/` folder
- [ ] Correct compiler version (0.8.19)
- [ ] Optimization enabled (200 runs)
- [ ] All dependencies and imports included
- [ ] Constructor arguments properly encoded (if any)
- [ ] Network selection correct (kairos/kaia)

## Resources

- **Kairos Testnet Explorer**: https://kairos.klaytnscope.com
- **Kaia Mainnet Explorer**: https://klaytnscope.com
- **Kaia Documentation**: https://docs.kaia.io
- **Hardhat Verify Plugin**: https://hardhat.org/hardhat-runner/plugins/nomicfoundation-hardhat-verify
- **Sourcify**: https://sourcify.dev

## Support

For issues with verification:
1. Check this guide first
2. Try manual verification via web interface
3. Consult Kaia documentation
4. Ask in Kaia Discord/Telegram community