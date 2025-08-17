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
    const network = hre.network.name;
    
    // Validate network
    if (!["kairos", "kaia"].includes(network)) {
      throw new Error(`Network ${network} not supported. Use 'kairos' or 'kaia'`);
    }

    log(`\n${"=".repeat(50)}`, "bright");
    log(`🔍 Contract Verification on ${network.toUpperCase()}`, "cyan");
    log(`${"=".repeat(50)}`, "bright");

    // Load deployment info
    const deploymentPath = path.join(__dirname, `../deployments/${network}-latest.json`);
    
    if (!fs.existsSync(deploymentPath)) {
      throw new Error(`No deployment found for ${network}. Deploy first with: npm run deploy:${network}`);
    }

    const deployment = JSON.parse(fs.readFileSync(deploymentPath, 'utf8'));
    
    log("\n📋 Deployment Information:", "yellow");
    console.log(`   Network: ${deployment.network}`);
    console.log(`   Chain ID: ${deployment.chainId}`);
    console.log(`   Deployed at: ${deployment.timestamp}`);
    
    log("\n📍 Contract Addresses:", "yellow");
    console.log(`   SageRegistry: ${deployment.contracts.SageRegistry.address}`);
    console.log(`   SageVerificationHook: ${deployment.contracts.SageVerificationHook.address}`);

    // Prepare verification
    log(`\n${"=".repeat(50)}`, "bright");
    log("🚀 Starting Verification Process...", "cyan");
    log(`${"=".repeat(50)}`, "bright");

    // Note about Kaia/Kairos verification
    log("\n📌 IMPORTANT: Kaia Network Verification", "yellow");
    console.log("\nKaia (including Kairos testnet) uses Klaytnscope for contract verification.");
    console.log("Unlike Etherscan, Klaytnscope doesn't require API keys.\n");

    // Verify SageRegistry
    log("[1/2] Verifying SageRegistry...", "blue");
    try {
      await hre.run("verify:verify", {
        address: deployment.contracts.SageRegistry.address,
        constructorArguments: [],
        contract: "contracts/SageRegistry.sol:SageRegistry"
      });
      log("✅ SageRegistry verified successfully!", "green");
    } catch (error) {
      if (error.message.includes("already verified")) {
        log("ℹ️  SageRegistry is already verified", "yellow");
      } else if (error.message.includes("does not have bytecode")) {
        log("⚠️  SageRegistry not found on chain. Deploy it first.", "red");
      } else {
        log(`❌ SageRegistry verification failed: ${error.message}`, "red");
        
        // Provide manual verification instructions
        log("\n💡 Manual Verification Alternative:", "yellow");
        console.log("\n1. Go to Klaytnscope:");
        if (network === "kairos") {
          console.log(`   https://kairos.klaytnscope.com/account/${deployment.contracts.SageRegistry.address}`);
        } else {
          console.log(`   https://klaytnscope.com/account/${deployment.contracts.SageRegistry.address}`);
        }
        console.log("\n2. Click on 'Contract' tab");
        console.log("3. Click 'Verify and Publish'");
        console.log("4. Fill in:");
        console.log("   - Compiler: v0.8.19");
        console.log("   - Optimization: Yes (200 runs)");
        console.log("   - Contract name: SageRegistry");
        console.log("5. Paste the contract source code");
        console.log("6. Submit for verification\n");
      }
    }

    // Verify SageVerificationHook
    log("[2/2] Verifying SageVerificationHook...", "blue");
    try {
      await hre.run("verify:verify", {
        address: deployment.contracts.SageVerificationHook.address,
        constructorArguments: [],
        contract: "contracts/SageVerificationHook.sol:SageVerificationHook"
      });
      log("✅ SageVerificationHook verified successfully!", "green");
    } catch (error) {
      if (error.message.includes("already verified")) {
        log("ℹ️  SageVerificationHook is already verified", "yellow");
      } else if (error.message.includes("does not have bytecode")) {
        log("⚠️  SageVerificationHook not found on chain. Deploy it first.", "red");
      } else {
        log(`❌ SageVerificationHook verification failed: ${error.message}`, "red");
        
        // Provide manual verification instructions
        log("\n💡 Manual Verification Alternative:", "yellow");
        console.log("\n1. Go to Klaytnscope:");
        if (network === "kairos") {
          console.log(`   https://kairos.klaytnscope.com/account/${deployment.contracts.SageVerificationHook.address}`);
        } else {
          console.log(`   https://klaytnscope.com/account/${deployment.contracts.SageVerificationHook.address}`);
        }
        console.log("\n2. Click on 'Contract' tab");
        console.log("3. Click 'Verify and Publish'");
        console.log("4. Fill in:");
        console.log("   - Compiler: v0.8.19");
        console.log("   - Optimization: Yes (200 runs)");
        console.log("   - Contract name: SageVerificationHook");
        console.log("5. Paste the contract source code");
        console.log("6. Submit for verification\n");
      }
    }

    // Summary
    log(`\n${"=".repeat(50)}`, "bright");
    log("📊 Verification Summary", "cyan");
    log(`${"=".repeat(50)}`, "bright");
    
    log("\n🔗 View Verified Contracts:", "yellow");
    if (network === "kairos") {
      console.log(`   Registry: https://kairos.klaytnscope.com/account/${deployment.contracts.SageRegistry.address}?tabId=contractCode`);
      console.log(`   Hook: https://kairos.klaytnscope.com/account/${deployment.contracts.SageVerificationHook.address}?tabId=contractCode`);
    } else {
      console.log(`   Registry: https://klaytnscope.com/account/${deployment.contracts.SageRegistry.address}?tabId=contractCode`);
      console.log(`   Hook: https://klaytnscope.com/account/${deployment.contracts.SageVerificationHook.address}?tabId=contractCode`);
    }
    
    log("\n✨ Tips:", "yellow");
    console.log("   - Verification might take a few minutes to process");
    console.log("   - If automatic verification fails, use manual verification via web interface");
    console.log("   - Verified contracts show a green checkmark on Klaytnscope");
    console.log("   - You can interact with verified contracts directly on Klaytnscope");
    
    log(`\n${"=".repeat(50)}\n`, "bright");

  } catch (error) {
    log(`\n${"=".repeat(50)}`, "bright");
    log("❌ VERIFICATION FAILED", "red");
    log(`${"=".repeat(50)}`, "bright");
    console.error("\nError:", error.message);
    
    if (error.message.includes("No deployment found")) {
      log("\n💡 Solution: Deploy contracts first", "yellow");
      console.log(`   Run: npm run deploy:${hre.network.name}`);
    }
    
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