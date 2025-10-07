const hre = require("hardhat");
const fs = require('fs');

/**
 * Sepolia Testnet Deployment Script
 *
 * Deploys TWO complete systems:
 * 1. SAGE Core System (with ERC-8004 adapters)
 * 2. ERC-8004 Standalone System (independent)
 */

async function main() {
  console.log("\n🚀 SAGE Platform - Sepolia Testnet Deployment");
  console.log("=".repeat(80));

  // Get deployer
  const [deployer] = await hre.ethers.getSigners();

  console.log("\n📍 Network Information:");
  const network = await hre.ethers.provider.getNetwork();
  console.log("  Network:", network.name);
  console.log("  Chain ID:", network.chainId.toString());
  console.log("\n👤 Deployer Account:");
  console.log("  Address:", deployer.address);

  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("  Balance:", hre.ethers.formatEther(balance), "ETH");

  if (balance < hre.ethers.parseEther("0.3")) {
    console.log("\n⚠️  WARNING: Balance may be insufficient for deployment");
    console.log("   Recommended: 0.3 ETH minimum");
  }

  console.log("\n" + "=".repeat(80));

  // Track deployed contracts
  const deployed = {
    sage: {},
    erc8004Standalone: {}
  };
  let totalGasUsed = 0n;

  // ==========================================================================
  // SYSTEM 1: SAGE CORE CONTRACTS
  // ==========================================================================
  console.log("\n\n📦 SYSTEM 1: Deploying SAGE Core Contracts\n");

  // 1.1 Deploy SageRegistryV2 (Security Enhanced)
  console.log("1️⃣  Deploying SageRegistryV2 (Security Enhanced)...");
  const SageRegistryV2 = await hre.ethers.getContractFactory("SageRegistryV2");
  const sageRegistryV2 = await SageRegistryV2.deploy();
  await sageRegistryV2.waitForDeployment();
  deployed.sage.sageRegistryV2 = await sageRegistryV2.getAddress();
  console.log("   ✅ SageRegistryV2:", deployed.sage.sageRegistryV2);
  console.log("   📝 Features: ReentrancyGuard, Ownable2Step, Pausable, Hook Gas Limit");

  // Wait for block confirmation
  await new Promise(resolve => setTimeout(resolve, 5000));

  // 1.2 Deploy SageRegistryV3 (Commit-Reveal)
  console.log("\n2️⃣  Deploying SageRegistryV3 (Commit-Reveal)...");
  const SageRegistryV3 = await hre.ethers.getContractFactory("SageRegistryV3");
  const sageRegistryV3 = await SageRegistryV3.deploy();
  await sageRegistryV3.waitForDeployment();
  deployed.sage.sageRegistryV3 = await sageRegistryV3.getAddress();
  console.log("   ✅ SageRegistryV3:", deployed.sage.sageRegistryV3);
  console.log("   📝 Features: Front-running protection, Timing validation");

  await new Promise(resolve => setTimeout(resolve, 5000));

  // 1.3 Deploy SageVerificationHook
  console.log("\n3️⃣  Deploying SageVerificationHook...");
  const SageVerificationHook = await hre.ethers.getContractFactory("SageVerificationHook");
  const verificationHook = await SageVerificationHook.deploy();
  await verificationHook.waitForDeployment();
  deployed.sage.verificationHook = await verificationHook.getAddress();
  console.log("   ✅ SageVerificationHook:", deployed.sage.verificationHook);

  await new Promise(resolve => setTimeout(resolve, 5000));

  // 1.4 Deploy ERC8004IdentityRegistry (Adapter)
  console.log("\n4️⃣  Deploying ERC8004IdentityRegistry (Adapter for SAGE)...");
  const ERC8004IdentityRegistry = await hre.ethers.getContractFactory(
    "contracts/erc-8004/ERC8004IdentityRegistry.sol:ERC8004IdentityRegistry"
  );
  const identityRegistry = await ERC8004IdentityRegistry.deploy(deployed.sage.sageRegistryV2);
  await identityRegistry.waitForDeployment();
  deployed.sage.identityRegistry = await identityRegistry.getAddress();
  console.log("   ✅ ERC8004IdentityRegistry:", deployed.sage.identityRegistry);
  console.log("   📝 Features: O(1) deactivation, Adapter for SageRegistryV2");

  await new Promise(resolve => setTimeout(resolve, 5000));

  // 1.5 Deploy ERC8004ReputationRegistryV2 (with Commit-Reveal)
  console.log("\n5️⃣  Deploying ERC8004ReputationRegistryV2 (Commit-Reveal)...");
  const ERC8004ReputationRegistryV2 = await hre.ethers.getContractFactory("ERC8004ReputationRegistryV2");
  const reputationRegistryV2 = await ERC8004ReputationRegistryV2.deploy(deployed.sage.identityRegistry);
  await reputationRegistryV2.waitForDeployment();
  deployed.sage.reputationRegistryV2 = await reputationRegistryV2.getAddress();
  console.log("   ✅ ERC8004ReputationRegistryV2:", deployed.sage.reputationRegistryV2);
  console.log("   📝 Features: Task authorization commit-reveal, Deadline validation (1h-30d)");

  await new Promise(resolve => setTimeout(resolve, 5000));

  // 1.6 Deploy ERC8004ValidationRegistry (Security Enhanced)
  console.log("\n6️⃣  Deploying ERC8004ValidationRegistry (Security Enhanced)...");
  const ERC8004ValidationRegistry = await hre.ethers.getContractFactory(
    "contracts/erc-8004/ERC8004ValidationRegistry.sol:ERC8004ValidationRegistry"
  );
  const validationRegistry = await ERC8004ValidationRegistry.deploy(
    deployed.sage.identityRegistry,
    deployed.sage.reputationRegistryV2
  );
  await validationRegistry.waitForDeployment();
  deployed.sage.validationRegistry = await validationRegistry.getAddress();
  console.log("   ✅ ERC8004ValidationRegistry:", deployed.sage.validationRegistry);
  console.log("   📝 Features: ReentrancyGuard, Pull Payment, Expiry handling, Reputation staking");

  await new Promise(resolve => setTimeout(resolve, 5000));

  // 1.7 Configure Hooks on SageRegistryV2
  console.log("\n7️⃣  Configuring hooks on SageRegistryV2...");
  let tx = await sageRegistryV2.setBeforeRegisterHook(deployed.sage.verificationHook);
  let receipt = await tx.wait();
  totalGasUsed += receipt.gasUsed;
  console.log("   ✅ BeforeRegisterHook configured (Gas:", receipt.gasUsed.toString(), ")");

  tx = await sageRegistryV2.setAfterRegisterHook(deployed.sage.verificationHook);
  receipt = await tx.wait();
  totalGasUsed += receipt.gasUsed;
  console.log("   ✅ AfterRegisterHook configured (Gas:", receipt.gasUsed.toString(), ")");

  // 1.8 Link ValidationRegistry to ReputationRegistry
  console.log("\n8️⃣  Linking ValidationRegistry to ReputationRegistry...");
  tx = await reputationRegistryV2.setValidationRegistry(deployed.sage.validationRegistry);
  receipt = await tx.wait();
  totalGasUsed += receipt.gasUsed;
  console.log("   ✅ ValidationRegistry linked (Gas:", receipt.gasUsed.toString(), ")");

  // ==========================================================================
  // SYSTEM 2: ERC-8004 STANDALONE CONTRACTS
  // ==========================================================================
  console.log("\n\n📦 SYSTEM 2: Deploying ERC-8004 Standalone System\n");

  await new Promise(resolve => setTimeout(resolve, 5000));

  // 2.1 Deploy Standalone ERC8004IdentityRegistry
  console.log("9️⃣  Deploying Standalone ERC8004IdentityRegistry...");
  const StandaloneIdentity = await hre.ethers.getContractFactory(
    "contracts/erc-8004/standalone/ERC8004IdentityRegistry.sol:ERC8004IdentityRegistry"
  );
  const standaloneIdentity = await StandaloneIdentity.deploy();
  await standaloneIdentity.waitForDeployment();
  deployed.erc8004Standalone.identityRegistry = await standaloneIdentity.getAddress();
  console.log("   ✅ Standalone IdentityRegistry:", deployed.erc8004Standalone.identityRegistry);
  console.log("   📝 Zero SAGE dependencies - Fully independent");

  await new Promise(resolve => setTimeout(resolve, 5000));

  // 2.2 Deploy Standalone ERC8004ValidationRegistry (before Reputation due to constructor dependency)
  console.log("\n🔟 Deploying Standalone ERC8004ValidationRegistry...");
  const StandaloneValidation = await hre.ethers.getContractFactory(
    "contracts/erc-8004/standalone/ERC8004ValidationRegistry.sol:ERC8004ValidationRegistry"
  );
  // Standalone uses different constructor: (minStake, minValidators, consensusThreshold)
  const standaloneValidation = await StandaloneValidation.deploy(
    hre.ethers.parseEther("0.01"), // minStake: 0.01 ETH
    1, // minValidators: 1
    66 // consensusThreshold: 66%
  );
  await standaloneValidation.waitForDeployment();
  deployed.erc8004Standalone.validationRegistry = await standaloneValidation.getAddress();
  console.log("   ✅ Standalone ValidationRegistry:", deployed.erc8004Standalone.validationRegistry);
  console.log("   📝 Config: 0.01 ETH minStake, 1 minValidator, 66% consensus");

  await new Promise(resolve => setTimeout(resolve, 5000));

  // 2.3 Deploy Standalone ERC8004ReputationRegistry (with ValidationRegistry address)
  console.log("\n1️⃣1️⃣  Deploying Standalone ERC8004ReputationRegistry...");
  const StandaloneReputation = await hre.ethers.getContractFactory(
    "contracts/erc-8004/standalone/ERC8004ReputationRegistry.sol:ERC8004ReputationRegistry"
  );
  const standaloneReputation = await StandaloneReputation.deploy(deployed.erc8004Standalone.validationRegistry);
  await standaloneReputation.waitForDeployment();
  deployed.erc8004Standalone.reputationRegistry = await standaloneReputation.getAddress();
  console.log("   ✅ Standalone ReputationRegistry:", deployed.erc8004Standalone.reputationRegistry);

  console.log("\n   ℹ️  Standalone system is fully independent - no cross-contract linking needed");

  // ==========================================================================
  // DEPLOYMENT SUMMARY
  // ==========================================================================
  console.log("\n\n" + "=".repeat(80));
  console.log("✅ Sepolia Deployment Complete!");
  console.log("=".repeat(80));

  console.log("\n📍 SYSTEM 1: SAGE Core Contracts (with ERC-8004 Adapters)\n");
  console.log("  SageRegistryV2 (Security Enhanced):", deployed.sage.sageRegistryV2);
  console.log("  SageRegistryV3 (Commit-Reveal):", deployed.sage.sageRegistryV3);
  console.log("  SageVerificationHook:", deployed.sage.verificationHook);
  console.log("  ERC8004IdentityRegistry (Adapter):", deployed.sage.identityRegistry);
  console.log("  ERC8004ReputationRegistryV2:", deployed.sage.reputationRegistryV2);
  console.log("  ERC8004ValidationRegistry:", deployed.sage.validationRegistry);

  console.log("\n📍 SYSTEM 2: ERC-8004 Standalone Contracts\n");
  console.log("  ERC8004IdentityRegistry (Standalone):", deployed.erc8004Standalone.identityRegistry);
  console.log("  ERC8004ReputationRegistry (Standalone):", deployed.erc8004Standalone.reputationRegistry);
  console.log("  ERC8004ValidationRegistry (Standalone):", deployed.erc8004Standalone.validationRegistry);

  console.log("\n📊 Deployment Statistics:\n");
  console.log("  Network:", network.name);
  console.log("  Chain ID:", network.chainId.toString());
  console.log("  Deployer:", deployer.address);
  console.log("  Total Gas Used (transactions):", totalGasUsed.toString());
  console.log("  Block:", await hre.ethers.provider.getBlockNumber());

  const finalBalance = await hre.ethers.provider.getBalance(deployer.address);
  const gasSpent = balance - finalBalance;
  console.log("  ETH Spent:", hre.ethers.formatEther(gasSpent), "ETH");
  console.log("  Final Balance:", hre.ethers.formatEther(finalBalance), "ETH");

  console.log("\n🔐 Security Features Deployed:\n");
  console.log("  ✅ ReentrancyGuard: Active on payable functions");
  console.log("  ✅ Pull Payment: pendingWithdrawals mapping in use");
  console.log("  ✅ Ownable2Step: Two-step ownership transfer");
  console.log("  ✅ Pausable: Emergency stop mechanism");
  console.log("  ✅ Hook Gas Limit: 50,000 gas limit enforced");
  console.log("  ✅ Deadline Validation: 1 hour - 30 days enforced");
  console.log("  ✅ Commit-Reveal: Front-running protection (V3 & ReputationV2)");

  console.log("\n📚 Next Steps:\n");
  console.log("  1. Verify contracts on Etherscan:");
  console.log("     npx hardhat verify --network sepolia <CONTRACT_ADDRESS>");
  console.log("  2. Update documentation with deployment addresses");
  console.log("  3. Test agent registration on Sepolia");
  console.log("  4. Test validation flow on Sepolia");

  console.log("\n🔗 Etherscan Links:\n");
  console.log("  SAGE System:");
  console.log("    https://sepolia.etherscan.io/address/" + deployed.sage.sageRegistryV2);
  console.log("  ERC-8004 Standalone:");
  console.log("    https://sepolia.etherscan.io/address/" + deployed.erc8004Standalone.identityRegistry);

  console.log("\n" + "=".repeat(80) + "\n");

  // Save deployment info to file
  const deploymentInfo = {
    network: network.name,
    chainId: Number(network.chainId),
    timestamp: new Date().toISOString(),
    deployer: deployer.address,
    sage: deployed.sage,
    erc8004Standalone: deployed.erc8004Standalone,
    gasUsed: totalGasUsed.toString(),
    ethSpent: hre.ethers.formatEther(gasSpent)
  };

  const deploymentDir = './deployments';
  if (!fs.existsSync(deploymentDir)) {
    fs.mkdirSync(deploymentDir, { recursive: true });
  }

  fs.writeFileSync(
    `${deploymentDir}/sepolia-deployment.json`,
    JSON.stringify(deploymentInfo, null, 2)
  );
  console.log("💾 Deployment info saved to: ./deployments/sepolia-deployment.json\n");
}

main().catch((error) => {
  console.error("\n❌ Deployment failed:", error);
  process.exitCode = 1;
});
