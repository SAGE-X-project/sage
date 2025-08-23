# Contract Verification Instructions for Klaytnscope

## Prerequisites
- Contract addresses from deployment
- Klaytnscope explorer access

## Steps to Verify

### 1. Visit Klaytnscope
- Testnet (Kairos): https://kairos.klaytnscope.com
- Mainnet (Cypress): https://klaytnscope.com

### 2. Search for Your Contract
- Enter the contract address in the search bar
- Navigate to the contract page

### 3. Click "Contract" Tab
- If not verified, you'll see the bytecode
- Click "Verify and Publish" button

### 4. Fill Verification Form

#### For SageRegistryV2:
- **Contract Name**: SageRegistryV2
- **Compiler Version**: v0.8.19+commit.7dd6d404
- **EVM Version**: paris (or default)
- **Optimization**: Yes
- **Optimization Runs**: 200
- **Source Code**: Copy entire content from `SageRegistryV2_flat.sol`

#### For SageVerificationHook:
- **Contract Name**: SageVerificationHook
- **Compiler Version**: v0.8.19+commit.7dd6d404
- **EVM Version**: paris (or default)
- **Optimization**: Yes
- **Optimization Runs**: 200
- **Source Code**: Copy entire content from `SageVerificationHook_flat.sol`

### 5. Additional Settings
- **Constructor Arguments**: Leave empty (both contracts have no constructor parameters)
- **Libraries**: Not used

### 6. Submit Verification
- Click "Verify and Publish"
- Wait for verification to complete

## Alternative: Using Hardhat Verify

```bash
# Set contract addresses in .env or as environment variables
export SAGE_REGISTRY_ADDRESS=0x...
export SAGE_VERIFICATION_HOOK_ADDRESS=0x...

# Run verification script
npx hardhat run scripts/verify-contracts.js --network kairos
```

## Troubleshooting

### "Similar bytecode already verified"
- The contract is already verified, check the "Contract" tab

### "Bytecode does not match"
- Ensure compiler version matches exactly: v0.8.19
- Optimization must be enabled with 200 runs
- Use the flattened source files provided

### Network Connection Issues
- Try using a VPN if Klaytnscope is slow
- Alternatively, use the Hardhat verify plugin
