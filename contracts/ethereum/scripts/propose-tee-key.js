const hre = require("hardhat");
const fs = require('fs');

/**
 * Propose TEE Key Script
 *
 * Submits a TEE key proposal to the governance system.
 * Requires proposalStake (default 1 ETH).
 */

async function main() {
  console.log("\n📝 Propose TEE Key to Governance");
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

  // Get proposer
  const [proposer] = await hre.ethers.getSigners();
  console.log("👤 Proposer:", proposer.address);

  const balance = await hre.ethers.provider.getBalance(proposer.address);
  console.log("💰 Balance:", hre.ethers.formatEther(balance), "ETH");

  // Get contract instance
  const TEEKeyRegistry = await hre.ethers.getContractFactory("TEEKeyRegistry");
  const teeKeyRegistry = TEEKeyRegistry.attach(teeKeyRegistryAddress);

  // Get proposal stake requirement
  const proposalStake = await teeKeyRegistry.proposalStake();
  console.log("💵 Required Stake:", hre.ethers.formatEther(proposalStake), "ETH");

  if (balance < proposalStake) {
    throw new Error("Insufficient balance for proposal stake");
  }

  // Configuration - Update these values for your TEE key
  const TEE_PUBLIC_KEY = process.env.TEE_PUBLIC_KEY ||
    "0x" + "04" + // Uncompressed public key prefix
    "1234567890123456789012345678901234567890123456789012345678901234" + // x coordinate
    "5678901234567890123456789012345678901234567890123456789012345678"; // y coordinate

  const ATTESTATION_DATA = process.env.ATTESTATION_DATA ||
    "0x" + "a1b2c3d4".repeat(32); // Mock attestation data

  console.log("\n📋 Proposal Details:");
  console.log("  TEE Public Key:", TEE_PUBLIC_KEY);
  console.log("  Attestation Data:", ATTESTATION_DATA.substring(0, 20) + "...");
  console.log("  Stake Amount:", hre.ethers.formatEther(proposalStake), "ETH");

  // Submit proposal
  console.log("\n📝 Submitting proposal...");
  const tx = await teeKeyRegistry.proposeTEEKey(TEE_PUBLIC_KEY, ATTESTATION_DATA, {
    value: proposalStake
  });
  console.log("  Transaction hash:", tx.hash);

  const receipt = await tx.wait();
  console.log("  ✅ Transaction confirmed");
  console.log("  Gas used:", receipt.gasUsed.toString());
  console.log("  Block:", receipt.blockNumber);

  // Get proposal ID from event
  const proposalEvent = receipt.logs.find(log => {
    try {
      const parsed = teeKeyRegistry.interface.parseLog(log);
      return parsed.name === "TEEKeyProposed";
    } catch {
      return false;
    }
  });

  let proposalId;
  if (proposalEvent) {
    const parsed = teeKeyRegistry.interface.parseLog(proposalEvent);
    proposalId = parsed.args.proposalId;
    console.log("\n✅ Proposal Created!");
    console.log("  Proposal ID:", proposalId.toString());
  }

  // Get proposal details
  if (proposalId !== undefined) {
    const proposal = await teeKeyRegistry.proposals(proposalId);
    const votingPeriod = await teeKeyRegistry.votingPeriod();
    const endTime = new Date(Number(proposal.startTime + votingPeriod) * 1000);

    console.log("\n📊 Proposal Status:");
    console.log("  Proposal ID:", proposalId.toString());
    console.log("  TEE Key:", proposal.teePublicKey);
    console.log("  Proposer:", proposal.proposer);
    console.log("  Stake:", hre.ethers.formatEther(proposal.stake), "ETH");
    console.log("  Voting Ends:", endTime.toISOString());
    console.log("  Status:", proposal.executed ? "Executed" : "Active");
    console.log("  Votes For:", proposal.votesFor.toString());
    console.log("  Votes Against:", proposal.votesAgainst.toString());
  }

  console.log("\n📚 Next Steps:");
  console.log("  1. Wait for voting period to start");
  console.log("  2. Vote on proposal:");
  console.log("     node scripts/vote-on-proposal.js " + (proposalId || "0"));
  console.log("  3. After voting period ends, execute proposal:");
  console.log("     node scripts/execute-proposal.js " + (proposalId || "0"));
  console.log("\n" + "=".repeat(60) + "\n");
}

main().catch((error) => {
  console.error("\n❌ Proposal failed:", error);
  process.exitCode = 1;
});
