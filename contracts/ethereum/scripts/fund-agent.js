import hre from "hardhat";
import { network } from "hardhat";

// Initialize ethers from network connection (Hardhat 3.x pattern)
const { ethers } = await network.connect();

async function main() {
  console.log("\n💰 Funding Agent Address");
  console.log("=".repeat(80));

  // Get default signer (first Hardhat account with 10000 ETH)
  const [signer] = await ethers.getSigners();
  console.log(`  Funding from: ${signer.address}`);

  // Payment Agent address
  const agentAddress = "0xE230795E3DDef701fe38dB02D70E796d352068a0";
  console.log(`  Funding to: ${agentAddress}`);

  // Send 10 ETH to agent
  const amount = ethers.parseEther("10.0");
  console.log(`  Amount: ${ethers.formatEther(amount)} ETH`);

  const tx = await signer.sendTransaction({
    to: agentAddress,
    value: amount,
  });

  console.log(`  Transaction sent: ${tx.hash}`);
  await tx.wait();
  console.log(`  ✓ Transaction confirmed`);

  // Check balance
  const balance = await ethers.provider.getBalance(agentAddress);
  console.log(`  Agent balance: ${ethers.formatEther(balance)} ETH`);

  console.log("\n" + "=".repeat(80));
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
