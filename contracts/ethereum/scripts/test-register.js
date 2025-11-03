import hre from "hardhat";
import { network } from "hardhat";

// Initialize ethers from network connection
const { ethers } = await network.connect();

async function main() {
  console.log("\n🧪 Testing Agent Registration");
  console.log("=".repeat(80));

  const [signer] = await ethers.getSigners();
  const registryAddress = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";

  const Registry = await ethers.getContractFactory("AgentCardRegistry");
  const registry = Registry.attach(registryAddress);

  // Agent parameters
  const did = "did:sage:ethereum:0xE230795E3DDef701fe38dB02D70E796d352068a0";
  const keys = [
    "0x04e38c1b8a3a3e088c7a5c42db5a3f1e59b8e63e89d6f5e67a2c9e9f3a7e8c9d1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4",
    "0x64f44fe9b266af6f062575f075202b0cefb22c3dfa9560c92de32d9163383538"
  ];
  const salt = ethers.keccak256(ethers.toUtf8Bytes("test-salt"));

  const params = {
    did,
    name: "Official Payment Agent",
    description: "Blockchain-based secure payment processing agent",
    endpoint: "http://localhost:19083",
    capabilities: "crypto_payment,stablecoin,rfc9421_signature,hpke_encryption",
    keys,
    keyTypes: [0, 1],
    signatures: ["0x" + "00".repeat(65)],
    salt
  };

  try {
    // Step 1: Commit
    const commitHash = ethers.keccak256(
      ethers.AbiCoder.defaultAbiCoder().encode(
        ["string", "bytes[]", "address", "bytes32", "uint256"],
        [params.did, params.keys, signer.address, params.salt, 31337]
      )
    );

    console.log(`  Commit hash: ${commitHash}`);
    const commitTx = await registry.commitRegistration(commitHash, {
      value: ethers.parseEther("0.01")
    });
    await commitTx.wait();
    console.log("  ✓ Commit successful");

    // Wait 2 seconds
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 2: Register
    console.log("  Attempting registration...");
    const registerTx = await registry.registerAgentWithParams(params);
    await registerTx.wait();
    console.log("  ✓ Registration successful!");
  } catch (error) {
    console.error("  ❌ Error:", error.message);
    if (error.data) {
      console.error("  Error data:", error.data);
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
