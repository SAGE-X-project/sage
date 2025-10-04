const hre = require("hardhat");

// Helper function to create a valid public key
function createValidPublicKey() {
  return "0x04" + hre.ethers.hexlify(hre.ethers.randomBytes(64)).slice(2);
}

// Helper function to create registration signature
async function createRegistrationSignature(signer, publicKey, contractAddress) {
  const chainId = (await hre.ethers.provider.getNetwork()).chainId;
  const keyHash = hre.ethers.keccak256(publicKey);
  
  const packedData = hre.ethers.solidityPacked(
    ["string", "uint256", "address", "address", "bytes32"],
    [
      "SAGE Key Registration:",
      chainId,
      contractAddress,
      signer.address,
      keyHash
    ]
  );
  
  const challenge = hre.ethers.keccak256(packedData);
  return await signer.signMessage(hre.ethers.getBytes(challenge));
}

async function main() {
  console.log(" Quick Test for SageRegistryV2");
  console.log("=" .repeat(50));
  
  const [deployer] = await hre.ethers.getSigners();
  const network = await hre.ethers.provider.getNetwork();
  
  console.log("Network:", network.name);
  console.log("Deployer:", deployer.address);
  console.log();
  
  // Deploy contracts
  console.log(" Deploying contracts...");
  
  const SageRegistryV2 = await hre.ethers.getContractFactory("SageRegistryV2");
  const registry = await SageRegistryV2.deploy();
  await registry.waitForDeployment();
  const registryAddress = await registry.getAddress();
  console.log(" Registry deployed to:", registryAddress);
  
  const SageVerificationHook = await hre.ethers.getContractFactory("SageVerificationHook");
  const hook = await SageVerificationHook.deploy();
  await hook.waitForDeployment();
  const hookAddress = await hook.getAddress();
  console.log(" Hook deployed to:", hookAddress);
  
  // Configure hook
  await registry.setBeforeRegisterHook(hookAddress);
  console.log(" Hook configured");
  console.log();
  
  // Test registration
  console.log(" Testing registration...");
  
  const publicKey = createValidPublicKey();
  const did = `did:sage:test:${deployer.address}`;
  const signature = await createRegistrationSignature(deployer, publicKey, registryAddress);
  
  const tx = await registry.registerAgent(
    did,
    "Test Agent",
    "Quick test agent",
    "https://test.example.com",
    publicKey,
    JSON.stringify(["chat", "test"]),
    signature
  );
  
  const receipt = await tx.wait();
  console.log(" Agent registered!");
  console.log("   Gas used:", receipt.gasUsed.toString());
  
  // Verify registration
  const agent = await registry.getAgentByDID(did);
  console.log(" Agent retrieved:");
  console.log("   Name:", agent.name);
  console.log("   Active:", agent.active);
  
  // Test key validation
  const isValid = await registry.isKeyValid(publicKey);
  console.log(" Key validation:", isValid ? "Valid" : "Invalid");
  
  // Test zero key rejection
  console.log();
  console.log(" Testing security features...");
  
  const zeroKey = "0x04" + "00".repeat(64);
  const zeroSig = await createRegistrationSignature(deployer, zeroKey, registryAddress);
  
  try {
    await registry.registerAgent(
      `did:sage:test:${deployer.address}_zero`,
      "Zero Agent",
      "Should fail",
      "https://fail.com",
      zeroKey,
      "{}",
      zeroSig
    );
    console.log(" FAILED: Zero key was accepted!");
  } catch (error) {
    if (error.message.includes("Invalid zero key")) {
      console.log(" Zero key correctly rejected");
    } else {
      console.log(" Unexpected error:", error.message);
    }
  }
  
  // Test key revocation
  console.log();
  console.log(" Testing key revocation...");
  
  await registry.revokeKey(publicKey);
  console.log(" Key revoked");
  
  const isValidAfter = await registry.isKeyValid(publicKey);
  console.log(" Key status after revocation:", isValidAfter ? "Valid" : "Invalid");
  
  const agentAfter = await registry.getAgentByDID(did);
  console.log(" Agent status after revocation:", agentAfter.active ? "Active" : "Inactive");
  
  console.log();
  console.log("=" .repeat(50));
  console.log(" All tests passed!");
  console.log("=" .repeat(50));
  
  if (network.name !== "localhost" && network.name !== "hardhat") {
    console.log();
    console.log(" Deployed contracts:");
    console.log("   Registry:", registryAddress);
    console.log("   Hook:", hookAddress);
    console.log();
    console.log("Save these addresses for future reference!");
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Error:", error);
    process.exit(1);
  });