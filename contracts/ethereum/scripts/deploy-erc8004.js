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
 * ERC-8004 Multi-Contract Deployment Script
 *
 * Deploys ERC-8004 standalone contracts:
 * - ERC8004IdentityRegistry
 * - ERC8004ReputationRegistry
 * - ERC8004ValidationRegistry
 *
 * Usage:
 *   npx hardhat run scripts/deploy-erc8004.js --network <network>
 */

// Network name mapping for deployment files
const NETWORK_NAME_MAP = {
  mainnet: 'ethereum-mainnet',
  sepolia: 'ethereum-sepolia',
  cypress: 'kaia-mainnet',
  kaia: 'kaia-mainnet',
  kairos: 'kaia-kairos',
  bsc: 'bsc-mainnet',
  bscTestnet: 'bsc-testnet',
  base: 'base-mainnet',
  baseSepolia: 'base-sepolia',
  arbitrumOne: 'arbitrum-mainnet',
  arbitrum: 'arbitrum-mainnet',
  arbitrumSepolia: 'arbitrum-sepolia',
  optimism: 'optimism-mainnet',
  optimisticEthereum: 'optimism-mainnet',
  optimismSepolia: 'optimism-sepolia',
  localhost: 'localhost',
  hardhat: 'hardhat'
};

async function main() {
  console.log("\n ERC-8004 Standalone Contracts Deployment");
  console.log("=".repeat(80));

  // Get network info
  const networkInfo = await ethers.provider.getNetwork();
  const networkName = hre.network.name || "localhost";
  const deploymentNetworkName = NETWORK_NAME_MAP[networkName] || networkName;

  console.log(` Network: ${networkName} (Chain ID: ${networkInfo.chainId})`);
  console.log(` Deployment ID: ${deploymentNetworkName}`);

  // Get deployer account
  const [deployer] = await ethers.getSigners();
  console.log(` Deployer: ${deployer.address}`);

  const balance = await ethers.provider.getBalance(deployer.address);
  console.log(` Balance: ${ethers.formatEther(balance)} ETH`);

  if (balance === 0n) {
    console.error("\n Error: Deployer has no balance!");
    process.exit(1);
  }

  console.log("=".repeat(80));

  const deployedContracts = {};

  // Deploy contracts in correct order:
  // 1. ValidationRegistry (no dependencies)
  // 2. ReputationRegistry (depends on ValidationRegistry)
  // 3. IdentityRegistry (no dependencies)

  // Deploy ERC8004ValidationRegistry first
  console.log("\n [1/3] Deploying ERC8004ValidationRegistry...");
  const minStake = ethers.parseEther("0.01"); // 0.01 ETH minimum stake
  const minValidators = 3; // Minimum 3 validators
  const consensusThreshold = 66; // 66% consensus threshold

  console.log(`    Constructor args: minStake=${ethers.formatEther(minStake)} ETH, minValidators=${minValidators}, consensusThreshold=${consensusThreshold}%`);

  const ValidationRegistry = await ethers.getContractFactory("ERC8004ValidationRegistry");
  const validationRegistry = await ValidationRegistry.deploy(minStake, minValidators, consensusThreshold);
  await validationRegistry.waitForDeployment();
  const validationAddress = await validationRegistry.getAddress();

  console.log(`    ERC8004ValidationRegistry: ${validationAddress}`);

  const validationDeployTx = validationRegistry.deploymentTransaction();
  const validationReceipt = await validationDeployTx.wait();
  console.log(`    Gas used: ${validationReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${validationReceipt.blockNumber}`);
  console.log(`    Tx: ${validationReceipt.hash}`);

  deployedContracts.ERC8004ValidationRegistry = {
    address: validationAddress,
    deployer: deployer.address,
    blockNumber: validationReceipt.blockNumber,
    transactionHash: validationReceipt.hash,
    gasUsed: validationReceipt.gasUsed.toString(),
    constructorArgs: [minStake.toString(), minValidators, consensusThreshold],
    verified: false
  };

  // Deploy ERC8004ReputationRegistry with ValidationRegistry address
  console.log("\n [2/3] Deploying ERC8004ReputationRegistry...");
  console.log(`    Constructor args: validationRegistry=${validationAddress}`);

  const ReputationRegistry = await ethers.getContractFactory("ERC8004ReputationRegistry");
  const reputationRegistry = await ReputationRegistry.deploy(validationAddress);
  await reputationRegistry.waitForDeployment();
  const reputationAddress = await reputationRegistry.getAddress();

  console.log(`    ERC8004ReputationRegistry: ${reputationAddress}`);

  const reputationDeployTx = reputationRegistry.deploymentTransaction();
  const reputationReceipt = await reputationDeployTx.wait();
  console.log(`    Gas used: ${reputationReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${reputationReceipt.blockNumber}`);
  console.log(`    Tx: ${reputationReceipt.hash}`);

  deployedContracts.ERC8004ReputationRegistry = {
    address: reputationAddress,
    deployer: deployer.address,
    blockNumber: reputationReceipt.blockNumber,
    transactionHash: reputationReceipt.hash,
    gasUsed: reputationReceipt.gasUsed.toString(),
    constructorArgs: [validationAddress],
    verified: false
  };

  // Deploy ERC8004IdentityRegistry (no constructor args)
  console.log("\n [3/3] Deploying ERC8004IdentityRegistry...");
  const IdentityRegistry = await ethers.getContractFactory("ERC8004IdentityRegistry");
  const identityRegistry = await IdentityRegistry.deploy();
  await identityRegistry.waitForDeployment();
  const identityAddress = await identityRegistry.getAddress();

  console.log(`    ERC8004IdentityRegistry: ${identityAddress}`);

  const identityDeployTx = identityRegistry.deploymentTransaction();
  const identityReceipt = await identityDeployTx.wait();
  console.log(`    Gas used: ${identityReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${identityReceipt.blockNumber}`);
  console.log(`    Tx: ${identityReceipt.hash}`);

  deployedContracts.ERC8004IdentityRegistry = {
    address: identityAddress,
    deployer: deployer.address,
    blockNumber: identityReceipt.blockNumber,
    transactionHash: identityReceipt.hash,
    gasUsed: identityReceipt.gasUsed.toString(),
    constructorArgs: [],
    verified: false
  };

  // Prepare deployment info
  const timestamp = Date.now();
  const deploymentInfo = {
    network: deploymentNetworkName,
    chainId: networkInfo.chainId.toString(),
    timestamp: timestamp,
    deployedAt: new Date(timestamp).toISOString(),
    contracts: deployedContracts,
    standard: "ERC-8004",
    version: "1.0.0",
    features: {
      identityRegistry: true,
      reputationRegistry: true,
      validationRegistry: true,
      standalone: true
    }
  };

  // Save deployment info
  const deploymentPath = path.join(__dirname, "../deployments");
  if (!fs.existsSync(deploymentPath)) {
    fs.mkdirSync(deploymentPath, { recursive: true });
  }

  // Save timestamped deployment file
  const fileName = `${deploymentNetworkName}-erc8004-${timestamp}.json`;
  const filePath = path.join(deploymentPath, fileName);
  fs.writeFileSync(filePath, JSON.stringify(deploymentInfo, null, 2));
  console.log(`\n Deployment info saved: deployments/${fileName}`);

  // Save latest deployment file
  const latestFileName = `${deploymentNetworkName}-erc8004-latest.json`;
  const latestFilePath = path.join(deploymentPath, latestFileName);
  fs.writeFileSync(latestFilePath, JSON.stringify(deploymentInfo, null, 2));
  console.log(` Latest deployment: deployments/${latestFileName}`);

  // Calculate total gas used
  const totalGas = BigInt(identityReceipt.gasUsed) +
                   BigInt(reputationReceipt.gasUsed) +
                   BigInt(validationReceipt.gasUsed);

  // Print summary
  console.log("\n" + "=".repeat(80));
  console.log(" Deployment Complete!");
  console.log("=".repeat(80));
  console.log("\n Summary:");
  console.log(`   Network:                      ${deploymentNetworkName}`);
  console.log(`   Chain ID:                     ${networkInfo.chainId}`);
  console.log(`   ERC8004IdentityRegistry:      ${identityAddress}`);
  console.log(`   ERC8004ReputationRegistry:    ${reputationAddress}`);
  console.log(`   ERC8004ValidationRegistry:    ${validationAddress}`);
  console.log(`   Total Gas Used:               ${totalGas.toString()}`);

  console.log("\n Next Steps:");
  console.log(`   1. Verify contracts on block explorer`);
  console.log(`   2. Run: npx hardhat run scripts/verify-erc8004.js --network ${networkName}`);
  console.log(`   3. Test agent registration`);

  if (networkName !== 'localhost' && networkName !== 'hardhat') {
    console.log("\n Block Explorer URLs:");
    const explorerUrls = getExplorerUrls(deploymentNetworkName, identityAddress, reputationAddress, validationAddress);
    console.log(`   Identity:    ${explorerUrls.identity}`);
    console.log(`   Reputation:  ${explorerUrls.reputation}`);
    console.log(`   Validation:  ${explorerUrls.validation}`);
  }

  console.log("\n");
}

