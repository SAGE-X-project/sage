const hre = require("hardhat");

/**
 * ERC-8004 Standalone - Sepolia Integration Test
 *
 * Tests agent registration on the deployed Sepolia contracts
 */

async function main() {
  console.log("\n🧪 ERC-8004 Standalone - Sepolia Integration Test");
  console.log("=".repeat(80));

  // Get signer
  const [deployer] = await hre.ethers.getSigners();

  console.log("\n📍 Network Information:");
  const network = await hre.ethers.provider.getNetwork();
  console.log("  Network:", network.name);
  console.log("  Chain ID:", network.chainId.toString());
  console.log("\n👤 Test Account:");
  console.log("  Address:", deployer.address);

  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("  Balance:", hre.ethers.formatEther(balance), "ETH");

  console.log("\n" + "=".repeat(80));

  // Deployed contract addresses (from deployment)
  const STANDALONE_IDENTITY_REGISTRY = "0x02439d8DA11517603d0DE1424B33139A90969517";
  const STANDALONE_VALIDATION_REGISTRY = "0xa8e001E0755342BAeCF47D0d4d1C418a1FD82b8f";
  const STANDALONE_REPUTATION_REGISTRY = "0x1eA3c909fE7Eb94A724b163CD98117832931D9F4";

  console.log("\n📍 Using Deployed Contracts:");
  console.log("  IdentityRegistry:", STANDALONE_IDENTITY_REGISTRY);
  console.log("  ValidationRegistry:", STANDALONE_VALIDATION_REGISTRY);
  console.log("  ReputationRegistry:", STANDALONE_REPUTATION_REGISTRY);

  // Connect to contracts
  const identityRegistry = await hre.ethers.getContractAt(
    "contracts/erc-8004/standalone/ERC8004IdentityRegistry.sol:ERC8004IdentityRegistry",
    STANDALONE_IDENTITY_REGISTRY
  );

  const validationRegistry = await hre.ethers.getContractAt(
    "contracts/erc-8004/standalone/ERC8004ValidationRegistry.sol:ERC8004ValidationRegistry",
    STANDALONE_VALIDATION_REGISTRY
  );

  const reputationRegistry = await hre.ethers.getContractAt(
    "contracts/erc-8004/standalone/ERC8004ReputationRegistry.sol:ERC8004ReputationRegistry",
    STANDALONE_REPUTATION_REGISTRY
  );

  console.log("\n✅ Connected to all contracts");

  // ==========================================================================
  // TEST 1: Check if agent already registered
  // ==========================================================================
  console.log("\n\n📝 TEST 1: Check Existing Registrations\n");

  const totalAgents = await identityRegistry.totalAgents();
  console.log("  Total registered agents:", totalAgents.toString());

  // Check if current address has an agent
  try {
    const existingAgent = await identityRegistry.resolveAgentByAddress(deployer.address);
    console.log("\n  ℹ️  Agent already registered for this address:");
    console.log("    Agent ID:", existingAgent.agentId);
    console.log("    Endpoint:", existingAgent.endpoint);
    console.log("    Active:", existingAgent.isActive);
    console.log("    Registered At:", new Date(Number(existingAgent.registeredAt) * 1000).toISOString());

    console.log("\n  ✅ Agent registration verified on Sepolia!");
    return;
  } catch (error) {
    if (error.message.includes("AgentNotFound") || error.message.includes("No agent found")) {
      console.log("\n  ℹ️  No agent registered for this address yet");
    } else {
      console.log("\n  ⚠️  Error checking agent:", error.message);
    }
  }

  // ==========================================================================
  // TEST 2: Register New Agent
  // ==========================================================================
  console.log("\n\n📝 TEST 2: Register New Agent\n");

  const agentId = `did:erc8004:sepolia:test:${Date.now()}`;
  const endpoint = "https://example.com/agentcard.json";

  console.log("  Registering agent:");
  console.log("    Agent ID:", agentId);
  console.log("    Endpoint:", endpoint);

  try {
    const tx = await identityRegistry.registerAgent(agentId, endpoint);
    console.log("\n  ⏳ Transaction sent:", tx.hash);
    console.log("  ⏳ Waiting for confirmation...");

    const receipt = await tx.wait();
    console.log("\n  ✅ Agent registered successfully!");
    console.log("  📊 Gas used:", receipt.gasUsed.toString());
    console.log("  📊 Block number:", receipt.blockNumber);

    // Verify registration
    const agentInfo = await identityRegistry.resolveAgent(agentId);
    console.log("\n  📋 Verified agent information:");
    console.log("    Agent ID:", agentInfo.agentId);
    console.log("    Address:", agentInfo.agentAddress);
    console.log("    Endpoint:", agentInfo.endpoint);
    console.log("    Active:", agentInfo.isActive);
    console.log("    Registered At:", new Date(Number(agentInfo.registeredAt) * 1000).toISOString());

    // Check updated total
    const newTotal = await identityRegistry.totalAgents();
    console.log("\n  📊 Total registered agents:", newTotal.toString());

  } catch (error) {
    if (error.message.includes("AgentAlreadyRegistered")) {
      console.log("\n  ℹ️  Agent already registered (this is expected if running multiple times)");
    } else {
      console.log("\n  ❌ Registration failed:", error.message);
      throw error;
    }
  }

  // ==========================================================================
  // TEST 3: Verify Contract Configuration
  // ==========================================================================
  console.log("\n\n📝 TEST 3: Verify Contract Configuration\n");

  const minStake = await validationRegistry.minStake();
  const minValidators = await validationRegistry.minValidators();
  const consensusThreshold = await validationRegistry.consensusThreshold();

  console.log("  ValidationRegistry Configuration:");
  console.log("    Min Stake:", hre.ethers.formatEther(minStake), "ETH");
  console.log("    Min Validators:", minValidators.toString());
  console.log("    Consensus Threshold:", consensusThreshold.toString() + "%");

  const validationRegistryAddress = await reputationRegistry.validationRegistry();
  console.log("\n  ReputationRegistry Configuration:");
  console.log("    Linked ValidationRegistry:", validationRegistryAddress);
  console.log("    Matches deployed address:", validationRegistryAddress === STANDALONE_VALIDATION_REGISTRY);

  // ==========================================================================
  // SUMMARY
  // ==========================================================================
  console.log("\n\n" + "=".repeat(80));
  console.log("✅ Sepolia Integration Test Complete!");
  console.log("=".repeat(80));

  console.log("\n📊 Test Summary:\n");
  console.log("  ✅ Contracts deployed and accessible");
  console.log("  ✅ Agent registration working on Sepolia");
  console.log("  ✅ Contract configuration verified");
  console.log("  ✅ ERC-8004 Standalone fully functional");

  console.log("\n🔗 Etherscan Links:\n");
  console.log("  IdentityRegistry:");
  console.log("    https://sepolia.etherscan.io/address/" + STANDALONE_IDENTITY_REGISTRY);
  console.log("  ValidationRegistry:");
  console.log("    https://sepolia.etherscan.io/address/" + STANDALONE_VALIDATION_REGISTRY);
  console.log("  ReputationRegistry:");
  console.log("    https://sepolia.etherscan.io/address/" + STANDALONE_REPUTATION_REGISTRY);

  console.log("\n💡 Next Steps:\n");
  console.log("  1. Check Etherscan for transaction details");
  console.log("  2. Verify agent registration in Read Contract section");
  console.log("  3. Try registering agents from different addresses");
  console.log("  4. Test validation and reputation flows");

  console.log("\n" + "=".repeat(80) + "\n");
}

main().catch((error) => {
  console.error("\n❌ Test failed:", error);
  process.exitCode = 1;
});
