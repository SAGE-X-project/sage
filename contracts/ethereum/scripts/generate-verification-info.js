const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

// Color codes for console output
const colors = {
  reset: "\x1b[0m",
  bright: "\x1b[1m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
  red: "\x1b[31m",
  cyan: "\x1b[36m"
};

function log(message, color = "reset") {
  console.log(colors[color] + message + colors.reset);
}

async function main() {
  try {
    log(`\n${"=".repeat(50)}`, "bright");
    log(`📋 Generating Verification Information`, "cyan");
    log(`${"=".repeat(50)}`, "bright");
    
    // Get compiler settings from hardhat config
    const compilerSettings = hre.config.solidity;
    
    log("\n🔧 Compiler Settings:", "yellow");
    console.log("   Solidity Version: 0.8.19");
    console.log("   Optimization: Enabled");
    console.log("   Optimization Runs: 200");
    console.log("   Via IR: true (IMPORTANT!)");
    console.log("   EVM Version: paris (default)");
    
    log("\n⚠️  IMPORTANT for Klaytnscope Verification:", "red");
    console.log("   1. Use EXACT compiler version: v0.8.19+commit.7dd6d404");
    console.log("   2. Enable 'Via IR' compilation");
    console.log("   3. Set optimization runs to 200");
    console.log("   4. Use the flattened source files");
    
    // Generate Standard JSON Input for verification
    const standardJsonInput = {
      language: "Solidity",
      sources: {
        "SageRegistryV2.sol": {
          content: fs.readFileSync(
            path.join(__dirname, "../flattened/SageRegistryV2_flat.sol"),
            "utf8"
          )
        }
      },
      settings: {
        optimizer: {
          enabled: true,
          runs: 200,
          details: {
            yul: true,
            yulDetails: {
              stackAllocation: true,
              optimizerSteps: "dhfoDgvulfnTUtnIf"
            }
          }
        },
        viaIR: true,
        outputSelection: {
          "*": {
            "*": ["abi", "evm.bytecode", "evm.deployedBytecode", "evm.methodIdentifiers"]
          }
        },
        evmVersion: "paris"
      }
    };
    
    // Save Standard JSON Input
    const verificationDir = path.join(__dirname, "../verification");
    fs.mkdirSync(verificationDir, { recursive: true });
    
    fs.writeFileSync(
      path.join(verificationDir, "standard-json-input.json"),
      JSON.stringify(standardJsonInput, null, 2)
    );
    
    log("\n📁 Files Generated:", "green");
    console.log("   verification/standard-json-input.json");
    
    // Create verification instructions
    const instructions = `
# Klaytnscope Contract Verification Guide

## Contract Addresses (Kairos Testnet)
- SageRegistryV2: 0xF1f53bd1dEc3f91Ffba5E66f4397aF2ec3eAF7fE
- SageVerificationHook: 0xDA0777245c125a7F8a733945d93bAe100F814093

## Method 1: Web Interface Verification

### Step 1: Navigate to Klaytnscope
1. Go to: https://kairos.klaytnscope.com
2. Search for the contract address
3. Click on "Contract" tab
4. Click "Verify and Publish"

### Step 2: Fill Verification Form
**CRITICAL SETTINGS:**
- Contract Name: SageRegistryV2 (or SageVerificationHook)
- Compiler Type: Solidity (Single file)
- Compiler Version: v0.8.19+commit.7dd6d404
- Open Source License: MIT

**Optimization Settings:**
- Optimization: Yes
- Runs: 200
- Via IR: YES ✅ (This is critical!)

**Advanced Settings (if available):**
- EVM Version: paris
- Enable Yul Optimizer: Yes

### Step 3: Paste Source Code
Use the flattened source from:
- flattened/SageRegistryV2_flat.sol
- flattened/SageVerificationHook_flat.sol

### Step 4: Constructor Arguments
Leave empty (both contracts have no constructor parameters)

## Method 2: Using Standard JSON Input

If the web interface fails, try using Standard JSON Input:

1. Use the generated file: verification/standard-json-input.json
2. In Klaytnscope, select "Standard JSON Input" option
3. Upload or paste the JSON file content
4. Submit for verification

## Common Issues and Solutions

### "Cannot generate bytecode" Error
**Cause**: Compiler settings mismatch
**Solution**: 
- Ensure "Via IR" is enabled
- Use exact compiler version v0.8.19+commit.7dd6d404
- Check optimization is set to 200 runs

### "Bytecode does not match"
**Cause**: Missing advanced optimizer settings
**Solution**:
- Enable Yul optimizer
- Set optimizer steps if possible
- Use Standard JSON Input method

### Alternative: Remix Verification
1. Open https://remix.ethereum.org
2. Create new file with flattened source
3. Set compiler to 0.8.19
4. Enable optimization (200 runs)
5. Enable "Via IR" in Advanced Configuration
6. Compile
7. Use Remix plugin for Klaytnscope verification

## Compiler Configuration Summary
\`\`\`json
{
  "version": "0.8.19",
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
    "evmVersion": "paris"
  }
}
\`\`\`
`;
    
    fs.writeFileSync(
      path.join(verificationDir, "VERIFICATION_GUIDE.md"),
      instructions
    );
    
    log("\n📝 Verification Instructions:", "cyan");
    console.log("   verification/VERIFICATION_GUIDE.md");
    
    // Try to compile locally to get exact bytecode
    log("\n🔨 Compiling contracts locally...", "yellow");
    await hre.run("compile");
    
    // Get build info
    const buildInfoDir = path.join(__dirname, "../artifacts/build-info");
    const buildFiles = fs.readdirSync(buildInfoDir);
    if (buildFiles.length > 0) {
      const latestBuild = buildFiles[buildFiles.length - 1];
      log(`\n✅ Build info available: ${latestBuild}`, "green");
      console.log("   This contains exact compiler settings used");
    }
    
    log("\n✨ Next Steps:", "yellow");
    log("1. Try web verification with Via IR enabled", "reset");
    log("2. If that fails, use Standard JSON Input method", "reset");
    log("3. As last resort, use Remix IDE for verification", "reset");
    
    log("\n🔑 Key Points:", "red");
    log("   - Via IR MUST be enabled", "reset");
    log("   - Use exact compiler v0.8.19+commit.7dd6d404", "reset");
    log("   - Optimization must be 200 runs", "reset");
    
  } catch (error) {
    log(`\n❌ Error: ${error.message}`, "red");
    console.error(error);
    process.exit(1);
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });