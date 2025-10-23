const hre = require("hardhat");
const { ethers } = require("hardhat");

async function main() {
    console.log("\n=== Testing Update Signature ===\n");
    
    const [deployer] = await ethers.getSigners();
    console.log("Deployer:", deployer.address);
    
    // Get deployed contract
    const contractAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3";
    const SageRegistryV4 = await ethers.getContractFactory("SageRegistryV4");
    const contract = SageRegistryV4.attach(contractAddress);
    
    // Create test wallet (agent)
    const agentWallet = ethers.Wallet.createRandom().connect(ethers.provider);
    console.log("Agent address:", agentWallet.address);
    
    // Fund agent
    await deployer.sendTransaction({
        to: agentWallet.address,
        value: ethers.parseEther("10")
    });
    console.log("Agent funded");
    
    // Prepare registration data
    const did = "did:sage:ethereum:test-update-001";
    const name = "Test Agent";
    const description = "Initial description";
    const endpoint = "http://localhost:8080";
    const capabilities = JSON.stringify({version: "1.0.0"});
    
    // Get agent's public key (uncompressed, 64 bytes without 0x04 prefix)
    // ethers v6 provides signingKey.publicKey which includes 0x04 prefix
    const fullPublicKey = agentWallet.signingKey.publicKey; // "0x04" + 64 bytes
    const keyData = ethers.getBytes("0x" + fullPublicKey.slice(4)); // Remove "0x04"
    
    console.log("\nPublic key length:", keyData.length);
    console.log("Public key:", ethers.hexlify(keyData).slice(0, 20) + "...");
    
    // Calculate agentId
    const agentId = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
            ["string", "bytes"],
            [did, keyData]
        )
    );
    console.log("Agent ID:", agentId);
    
    // Generate registration signature
    const regMessageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
            ["bytes32", "bytes", "address", "uint256"],
            [agentId, keyData, agentWallet.address, 0]
        )
    );
    
    const regSignature = await agentWallet.signMessage(ethers.getBytes(regMessageHash));
    
    console.log("\n=== Registration ===");
    console.log("Message hash:", regMessageHash);
    console.log("Signature:", regSignature.slice(0, 20) + "...");
    
    // Register agent
    const regTx = await contract.connect(agentWallet).registerAgent({
        did: did,
        name: name,
        description: description,
        endpoint: endpoint,
        keyTypes: [1], // ECDSA
        keyData: [keyData],
        signatures: [regSignature],
        capabilities: capabilities
    });
    await regTx.wait();
    console.log("✓ Agent registered");
    
    // Get nonce
    const nonce = await contract.getNonce(agentId);
    console.log("\nCurrent nonce:", nonce.toString());
    
    // Prepare update
    const newName = "Updated Agent";
    const newDescription = "Updated description";
    const newEndpoint = "http://localhost:9090";
    const newCapabilities = JSON.stringify({version: "2.0.0"});
    
    // Generate update signature
    const updateMessageHash = ethers.keccak256(
        ethers.AbiCoder.defaultAbiCoder().encode(
            ["bytes32", "string", "string", "string", "string", "address", "uint256"],
            [agentId, newName, newDescription, newEndpoint, newCapabilities, agentWallet.address, nonce]
        )
    );
    
    const updateSignature = await agentWallet.signMessage(ethers.getBytes(updateMessageHash));
    
    console.log("\n=== Update ===");
    console.log("Message hash:", updateMessageHash);
    console.log("Signature:", updateSignature.slice(0, 20) + "...");
    
    // Try update
    try {
        const updateTx = await contract.connect(agentWallet).updateAgent(
            agentId,
            newName,
            newDescription,
            newEndpoint,
            newCapabilities,
            updateSignature
        );
        await updateTx.wait();
        console.log("✓ Update successful!");
        
        // Verify
        const agent = await contract.getAgent(agentId);
        console.log("\nUpdated agent name:", agent.name);
        console.log("Updated nonce:", (await contract.getNonce(agentId)).toString());
        
    } catch (error) {
        console.log("✗ Update failed:");
        console.log(error.message);
    }
}

main()
    .then(() => process.exit(0))
    .catch((error) => {
        console.error(error);
        process.exit(1);
    });
