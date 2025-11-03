import hre from "hardhat";
import { network } from "hardhat";

const { ethers } = await network.connect();

async function main() {
  console.log("\n🧪 Simple Registration Test");
  console.log("=".repeat(80));

  const [signer] = await ethers.getSigners();
  console.log(`  Signer: ${signer.address}`);

  const registryAddress = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";
  const Registry = await ethers.getContractFactory("AgentCardRegistry");
  const registry = Registry.attach(registryAddress);

  // Simple test parameters
  const did = "did:sage:ethereum:" + signer.address;
  const chainId = 31337;

  // Generate a test ECDSA key (just use signer's key for simplicity)
  const testMessage = ethers.solidityPackedKeccak256(
    ["string", "uint256", "address", "address"],
    ["SAGE Agent Registration:", chainId, registryAddress, signer.address]
  );

  const ecdsaSignature = await signer.signMessage(ethers.getBytes(testMessage));

  // Create minimal params
  const keys = [ethers.hexlify(ethers.randomBytes(65))]; // Fake ECDSA public key
  const salt = ethers.randomBytes(32);

  const params = {
    did,
    name: "Test Agent",
    description: "Test",
    endpoint: "http://localhost:8080",
    capabilities: "test",
    keys,
    keyTypes: [0], // ECDSA only
    signatures: [ecdsaSignature],
    salt
  };

  try {
    // Step 1: Commit
    const commitHash = ethers.keccak256(
      ethers.AbiCoder.defaultAbiCoder().encode(
        ["string", "bytes[]", "address", "bytes32", "uint256"],
        [params.did, params.keys, signer.address, params.salt, chainId]
      )
    );

    console.log(`\n  📝 Commitment hash: ${commitHash}`);
    const commitTx = await registry.commitRegistration(commitHash, {
      value: ethers.parseEther("0.01")
    });
    const commitReceipt = await commitTx.wait();
    console.log(`  ✓ Commit successful (block ${commitReceipt.blockNumber})`);

    // Wait 2 seconds
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 2: Register
    console.log(`\n  📤 Attempting registration...`);
    const registerTx = await registry.registerAgentWithParams(params);
    const registerReceipt = await registerTx.wait();
    console.log(`  ✓ Registration successful! (block ${registerReceipt.blockNumber})`);
    console.log(`  ✓ Gas used: ${registerReceipt.gasUsed.toString()}`);

    // Query the registered agent
    const agentIds = await registry.getAgentsByOwner(signer.address);
    console.log(`\n  📋 Registered agents: ${agentIds.length}`);

  } catch (error) {
    console.error("\n  ❌ Error:", error.shortMessage || error.message);
    if (error.data) {
      console.error("  Error data:", error.data);
    }
    if (error.reason) {
      console.error("  Reason:", error.reason);
    }
  }

  console.log("\n" + "=".repeat(80));
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
