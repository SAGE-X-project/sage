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

async function verifyContract(contractName, address, constructorArgs = []) {
  try {
    log(`\n Verifying ${contractName}...`, "blue");
    console.log(`   Address: ${address}`);
    console.log(`   Network: ${hre.network.name}`);
    
    await hre.run("verify:verify", {
      address: address,
      constructorArguments: constructorArgs,
    });
    
    log(` ${contractName} verified successfully!`, "green");
    return true;
  } catch (error) {
    if (error.message.includes("Already Verified")) {
      log(`ℹ️  ${contractName} is already verified`, "yellow");
      return true;
    } else if (error.message.includes("does not have bytecode")) {
      log(` ${contractName} not found at address ${address}`, "red");
      return false;
    } else {
      log(` Failed to verify ${contractName}: ${error.message}`, "red");
      return false;
    }
  }
}

async function main() {
  try {
    const network = hre.network.name;
    
    log(`\n${"=".repeat(50)}`, "bright");
    log(` Contract Verification on ${network.toUpperCase()}`, "cyan");
    log(`${"=".repeat(50)}`, "bright");
    
    // Check network
    if (!["kairos", "kaia", "cypress"].includes(network)) {
      throw new Error("Verification only supported on Kaia networks (kairos, kaia, cypress)");
    }
    
    // Try to load deployment info
    let deploymentInfo = null;
    let registryAddress = process.env.SAGE_REGISTRY_ADDRESS;
    let hookAddress = process.env.SAGE_VERIFICATION_HOOK_ADDRESS;
    
    // First try to load from latest deployment file
    const latestPath = path.join(__dirname, `../deployments/${network}-v2-latest.json`);
    if (fs.existsSync(latestPath)) {
      deploymentInfo = JSON.parse(fs.readFileSync(latestPath, 'utf8'));
      registryAddress = deploymentInfo.contracts.SageRegistryV2;
      hookAddress = deploymentInfo.contracts.SageVerificationHook;
      log("\n Loaded deployment info from file", "green");
    }
    
    // Override with environment variables if set
    if (process.env.SAGE_REGISTRY_ADDRESS) {
      registryAddress = process.env.SAGE_REGISTRY_ADDRESS;
      log(" Using registry address from environment", "yellow");
    }
    if (process.env.SAGE_VERIFICATION_HOOK_ADDRESS) {
      hookAddress = process.env.SAGE_VERIFICATION_HOOK_ADDRESS;
      log(" Using hook address from environment", "yellow");
    }
    
    // Validate addresses
    if (!registryAddress || !hookAddress) {
      throw new Error("Contract addresses not found. Deploy contracts first or set environment variables.");
    }
    
    log("\n📍 Contracts to verify:", "cyan");
    console.log(`   SageRegistryV2: ${registryAddress}`);
    console.log(`   SageVerificationHook: ${hookAddress}`);
    
    // Get network info
    const chainId = (await hre.ethers.provider.getNetwork()).chainId;
    const explorerBase = network === "kairos" ? "https://kairos.klaytnscope.com" : 
                        network === "kaia" || network === "cypress" ? "https://klaytnscope.com" : "";
    
    log("\n Network Info:", "yellow");
    console.log(`   Network: ${network}`);
    console.log(`   Chain ID: ${chainId}`);
    console.log(`   Explorer: ${explorerBase}`);
    
    // Verify contracts
    log(`\n${"=".repeat(50)}`, "bright");
    log(" Starting Verification Process", "cyan");
    log(`${"=".repeat(50)}`, "bright");
    
    const results = [];
    
    // Verify SageRegistryV2 (no constructor arguments)
    const registryVerified = await verifyContract(
      "SageRegistryV2",
      registryAddress,
      []
    );
    results.push({ name: "SageRegistryV2", address: registryAddress, verified: registryVerified });
    
    // Verify SageVerificationHook (no constructor arguments)
    const hookVerified = await verifyContract(
      "SageVerificationHook",
      hookAddress,
      []
    );
    results.push({ name: "SageVerificationHook", address: hookAddress, verified: hookVerified });
    
    // Summary
    log(`\n${"=".repeat(50)}`, "bright");
    log(" Verification Summary", "cyan");
    log(`${"=".repeat(50)}`, "bright");
    
    const successCount = results.filter(r => r.verified).length;
    const failCount = results.filter(r => !r.verified).length;
    
    console.log(`\n Successfully verified: ${successCount}/${results.length}`);
    if (failCount > 0) {
      console.log(` Failed to verify: ${failCount}/${results.length}`);
    }
    
    results.forEach(result => {
      const status = result.verified ? "" : "";
      console.log(`   ${status} ${result.name}: ${result.address}`);
    });
    
    if (explorerBase) {
      log("\n View on Explorer:", "cyan");
      results.forEach(result => {
        if (result.verified) {
          console.log(`   ${result.name}: ${explorerBase}/account/${result.address}?tab=code`);
        }
      });
    }
    
    // Alternative verification methods for Kaia
    if (network === "kairos" || network === "kaia" || network === "cypress") {
      log("\n Alternative Verification Methods:", "yellow");
      log("\n1. Manual verification on Klaytnscope:", "reset");
      console.log(`   - Visit ${explorerBase}`);
      console.log("   - Search for your contract address");
      console.log("   - Click 'Contract' tab");
      console.log("   - Click 'Verify and Publish'");
      console.log("   - Select compiler version: 0.8.19");
      console.log("   - Enable optimization: Yes (200 runs)");
      console.log("   - Paste contract source code");
      
      log("\n2. Using Sourcify (automatic):", "reset");
      console.log("   npx hardhat verify --network", network, registryAddress);
      console.log("   npx hardhat verify --network", network, hookAddress);
      
      log("\n3. Using flattened source:", "reset");
      console.log("   npx hardhat flatten contracts/SageRegistryV2.sol > SageRegistryV2_flat.sol");
      console.log("   npx hardhat flatten contracts/SageVerificationHook.sol > SageVerificationHook_flat.sol");
      console.log("   Then upload the flattened files to Klaytnscope");
    }
    
    log("\n Verification process complete!", "green");
    
  } catch (error) {
    log(`\n Verification failed: ${error.message}`, "red");
    console.error(error);
    process.exit(1);
  }
}

// Execute verification
main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });