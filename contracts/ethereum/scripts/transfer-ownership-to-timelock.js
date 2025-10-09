/**
 * Transfer Ownership of SAGE Contracts to Timelock
 *
 * This script transfers ownership of all SAGE contracts to the TimelockController,
 * ensuring that all admin functions require multi-sig approval with time delay.
 *
 * Usage:
 *   npx hardhat run scripts/transfer-ownership-to-timelock.js --network sepolia
 *
 * Prerequisites:
 *   - Multi-sig and Timelock already deployed
 *   - Deployer is current owner of all contracts
 *   - Set TIMELOCK_ADDRESS in .env or use governance deployment file
 */

const { ethers } = require("hardhat");
const fs = require("fs");
const path = require("path");

// Contract addresses (will be loaded from deployment files)
let CONTRACTS = {
    SAGE_REGISTRY_V2: "",
    ERC8004_IDENTITY_REGISTRY: "",
    ERC8004_REPUTATION_REGISTRY: "",
    ERC8004_VALIDATION_REGISTRY: "",
    TIMELOCK: "",
    MULTISIG: ""
};

/**
 * Load governance deployment info
 */
function loadGovernanceDeployment(network) {
    const deployDir = path.join(__dirname, "..", "deployments", "governance");
    const latestFile = path.join(deployDir, `governance-${network}-latest.json`);

    if (!fs.existsSync(latestFile)) {
        console.log(`⚠️  Governance deployment file not found: ${latestFile}`);
        return null;
    }

    const deployment = JSON.parse(fs.readFileSync(latestFile, "utf8"));
    return deployment;
}

/**
 * Load contract deployment info
 */
function loadContractDeployments(network) {
    const deployDir = path.join(__dirname, "..", "deployments");

    // Try to find latest deployment file
    const possibleFiles = [
        `sepolia-deployment-latest.json`,
        `${network}-deployment-latest.json`,
        `erc8004-${network}-latest.json`
    ];

    for (const filename of possibleFiles) {
        const filepath = path.join(deployDir, filename);
        if (fs.existsSync(filepath)) {
            console.log(`📄 Loading contracts from: ${filename}`);
            return JSON.parse(fs.readFileSync(filepath, "utf8"));
        }
    }

    return null;
}

/**
 * Verify current ownership
 */
async function verifyCurrentOwnership(contractAddress, contractName) {
    console.log(`\n🔍 Verifying ${contractName} ownership...`);

    try {
        const contract = await ethers.getContractAt("Ownable2Step", contractAddress);
        const currentOwner = await contract.owner();

        const [deployer] = await ethers.getSigners();
        const isOwner = currentOwner.toLowerCase() === deployer.address.toLowerCase();

        console.log(`   Current owner: ${currentOwner}`);
        console.log(`   Deployer: ${deployer.address}`);
        console.log(`   Deployer is owner: ${isOwner ? '✅' : '❌'}`);

        return { currentOwner, isOwner };
    } catch (error) {
        console.log(`   ❌ Error checking ownership: ${error.message}`);
        return { currentOwner: null, isOwner: false };
    }
}

/**
 * Transfer ownership using Ownable2Step
 */
async function transferOwnership(contractAddress, contractName, newOwner) {
    console.log(`\n📝 Transferring ${contractName} ownership...`);
    console.log(`   From: current owner (via deployer)`);
    console.log(`   To: ${newOwner}`);

    try {
        const contract = await ethers.getContractAt("Ownable2Step", contractAddress);

        // Step 1: Transfer ownership (propose)
        console.log(`\n   Step 1: Proposing ownership transfer...`);
        const tx1 = await contract.transferOwnership(newOwner);
        await tx1.wait();
        console.log(`   ✅ Ownership transfer proposed`);
        console.log(`      Tx: ${tx1.hash}`);

        // Check pending owner
        const pendingOwner = await contract.pendingOwner();
        console.log(`   ✅ Pending owner: ${pendingOwner}`);

        // Step 2: Accept ownership (must be done by new owner - Timelock)
        console.log(`\n   Step 2: Accepting ownership...`);
        console.log(`   ⚠️  This requires Timelock to call acceptOwnership()`);
        console.log(`   ⚠️  Use multi-sig to propose this transaction through Timelock`);

        return {
            success: true,
            pendingOwner,
            acceptanceRequired: true
        };
    } catch (error) {
        console.log(`   ❌ Error: ${error.message}`);
        return { success: false, error: error.message };
    }
}

/**
 * Create acceptance transaction data for Timelock
 */
function createAcceptanceTransaction(contractAddress, contractName, timelockAddress) {
    const contract = new ethers.Interface([
        "function acceptOwnership()"
    ]);

    const calldata = contract.encodeFunctionData("acceptOwnership", []);

    const transaction = {
        target: contractAddress,
        value: 0,
        data: calldata,
        predecessor: ethers.ZeroHash,
        salt: ethers.ZeroHash,
        delay: 2 * 24 * 60 * 60  // 48 hours
    };

    console.log(`\n📋 Acceptance Transaction for ${contractName}:`);
    console.log(`   Target: ${contractAddress}`);
    console.log(`   Value: 0`);
    console.log(`   Data: ${calldata}`);
    console.log(`   Delay: 48 hours`);

    return transaction;
}

/**
 * Save ownership transfer info
 */
function saveTransferInfo(network, transfers) {
    const transferInfo = {
        network: network.name,
        chainId: network.chainId,
        timestamp: new Date().toISOString(),
        transfers
    };

    const deployDir = path.join(__dirname, "..", "deployments", "governance");
    if (!fs.existsSync(deployDir)) {
        fs.mkdirSync(deployDir, { recursive: true });
    }

    const filename = `ownership-transfer-${network.name}-${Date.now()}.json`;
    const filepath = path.join(deployDir, filename);
    fs.writeFileSync(filepath, JSON.stringify(transferInfo, null, 2));

    console.log(`\n💾 Transfer info saved to: ${filepath}`);

    return filepath;
}

/**
 * Main function
 */
