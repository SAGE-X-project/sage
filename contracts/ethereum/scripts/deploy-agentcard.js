import hre from "hardhat";
import { network } from "hardhat";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Initialize ethers from network connection (Hardhat 3.x pattern)
const { ethers } = await network.connect();

/**
 * AgentCard Multi-Chain Deployment Script
 *
 * Deploys AgentCardRegistry and AgentCardVerifyHook to any supported network
 *
 * Supported Networks:
 * - Ethereum: mainnet, sepolia
 * - Kaia: cypress (mainnet), kairos (testnet)
 * - BSC: mainnet, testnet
 * - Base: mainnet, sepolia
 * - Arbitrum: mainnet, sepolia
 * - Optimism: mainnet, sepolia
 *
 * Usage:
 *   npx hardhat run scripts/deploy-agentcard.js --network <network>
 *
 * Example:
 *   npx hardhat run scripts/deploy-agentcard.js --network kairos
 */

// Network name mapping for deployment files
const NETWORK_NAME_MAP = {
  // Ethereum
  mainnet: 'ethereum-mainnet',
  sepolia: 'ethereum-sepolia',

  // Kaia
  cypress: 'kaia-mainnet',
  kaia: 'kaia-mainnet',
  kairos: 'kaia-kairos',

  // BSC
  bsc: 'bsc-mainnet',
  bscTestnet: 'bsc-testnet',

  // Base
  base: 'base-mainnet',
  baseSepolia: 'base-sepolia',

  // Arbitrum
  arbitrumOne: 'arbitrum-mainnet',
  arbitrum: 'arbitrum-mainnet',
  arbitrumSepolia: 'arbitrum-sepolia',

  // Optimism
  optimism: 'optimism-mainnet',
  optimisticEthereum: 'optimism-mainnet',
  optimismSepolia: 'optimism-sepolia',

  // Local
  localhost: 'localhost',
  hardhat: 'hardhat'
};

