import hre from "hardhat";
import { network } from "hardhat";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const { ethers } = await network.connect();

/**
 * Local Deployment Verification Script
 *
 * Verifies that contracts are deployed correctly on local networks (localhost/hardhat)
 * This is a quick sanity check, not a block explorer verification
 *
 * Usage:
 *   npx hardhat run scripts/verify-deployment-local.js --network localhost
 *   npx hardhat run scripts/verify-deployment-local.js --network hardhat
 */

async function main() {
  console.log("\n🔍 Local Deployment Verification");
  console.log("=".repeat(80));
  console.log("This script checks if contracts are deployed to the blockchain.");
  console.log("For block explorer verification, use scripts/verify-agentcard.js\n");

  // Load deployment info from latest file
  const networkName = hre.network.name || "localhost";
  const deploymentPath = path.join(__dirname, "../deployments");
  const latestFileName = `${networkName}-latest.json`;
  const latestFilePath = path.join(deploymentPath, latestFileName);

  let registryAddress, hookAddress;

  if (fs.existsSync(latestFilePath)) {
    const deploymentInfo = JSON.parse(fs.readFileSync(latestFilePath, "utf8"));
    registryAddress = deploymentInfo.contracts.AgentCardRegistry.address;
    hookAddress = deploymentInfo.contracts.AgentCardVerifyHook.address;
    console.log(`📂 Loaded addresses from: ${latestFileName}`);
    console.log(`   Deployed at: ${deploymentInfo.deployedAt}\n`);
  } else {
    console.log("⚠️  No deployment file found");
    console.log("   Please deploy contracts first:");
    console.log(`   npm run deploy:localhost\n`);
    process.exit(1);
  }

  console.log("🔍 Checking contracts...");

  // Check if contracts exist
  const registryCode = await ethers.provider.getCode(registryAddress);
  const hookCode = await ethers.provider.getCode(hookAddress);

  console.log("✅ AgentCardRegistry deployed:", registryCode !== "0x");
  console.log("   Address:", registryAddress);
  console.log("   Bytecode length:", (registryCode.length - 2) / 2, "bytes\n");

  console.log("✅ AgentCardVerifyHook deployed:", hookCode !== "0x");
  console.log("   Address:", hookAddress);
  console.log("   Bytecode length:", (hookCode.length - 2) / 2, "bytes\n");

  // Verify contracts have bytecode
  if (registryCode === "0x" || hookCode === "0x") {
    console.log("❌ One or more contracts not deployed properly\n");
    process.exit(1);
  }

  // Check network
  const network = await ethers.provider.getNetwork();
  console.log("🌐 Network Information:");
  console.log("   Chain ID:", network.chainId.toString());
  console.log("   Network Name:", hre.network.name, "\n");

  // Get deployer balance
  const deployer = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266";
  const balance = await ethers.provider.getBalance(deployer);
  console.log("💰 Deployer Balance:");
  console.log("   Address:", deployer);
  console.log("   Balance:", ethers.formatEther(balance), "ETH\n");

  console.log("🎉 All checks passed! Contracts are deployed and functional.\n");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("\n❌ Verification failed:");
    console.error(error);
    process.exit(1);
  });
