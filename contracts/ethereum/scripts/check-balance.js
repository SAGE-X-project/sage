const hre = require("hardhat");

async function main() {
  const [deployer] = await hre.ethers.getSigners();
  const network = await hre.ethers.provider.getNetwork();
  
  console.log("=" .repeat(50));
  console.log(" Account Balance Check");
  console.log("=" .repeat(50));
  console.log("Network:", network.name);
  console.log("Chain ID:", network.chainId);
  console.log("Account:", deployer.address);
  
  const balance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("Balance:", hre.ethers.formatEther(balance), "KLAY");
  console.log("=" .repeat(50));
  
  if (balance === 0n) {
    console.log("  Your account has no KLAY!");
    console.log("Get test KLAY from: https://kairos.wallet.kaia.io/faucet");
  } else if (balance < hre.ethers.parseEther("0.1")) {
    console.log("  Low balance! Consider getting more test KLAY.");
  } else {
    console.log(" Sufficient balance for deployment and testing.");
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });