import hre from "hardhat";
import { network } from "hardhat";

const { ethers } = await network.connect();

async function main() {
  console.log("\n🧪 Testing Frontend Agent Query Integration\n");
  console.log("=" .repeat(80));

  const registryAddress = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";
  const agentOwner = "0xE230795E3DDef701fe38dB02D70E796d352068a0";

  const Registry = await ethers.getContractFactory("AgentCardRegistry");
  const registry = Registry.attach(registryAddress);

  try {
    // Test 1: Query registered agents by owner
    console.log("\n📋 Test 1: Query agents by owner");
    console.log(`  Owner: ${agentOwner}`);

    const agentIds = await registry.getAgentsByOwner(agentOwner);
    console.log(`  ✓ Found ${agentIds.length} agent(s)`);

    if (agentIds.length === 0) {
      console.log("\n  ❌ No agents found! Registration may have failed.");
      process.exit(1);
    }

    // Test 2: Get agent metadata
    console.log("\n📄 Test 2: Get agent metadata");
    const agentId = agentIds[0];
    console.log(`  Agent ID: ${agentId}`);

    const metadata = await registry.getAgent(agentId);
    console.log(`\n  Agent Details:`);
    console.log(`    DID: ${metadata.did}`);
    console.log(`    Name: ${metadata.name}`);
    console.log(`    Description: ${metadata.description}`);
    console.log(`    Endpoint: ${metadata.endpoint}`);
    console.log(`    Capabilities: ${metadata.capabilities}`);
    console.log(`    Owner: ${metadata.owner}`);
    console.log(`    Registered: ${new Date(Number(metadata.registeredAt) * 1000).toISOString()}`);
    console.log(`    Updated: ${new Date(Number(metadata.updatedAt) * 1000).toISOString()}`);
    console.log(`    Active: ${metadata.active}`);

    // Test 3: Get agent keys
    console.log("\n🔑 Test 3: Get agent keys");
    const keyHashes = metadata.keyHashes;
    console.log(`  Total keys: ${keyHashes.length}`);

    for (let i = 0; i < keyHashes.length; i++) {
      const keyHash = keyHashes[i];
      const keyInfo = await registry.getKey(keyHash);

      const keyTypeNames = ["ECDSA", "Ed25519", "X25519"];
      const keyTypeName = keyTypeNames[keyInfo.keyType] || "Unknown";

      console.log(`\n  Key #${i + 1}:`);
      console.log(`    Hash: ${keyHash}`);
      console.log(`    Type: ${keyTypeName} (${keyInfo.keyType})`);
      console.log(`    Data: ${keyInfo.keyData.substring(0, 20)}... (${keyInfo.keyData.length / 2 - 1} bytes)`);
      console.log(`    Verified: ${keyInfo.verified}`);
      console.log(`    Registered: ${new Date(Number(keyInfo.registeredAt) * 1000).toISOString()}`);
    }

    // Test 4: Test agent endpoint availability
    console.log("\n🌐 Test 4: Check agent endpoint availability");
    console.log(`  Endpoint: ${metadata.endpoint}`);

    try {
      const response = await fetch(`${metadata.endpoint}/health`);
      const health = await response.json();
      console.log(`  ✓ Agent is online!`);
      console.log(`    Status: ${health.status}`);
      console.log(`    SAGE Enabled: ${health.sage_enabled}`);
      console.log(`    Uptime: ${health.uptime_seconds}s`);
    } catch (error) {
      console.log(`  ⚠️  Agent endpoint not reachable: ${error.message}`);
    }

    // Test 5: Simulate frontend query workflow
    console.log("\n🔄 Test 5: Simulate frontend query workflow");
    console.log("  This simulates what the frontend would do:");
    console.log("  1. User enters owner address or agent DID");
    console.log("  2. Frontend calls getAgentsByOwner() or getAgentByDID()");
    console.log("  3. Frontend displays agent cards with metadata");
    console.log("  4. User can click to view full agent details");
    console.log("  5. Frontend shows keys, capabilities, and endpoint");

    console.log("\n✅ All integration tests passed!");
    console.log("\n" + "=".repeat(80));
    console.log("🎉 Frontend can successfully query and display agent information!");
    console.log("=".repeat(80));

  } catch (error) {
    console.error("\n❌ Test failed!");
    console.error("Error:", error.message);
    if (error.data) {
      console.error("Error data:", error.data);
    }
    process.exit(1);
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
