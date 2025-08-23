#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔧 Flattening Smart Contracts for Verification${NC}"
echo "================================================"

# Create flattened directory if it doesn't exist
mkdir -p flattened

# Flatten SageRegistryV2
echo -e "\n${YELLOW}📄 Flattening SageRegistryV2.sol...${NC}"
npx hardhat flatten contracts/SageRegistryV2.sol > flattened/SageRegistryV2_flat.sol

# Remove duplicate SPDX license identifiers (Klaytnscope doesn't like multiple)
sed -i.bak '/SPDX-License-Identifier/d' flattened/SageRegistryV2_flat.sol
# Add single SPDX license at the top
sed -i.bak '1s/^/\/\/ SPDX-License-Identifier: MIT\n/' flattened/SageRegistryV2_flat.sol
rm flattened/SageRegistryV2_flat.sol.bak

echo -e "${GREEN}✅ SageRegistryV2 flattened${NC}"

# Flatten SageVerificationHook
echo -e "\n${YELLOW}📄 Flattening SageVerificationHook.sol...${NC}"
npx hardhat flatten contracts/SageVerificationHook.sol > flattened/SageVerificationHook_flat.sol

# Remove duplicate SPDX license identifiers
sed -i.bak '/SPDX-License-Identifier/d' flattened/SageVerificationHook_flat.sol
# Add single SPDX license at the top
sed -i.bak '1s/^/\/\/ SPDX-License-Identifier: MIT\n/' flattened/SageVerificationHook_flat.sol
rm flattened/SageVerificationHook_flat.sol.bak

echo -e "${GREEN}✅ SageVerificationHook flattened${NC}"

# Create verification instructions
cat > flattened/VERIFICATION_INSTRUCTIONS.md << 'EOF'
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
EOF

echo -e "\n${GREEN}✅ All contracts flattened successfully!${NC}"
echo -e "\n${BLUE}📁 Flattened contracts saved in:${NC}"
echo "   - flattened/SageRegistryV2_flat.sol"
echo "   - flattened/SageVerificationHook_flat.sol"
echo "   - flattened/VERIFICATION_INSTRUCTIONS.md"

echo -e "\n${YELLOW}📝 Next Steps:${NC}"
echo "1. Review the flattened contracts"
echo "2. Follow instructions in VERIFICATION_INSTRUCTIONS.md"
echo "3. Or run: npx hardhat run scripts/verify-contracts.js --network kairos"