async function main() {
  console.log("\n🚀 AgentCard Multi-Chain Deployment");
  console.log("=".repeat(80));

  // Get network info
  const networkInfo = await ethers.provider.getNetwork();
  const networkName = hre.network.name || "localhost";
  const deploymentNetworkName = NETWORK_NAME_MAP[networkName] || networkName;

  console.log(`📍 Network: ${networkName} (Chain ID: ${networkInfo.chainId})`);
  console.log(`📝 Deployment ID: ${deploymentNetworkName}`);

  // Get deployer account
  const [deployer] = await ethers.getSigners();
  console.log(`👤 Deployer: ${deployer.address}`);

  const balance = await ethers.provider.getBalance(deployer.address);
  console.log(`💰 Balance: ${ethers.formatEther(balance)} ETH`);

  if (balance === 0n) {
    console.error("\n❌ Error: Deployer has no balance!");
    process.exit(1);
  }

  console.log("=".repeat(80));

  // Deploy AgentCardVerifyHook first (required for Registry constructor)
  console.log("\n📦 [1/2] Deploying AgentCardVerifyHook...");
  const AgentCardVerifyHook = await ethers.getContractFactory("AgentCardVerifyHook");
  const hook = await AgentCardVerifyHook.deploy();
  await hook.waitForDeployment();
  const hookAddress = await hook.getAddress();

  console.log(`   ✅ AgentCardVerifyHook: ${hookAddress}`);

  const hookDeployTx = hook.deploymentTransaction();
  const hookReceipt = await hookDeployTx.wait();
  console.log(`   📊 Gas used: ${hookReceipt.gasUsed.toString()}`);
  console.log(`   🔗 Block: ${hookReceipt.blockNumber}`);
  console.log(`   📜 Tx: ${hookReceipt.hash}`);

  // Deploy AgentCardRegistry with hook address
  console.log("\n📦 [2/2] Deploying AgentCardRegistry...");
  const AgentCardRegistry = await ethers.getContractFactory("AgentCardRegistry");
  const registry = await AgentCardRegistry.deploy(hookAddress);
  await registry.waitForDeployment();
  const registryAddress = await registry.getAddress();

  console.log(`   ✅ AgentCardRegistry: ${registryAddress}`);

  // Get deployment transaction info
  const registryDeployTx = registry.deploymentTransaction();
  const registryReceipt = await registryDeployTx.wait();
  console.log(`   📊 Gas used: ${registryReceipt.gasUsed.toString()}`);
  console.log(`   🔗 Block: ${registryReceipt.blockNumber}`);
  console.log(`   📜 Tx: ${registryReceipt.hash}`);

  console.log("\n✅ Hook configured in constructor");

  // Prepare deployment info
  const timestamp = Date.now();
  const deploymentInfo = {
    network: deploymentNetworkName,
    chainId: networkInfo.chainId.toString(),
    timestamp: timestamp,
    deployedAt: new Date(timestamp).toISOString(),
    contracts: {
      AgentCardRegistry: {
        address: registryAddress,
        deployer: deployer.address,
        blockNumber: registryReceipt.blockNumber,
        transactionHash: registryReceipt.hash,
        gasUsed: registryReceipt.gasUsed.toString(),
        constructorArgs: [hookAddress],
        verified: false
      },
      AgentCardVerifyHook: {
        address: hookAddress,
        deployer: deployer.address,
        blockNumber: hookReceipt.blockNumber,
        transactionHash: hookReceipt.hash,
        gasUsed: hookReceipt.gasUsed.toString(),
        constructorArgs: [],
        verified: false
      }
    },
    configuration: {
      hookConfigured: true,
      hookAddress: hookAddress
    },
    features: {
      erc8004Compliant: true,
      commitRevealPattern: true,
      multiKeySupport: true,
      maxKeysPerAgent: 10,
      supportedKeyTypes: ["ECDSA", "Ed25519", "X25519"]
    }
  };

  // Save deployment info
  const deploymentPath = path.join(__dirname, "../deployments");
  if (!fs.existsSync(deploymentPath)) {
    fs.mkdirSync(deploymentPath, { recursive: true });
  }

  // Save timestamped deployment file
  const fileName = `${deploymentNetworkName}-agentcard-${timestamp}.json`;
  const filePath = path.join(deploymentPath, fileName);
  fs.writeFileSync(filePath, JSON.stringify(deploymentInfo, null, 2));
  console.log(`\n💾 Deployment info saved: deployments/${fileName}`);

  // Save latest deployment file
  const latestFileName = `${deploymentNetworkName}-latest.json`;
  const latestFilePath = path.join(deploymentPath, latestFileName);
  fs.writeFileSync(latestFilePath, JSON.stringify(deploymentInfo, null, 2));
  console.log(`💾 Latest deployment: deployments/${latestFileName}`);

  // Print summary
  console.log("\n" + "=".repeat(80));
  console.log("✅ Deployment Complete!");
  console.log("=".repeat(80));
  console.log("\n📋 Summary:");
  console.log(`   Network:              ${deploymentNetworkName}`);
  console.log(`   Chain ID:             ${networkInfo.chainId}`);
  console.log(`   AgentCardRegistry:    ${registryAddress}`);
  console.log(`   AgentCardVerifyHook:  ${hookAddress}`);
  console.log(`   Total Gas Used:       ${(registryReceipt.gasUsed + hookReceipt.gasUsed).toString()}`);

  console.log("\n📝 Next Steps:");
  console.log(`   1. Verify contracts on block explorer`);
  console.log(`   2. Run: npx hardhat run scripts/verify-agentcard.js --network ${networkName}`);
  console.log(`   3. Test agent registration`);

  if (networkName !== 'localhost' && networkName !== 'hardhat') {
    console.log("\n🔍 Block Explorer URLs:");
    const explorerUrls = getExplorerUrls(deploymentNetworkName, registryAddress, hookAddress);
    console.log(`   Registry:  ${explorerUrls.registry}`);
    console.log(`   Hook:      ${explorerUrls.hook}`);
  }

  console.log("\n");
}

/**
 * Get block explorer URLs for the deployed contracts
 */
function getExplorerUrls(network, registryAddress, hookAddress) {
  const explorers = {
    // Ethereum
    'ethereum-mainnet': 'https://etherscan.io/address/',
    'ethereum-sepolia': 'https://sepolia.etherscan.io/address/',

    // Kaia
    'kaia-mainnet': 'https://kaiascan.io/account/',
    'kaia-kairos': 'https://kairos.kaiascan.io/account/',

    // BSC
    'bsc-mainnet': 'https://bscscan.com/address/',
    'bsc-testnet': 'https://testnet.bscscan.com/address/',

    // Base
    'base-mainnet': 'https://basescan.org/address/',
    'base-sepolia': 'https://sepolia.basescan.org/address/',

    // Arbitrum
    'arbitrum-mainnet': 'https://arbiscan.io/address/',
    'arbitrum-sepolia': 'https://sepolia.arbiscan.io/address/',

    // Optimism
    'optimism-mainnet': 'https://optimistic.etherscan.io/address/',
    'optimism-sepolia': 'https://sepolia-optimistic.etherscan.io/address/'
  };

  const baseUrl = explorers[network] || 'Unknown explorer';

  return {
    registry: baseUrl + registryAddress,
    hook: baseUrl + hookAddress
  };
}

// Execute deployment
main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("\n❌ Deployment failed:");
    console.error(error);
    process.exit(1);
  });

export default main;
