const hre = require("hardhat");

async function main() {
  const networkName = hre.network.name;
  const deploymentInfo = require(`../deployments/${networkName}.json`);

  console.log(`Verifying contracts on ${networkName}...`);

  // Verify SageRegistry
  try {
    await hre.run("verify:verify", {
      address: deploymentInfo.contracts.SageRegistry,
      constructorArguments: [],
    });
    console.log("SageRegistry verified successfully");
  } catch (error) {
    console.log("SageRegistry verification failed:", error.message);
  }

  // Verify SageVerificationHook
  try {
    await hre.run("verify:verify", {
      address: deploymentInfo.contracts.SageVerificationHook,
      constructorArguments: [],
    });
    console.log("SageVerificationHook verified successfully");
  } catch (error) {
    console.log("SageVerificationHook verification failed:", error.message);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});