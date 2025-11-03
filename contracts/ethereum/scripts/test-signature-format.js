import hre from "hardhat";
import { network } from "hardhat";

const { ethers } = await network.connect();

async function main() {
  console.log("\n🧪 Testing Signature Format\n");

  const [signer] = await ethers.getSigners();
  console.log(`Signer: ${signer.address}`);

  const chainId = 31337;
  const contractAddress = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";

  // Build message using abi.encodePacked format
  // keccak256(abi.encodePacked("SAGE Agent Registration:", chainid, contract, owner))
  const messageHash = ethers.solidityPackedKeccak256(
    ["string", "uint256", "address", "address"],
    ["SAGE Agent Registration:", chainId, contractAddress, signer.address]
  );

  console.log(`Message hash: ${messageHash}`);

  // Sign the message hash
  // ethers will automatically add "\x19Ethereum Signed Message:\n32" prefix
  const signature = await signer.signMessage(ethers.getBytes(messageHash));

  console.log(`Signature: ${signature}`);
  console.log(`Signature length: ${signature.length}`);

  // Verify the signature
  const recoveredAddress = ethers.verifyMessage(
    ethers.getBytes(messageHash),
    signature
  );

  console.log(`\nRecovered address: ${recoveredAddress}`);
  console.log(`Expected address: ${signer.address}`);
  console.log(`Match: ${recoveredAddress.toLowerCase() === signer.address.toLowerCase()}`);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
