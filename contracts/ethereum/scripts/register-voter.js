const hre = require("hardhat");
const fs = require('fs');

/**
 * Register Voter Script
 *
 * Registers a voter in the TEEKeyRegistry with voting weight.
 * Only the owner can register voters.
 */

async function main() {
  console.log("\n🗳️  Register Voter in TEEKeyRegistry");
  console.log("=".repeat(60));

  // Load deployment addresses
  const deploymentFile = './deployments/sepolia-deployment.json';
  if (!fs.existsSync(deploymentFile)) {
    throw new Error("Deployment file not found. Deploy governance contracts first.");
  }

  const deployment = JSON.parse(fs.readFileSync(deploymentFile, 'utf8'));
  const teeKeyRegistryAddress = deployment.governance?.teeKeyRegistry;

  if (!teeKeyRegistryAddress) {
    throw new Error("TEEKeyRegistry not found in deployment file. Deploy governance first.");
  }

  console.log("\n📍 Network:", (await hre.ethers.provider.getNetwork()).name);
  console.log("📍 TEEKeyRegistry:", teeKeyRegistryAddress);

  // Get deployer (owner)
  const [owner] = await hre.ethers.getSigners();
  console.log("👤 Owner:", owner.address);

  // Get contract instance
  const TEEKeyRegistry = await hre.ethers.getContractFactory("TEEKeyRegistry");
  const teeKeyRegistry = TEEKeyRegistry.attach(teeKeyRegistryAddress);

  // Configuration - Update these values as needed
  const VOTER_ADDRESS = process.env.VOTER_ADDRESS || owner.address;
  const VOTER_WEIGHT = process.env.VOTER_WEIGHT || "100";

  console.log("\n📋 Registration Details:");
  console.log("  Voter Address:", VOTER_ADDRESS);
  console.log("  Voting Weight:", VOTER_WEIGHT);

  // Check if already registered
  const currentWeight = await teeKeyRegistry.voterWeights(VOTER_ADDRESS);
  if (currentWeight > 0n) {
    console.log("\n⚠️  Voter already registered with weight:", currentWeight.toString());
    console.log("   Updating weight...");
  }

  // Register voter
  console.log("\n📝 Registering voter...");
  const tx = await teeKeyRegistry.registerVoter(VOTER_ADDRESS, VOTER_WEIGHT);
  console.log("  Transaction hash:", tx.hash);

  const receipt = await tx.wait();
  console.log("  ✅ Transaction confirmed");
  console.log("  Gas used:", receipt.gasUsed.toString());
  console.log("  Block:", receipt.blockNumber);

  // Verify registration
  const newWeight = await teeKeyRegistry.voterWeights(VOTER_ADDRESS);
  const totalVotingPower = await teeKeyRegistry.totalVotingPower();

  console.log("\n✅ Registration Complete!");
  console.log("\n📊 Voter Status:");
  console.log("  Address:", VOTER_ADDRESS);
  console.log("  Weight:", newWeight.toString());
  console.log("  Total Voting Power:", totalVotingPower.toString());
  console.log("  Share of Voting Power:", ((Number(newWeight) / Number(totalVotingPower)) * 100).toFixed(2) + "%");

  console.log("\n📚 Next Steps:");
  console.log("  1. Register additional voters (if needed)");
  console.log("  2. Propose a TEE key:");
  console.log("     node scripts/propose-tee-key.js");
  console.log("\n" + "=".repeat(60) + "\n");
}

main().catch((error) => {
  console.error("\n❌ Registration failed:", error);
  process.exitCode = 1;
});
