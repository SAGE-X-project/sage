const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

async function main() {
  console.log("🚀 Deploying SageRegistryV2 with Enhanced Public Key Validation...");
  console.log("=" .repeat(60));
  
  // Get the deployer account
  const [deployer] = await hre.ethers.getSigners();
  const network = await hre.ethers.provider.getNetwork();
  
  console.log("📍 Deploying to network:", network.name);
  console.log("👤 Deployer address:", deployer.address);
  
  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("💰 Deployer balance:", hre.ethers.formatEther(balance), "ETH");
  console.log();

  // Deploy SageRegistryV2
  console.log("📝 Deploying SageRegistryV2...");
  const SageRegistryV2 = await hre.ethers.getContractFactory("SageRegistryV2");
  const sageRegistry = await SageRegistryV2.deploy();
  await sageRegistry.waitForDeployment();
  
  const registryAddress = await sageRegistry.getAddress();
  console.log("✅ SageRegistryV2 deployed to:", registryAddress);
  console.log();

  // Deploy SageVerificationHook
  console.log("📝 Deploying SageVerificationHook...");
  const SageVerificationHook = await hre.ethers.getContractFactory("SageVerificationHook");
  const verificationHook = await SageVerificationHook.deploy();
  await verificationHook.waitForDeployment();
  
  const hookAddress = await verificationHook.getAddress();
  console.log("✅ SageVerificationHook deployed to:", hookAddress);
  console.log();

  // Configure hooks
  console.log("🔧 Configuring hooks...");
  const tx = await sageRegistry.setBeforeRegisterHook(hookAddress);
  await tx.wait();
  console.log("✅ BeforeRegisterHook set to:", hookAddress);
  console.log();

  // Save deployment info
  const deploymentInfo = {
    network: network.name,
    chainId: network.chainId.toString(),
    deployedAt: new Date().toISOString(),
    contracts: {
      SageRegistryV2: {
        address: registryAddress,
        deployer: deployer.address
      },
      SageVerificationHook: {
        address: hookAddress,
        deployer: deployer.address
      }
    },
    enhancements: {
      publicKeyValidation: {
        formatValidation: true,
        zeroKeyPrevention: true,
        ownershipProof: true,
        keyRevocation: true,
        ed25519Rejection: true
      },
      gasUsage: {
        registration: "~620K gas",
        update: "~50K gas",
        revocation: "~30K gas"
      }
    }
  };

  const deploymentPath = path.join(__dirname, "../deployments");
  if (!fs.existsSync(deploymentPath)) {
    fs.mkdirSync(deploymentPath, { recursive: true });
  }
  
  const fileName = `${network.name}-v2-${Date.now()}.json`;
  fs.writeFileSync(
    path.join(deploymentPath, fileName),
    JSON.stringify(deploymentInfo, null, 2)
  );

  console.log("=" .repeat(60));
  console.log("🎉 Deployment Complete!");
  console.log("=" .repeat(60));
  console.log();
  console.log("📋 Summary:");
  console.log("  • SageRegistryV2:", registryAddress);
  console.log("  • SageVerificationHook:", hookAddress);
  console.log("  • Network:", network.name);
  console.log("  • Chain ID:", network.chainId);
  console.log();
  console.log("✨ Key Improvements:");
  console.log("  ✅ Public key format validation (0x04, 0x02, 0x03)");
  console.log("  ✅ Zero key prevention");
  console.log("  ✅ Key ownership proof via signature");
  console.log("  ✅ Key revocation functionality");
  console.log("  ✅ Ed25519 rejection (on-chain limitation)");
  console.log();
  console.log("📝 Deployment info saved to:", path.join(deploymentPath, fileName));
  
  // Verify contracts if on testnet/mainnet
  if (network.name !== "localhost" && network.name !== "hardhat") {
    console.log();
    console.log("🔍 Preparing for contract verification...");
    console.log("Run the following commands to verify:");
    console.log();
    console.log(`npx hardhat verify --network ${network.name} ${registryAddress}`);
    console.log(`npx hardhat verify --network ${network.name} ${hookAddress}`);
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });