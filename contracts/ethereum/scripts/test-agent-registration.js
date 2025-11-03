import hre from "hardhat";
import { network } from "hardhat";

const { ethers } = await network.connect();

async function main() {
  console.log("\n🧪 Testing Agent Registration with Actual Keys\n");

  // Agent's actual private key
  const agentPrivateKey = "0x775d254fa83309f2fb69a3ccbf272ed202c1a0a395b2bee01e758045d3be8d32";
  const agentWallet = new ethers.Wallet(agentPrivateKey, ethers.provider);

  console.log(`Agent Address: ${agentWallet.address}`);

  // Fund agent if needed
  const [funder] = await ethers.getSigners();
  const balance = await ethers.provider.getBalance(agentWallet.address);
  console.log(`Agent Balance: ${ethers.formatEther(balance)} ETH`);

  if (balance < ethers.parseEther("0.02")) {
    console.log(`Funding agent...`);
    const tx = await funder.sendTransaction({
      to: agentWallet.address,
      value: ethers.parseEther("1.0")
    });
    await tx.wait();
    console.log(`✓ Funded agent with 1 ETH`);
  }

  // Get agent's public key
  const publicKey = ethers.SigningKey.computePublicKey(agentPrivateKey, false);  // false = uncompressed
  console.log(`\nAgent Public Key: ${publicKey}`);
  console.log(`Public Key Length: ${publicKey.length} chars (${(publicKey.length - 2) / 2} bytes)`);

  // X25519 public key (from agent logs)
  const x25519Key = "0x64f44fe9b266af6f062575f075202b0cefb22c3dfa9560c92de32d9163383538";

  const keys = [publicKey, x25519Key];

  // Get registry
  const registryAddress = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";
  const Registry = await ethers.getContractFactory("AgentCardRegistry");
  const registry = Registry.attach(registryAddress).connect(agentWallet);

  const chainId = 31337;
  const did = `did:sage:ethereum:${agentWallet.address}`;

  // Generate salt
  const salt = ethers.randomBytes(32);

  // Create commitment
  const commitHash = ethers.keccak256(
    ethers.AbiCoder.defaultAbiCoder().encode(
      ["string", "bytes[]", "address", "bytes32", "uint256"],
      [did, keys, agentWallet.address, salt, chainId]
    )
  );

  try {
    // Step 1: Commit
    console.log(`\n📝 Committing...`);
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
      ["SAGE Agent Registration:", chainId, registryAddress, agentWallet.address]
    );
    const ecdsaSignature = await agentWallet.signMessage(ethers.getBytes(ecdsaMessage));
    console.log(`\n✓ ECDSA signature generated`);

    // X25519 ownership proof
    const x25519Message = ethers.solidityPackedKeccak256(
      ["string", "bytes", "uint256", "address", "address"],
      ["SAGE X25519 Ownership:", x25519Key, chainId, registryAddress, agentWallet.address]
    );
    const x25519Signature = await agentWallet.signMessage(ethers.getBytes(x25519Message));
    console.log(`✓ X25519 signature generated`);

    // Step 3: Register
    const params = {
      did,
      name: "Official Payment Agent",
      description: "Blockchain-based secure payment processing agent",
      endpoint: "http://localhost:19083",
      capabilities: "crypto_payment,stablecoin,rfc9421_signature,hpke_encryption",
      keys,
      keyTypes: [0, 2],  // ECDSA, X25519
      signatures: [ecdsaSignature, x25519Signature],
      salt
    };

    console.log(`\n📤 Registering agent...`);
    const registerTx = await registry.registerAgentWithParams(params);
    const receipt = await registerTx.wait();

    console.log(`\n✅ SUCCESS! Agent registered on blockchain!`);
    console.log(`  Block: ${receipt.blockNumber}`);
    console.log(`  Gas used: ${receipt.gasUsed.toString()}`);
    console.log(`  TX: ${receipt.hash}`);

    // Query agent
    const agentIds = await registry.getAgentsByOwner(agentWallet.address);
    console.log(`\n📋 Total registered agents: ${agentIds.length}`);
    if (agentIds.length > 0) {
      console.log(`  Agent ID: ${agentIds[0]}`);
    }

  } catch (error) {
    console.error("\n❌ Registration failed!");
    console.error("Error:", error.message);

    if (error.data) {
      console.error("\nError data:", error.data);
    }

    // Try to get revert reason
    if (error.reason) {
      console.error("Revert reason:", error.reason);
    }
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
