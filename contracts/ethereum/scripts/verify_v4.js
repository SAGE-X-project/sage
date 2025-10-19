const hre = require("hardhat");
const fs = require("fs");
const path = require("path");

/**
 * Verify SageRegistryV4 contract on Etherscan
 *
 * Usage:
 *   npx hardhat run scripts/verify_v4.js --network sepolia
 *   npx hardhat run scripts/verify_v4.js --network mainnet
 *
 * Environment variables:
 *   CONTRACT_ADDRESS - Contract address to verify (required if not in deployment file)
 *   DEPLOYMENT_FILE - Path to deployment JSON file (optional)
 */
async function main() {
  console.log("\n╔═══════════════════════════════════════════════════════════╗");
  console.log("║     SageRegistryV4 Verification Script                   ║");
  console.log("╚═══════════════════════════════════════════════════════════╝\n");

  const network = hre.network.name;
  const chainId = (await hre.ethers.provider.getNetwork()).chainId;

  console.log("📋 Verification Configuration");
  console.log("─".repeat(60));
  console.log("Network:  ", network);
  console.log("Chain ID: ", chainId.toString());
  console.log();

  // Get contract address
  let contractAddress = process.env.CONTRACT_ADDRESS;

  // Try to load from deployment file if not provided
  if (!contractAddress) {
    const deploymentFile = process.env.DEPLOYMENT_FILE ||
                          path.join(__dirname, "..", "deployments", `v4_${network}.json`);

    if (fs.existsSync(deploymentFile)) {
      console.log("📂 Loading deployment info from:", deploymentFile);
      const deploymentInfo = JSON.parse(fs.readFileSync(deploymentFile, "utf8"));
      contractAddress = deploymentInfo.contractAddress;
      console.log("   Found contract address:", contractAddress);
      console.log();
    }
  }

  if (!contractAddress) {
    console.error("❌ Contract address not found");
    console.error("   Please provide CONTRACT_ADDRESS environment variable");
    console.error("   or ensure deployment file exists at:");
    console.error("   contracts/ethereum/deployments/v4_" + network + ".json");
    process.exit(1);
  }

  // Verify that the contract exists
  console.log("🔍 Checking if contract exists at", contractAddress);
  const code = await hre.ethers.provider.getCode(contractAddress);

  if (code === "0x") {
    console.error("❌ No contract found at address:", contractAddress);
    console.error("   Please check the address and network");
    process.exit(1);
  }

  console.log("✅ Contract code found (", code.length, "bytes)");
  console.log();

  // Check if already verified
  console.log("🔍 Checking if contract is already verified...");
  try {
    const alreadyVerified = await isContractVerified(contractAddress, network);
    if (alreadyVerified) {
      console.log("ℹ️  Contract is already verified on Etherscan");
      console.log();
      displayExplorerLink(contractAddress, network);
      return;
    }
  } catch (error) {
    console.log("⚠️  Could not check verification status, continuing...");
    console.log();
  }

  // Prepare constructor arguments
  // SageRegistryV4 has no constructor arguments
  const constructorArguments = [];

  console.log("📝 Verification Parameters");
  console.log("─".repeat(60));
  console.log("Contract:      SageRegistryV4");
  console.log("Address:       ", contractAddress);
  console.log("Constructor:   ", constructorArguments.length === 0 ? "(no arguments)" : JSON.stringify(constructorArguments));
  console.log();

  // Verify contract on Etherscan
  console.log("🚀 Submitting verification to Etherscan...");
  console.log("   This may take a few moments...");
  console.log();

  try {
    await hre.run("verify:verify", {
      address: contractAddress,
      constructorArguments: constructorArguments,
      contract: "contracts/SageRegistryV4.sol:SageRegistryV4"
    });

    console.log("✅ Contract verified successfully!");
    console.log();

    // Save verification info
    const verificationInfo = {
      network: network,
      chainId: chainId.toString(),
      contractAddress: contractAddress,
      verified: true,
      verifiedAt: new Date().toISOString(),
      constructorArguments: constructorArguments
    };

    const verificationFile = path.join(
      __dirname,
      "..",
      "deployments",
      `v4_${network}_verification.json`
    );

    fs.writeFileSync(verificationFile, JSON.stringify(verificationInfo, null, 2));
    console.log("💾 Verification info saved to:", verificationFile);
    console.log();

  } catch (error) {
    if (error.message.includes("Already Verified")) {
      console.log("ℹ️  Contract is already verified on Etherscan");
      console.log();
    } else if (error.message.includes("does not have bytecode")) {
      console.error("❌ Contract not found at address:", contractAddress);
      console.error("   Please check the address and network");
      process.exit(1);
    } else {
      console.error("❌ Verification failed:", error.message);
      console.error();
      console.error("Troubleshooting:");
      console.error("1. Ensure ETHERSCAN_API_KEY is set in .env file");
      console.error("2. Wait a few blocks after deployment before verifying");
      console.error("3. Check that the contract source matches exactly");
      console.error("4. Try again in a few minutes if Etherscan is busy");
      console.error();
      process.exit(1);
    }
  }

  // Display explorer link
  console.log("╔═══════════════════════════════════════════════════════════╗");
  console.log("║     Verification Complete!                                ║");
  console.log("╚═══════════════════════════════════════════════════════════╝\n");

  displayExplorerLink(contractAddress, network);

  console.log("📋 Next Steps:");
  console.log();
  console.log("1. View the verified contract source on Etherscan");
  console.log("2. Test read functions directly on Etherscan");
  console.log("3. Update documentation with verified contract address");
  console.log("4. Share the Etherscan link in DEPLOYED_ADDRESSES.md");
  console.log();
}

/**
 * Check if contract is already verified on Etherscan
 */
async function isContractVerified(address, network) {
  // This is a best-effort check using the Etherscan API
  // Returns true if verified, false if not, throws on error

  let apiUrl;
  let apiKey = process.env.ETHERSCAN_API_KEY;

  if (!apiKey) {
    throw new Error("ETHERSCAN_API_KEY not set");
  }

  switch (network) {
    case "mainnet":
      apiUrl = "https://api.etherscan.io/api";
      break;
    case "sepolia":
      apiUrl = "https://api-sepolia.etherscan.io/api";
      break;
    case "goerli":
      apiUrl = "https://api-goerli.etherscan.io/api";
      break;
    default:
      throw new Error("Unsupported network for verification: " + network);
  }

  const url = `${apiUrl}?module=contract&action=getsourcecode&address=${address}&apikey=${apiKey}`;

  const response = await fetch(url);
  const data = await response.json();

  if (data.status === "1" && data.result && data.result[0]) {
    const sourceCode = data.result[0].SourceCode;
    return sourceCode && sourceCode.length > 0;
  }

  return false;
}

/**
 * Display Etherscan explorer link
 */
function displayExplorerLink(address, network) {
  let explorerUrl = "";
  let explorerName = "";

  switch (network) {
    case "mainnet":
      explorerUrl = `https://etherscan.io/address/${address}#code`;
      explorerName = "Etherscan";
      break;
    case "sepolia":
      explorerUrl = `https://sepolia.etherscan.io/address/${address}#code`;
      explorerName = "Sepolia Etherscan";
      break;
    case "goerli":
      explorerUrl = `https://goerli.etherscan.io/address/${address}#code`;
      explorerName = "Goerli Etherscan";
      break;
    default:
      return;
  }

  console.log("🔍 View on", explorerName + ":");
  console.log("   " + explorerUrl);
  console.log();
}

// Execute verification
if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error("\n❌ Verification failed:", error);
      process.exit(1);
    });
}

module.exports = { main };
