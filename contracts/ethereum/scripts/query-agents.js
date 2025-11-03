import hre from "hardhat";
import { network } from "hardhat";

// Initialize ethers from network connection (Hardhat 3.x pattern)
const { ethers } = await network.connect();

async function main() {
  console.log("\n📋 Querying Registered Agents");
  console.log("=".repeat(80));

  // Load contract address from deployment file
  const deploymentData = JSON.parse(
    await import("fs").then(fs =>
      fs.promises.readFile("./deployments/localhost-latest.json", "utf8")
    )
  );

  const registryAddress = deploymentData.contracts.AgentCardRegistry.address;
  console.log(`  Registry Address: ${registryAddress}`);

  // Get contract instance
  const Registry = await ethers.getContractFactory("AgentCardRegistry");
  const registry = Registry.attach(registryAddress);

  // Query Payment Agent owner address
  const paymentAgentOwner = "0xE230795E3DDef701fe38dB02D70E796d352068a0";
  console.log(`\n  Querying agents for owner: ${paymentAgentOwner}`);

  try {
    const agentIds = await registry.getAgentsByOwner(paymentAgentOwner);
    console.log(`  Found ${agentIds.length} agent(s)`);

    if (agentIds.length === 0) {
      console.log("\n⚠️  No agents registered yet.");
      console.log("   This is expected if agent registration is simulated.");
    } else {
      for (let i = 0; i < agentIds.length; i++) {
        const agentId = agentIds[i];
        console.log(`\n  Agent #${i + 1}:`);
        console.log(`    ID: ${agentId}`);

        const agent = await registry.getAgent(agentId);
        console.log(`    DID: ${agent.did}`);
        console.log(`    Name: ${agent.name}`);
        console.log(`    Endpoint: ${agent.endpoint}`);
        console.log(`    Active: ${agent.active}`);
      }
    }
  } catch (error) {
    console.error(`\n❌ Error querying agents: ${error.message}`);
  }

  console.log("\n" + "=".repeat(80));
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
