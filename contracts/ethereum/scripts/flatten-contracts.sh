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

# Flatten AgentCardRegistry
echo -e "\n${YELLOW}📄 Flattening AgentCardRegistry.sol...${NC}"
npx hardhat flatten contracts/AgentCardRegistry.sol > flattened/AgentCardRegistry_flat.sol

# Remove duplicate SPDX license identifiers (explorers don't like multiple)
sed -i.bak '/SPDX-License-Identifier/d' flattened/AgentCardRegistry_flat.sol
# Add single SPDX license at the top
sed -i.bak '1s/^/\/\/ SPDX-License-Identifier: MIT\n/' flattened/AgentCardRegistry_flat.sol
rm flattened/AgentCardRegistry_flat.sol.bak

echo -e "${GREEN}✅ AgentCardRegistry flattened${NC}"

# Flatten AgentCardVerifyHook
echo -e "\n${YELLOW}📄 Flattening AgentCardVerifyHook.sol...${NC}"
npx hardhat flatten contracts/AgentCardVerifyHook.sol > flattened/AgentCardVerifyHook_flat.sol

# Remove duplicate SPDX license identifiers
sed -i.bak '/SPDX-License-Identifier/d' flattened/AgentCardVerifyHook_flat.sol
# Add single SPDX license at the top
sed -i.bak '1s/^/\/\/ SPDX-License-Identifier: MIT\n/' flattened/AgentCardVerifyHook_flat.sol
rm flattened/AgentCardVerifyHook_flat.sol.bak

echo -e "${GREEN}✅ AgentCardVerifyHook flattened${NC}"

# Flatten AgentCardStorage
echo -e "\n${YELLOW}📄 Flattening AgentCardStorage.sol...${NC}"
npx hardhat flatten contracts/AgentCardStorage.sol > flattened/AgentCardStorage_flat.sol

# Remove duplicate SPDX license identifiers
sed -i.bak '/SPDX-License-Identifier/d' flattened/AgentCardStorage_flat.sol
# Add single SPDX license at the top
sed -i.bak '1s/^/\/\/ SPDX-License-Identifier: MIT\n/' flattened/AgentCardStorage_flat.sol
rm flattened/AgentCardStorage_flat.sol.bak

echo -e "${GREEN}✅ AgentCardStorage flattened${NC}"

# Create verification instructions
cat > flattened/VERIFICATION_INSTRUCTIONS.md << 'EOF'
# Contract Verification Instructions

## Prerequisites
- Contract addresses from deployment
- Block explorer access (Etherscan, Kaiascan, etc.)
- API key configured in .env

## Automated Verification (Recommended)

Use the verify-agentcard.js script:

```bash
npx hardhat run scripts/verify-agentcard.js --network <network>
```

Supported networks:
- Ethereum: mainnet, sepolia
- Kaia: cypress (mainnet), kairos (testnet)
- BSC: bsc, bscTestnet
- Base: base, baseSepolia
- Arbitrum: arbitrum, arbitrumSepolia
- Optimism: optimism, optimismSepolia

## Manual Verification (Fallback)

If automated verification fails, use flattened contracts from this directory:

### 1. Visit Block Explorer

- Ethereum: https://etherscan.io (mainnet) or https://sepolia.etherscan.io (testnet)
- Kaia: https://kaiascan.io (mainnet) or https://kairos.kaiascan.io (testnet)
- BSC: https://bscscan.com (mainnet) or https://testnet.bscscan.com (testnet)
- Base: https://basescan.org (mainnet) or https://sepolia.basescan.org (testnet)
- Arbitrum: https://arbiscan.io (mainnet) or https://sepolia.arbiscan.io (testnet)
- Optimism: https://optimistic.etherscan.io (mainnet) or https://sepolia-optimistic.etherscan.io (testnet)

### 2. Navigate to Contract Verification

1. Go to your contract address page
2. Click "Contract" tab
3. Click "Verify and Publish"