/**
 * Get block explorer URLs for the deployed contracts
 */
function getExplorerUrls(network, identityAddress, reputationAddress, validationAddress) {
  const explorers = {
    'ethereum-mainnet': 'https://etherscan.io/address/',
    'ethereum-sepolia': 'https://sepolia.etherscan.io/address/',
    'kaia-mainnet': 'https://kaiascan.io/account/',
    'kaia-kairos': 'https://kairos.kaiascan.io/account/',
    'bsc-mainnet': 'https://bscscan.com/address/',
    'bsc-testnet': 'https://testnet.bscscan.com/address/',
    'base-mainnet': 'https://basescan.org/address/',
    'base-sepolia': 'https://sepolia.basescan.org/address/',
    'arbitrum-mainnet': 'https://arbiscan.io/address/',
    'arbitrum-sepolia': 'https://sepolia.arbiscan.io/address/',
    'optimism-mainnet': 'https://optimistic.etherscan.io/address/',
    'optimism-sepolia': 'https://sepolia-optimistic.etherscan.io/address/'
  };

  const baseUrl = explorers[network] || 'Unknown explorer';

  return {
    identity: baseUrl + identityAddress,
    reputation: baseUrl + reputationAddress,
    validation: baseUrl + validationAddress
  };
}

// Execute deployment
main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("\n Deployment failed:");
    console.error(error);
    process.exit(1);
  });

export default main;
