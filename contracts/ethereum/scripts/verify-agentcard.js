const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

/**
 * AgentCard Contract Verification Script
 *
 * Verifies deployed AgentCard contracts on block explorers
 *
 * Usage:
 *   npx hardhat run scripts/verify-agentcard.js --network <network>
 *
 * Example:
 *   npx hardhat run scripts/verify-agentcard.js --network kairos
 *
 * Prerequisites:
 * - Contracts must be deployed (deploy-agentcard.js)
 * - API keys configured in .env (ETHERSCAN_API_KEY, etc.)
 */

// Network name mapping (same as deploy script)
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
  console.log("\n AgentCard Contract Verification");
  console.log("=".repeat(80));

  const networkName = hre.network.name;
  const deploymentNetworkName = NETWORK_NAME_MAP[networkName] || networkName;

  console.log(` Network: ${networkName}`);
  console.log(` Deployment ID: ${deploymentNetworkName}`);

  // Skip verification for local networks
  if (networkName === 'localhost' || networkName === 'hardhat') {
    console.log("\n  Skipping verification for local network");
    console.log("   Local networks don't have block explorers");
    process.exit(0);
  }

  // Load deployment info
  const deploymentPath = path.join(__dirname, "../deployments");
  const latestFileName = `${deploymentNetworkName}-latest.json`;
  const latestFilePath = path.join(deploymentPath, latestFileName);

  if (!fs.existsSync(latestFilePath)) {
    console.error(`\n Error: Deployment file not found: ${latestFileName}`);
    console.error("   Please deploy contracts first using:");
    console.error(`   npx hardhat run scripts/deploy-agentcard.js --network ${networkName}`);
    process.exit(1);
  }

  const deploymentInfo = JSON.parse(fs.readFileSync(latestFilePath, 'utf8'));
  console.log(` Loaded deployment from: ${latestFileName}`);
  console.log(`   Deployed at: ${deploymentInfo.deployedAt}`);

  console.log("=".repeat(80));

  // Verify AgentCardRegistry
  console.log("\n [1/2] Verifying AgentCardRegistry...");
  const registryAddress = deploymentInfo.contracts.AgentCardRegistry.address;
  const hookAddress = deploymentInfo.contracts.AgentCardVerifyHook.address;
  console.log(`   Address: ${registryAddress}`);
  console.log(`   Constructor arg (hook): ${hookAddress}`);

  try {
    await hre.run("verify:verify", {
      address: registryAddress,
      constructorArguments: [hookAddress],
      contract: "contracts/AgentCardRegistry.sol:AgentCardRegistry"
    });

    console.log("    AgentCardRegistry verified!");
    deploymentInfo.contracts.AgentCardRegistry.verified = true;
  } catch (error) {
    if (error.message.includes("Already Verified")) {
      console.log("   ℹ  Already verified");
      deploymentInfo.contracts.AgentCardRegistry.verified = true;
    } else {
      console.error("    Verification failed:");
      console.error("   " + error.message);
      deploymentInfo.contracts.AgentCardRegistry.verificationError = error.message;
    }
  }

  // Verify AgentCardVerifyHook
  console.log("\n [2/2] Verifying AgentCardVerifyHook...");
  console.log(`   Address: ${hookAddress}`);

  try {
    await hre.run("verify:verify", {
      address: hookAddress,
      constructorArguments: [],
      contract: "contracts/AgentCardVerifyHook.sol:AgentCardVerifyHook"
    });

    console.log("    AgentCardVerifyHook verified!");
    deploymentInfo.contracts.AgentCardVerifyHook.verified = true;
  } catch (error) {
    if (error.message.includes("Already Verified")) {
      console.log("   ℹ  Already verified");
      deploymentInfo.contracts.AgentCardVerifyHook.verified = true;
    } else {
      console.error("    Verification failed:");
      console.error("   " + error.message);
      deploymentInfo.contracts.AgentCardVerifyHook.verificationError = error.message;
    }
  }

  // Update deployment file with verification status
  deploymentInfo.verifiedAt = new Date().toISOString();
  fs.writeFileSync(latestFilePath, JSON.stringify(deploymentInfo, null, 2));

  // Also update the timestamped file if it exists
  const timestampedFileName = `${deploymentNetworkName}-agentcard-${deploymentInfo.timestamp}.json`;
  const timestampedFilePath = path.join(deploymentPath, timestampedFileName);
  if (fs.existsSync(timestampedFilePath)) {
    fs.writeFileSync(timestampedFilePath, JSON.stringify(deploymentInfo, null, 2));
  }

  // Print summary
  console.log("\n" + "=".repeat(80));
  console.log(" Verification Complete!");
  console.log("=".repeat(80));

  const registryVerified = deploymentInfo.contracts.AgentCardRegistry.verified;
  const hookVerified = deploymentInfo.contracts.AgentCardVerifyHook.verified;

  console.log("\n Summary:");
  console.log(`   Network:              ${deploymentNetworkName}`);
  console.log(`   AgentCardRegistry:    ${registryVerified ? ' Verified' : ' Failed'}`);
  console.log(`   AgentCardVerifyHook:  ${hookVerified ? ' Verified' : ' Failed'}`);

  if (registryVerified && hookVerified) {
    console.log("\n All contracts verified successfully!");
    console.log("\n Block Explorer URLs:");
    const explorerUrls = getExplorerUrls(deploymentNetworkName, registryAddress, hookAddress);
    console.log(`   Registry:  ${explorerUrls.registry}`);
    console.log(`   Hook:      ${explorerUrls.hook}`);
  } else {
    console.log("\n  Some contracts failed verification");
    console.log("   Check the errors above and ensure:");
    console.log("   - Correct API key is configured in .env");
    console.log("   - Network configuration is correct in hardhat.config.js");
    console.log("   - Sufficient time has passed since deployment");
  }

  console.log("\n");
}

/**
 * Get block explorer URLs for the verified contracts
 */
function getExplorerUrls(network, registryAddress, hookAddress) {
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
    registry: baseUrl + registryAddress,
    hook: baseUrl + hookAddress
  };
}

// Execute verification
if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error("\n Verification failed:");
      console.error(error);
      process.exit(1);
    });
}

module.exports = main;
