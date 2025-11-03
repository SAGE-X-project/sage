import hre from "hardhat";
import { network } from "hardhat";

const { ethers } = await network.connect();

async function main() {
  console.log("\n🧪 Testing Registration with Detailed Errors\n");

  // Get signer (same as Go agent owner)
  const [defaultSigner] = await ethers.getSigners();

  // Agent's actual address
  const agentAddress = "0xE230795E3DDef701fe38dB02D70E796d352068a0";

  // Get registry
  const registryAddress = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";
  const Registry = await ethers.getContractFactory("AgentCardRegistry");
  const registry = Registry.attach(registryAddress);

  const chainId = 31337;

  // Agent's ECDSA public key (65 bytes) - from earlier logs
  const ecdsaKey = "0x04e38c1b8a3a3e088c7a5c42db5a3f1e59b8e63e89d6f5e67a2c9e9f3a7e8c9d1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4";

  // Agent's X25519 public key (32 bytes) - from earlier logs
  const x25519Key = "0x64f44fe9b266af6f062575f075202b0cefb22c3dfa9560c92de32d9163383538";

  const keys = [ecdsaKey, x25519Key];

  // Generate salt
  const salt = ethers.randomBytes(32);

  // Create commitment
  const commitHash = ethers.keccak256(
    ethers.AbiCoder.defaultAbiCoder().encode(
      ["string", "bytes[]", "address", "bytes32", "uint256"],
      [`did:sage:ethereum:${agentAddress}`, keys, defaultSigner.address, salt, chainId]
    )
  );

  try {
    // Step 1: Commit
    console.log(`📝 Committing with hash: ${commitHash}`);
    const commitTx = await registry.commitRegistration(commitHash, {
      value: ethers.parseEther("0.01")
    });
    await commitTx.wait();
    console.log(`✓ Commitment successful`);

    // Wait 2 seconds
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 2: Generate signatures
    // ECDSA signature
    const ecdsaMessage = ethers.solidityPackedKeccak256(
      ["string", "uint256", "address", "address"],
      ["SAGE Agent Registration:", chainId, registryAddress, defaultSigner.address]
    );
    const ecdsaSignature = await defaultSigner.signMessage(ethers.getBytes(ecdsaMessage));
    console.log(`\n✓ ECDSA signature generated: ${ecdsaSignature.substring(0, 20)}...`);

    // X25519 ownership proof
    const x25519Message = ethers.solidityPackedKeccak256(
      ["string", "bytes", "uint256", "address", "address"],
      ["SAGE X25519 Ownership:", x25519Key, chainId, registryAddress, defaultSigner.address]
    );
    const x25519Signature = await defaultSigner.signMessage(ethers.getBytes(x25519Message));
    console.log(`✓ X25519 signature generated: ${x25519Signature.substring(0, 20)}...`);

    // Step 3: Register
    const params = {
      did: `did:sage:ethereum:${agentAddress}`,
      name: "Official Payment Agent",
      description: "Blockchain-based secure payment processing agent",
      endpoint: "http://localhost:19083",
      capabilities: "crypto_payment,stablecoin,rfc9421_signature,hpke_encryption",
      keys,
      keyTypes: [0, 2],  // ECDSA, X25519
      signatures: [ecdsaSignature, x25519Signature],
      salt
    };

    console.log(`\n📤 Attempting registration...`);
    console.log(`  DID: ${params.did}`);
    console.log(`  Keys: ${params.keys.length}`);
    console.log(`  KeyTypes: [${params.keyTypes}]`);
    console.log(`  Signatures: ${params.signatures.length}`);

    const registerTx = await registry.registerAgentWithParams(params);
    const receipt = await registerTx.wait();

    console.log(`\n✅ Registration successful!`);
    console.log(`  Block: ${receipt.blockNumber}`);
    console.log(`  Gas used: ${receipt.gasUsed.toString()}`);

    // Query agent
    const agentIds = await registry.getAgentsByOwner(defaultSigner.address);
    console.log(`\n📋 Registered agents: ${agentIds.length}`);

  } catch (error) {
    console.error("\n❌ Registration failed!");
    console.error("Error:", error.message);

    if (error.data) {
      console.error("Error data:", error.data);
    }

    if (error.transaction) {
      console.error("\nTransaction details:");
      console.error("  From:", error.transaction.from);
      console.error("  To:", error.transaction.to);
      console.error("  Data length:", error.transaction.data.length);
    }

    // Try to decode revert reason
    try {
      const reason = ethers.toUtf8String("0x" + error.data.slice(138));
      console.error("\nRevert reason:", reason);
    } catch (e) {
      console.error("\nCould not decode revert reason");
    }
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
