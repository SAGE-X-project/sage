const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

/**
 * Deploy SageRegistryV4 to specified network
 *
 * Usage:
 *   npx hardhat run scripts/deploy_v4.js --network sepolia
 *   npx hardhat run scripts/deploy_v4.js --network mainnet
 *   npx hardhat run scripts/deploy_v4.js --network localhost
 *
 * Environment variables:
 *   INITIAL_OWNER - Optional initial owner address (defaults to deployer)
 *   DEPLOYED_ADDRESSES_FILE - Optional path to save deployment info
 */
async function main() {
  console.log("\n╔═══════════════════════════════════════════════════════════╗");
  console.log("║     SageRegistryV4 Deployment Script                     ║");
  console.log("╚═══════════════════════════════════════════════════════════╝\n");

  // Get deployment configuration
  const network = hre.network.name;
  const [deployer] = await hre.ethers.getSigners();
  const chainId = (await hre.ethers.provider.getNetwork()).chainId;

  console.log("📋 Deployment Configuration");
  console.log("─".repeat(60));
  console.log("Network:        ", network);
  console.log("Chain ID:       ", chainId.toString());
  console.log("Deployer:       ", deployer.address);

  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("Deployer Balance:", hre.ethers.formatEther(balance), "ETH");

  // Get initial owner (optional)
  const initialOwner = process.env.INITIAL_OWNER || deployer.address;
  if (initialOwner !== deployer.address) {
    console.log("Initial Owner:  ", initialOwner);
  }
  console.log();

  // Check if we have enough balance
  const minimumBalance = hre.ethers.parseEther("0.05"); // 0.05 ETH should be enough
  if (balance < minimumBalance) {
    console.error("❌ Insufficient balance for deployment");
    console.error(`   Required: ${hre.ethers.formatEther(minimumBalance)} ETH`);
    console.error(`   Available: ${hre.ethers.formatEther(balance)} ETH`);
    process.exit(1);
  }

  // Estimate gas cost
  console.log("📊 Estimating deployment cost...");
  const SageRegistryV4 = await hre.ethers.getContractFactory("SageRegistryV4");
  const deploymentData = SageRegistryV4.getDeployTransaction();

  try {
    const gasEstimate = await hre.ethers.provider.estimateGas({
      data: deploymentData.data,
      from: deployer.address
    });

    const feeData = await hre.ethers.provider.getFeeData();
    const estimatedCost = gasEstimate * (feeData.gasPrice || feeData.maxFeePerGas || 0n);

    console.log("Gas Estimate:   ", gasEstimate.toString());
    console.log("Estimated Cost: ", hre.ethers.formatEther(estimatedCost), "ETH");
    console.log();
  } catch (error) {
    console.log("⚠️  Could not estimate gas (continuing anyway)");
    console.log();
  }

  // Confirmation prompt for non-local networks
  if (network !== "localhost" && network !== "hardhat") {
    console.log("⚠️  You are about to deploy to", network.toUpperCase());
    console.log("   This will cost real funds.");
    console.log();
    console.log("Press Ctrl+C to cancel, or wait 5 seconds to continue...");

    await new Promise(resolve => setTimeout(resolve, 5000));
    console.log();
  }

  // Deploy contract
  console.log("🚀 Deploying SageRegistryV4...");
  const startTime = Date.now();

  const sageRegistry = await SageRegistryV4.deploy();
  await sageRegistry.waitForDeployment();

  const deployTime = ((Date.now() - startTime) / 1000).toFixed(2);
  const registryAddress = await sageRegistry.getAddress();

  console.log("✅ SageRegistryV4 deployed successfully!");
  console.log("   Address:", registryAddress);
  console.log("   Time:   ", deployTime, "seconds");
  console.log();

  // Get deployment transaction details
  const deployTx = sageRegistry.deploymentTransaction();
  if (deployTx) {
    const receipt = await deployTx.wait();
    console.log("📝 Transaction Details");
    console.log("─".repeat(60));
    console.log("Transaction Hash:", receipt.hash);
    console.log("Block Number:    ", receipt.blockNumber);
    console.log("Gas Used:        ", receipt.gasUsed.toString());
    console.log("Effective Gas Price:", hre.ethers.formatUnits(receipt.gasPrice || 0n, "gwei"), "Gwei");

    const actualCost = receipt.gasUsed * (receipt.gasPrice || 0n);
    console.log("Actual Cost:     ", hre.ethers.formatEther(actualCost), "ETH");
    console.log();
  }

  // Transfer ownership if needed
  if (initialOwner !== deployer.address) {
    console.log("🔐 Transferring ownership to", initialOwner);
    const transferTx = await sageRegistry.transferOwnership(initialOwner);
    await transferTx.wait();
    console.log("✅ Ownership transferred");
    console.log();
  }

  // Verify contract owner
  const contractOwner = await sageRegistry.owner();
  console.log("Contract Owner:  ", contractOwner);
  console.log();

  // Save deployment information
  const deploymentInfo = {
    network: network,
    chainId: chainId.toString(),
    contractAddress: registryAddress,
    deployer: deployer.address,
    owner: contractOwner,
    transactionHash: deployTx?.hash,
    blockNumber: deployTx ? (await deployTx.wait()).blockNumber : null,
    timestamp: new Date().toISOString(),
    compiler: {
      version: hre.config.solidity.compilers[0].version,
      settings: hre.config.solidity.compilers[0].settings
    }
  };

  const deploymentFile = process.env.DEPLOYED_ADDRESSES_FILE ||
                         path.join(__dirname, "..", "deployments", `v4_${network}.json`);

  // Ensure deployments directory exists
  const deploymentsDir = path.dirname(deploymentFile);
  if (!fs.existsSync(deploymentsDir)) {
    fs.mkdirSync(deploymentsDir, { recursive: true });
  }

  fs.writeFileSync(deploymentFile, JSON.stringify(deploymentInfo, null, 2));
  console.log("💾 Deployment info saved to:", deploymentFile);
  console.log();

  // Display next steps
  console.log("╔═══════════════════════════════════════════════════════════╗");
  console.log("║     Deployment Complete!                                  ║");
  console.log("╚═══════════════════════════════════════════════════════════╝\n");

  console.log("📋 Next Steps:");
  console.log();
  console.log("1. Verify contract on Etherscan:");
  console.log("   npx hardhat run scripts/verify_v4.js --network", network);
  console.log();
  console.log("2. Update DEPLOYED_ADDRESSES.md:");
  console.log("   Add the following information:");
  console.log("   Network:", network);
  console.log("   Address:", registryAddress);
  console.log("   Chain ID:", chainId.toString());
  console.log();
  console.log("3. Test the deployment:");
  console.log("   REGISTRY_ADDRESS=" + registryAddress, "go test ./pkg/agent/did/ethereum -v");
  console.log();
  console.log("4. Configure Go SDK:");
  console.log("   Update default contract addresses in:");
  console.log("   - cmd/sage-did/register.go");
  console.log("   - pkg/agent/did/ethereum/client.go");
  console.log();

  // Display explorer link
  let explorerUrl = "";
  switch (network) {
    case "mainnet":
      explorerUrl = `https://etherscan.io/address/${registryAddress}`;
      break;
    case "sepolia":
      explorerUrl = `https://sepolia.etherscan.io/address/${registryAddress}`;
      break;
    case "goerli":
      explorerUrl = `https://goerli.etherscan.io/address/${registryAddress}`;
      break;
    default:
      explorerUrl = null;
  }

  if (explorerUrl) {
    console.log("🔍 View on Explorer:");
    console.log("   " + explorerUrl);
    console.log();
  }

  return {
    address: registryAddress,
    deploymentInfo: deploymentInfo
  };
}

// Execute deployment
if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error("\n❌ Deployment failed:", error);
      process.exit(1);
    });
}

module.exports = { main };
