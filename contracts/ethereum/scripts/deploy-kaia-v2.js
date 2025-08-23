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
    log(`🚀 SAGE Registry V2 Deployment to ${network.toUpperCase()}`, "cyan");
    log(`${"=".repeat(50)}`, "bright");
    
    // Get deployer account
    const [deployer] = await hre.ethers.getSigners();
    const balance = await deployer.provider.getBalance(deployer.address);
    
    // Network information
    const chainId = (await deployer.provider.getNetwork()).chainId;
    const blockNumber = await deployer.provider.getBlockNumber();
    
    log("\n📍 Network Information:", "yellow");
    console.log(`   Network Name: ${network}`);
    console.log(`   Chain ID: ${chainId}`);
    console.log(`   Current Block: ${blockNumber}`);
    console.log(`   RPC URL: ${hre.network.config.url || "local"}`);
    
    log("\n👤 Deployer Information:", "yellow");
    console.log(`   Address: ${deployer.address}`);
    console.log(`   Balance: ${hre.ethers.formatEther(balance)} KLAY`);
    
    // Check minimum balance
    const minBalance = hre.ethers.parseEther("1");
    if (balance < minBalance) {
      throw new Error(`Insufficient balance. Need at least 1 KLAY, have ${hre.ethers.formatEther(balance)} KLAY`);
    }
    
    log(`\n${"=".repeat(50)}`, "bright");
    log("📦 Starting Contract Deployment...", "cyan");
    log(`${"=".repeat(50)}`, "bright");

    // Deploy SageRegistryV2
    log("\n[1/2] Deploying SageRegistryV2...", "blue");
    const SageRegistryV2 = await hre.ethers.getContractFactory("SageRegistryV2");
    const registry = await SageRegistryV2.deploy();
    await registry.waitForDeployment();
    const registryAddress = await registry.getAddress();
    
    log(`✅ SageRegistryV2 deployed!`, "green");
    console.log(`   Address: ${registryAddress}`);
    const deployTx = registry.deploymentTransaction();
    console.log(`   Transaction: ${deployTx.hash}`);
    
    // Wait for confirmation
    log("   ⏳ Waiting for confirmations...", "yellow");
    await deployTx.wait(2);
    log("   ✅ Confirmed!", "green");

    // Deploy SageVerificationHook
    log("\n[2/2] Deploying SageVerificationHook...", "blue");
    const SageVerificationHook = await hre.ethers.getContractFactory("SageVerificationHook");
    const verificationHook = await SageVerificationHook.deploy();
    await verificationHook.waitForDeployment();
    const hookAddress = await verificationHook.getAddress();
    
    log(`✅ SageVerificationHook deployed!`, "green");
    console.log(`   Address: ${hookAddress}`);
    const hookTx = verificationHook.deploymentTransaction();
    console.log(`   Transaction: ${hookTx.hash}`);
    
    // Wait for confirmation
    log("   ⏳ Waiting for confirmations...", "yellow");
    await hookTx.wait(2);
    log("   ✅ Confirmed!", "green");

    // Configure hooks
    log("\n🔧 Configuring hooks...", "blue");
    const setHookTx = await registry.setBeforeRegisterHook(hookAddress);
    await setHookTx.wait();
    log(`✅ BeforeRegisterHook set to: ${hookAddress}`, "green");

    // Save deployment info (convert BigInt to string for JSON serialization)
    const deploymentInfo = {
      network: network,
      chainId: chainId.toString(),
      deployedAt: new Date().toISOString(),
      deployer: deployer.address,
      contracts: {
        SageRegistryV2: registryAddress,
        SageVerificationHook: hookAddress
      },
      blockNumber: blockNumber.toString(),
      gasPrice: hre.network.config.gasPrice ? hre.network.config.gasPrice.toString() : "250000000000"
    };

    const deploymentPath = path.join(__dirname, `../deployments/${network}-v2-${Date.now()}.json`);
    fs.mkdirSync(path.dirname(deploymentPath), { recursive: true });
    fs.writeFileSync(deploymentPath, JSON.stringify(deploymentInfo, null, 2));

    log(`\n${"=".repeat(50)}`, "bright");
    log("🎉 Deployment Complete!", "green");
    log(`${"=".repeat(50)}`, "bright");
    
    log("\n📋 Summary:", "cyan");
    console.log(`   SageRegistryV2: ${registryAddress}`);
    console.log(`   SageVerificationHook: ${hookAddress}`);
    console.log(`   Network: ${network}`);
    console.log(`   Chain ID: ${chainId}`);
    
    log("\n📝 Deployment info saved to:", "yellow");
    console.log(`   ${deploymentPath}`);
    
    // Verification instructions
    if (network === "kairos" || network === "kaia") {
      log("\n🔍 To verify contracts on explorer:", "yellow");
      console.log(`   Visit: https://${network === "kairos" ? "kairos" : "www"}.klaytnscope.com`);
      console.log(`   Search for: ${registryAddress}`);
      console.log(`   Use the web interface to verify the contract`);
    }

    log("\n✨ Next Steps:", "cyan");
    log("   1. Save the contract addresses", "reset");
    log("   2. Update your .env file with the new addresses", "reset");
    log("   3. Register your production agents", "reset");
    
  } catch (error) {
    log(`\n❌ Deployment failed: ${error.message}`, "red");
    console.error(error);
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