### 3. Fill in Verification Form

**Compiler Configuration:**
- Compiler Type: Solidity (Single file)
- Compiler Version: v0.8.20+commit.a1b79de6
- License Type: MIT

**Contract Details:**

#### For AgentCardRegistry:
- Contract Name: AgentCardRegistry
- Flattened Source: Copy content from `flattened/AgentCardRegistry_flat.sol`
- Constructor Arguments: ABI-encoded hookAddress (get from deployment file)
- Optimization: Yes
- Runs: 200 (or check hardhat.config.js)
- EVM Version: shanghai

#### For AgentCardVerifyHook:
- Contract Name: AgentCardVerifyHook
- Flattened Source: Copy content from `flattened/AgentCardVerifyHook_flat.sol`
- Constructor Arguments: (none)
- Optimization: Yes
- Runs: 200 (or check hardhat.config.js)
- EVM Version: shanghai

#### For AgentCardStorage:
- Contract Name: AgentCardStorage
- Flattened Source: Copy content from `flattened/AgentCardStorage_flat.sol`
- Constructor Arguments: (none - library)
- Optimization: Yes
- Runs: 200 (or check hardhat.config.js)
- EVM Version: shanghai

### 4. Get Constructor Arguments

Constructor arguments must be ABI-encoded. Get them from deployment JSON file:

```bash
cat deployments/<network>-latest.json | jq '.contracts.AgentCardRegistry.constructorArgs'
```

Or use online ABI encoder:
- https://abi.hashex.org/

For AgentCardRegistry, encode:
- Type: `address`
- Value: `<hook_address>`

### 5. Submit and Wait

- Submit the verification form
- Wait for processing (usually 1-2 minutes)
- Check for success/failure messages

## Troubleshooting

### Common Issues

1. **Compiler version mismatch**
   - Solution: Check contracts for `pragma solidity ^0.8.20;`
   - Use exact compiler version from deployment

2. **Constructor arguments error**
   - Solution: Ensure ABI encoding is correct
   - Verify hookAddress is correctly encoded

3. **Optimization settings**
   - Solution: Match hardhat.config.js settings
   - Check `optimizer.runs` value

4. **Library not found**
   - Solution: Verify AgentCardStorage first if needed
   - Or use flattened version (already includes libraries)

### Get Help

- Check deployment file: `deployments/<network>-latest.json`
- View compiler settings: `hardhat.config.js`
- GitHub Issues: https://github.com/SAGE-X-project/sage/issues

## Files in this Directory

- `AgentCardRegistry_flat.sol` - Flattened main registry contract
- `AgentCardVerifyHook_flat.sol` - Flattened validation hook contract
- `AgentCardStorage_flat.sol` - Flattened storage library
- `VERIFICATION_INSTRUCTIONS.md` - This file

## Notes

- Flattened files have a single SPDX license identifier at the top
- All imports are inlined for single-file verification
- Original file structure is preserved in comments
- Optimization settings should match hardhat.config.js

---

**Last Updated**: 2025-11-01
**AgentCard Version**: v4.1
EOF

echo -e "\n${GREEN}✅ All contracts flattened successfully!${NC}"
echo -e "${BLUE}📁 Flattened contracts saved to: flattened/${NC}"
echo ""
echo "Files created:"
echo "  - flattened/AgentCardRegistry_flat.sol"
echo "  - flattened/AgentCardVerifyHook_flat.sol"
echo "  - flattened/AgentCardStorage_flat.sol"
echo "  - flattened/VERIFICATION_INSTRUCTIONS.md"
echo ""
echo -e "${YELLOW}ℹ️  For verification, use:${NC}"
echo "  1. Automated: npx hardhat run scripts/verify-agentcard.js --network <network>"
echo "  2. Manual: Follow instructions in flattened/VERIFICATION_INSTRUCTIONS.md"
echo ""
