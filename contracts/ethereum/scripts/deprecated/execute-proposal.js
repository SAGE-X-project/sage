const hre = require("hardhat");
const fs = require('fs');

/**
 * Execute Proposal Script
 *
 * Executes a TEE key proposal after voting period ends.
 * Can be called by anyone once voting period is complete.
 *
 * Usage: node scripts/execute-proposal.js <proposalId>
 * Example: node scripts/execute-proposal.js 0
 */

async function main() {
  console.log("\n⚡ Execute TEE Key Proposal");
  console.log("=".repeat(60));

  // Get command line argument
  const proposalId = process.argv[2] || "0";

  console.log("\n📋 Execution Details:");
  console.log("  Proposal ID:", proposalId);

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

  // Get executor (can be anyone)
  const [executor] = await hre.ethers.getSigners();
  console.log("👤 Executor:", executor.address);

  // Get contract instance
  const TEEKeyRegistry = await hre.ethers.getContractFactory("TEEKeyRegistry");
  const teeKeyRegistry = TEEKeyRegistry.attach(teeKeyRegistryAddress);

  // Get proposal details
  const proposal = await teeKeyRegistry.proposals(proposalId);
  const votingPeriod = await teeKeyRegistry.votingPeriod();
  const currentTime = Math.floor(Date.now() / 1000);
  const endTime = Number(proposal.startTime + votingPeriod);

  console.log("\n📊 Proposal Status:");
  console.log("  TEE Key:", proposal.teePublicKey.substring(0, 20) + "...");
  console.log("  Proposer:", proposal.proposer);
  console.log("  Stake:", hre.ethers.formatEther(proposal.stake), "ETH");
  console.log("  Votes For:", proposal.votesFor.toString());
  console.log("  Votes Against:", proposal.votesAgainst.toString());
  console.log("  Executed:", proposal.executed);
  console.log("  Voting Ended:", currentTime >= endTime ? "✅" : "❌");

  if (proposal.executed) {
    throw new Error("Proposal already executed.");
  }

  if (currentTime < endTime) {
    const remaining = endTime - currentTime;
    throw new Error(`Voting period not ended. Wait ${remaining} seconds.`);
  }

  // Calculate outcome
  const totalVotingPower = await teeKeyRegistry.totalVotingPower();
  const participation = ((Number(proposal.votesFor) + Number(proposal.votesAgainst)) /
                        Number(totalVotingPower) * 100);
  const approvalRate = Number(proposal.votesFor) /
                      (Number(proposal.votesFor) + Number(proposal.votesAgainst)) * 100;

  const quorumPercentage = await teeKeyRegistry.quorumPercentage();
  const approvalThreshold = await teeKeyRegistry.approvalThreshold();
  const slashPercentage = await teeKeyRegistry.slashPercentage();

  const quorumMet = participation >= Number(quorumPercentage);
  const approvalMet = approvalRate >= Number(approvalThreshold);
  const willApprove = quorumMet && approvalMet;

  console.log("\n📋 Voting Analysis:");
  console.log("  Total Voting Power:", totalVotingPower.toString());
  console.log("  Participation:", participation.toFixed(2) + "%");
  console.log("  Approval Rate:", approvalRate.toFixed(2) + "%");
  console.log("\n  Quorum Required:", quorumPercentage.toString() + "%");
  console.log("  Quorum Met:", quorumMet ? "✅" : "❌");
  console.log("  Approval Required:", approvalThreshold.toString() + "%");
  console.log("  Approval Met:", approvalMet ? "✅" : "❌");
  console.log("\n  🎯 Outcome:", willApprove ? "APPROVED ✅" : "REJECTED ❌");

  if (!willApprove) {
    const slashAmount = (proposal.stake * BigInt(slashPercentage)) / 100n;
    console.log("  ⚠️  Slash Amount:", hre.ethers.formatEther(slashAmount), "ETH");
    console.log("  💰 Return to Proposer:", hre.ethers.formatEther(proposal.stake - slashAmount), "ETH");
  } else {
    console.log("  ✅ Full stake returned:", hre.ethers.formatEther(proposal.stake), "ETH");
    console.log("  🔑 TEE key will be marked as trusted");
  }

  // Execute proposal
  console.log("\n⚡ Executing proposal...");
  const tx = await teeKeyRegistry.executeProposal(proposalId);
  console.log("  Transaction hash:", tx.hash);

  const receipt = await tx.wait();
  console.log("  ✅ Transaction confirmed");
  console.log("  Gas used:", receipt.gasUsed.toString());
  console.log("  Block:", receipt.blockNumber);

  // Parse events
  const approvedEvent = receipt.logs.find(log => {
    try {
      const parsed = teeKeyRegistry.interface.parseLog(log);
      return parsed.name === "TEEKeyApproved";
    } catch {
      return false;
    }
  });

  const rejectedEvent = receipt.logs.find(log => {
    try {
      const parsed = teeKeyRegistry.interface.parseLog(log);
      return parsed.name === "TEEKeyRejected";
    } catch {
      return false;
    }
  });

  if (approvedEvent) {
    const parsed = teeKeyRegistry.interface.parseLog(approvedEvent);
    console.log("\n✅ TEE KEY APPROVED!");
    console.log("  Proposal ID:", parsed.args.proposalId.toString());
    console.log("  TEE Key:", parsed.args.teePublicKey.substring(0, 20) + "...");
    console.log("  Status: TRUSTED ✅");

    // Verify key is trusted
    const isTrusted = await teeKeyRegistry.trustedTEEKeys(proposal.teePublicKey);
    console.log("  Verification:", isTrusted ? "TRUSTED" : "ERROR");
  }

  if (rejectedEvent) {
    const parsed = teeKeyRegistry.interface.parseLog(rejectedEvent);
    console.log("\n❌ TEE KEY REJECTED");
    console.log("  Proposal ID:", parsed.args.proposalId.toString());
    console.log("  TEE Key:", parsed.args.teePublicKey.substring(0, 20) + "...");
    console.log("  Slashed Amount:", hre.ethers.formatEther(parsed.args.slashedAmount), "ETH");
  }

  // Final status
  const updatedProposal = await teeKeyRegistry.proposals(proposalId);
  console.log("\n📊 Final Status:");
  console.log("  Executed:", updatedProposal.executed ? "✅" : "❌");
  console.log("  Outcome:", willApprove ? "APPROVED" : "REJECTED");

  console.log("\n📚 Next Steps:");
  if (willApprove) {
    console.log("  ✅ TEE key is now trusted and can be used");
    console.log("  ✅ Validators can submit TEE attestations with this key");
  } else {
    console.log("  ❌ TEE key was rejected");
    console.log("  💡 Consider proposing a different key");
  }
  console.log("\n" + "=".repeat(60) + "\n");
}

main().catch((error) => {
  console.error("\n❌ Execution failed:", error);
  process.exitCode = 1;
});
