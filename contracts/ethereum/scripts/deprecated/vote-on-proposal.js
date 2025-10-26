const hre = require("hardhat");
const fs = require('fs');

/**
 * Vote on Proposal Script
 *
 * Casts a vote on a TEE key proposal.
 * Only registered voters can vote.
 *
 * Usage: node scripts/vote-on-proposal.js <proposalId> <support>
 * Example: node scripts/vote-on-proposal.js 0 true
 */

async function main() {
  console.log("\n🗳️  Vote on TEE Key Proposal");
  console.log("=".repeat(60));

  // Get command line arguments
  const proposalId = process.argv[2] || "0";
  const support = process.argv[3] === "true" || process.argv[3] === undefined; // Default to true

  console.log("\n📋 Vote Details:");
  console.log("  Proposal ID:", proposalId);
  console.log("  Vote:", support ? "FOR ✅" : "AGAINST ❌");

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

  // Get voter
  const [voter] = await hre.ethers.getSigners();
  console.log("👤 Voter:", voter.address);

  // Get contract instance
  const TEEKeyRegistry = await hre.ethers.getContractFactory("TEEKeyRegistry");
  const teeKeyRegistry = TEEKeyRegistry.attach(teeKeyRegistryAddress);

  // Check voter weight
  const voterWeight = await teeKeyRegistry.voterWeights(voter.address);
  if (voterWeight === 0n) {
    throw new Error("Voter not registered. Register voter first.");
  }
  console.log("  Voting Weight:", voterWeight.toString());

  // Get proposal details
  const proposal = await teeKeyRegistry.proposals(proposalId);
  const votingPeriod = await teeKeyRegistry.votingPeriod();
  const currentTime = Math.floor(Date.now() / 1000);
  const endTime = Number(proposal.startTime + votingPeriod);

  console.log("\n📊 Current Proposal Status:");
  console.log("  TEE Key:", proposal.teePublicKey.substring(0, 20) + "...");
  console.log("  Proposer:", proposal.proposer);
  console.log("  Votes For:", proposal.votesFor.toString());
  console.log("  Votes Against:", proposal.votesAgainst.toString());
  console.log("  Executed:", proposal.executed);
  console.log("  Time Remaining:", Math.max(0, endTime - currentTime), "seconds");

  if (proposal.executed) {
    throw new Error("Proposal already executed. Cannot vote.");
  }

  if (currentTime > endTime) {
    throw new Error("Voting period ended. Cannot vote.");
  }

  // Check if already voted
  const hasVoted = await teeKeyRegistry.hasVoted(proposalId, voter.address);
  if (hasVoted) {
    throw new Error("Already voted on this proposal.");
  }

  // Cast vote
  console.log("\n📝 Casting vote...");
  const tx = await teeKeyRegistry.vote(proposalId, support);
  console.log("  Transaction hash:", tx.hash);

  const receipt = await tx.wait();
  console.log("  ✅ Transaction confirmed");
  console.log("  Gas used:", receipt.gasUsed.toString());
  console.log("  Block:", receipt.blockNumber);

  // Get updated proposal details
  const updatedProposal = await teeKeyRegistry.proposals(proposalId);
  const totalVotingPower = await teeKeyRegistry.totalVotingPower();
  const participation = ((Number(updatedProposal.votesFor) + Number(updatedProposal.votesAgainst)) /
                        Number(totalVotingPower) * 100);
  const approvalRate = Number(updatedProposal.votesFor) /
                      (Number(updatedProposal.votesFor) + Number(updatedProposal.votesAgainst)) * 100;

  console.log("\n✅ Vote Cast!");
  console.log("\n📊 Updated Proposal Status:");
  console.log("  Votes For:", updatedProposal.votesFor.toString());
  console.log("  Votes Against:", updatedProposal.votesAgainst.toString());
  console.log("  Total Voting Power:", totalVotingPower.toString());
  console.log("  Participation:", participation.toFixed(2) + "%");
  console.log("  Approval Rate:", approvalRate.toFixed(2) + "%");

  // Check if thresholds met
  const quorumPercentage = await teeKeyRegistry.quorumPercentage();
  const approvalThreshold = await teeKeyRegistry.approvalThreshold();

  console.log("\n📋 Threshold Status:");
  console.log("  Quorum Required:", quorumPercentage.toString() + "%");
  console.log("  Quorum Met:", participation >= Number(quorumPercentage) ? "✅" : "❌");
  console.log("  Approval Required:", approvalThreshold.toString() + "%");
  console.log("  Approval Met:", approvalRate >= Number(approvalThreshold) ? "✅" : "❌");

  console.log("\n📚 Next Steps:");
  if (currentTime < endTime) {
    console.log("  Voting period still active. Wait for:");
    console.log("  - More votes (optional)");
    console.log("  - Voting period to end (", Math.max(0, endTime - currentTime), "seconds )");
  }
  console.log("  Execute proposal after voting ends:");
  console.log("    node scripts/execute-proposal.js " + proposalId);
  console.log("\n" + "=".repeat(60) + "\n");
}

main().catch((error) => {
  console.error("\n❌ Vote failed:", error);
  process.exitCode = 1;
});