async function main() {
    console.log("=".repeat(70));
    console.log("SAGE Ownership Transfer to Timelock");
    console.log("=".repeat(70));

    // Get network info
    const network = await ethers.provider.getNetwork();
    console.log(`\n🌐 Network: ${network.name}`);
    console.log(`   Chain ID: ${network.chainId}`);

    // Get deployer
    const [deployer] = await ethers.getSigners();
    console.log(`\n👤 Deployer: ${deployer.address}`);

    // Load governance deployment
    const governance = loadGovernanceDeployment(network.name);
    if (!governance) {
        console.log("\n❌ Error: Governance deployment not found!");
        console.log("   Please deploy multi-sig and timelock first:");
        console.log("   npx hardhat run scripts/deploy-multisig-governance.js");
        process.exit(1);
    }

    console.log(`\n📋 Governance Contracts:`);
    console.log(`   Multi-sig: ${governance.multiSig.address}`);
    console.log(`   Timelock: ${governance.timelock.address}`);

    CONTRACTS.TIMELOCK = governance.timelock.address;
    CONTRACTS.MULTISIG = governance.multiSig.address;

    // Load contract deployments
    const contracts = loadContractDeployments(network.name);
    if (contracts) {
        CONTRACTS.SAGE_REGISTRY_V2 = contracts.sageRegistry || process.env.SAGE_REGISTRY_ADDRESS;
        CONTRACTS.ERC8004_IDENTITY_REGISTRY = contracts.identityRegistry || process.env.IDENTITY_REGISTRY_ADDRESS;
        CONTRACTS.ERC8004_REPUTATION_REGISTRY = contracts.reputationRegistry || process.env.REPUTATION_REGISTRY_ADDRESS;
        CONTRACTS.ERC8004_VALIDATION_REGISTRY = contracts.validationRegistry || process.env.VALIDATION_REGISTRY_ADDRESS;
    }

    console.log(`\n📋 SAGE Contracts:`);
    console.log(`   SageRegistryV2: ${CONTRACTS.SAGE_REGISTRY_V2 || '⚠️  Not set'}`);
    console.log(`   IdentityRegistry: ${CONTRACTS.ERC8004_IDENTITY_REGISTRY || '⚠️  Not set'}`);
    console.log(`   ReputationRegistry: ${CONTRACTS.ERC8004_REPUTATION_REGISTRY || '⚠️  Not set'}`);
    console.log(`   ValidationRegistry: ${CONTRACTS.ERC8004_VALIDATION_REGISTRY || '⚠️  Not set'}`);

    // Verify all addresses are set
    const contractsToTransfer = [];
    if (CONTRACTS.SAGE_REGISTRY_V2) {
        contractsToTransfer.push({ address: CONTRACTS.SAGE_REGISTRY_V2, name: "SageRegistryV2" });
    }
    if (CONTRACTS.ERC8004_IDENTITY_REGISTRY) {
        contractsToTransfer.push({ address: CONTRACTS.ERC8004_IDENTITY_REGISTRY, name: "IdentityRegistry" });
    }
    if (CONTRACTS.ERC8004_REPUTATION_REGISTRY) {
        contractsToTransfer.push({ address: CONTRACTS.ERC8004_REPUTATION_REGISTRY, name: "ReputationRegistry" });
    }
    if (CONTRACTS.ERC8004_VALIDATION_REGISTRY) {
        contractsToTransfer.push({ address: CONTRACTS.ERC8004_VALIDATION_REGISTRY, name: "ValidationRegistry" });
    }

    if (contractsToTransfer.length === 0) {
        console.log("\n❌ Error: No contracts to transfer!");
        console.log("   Please deploy SAGE contracts first or set addresses in .env");
        process.exit(1);
    }

    console.log(`\n📝 Contracts to transfer: ${contractsToTransfer.length}`);

    // Verify current ownership
    const ownershipChecks = [];
    for (const contract of contractsToTransfer) {
        const check = await verifyCurrentOwnership(contract.address, contract.name);
        ownershipChecks.push({ ...contract, ...check });
    }

    // Transfer ownership
    const transfers = [];
    const acceptanceTransactions = [];

    for (const contract of ownershipChecks) {
        if (!contract.isOwner) {
            console.log(`\n⚠️  Skipping ${contract.name} - deployer is not owner`);
            continue;
        }

        const result = await transferOwnership(
            contract.address,
            contract.name,
            CONTRACTS.TIMELOCK
        );

        transfers.push({
            contract: contract.name,
            address: contract.address,
            oldOwner: contract.currentOwner,
            newOwner: CONTRACTS.TIMELOCK,
            status: result.success ? "pending_acceptance" : "failed",
            ...result
        });

        if (result.success) {
            const acceptTx = createAcceptanceTransaction(
                contract.address,
                contract.name,
                CONTRACTS.TIMELOCK
            );
            acceptanceTransactions.push({
                contract: contract.name,
                ...acceptTx
            });
        }
    }

    // Save transfer info
    saveTransferInfo(network, transfers);

    // Final summary
    console.log("\n" + "=".repeat(70));
    console.log("✅ Ownership Transfer Initiated!");
    console.log("=".repeat(70));

    console.log(`\n📊 Transfer Summary:`);
    transfers.forEach((t, i) => {
        console.log(`   ${i + 1}. ${t.contract}: ${t.status}`);
    });

    console.log(`\n⚠️  IMPORTANT: Next Steps Required!`);
    console.log(`\n   The ownership transfer is a 2-step process:`);
    console.log(`   1. ✅ transferOwnership() called - DONE`);
    console.log(`   2. ⏳ acceptOwnership() must be called by Timelock - PENDING`);

    console.log(`\n   To complete the transfer:`);
    console.log(`   a) Multi-sig proposes acceptOwnership() to Timelock`);
    console.log(`   b) Multi-sig signers approve (need ${governance.multiSig.threshold}/${governance.multiSig.signers.length})`);
    console.log(`   c) Wait ${governance.timelock.minDelay / 3600} hours (Timelock delay)`);
    console.log(`   d) Execute acceptOwnership() through Timelock`);

    console.log(`\n📋 Acceptance Transactions to Propose:`);
    acceptanceTransactions.forEach((tx, i) => {
        console.log(`\n   ${i + 1}. ${tx.contract}:`);
        console.log(`      schedule(`);
        console.log(`        target: ${tx.target},`);
        console.log(`        value: ${tx.value},`);
        console.log(`        data: ${tx.data},`);
        console.log(`        predecessor: ${tx.predecessor},`);
        console.log(`        salt: ${tx.salt},`);
        console.log(`        delay: ${tx.delay}`);
        console.log(`      )`);
    });

    console.log(`\n💡 Helper Script:`);
    console.log(`   Use: npx hardhat run scripts/accept-ownership.js --network ${network.name}`);

    console.log("\n" + "=".repeat(70) + "\n");
}

// Execute
main()
    .then(() => process.exit(0))
    .catch((error) => {
        console.error("\n❌ Ownership transfer failed:");
        console.error(error);
        process.exit(1);
    });
