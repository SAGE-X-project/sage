const hre = require("hardhat");

async function main() {
  console.log("Deploying SAGE Registry contracts...");

  // Deploy the main registry
  const SageRegistry = await hre.ethers.getContractFactory("SageRegistry");
  const registry = await SageRegistry.deploy();
  await registry.waitForDeployment();
  const registryAddress = await registry.getAddress();
  console.log("SageRegistry deployed to:", registryAddress);

  // Deploy the verification hook
  const SageVerificationHook = await hre.ethers.getContractFactory("SageVerificationHook");
  const verificationHook = await SageVerificationHook.deploy();
  await verificationHook.waitForDeployment();
  const hookAddress = await verificationHook.getAddress();
  console.log("SageVerificationHook deployed to:", hookAddress);

  // Set the verification hook in the registry
  console.log("Setting verification hook...");
  const setHookTx = await registry.setBeforeRegisterHook(hookAddress);
  await setHookTx.wait();
  console.log("Verification hook set successfully");

  // Save deployment addresses
  const fs = require("fs");
  const deploymentInfo = {
    network: hre.network.name,
    contracts: {
      SageRegistry: registryAddress,
      SageVerificationHook: hookAddress,
    },
    deployer: (await hre.ethers.getSigners())[0].address,
    timestamp: new Date().toISOString(),
  };

  fs.writeFileSync(
    `deployments/${hre.network.name}.json`,
    JSON.stringify(deploymentInfo, null, 2)
  );

  console.log("\nDeployment complete!");
  console.log("Deployment info saved to:", `deployments/${hre.network.name}.json`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});