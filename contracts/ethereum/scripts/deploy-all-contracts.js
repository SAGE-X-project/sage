import hre from "hardhat";
import { network } from "hardhat";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const { ethers } = await network.connect();

/**
 * SAGE Complete Contract Deployment Script
 *
 * Deploys all SAGE contracts to the specified network:
 * 1. Governance Contracts (SimpleMultiSig, TEEKeyRegistry)
 * 2. ERC-8004 Registries (Identity, Reputation, Validation)
 * 3. AgentCard System (Registry, Hook, Storage)
 *
 * Usage:
 *   npx hardhat run scripts/deploy-all-contracts.js --network localhost
 *   npm run deploy:all
 */

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
  console.log("\n=".repeat(80));
  console.log(" SAGE Complete Contract Deployment");
  console.log("=".repeat(80));

  const networkInfo = await ethers.provider.getNetwork();
  const networkName = hre.network.name || "localhost";
  const deploymentNetworkName = NETWORK_NAME_MAP[networkName] || networkName;

  console.log(`\n Network: ${networkName} (Chain ID: ${networkInfo.chainId})`);
  console.log(` Deployment ID: ${deploymentNetworkName}`);

  const [deployer] = await ethers.getSigners();
  console.log(` Deployer: ${deployer.address}`);

  const balance = await ethers.provider.getBalance(deployer.address);
  console.log(` Balance: ${ethers.formatEther(balance)} ETH`);

  if (balance === 0n) {
    console.error("\n Error: Deployer has no balance!");
    process.exit(1);
  }

  console.log("=".repeat(80));

  const deploymentInfo = {
    network: deploymentNetworkName,
    chainId: networkInfo.chainId.toString(),
    timestamp: Date.now(),
    deployedAt: new Date().toISOString(),
    deployer: deployer.address,
    contracts: {}
  };

  // ============================================
  // 1. GOVERNANCE CONTRACTS
  // ============================================
  console.log("\n [PHASE 1] Deploying Governance Contracts");
  console.log("-".repeat(80));

  // 1.1 SimpleMultiSig
  console.log("\n [1/7] Deploying SimpleMultiSig...");
  const owners = [deployer.address];
  const threshold = 1;

  const SimpleMultiSig = await ethers.getContractFactory("SimpleMultiSig");
  const multiSig = await SimpleMultiSig.deploy(owners, threshold);
  await multiSig.waitForDeployment();
  const multiSigAddress = await multiSig.getAddress();

  console.log(`    Address: ${multiSigAddress}`);
  const multiSigTx = multiSig.deploymentTransaction();
  const multiSigReceipt = await multiSigTx.wait();
  console.log(`    Gas: ${multiSigReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${multiSigReceipt.blockNumber}`);

  deploymentInfo.contracts.SimpleMultiSig = {
    address: multiSigAddress,
    blockNumber: multiSigReceipt.blockNumber,
    transactionHash: multiSigReceipt.hash,
    gasUsed: multiSigReceipt.gasUsed.toString(),
    constructorArgs: [owners, threshold]
  };

  // 1.2 TEEKeyRegistry
  console.log("\n [2/7] Deploying TEEKeyRegistry...");
  const TEEKeyRegistry = await ethers.getContractFactory("TEEKeyRegistry");
  const teeRegistry = await TEEKeyRegistry.deploy();
  await teeRegistry.waitForDeployment();
  const teeRegistryAddress = await teeRegistry.getAddress();

  console.log(`    Address: ${teeRegistryAddress}`);
  const teeRegistryTx = teeRegistry.deploymentTransaction();
  const teeRegistryReceipt = await teeRegistryTx.wait();
  console.log(`    Gas: ${teeRegistryReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${teeRegistryReceipt.blockNumber}`);

  deploymentInfo.contracts.TEEKeyRegistry = {
    address: teeRegistryAddress,
    blockNumber: teeRegistryReceipt.blockNumber,
    transactionHash: teeRegistryReceipt.hash,
    gasUsed: teeRegistryReceipt.gasUsed.toString(),
    constructorArgs: []
  };

  // ============================================
  // 2. ERC-8004 REGISTRIES
  // ============================================
  console.log("\n [PHASE 2] Deploying ERC-8004 Registries");
  console.log("-".repeat(80));

  // 2.1 ERC8004IdentityRegistry
  console.log("\n [3/7] Deploying ERC8004IdentityRegistry...");
  const ERC8004IdentityRegistry = await ethers.getContractFactory("ERC8004IdentityRegistry");
  const identityRegistry = await ERC8004IdentityRegistry.deploy();
  await identityRegistry.waitForDeployment();
  const identityRegistryAddress = await identityRegistry.getAddress();

  console.log(`    Address: ${identityRegistryAddress}`);
  const identityRegistryTx = identityRegistry.deploymentTransaction();
  const identityRegistryReceipt = await identityRegistryTx.wait();
  console.log(`    Gas: ${identityRegistryReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${identityRegistryReceipt.blockNumber}`);

  deploymentInfo.contracts.ERC8004IdentityRegistry = {
    address: identityRegistryAddress,
    blockNumber: identityRegistryReceipt.blockNumber,
    transactionHash: identityRegistryReceipt.hash,
    gasUsed: identityRegistryReceipt.gasUsed.toString(),
    constructorArgs: []
  };

  // 2.2 ERC8004ReputationRegistry
  console.log("\n [4/7] Deploying ERC8004ReputationRegistry...");
  const ERC8004ReputationRegistry = await ethers.getContractFactory("ERC8004ReputationRegistry");
  const reputationRegistry = await ERC8004ReputationRegistry.deploy();
  await reputationRegistry.waitForDeployment();
  const reputationRegistryAddress = await reputationRegistry.getAddress();

  console.log(`    Address: ${reputationRegistryAddress}`);
  const reputationRegistryTx = reputationRegistry.deploymentTransaction();
  const reputationRegistryReceipt = await reputationRegistryTx.wait();
  console.log(`    Gas: ${reputationRegistryReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${reputationRegistryReceipt.blockNumber}`);

  deploymentInfo.contracts.ERC8004ReputationRegistry = {
    address: reputationRegistryAddress,
    blockNumber: reputationRegistryReceipt.blockNumber,
    transactionHash: reputationRegistryReceipt.hash,
    gasUsed: reputationRegistryReceipt.gasUsed.toString(),
    constructorArgs: []
  };

  // 2.3 ERC8004ValidationRegistry
  console.log("\n [5/7] Deploying ERC8004ValidationRegistry...");
  const ERC8004ValidationRegistry = await ethers.getContractFactory("ERC8004ValidationRegistry");
  const validationRegistry = await ERC8004ValidationRegistry.deploy();
  await validationRegistry.waitForDeployment();
  const validationRegistryAddress = await validationRegistry.getAddress();

  console.log(`    Address: ${validationRegistryAddress}`);
  const validationRegistryTx = validationRegistry.deploymentTransaction();
  const validationRegistryReceipt = await validationRegistryTx.wait();
  console.log(`    Gas: ${validationRegistryReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${validationRegistryReceipt.blockNumber}`);

  deploymentInfo.contracts.ERC8004ValidationRegistry = {
    address: validationRegistryAddress,
    blockNumber: validationRegistryReceipt.blockNumber,
    transactionHash: validationRegistryReceipt.hash,
    gasUsed: validationRegistryReceipt.gasUsed.toString(),
    constructorArgs: []
  };

  // ============================================
  // 3. AGENTCARD SYSTEM
  // ============================================
  console.log("\n [PHASE 3] Deploying AgentCard System");
  console.log("-".repeat(80));

  // 3.1 AgentCardVerifyHook
  console.log("\n [6/7] Deploying AgentCardVerifyHook...");
  const AgentCardVerifyHook = await ethers.getContractFactory("AgentCardVerifyHook");
  const hook = await AgentCardVerifyHook.deploy();
  await hook.waitForDeployment();
  const hookAddress = await hook.getAddress();

  console.log(`    Address: ${hookAddress}`);
  const hookTx = hook.deploymentTransaction();
  const hookReceipt = await hookTx.wait();
  console.log(`    Gas: ${hookReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${hookReceipt.blockNumber}`);

  deploymentInfo.contracts.AgentCardVerifyHook = {
    address: hookAddress,
    blockNumber: hookReceipt.blockNumber,
    transactionHash: hookReceipt.hash,
    gasUsed: hookReceipt.gasUsed.toString(),
    constructorArgs: []
  };

  // 3.2 AgentCardRegistry
  console.log("\n [7/7] Deploying AgentCardRegistry...");
  const AgentCardRegistry = await ethers.getContractFactory("AgentCardRegistry");
  const registry = await AgentCardRegistry.deploy(hookAddress);
  await registry.waitForDeployment();
  const registryAddress = await registry.getAddress();

  console.log(`    Address: ${registryAddress}`);
  const registryTx = registry.deploymentTransaction();
  const registryReceipt = await registryTx.wait();
  console.log(`    Gas: ${registryReceipt.gasUsed.toString()}`);
  console.log(`    Block: ${registryReceipt.blockNumber}`);

  deploymentInfo.contracts.AgentCardRegistry = {
    address: registryAddress,
    blockNumber: registryReceipt.blockNumber,
    transactionHash: registryReceipt.hash,
    gasUsed: registryReceipt.gasUsed.toString(),
    constructorArgs: [hookAddress]
  };

  // ============================================
  // SAVE DEPLOYMENT INFO
  // ============================================
  const deploymentPath = path.join(__dirname, "../deployments");
  if (!fs.existsSync(deploymentPath)) {
    fs.mkdirSync(deploymentPath, { recursive: true });
  }

  const timestamp = deploymentInfo.timestamp;
  const fileName = `${deploymentNetworkName}-complete-${timestamp}.json`;
  const filePath = path.join(deploymentPath, fileName);
  fs.writeFileSync(filePath, JSON.stringify(deploymentInfo, null, 2));

  const latestFileName = `${deploymentNetworkName}-complete-latest.json`;
  const latestFilePath = path.join(deploymentPath, latestFileName);
  fs.writeFileSync(latestFilePath, JSON.stringify(deploymentInfo, null, 2));

  // ============================================
  // SUMMARY
  // ============================================
  console.log("\n" + "=".repeat(80));
  console.log(" Deployment Complete!");
  console.log("=".repeat(80));

  console.log("\n Governance Contracts:");
  console.log(`   SimpleMultiSig:              ${multiSigAddress}`);
  console.log(`   TEEKeyRegistry:              ${teeRegistryAddress}`);

  console.log("\n ERC-8004 Registries:");
  console.log(`   IdentityRegistry:            ${identityRegistryAddress}`);
  console.log(`   ReputationRegistry:          ${reputationRegistryAddress}`);
  console.log(`   ValidationRegistry:          ${validationRegistryAddress}`);

  console.log("\n AgentCard System:");
  console.log(`   AgentCardRegistry:           ${registryAddress}`);
  console.log(`   AgentCardVerifyHook:         ${hookAddress}`);

  const totalGas =
    multiSigReceipt.gasUsed +
    teeRegistryReceipt.gasUsed +
    identityRegistryReceipt.gasUsed +
    reputationRegistryReceipt.gasUsed +
    validationRegistryReceipt.gasUsed +
    hookReceipt.gasUsed +
    registryReceipt.gasUsed;

  console.log(`\n Total Gas Used:                ${totalGas.toString()}`);
  console.log(`\n Deployment Files:`);
  console.log(`   Timestamped: deployments/${fileName}`);
  console.log(`   Latest:      deployments/${latestFileName}`);

  console.log("\n Next Steps:");
  console.log("   1. Verify contracts on block explorer");
  console.log("   2. Run integration tests");
  console.log("   3. Configure contract interactions");
  console.log("\n");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("\n Deployment failed:");
    console.error(error);
    process.exit(1);
  });

export default main;
