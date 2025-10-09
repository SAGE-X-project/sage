const hre = require("hardhat");
const fs = require('fs');

/**
 * Sepolia Governance Deployment Script
 *
 * Deploys governance infrastructure:
 * 1. TEEKeyRegistry - Decentralized TEE key approval
 * 2. SimpleMultiSig - Multi-signature wallet for admin control
 */

async function main() {
  console.log("\n🏛️  SAGE Governance - Sepolia Testnet Deployment");
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

  if (balance < hre.ethers.parseEther("0.1")) {
    console.log("\n⚠️  WARNING: Balance may be insufficient for deployment");
    console.log("   Recommended: 0.1 ETH minimum");
  }

  console.log("\n" + "=".repeat(80));

  // Load existing deployment addresses
  let existingDeployment = {};
  const deploymentFile = './deployments/sepolia-deployment.json';
  if (fs.existsSync(deploymentFile)) {
    existingDeployment = JSON.parse(fs.readFileSync(deploymentFile, 'utf8'));
    console.log("\n📋 Loaded existing deployment addresses");
  }

  // Track deployed contracts
  const deployed = {
    governance: {}
  };
  let totalGasUsed = 0n;

  // ==========================================================================
  // GOVERNANCE SYSTEM DEPLOYMENT
  // ==========================================================================
  console.log("\n\n📦 Deploying Governance Contracts\n");

  // 1. Deploy TEEKeyRegistry
  console.log("1️⃣  Deploying TEEKeyRegistry...");
  console.log("   Configuration:");
  console.log("   - Proposal Stake: 1 ETH");
  console.log("   - Voting Period: 7 days");
  console.log("   - Quorum: 10% (minimum participation)");
  console.log("   - Approval Threshold: 66% (2/3 majority)");
  console.log("   - Slash Percentage: 50% (for rejected proposals)");

  const TEEKeyRegistry = await hre.ethers.getContractFactory("TEEKeyRegistry");
  const teeKeyRegistry = await TEEKeyRegistry.deploy(
    hre.ethers.parseEther("1.0"),  // proposalStake: 1 ETH
    7 * 24 * 60 * 60,              // votingPeriod: 7 days in seconds
    10,                             // quorumPercentage: 10%
    66,                             // approvalThreshold: 66%
    50                              // slashPercentage: 50%
  );
  await teeKeyRegistry.waitForDeployment();
  deployed.governance.teeKeyRegistry = await teeKeyRegistry.getAddress();
  console.log("\n   ✅ TEEKeyRegistry:", deployed.governance.teeKeyRegistry);
  console.log("   📝 Features: Weighted voting, Proposal stake, Slashing");

  await new Promise(resolve => setTimeout(resolve, 5000));

  // 2. Deploy SimpleMultiSig
  console.log("\n2️⃣  Deploying SimpleMultiSig...");

  // For testnet, use deployer as initial owner and set 2-of-3 multi-sig
  // In production, these would be separate trusted addresses
  const owners = [
    deployer.address,
    "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", // Hardhat test account #2
    "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"  // Hardhat test account #3
  ];
  const requiredConfirmations = 2;

  console.log("   Initial Owners:", owners.length);
  console.log("   Required Confirmations:", requiredConfirmations);

  const SimpleMultiSig = await hre.ethers.getContractFactory("SimpleMultiSig");
  const multiSig = await SimpleMultiSig.deploy(owners, requiredConfirmations);
  await multiSig.waitForDeployment();
  deployed.governance.simpleMultiSig = await multiSig.getAddress();
  console.log("\n   ✅ SimpleMultiSig:", deployed.governance.simpleMultiSig);
  console.log("   📝 Features: Multi-signature, Transaction queue");

  await new Promise(resolve => setTimeout(resolve, 5000));

  // ==========================================================================
  // CONFIGURATION (Optional - if contracts already deployed)
  // ==========================================================================
  console.log("\n\n🔧 Configuration Options\n");

  if (existingDeployment.sage) {
    console.log("📍 Existing SAGE contracts detected:");
    console.log("   SageRegistryV2:", existingDeployment.sage.sageRegistryV2);
    console.log("   SageRegistryV3:", existingDeployment.sage.sageRegistryV3);
    console.log("   ValidationRegistry:", existingDeployment.sage.validationRegistry);
    console.log("\n⚠️  To transfer ownership to MultiSig, run:");
    console.log("   node scripts/transfer-ownership.js");
  }

  // ==========================================================================
  // DEPLOYMENT SUMMARY
  // ==========================================================================
  console.log("\n\n" + "=".repeat(80));
  console.log("✅ Governance Deployment Complete!");
  console.log("=".repeat(80));

  console.log("\n📍 Governance Contracts\n");
  console.log("  TEEKeyRegistry:", deployed.governance.teeKeyRegistry);
  console.log("  SimpleMultiSig:", deployed.governance.simpleMultiSig);

  console.log("\n📊 Deployment Statistics:\n");
  console.log("  Network:", network.name);
  console.log("  Chain ID:", network.chainId.toString());
  console.log("  Deployer:", deployer.address);
  console.log("  Block:", await hre.ethers.provider.getBlockNumber());

  const finalBalance = await hre.ethers.provider.getBalance(deployer.address);
  const gasSpent = balance - finalBalance;
  console.log("  ETH Spent:", hre.ethers.formatEther(gasSpent), "ETH");
  console.log("  Final Balance:", hre.ethers.formatEther(finalBalance), "ETH");

  console.log("\n🏛️  Governance Features:\n");
  console.log("  ✅ TEE Key Approval: Community-driven weighted voting");
  console.log("  ✅ Multi-Sig Control: 2-of-3 approval required");
  console.log("  ✅ Economic Security: 1 ETH proposal stake");
  console.log("  ✅ Slashing: 50% stake slashed for rejected proposals");
  console.log("  ✅ Quorum: 10% minimum participation");
  console.log("  ✅ Supermajority: 66% approval threshold");

  console.log("\n📚 Next Steps:\n");
  console.log("  1. Register initial voters in TEEKeyRegistry:");
  console.log("     await teeKeyRegistry.registerVoter(address, weight)");
  console.log("  2. Verify contracts on Etherscan:");
  console.log("     npx hardhat verify --network sepolia <CONTRACT_ADDRESS> <CONSTRUCTOR_ARGS>");
  console.log("  3. Test TEE key proposal flow");
  console.log("  4. (Optional) Transfer contract ownership to MultiSig");

  console.log("\n🔗 Etherscan Links:\n");
  console.log("  TEEKeyRegistry:");
  console.log("    https://sepolia.etherscan.io/address/" + deployed.governance.teeKeyRegistry);
  console.log("  SimpleMultiSig:");
  console.log("    https://sepolia.etherscan.io/address/" + deployed.governance.simpleMultiSig);

  console.log("\n🧪 Test Commands:\n");
  console.log("  # Test TEE Key Governance locally");
  console.log("  npx hardhat test test/security-features.test.js --grep \"TEE Key Governance\"");
  console.log("\n  # Register a voter");
  console.log("  npx hardhat run scripts/register-voter.js --network sepolia");
  console.log("\n  # Propose a TEE key");
  console.log("  npx hardhat run scripts/propose-tee-key.js --network sepolia");

  console.log("\n" + "=".repeat(80) + "\n");

  // Update deployment info
  const updatedDeployment = {
    ...existingDeployment,
    governance: deployed.governance,
    governanceDeploymentTimestamp: new Date().toISOString(),
    governanceGasSpent: hre.ethers.formatEther(gasSpent)
  };

  const deploymentDir = './deployments';
  if (!fs.existsSync(deploymentDir)) {
    fs.mkdirSync(deploymentDir, { recursive: true });
  }

  fs.writeFileSync(
    `${deploymentDir}/sepolia-deployment.json`,
    JSON.stringify(updatedDeployment, null, 2)
  );
  console.log("💾 Deployment info updated: ./deployments/sepolia-deployment.json\n");
}

main().catch((error) => {
  console.error("\n❌ Deployment failed:", error);
  process.exitCode = 1;
});
