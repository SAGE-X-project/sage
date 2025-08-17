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
    // Network validation
    const network = hre.network.name;
    const validNetworks = ["kairos", "kaia", "localhost", "hardhat"];
    
    if (!validNetworks.includes(network)) {
      throw new Error(`Invalid network: ${network}. Use one of: ${validNetworks.join(", ")}`);
    }

    log(`\n${"=".repeat(50)}`, "bright");
    log(`🚀 SAGE Registry Deployment to ${network.toUpperCase()}`, "cyan");
    log(`${"=".repeat(50)}`, "bright");
    
    // Get deployer account
    const [deployer] = await hre.ethers.getSigners();
    const balance = await deployer.getBalance();
    
    // Network information
    const chainId = hre.network.config.chainId;
    const blockNumber = await hre.ethers.provider.getBlockNumber();
    
    log("\n📍 Network Information:", "yellow");
    console.log(`   Network Name: ${network}`);
    console.log(`   Chain ID: ${chainId}`);
    console.log(`   Current Block: ${blockNumber}`);
    console.log(`   RPC URL: ${hre.network.config.url || "local"}`);
    
    log("\n👤 Deployer Information:", "yellow");
    console.log(`   Address: ${deployer.address}`);
    console.log(`   Balance: ${hre.ethers.utils.formatEther(balance)} KLAY`);
    
    // Check minimum balance
    const minBalance = hre.ethers.utils.parseEther("1");
    if (balance.lt(minBalance)) {
      throw new Error(`Insufficient balance. Need at least 1 KLAY, have ${hre.ethers.utils.formatEther(balance)} KLAY`);
    }
    
    log(`\n${"=".repeat(50)}`, "bright");
    log("📦 Starting Contract Deployment...", "cyan");
    log(`${"=".repeat(50)}`, "bright");

    // Deploy SageRegistry
    log("\n[1/3] Deploying SageRegistry...", "blue");
    const SageRegistry = await hre.ethers.getContractFactory("SageRegistry");
    const registry = await SageRegistry.deploy();
    await registry.deployed();
    
    log(`✅ SageRegistry deployed!`, "green");
    console.log(`   Address: ${registry.address}`);
    console.log(`   Transaction: ${registry.deployTransaction.hash}`);
    
    // Wait for confirmation
    log("   ⏳ Waiting for confirmations...", "yellow");
    await registry.deployTransaction.wait(2);
    log("   ✅ Confirmed!", "green");

    // Deploy SageVerificationHook
    log("\n[2/3] Deploying SageVerificationHook...", "blue");
    const SageVerificationHook = await hre.ethers.getContractFactory("SageVerificationHook");
    const verificationHook = await SageVerificationHook.deploy();
    await verificationHook.deployed();
    
    log(`✅ SageVerificationHook deployed!`, "green");
    console.log(`   Address: ${verificationHook.address}`);
    console.log(`   Transaction: ${verificationHook.deployTransaction.hash}`);
    
    // Wait for confirmation
    log("   ⏳ Waiting for confirmations...", "yellow");
    await verificationHook.deployTransaction.wait(2);
    log("   ✅ Confirmed!", "green");

    // Configure hook
    log("\n[3/3] Configuring verification hook...", "blue");
    const setHookTx = await registry.setBeforeRegisterHook(verificationHook.address);
    console.log(`   Transaction: ${setHookTx.hash}`);
    
    log("   ⏳ Waiting for confirmation...", "yellow");
    await setHookTx.wait(2);
    log("✅ Verification hook configured!", "green");

    // Verify configuration
    log("\n🔍 Verifying deployment...", "yellow");
    const registryOwner = await registry.owner();
    const beforeHook = await registry.beforeRegisterHook();
    
    console.log(`   Registry Owner: ${registryOwner}`);
    console.log(`   Before Register Hook: ${beforeHook}`);
    
    if (registryOwner !== deployer.address) {
      log("⚠️  Warning: Registry owner doesn't match deployer!", "red");
    }
    if (beforeHook !== verificationHook.address) {
      log("⚠️  Warning: Hook not properly set!", "red");
    }

    // Calculate gas used
    const endBalance = await deployer.getBalance();
    const gasUsed = balance.sub(endBalance);
    
    // Prepare deployment information
    const deploymentInfo = {
      network: network,
      chainId: chainId,
      deployer: deployer.address,
      timestamp: new Date().toISOString(),
      blockNumber: blockNumber,
      contracts: {
        SageRegistry: {
          address: registry.address,
          transactionHash: registry.deployTransaction.hash,
          blockNumber: registry.deployTransaction.blockNumber,
          gasUsed: registry.deployTransaction.gasLimit.toString()
        },
        SageVerificationHook: {
          address: verificationHook.address,
          transactionHash: verificationHook.deployTransaction.hash,
          blockNumber: verificationHook.deployTransaction.blockNumber,
          gasUsed: verificationHook.deployTransaction.gasLimit.toString()
        }
      },
      configuration: {
        beforeRegisterHook: verificationHook.address,
        setHookTransactionHash: setHookTx.hash
      },
      gasUsed: gasUsed.toString(),
      gasUsedETH: hre.ethers.utils.formatEther(gasUsed)
    };

    // Add explorer URLs for Kaia networks
    if (network === "kairos") {
      deploymentInfo.explorer = {
        registry: `https://kairos.klaytnscope.com/account/${registry.address}`,
        hook: `https://kairos.klaytnscope.com/account/${verificationHook.address}`
      };
    } else if (network === "kaia") {
      deploymentInfo.explorer = {
        registry: `https://klaytnscope.com/account/${registry.address}`,
        hook: `https://klaytnscope.com/account/${verificationHook.address}`
      };
    }

    // Save deployment information
    const deploymentsDir = path.join(__dirname, "../deployments");
    if (!fs.existsSync(deploymentsDir)) {
      fs.mkdirSync(deploymentsDir, { recursive: true });
    }

    // Save with timestamp
    const timestampFile = path.join(deploymentsDir, `${network}-${Date.now()}.json`);
    fs.writeFileSync(timestampFile, JSON.stringify(deploymentInfo, null, 2));
    
    // Save as latest
    const latestFile = path.join(deploymentsDir, `${network}-latest.json`);
    fs.writeFileSync(latestFile, JSON.stringify(deploymentInfo, null, 2));

    // Summary
    log(`\n${"=".repeat(50)}`, "bright");
    log("🎉 DEPLOYMENT SUCCESSFUL!", "green");
    log(`${"=".repeat(50)}`, "bright");
    
    log("\n📋 Contract Addresses:", "cyan");
    console.log(`   SageRegistry: ${registry.address}`);
    console.log(`   SageVerificationHook: ${verificationHook.address}`);
    
    if (deploymentInfo.explorer) {
      log("\n🔍 View on Explorer:", "cyan");
      console.log(`   Registry: ${deploymentInfo.explorer.registry}`);
      console.log(`   Hook: ${deploymentInfo.explorer.hook}`);
    }
    
    log("\n💾 Deployment files saved:", "cyan");
    console.log(`   ${timestampFile}`);
    console.log(`   ${latestFile}`);
    
    log("\n⛽ Gas Usage:", "cyan");
    console.log(`   Total: ${hre.ethers.utils.formatEther(gasUsed)} KLAY`);
    
    log("\n✨ Next Steps:", "yellow");
    console.log("   1. Verify contracts on Klaytnscope (run: npm run verify:kairos)");
    console.log("   2. Test the deployment (run: npm run test:kairos)");
    console.log("   3. Register your first agent!");
    
    log(`\n${"=".repeat(50)}\n`, "bright");

  } catch (error) {
    log(`\n${"=".repeat(50)}`, "bright");
    log("❌ DEPLOYMENT FAILED", "red");
    log(`${"=".repeat(50)}`, "bright");
    console.error("\nError details:", error.message);
    
    if (error.code === "INSUFFICIENT_FUNDS") {
      log("\n💡 Solution: Get test KLAY from https://faucet.kairos.kaia.io/", "yellow");
    } else if (error.code === "NETWORK_ERROR") {
      log("\n💡 Solution: Check your internet connection and RPC endpoint", "yellow");
    }
    
    process.exit(1);
  }
}

// Execute deployment
main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });