/**
 * Deploy Gnosis Safe Multi-sig and TimelockController for SAGE Governance
 *
 * This script deploys:
 * 1. Gnosis Safe multi-sig wallet (3/5 threshold)
 * 2. OpenZeppelin TimelockController (48-hour delay)
 * 3. Configures roles and permissions
 *
 * Usage:
 *   npx hardhat run scripts/deploy-multisig-governance.js --network sepolia
 *
 * Prerequisites:
 *   - Set SIGNER1_ADDRESS, SIGNER2_ADDRESS, ... SIGNER5_ADDRESS in .env
 *   - Ensure deployer has sufficient ETH for gas
 */

const { ethers } = require("hardhat");
const fs = require("fs");
const path = require("path");

// Configuration
const CONFIG = {
    // Multi-sig configuration
    MULTISIG_THRESHOLD: 3,  // Require 3 of 5 signatures
    MULTISIG_SIGNERS_COUNT: 5,

    // Timelock delays
    MIN_DELAY_NORMAL: 2 * 24 * 60 * 60,  // 48 hours for normal operations
    MIN_DELAY_EMERGENCY: 24 * 60 * 60,    // 24 hours for emergency pause

    // Save deployment info
    SAVE_DEPLOYMENT: true,
    DEPLOYMENT_DIR: "./deployments/governance"
};

/**
 * Get signer addresses from environment
 */
function getSignerAddresses() {
    const signers = [];

    for (let i = 1; i <= CONFIG.MULTISIG_SIGNERS_COUNT; i++) {
        const address = process.env[`SIGNER${i}_ADDRESS`];
        if (!address) {
            console.log(`⚠️  Warning: SIGNER${i}_ADDRESS not set in environment`);
            console.log(`   Using deployer address as placeholder`);
            return null;  // Use default addresses for testing
        }
        signers.push(address);
    }

    return signers;
}

/**
 * Deploy Gnosis Safe Multi-sig
 */
async function deployMultiSig(signers, threshold) {
    console.log("\n📦 Deploying Gnosis Safe Multi-sig...");
    console.log(`   Signers: ${signers.length}`);
    console.log(`   Threshold: ${threshold}`);

    // Note: Gnosis Safe requires factory deployment
    // For simplicity, we'll create a custom multi-sig contract
    // In production, use Gnosis Safe SDK or pre-deployed factory

    const MultiSigWallet = await ethers.getContractFactory("SimpleMultiSig");
    const multiSig = await MultiSigWallet.deploy(signers, threshold);
    await multiSig.waitForDeployment();

    const address = await multiSig.getAddress();
    console.log(`✅ Multi-sig deployed at: ${address}`);

    return { address, contract: multiSig };
}

/**
 * Deploy TimelockController
 */
async function deployTimelock(proposers, executors, minDelay) {
    console.log("\n📦 Deploying TimelockController...");
    console.log(`   Min Delay: ${minDelay / 3600} hours`);
    console.log(`   Proposers: ${proposers.length}`);
    console.log(`   Executors: ${executors.length}`);

    const TimelockController = await ethers.getContractFactory("TimelockController");

    // Deploy with multi-sig as both proposer and executor
    // No admin (address(0)) to prevent backdoors
    const timelock = await TimelockController.deploy(
        minDelay,
        proposers,   // Multi-sig can propose
        executors,   // Multi-sig can execute
        ethers.ZeroAddress  // No admin
    );

    await timelock.waitForDeployment();
    const address = await timelock.getAddress();

    console.log(`✅ Timelock deployed at: ${address}`);

    return { address, contract: timelock };
}

/**
 * Verify role configuration
 */
async function verifyRoles(timelock, multiSigAddress) {
    console.log("\n🔍 Verifying role configuration...");

    const PROPOSER_ROLE = await timelock.PROPOSER_ROLE();
    const EXECUTOR_ROLE = await timelock.EXECUTOR_ROLE();
    const TIMELOCK_ADMIN_ROLE = await timelock.TIMELOCK_ADMIN_ROLE();

    const hasProposerRole = await timelock.hasRole(PROPOSER_ROLE, multiSigAddress);
    const hasExecutorRole = await timelock.hasRole(EXECUTOR_ROLE, multiSigAddress);

    console.log(`   Multi-sig has PROPOSER role: ${hasProposerRole ? '✅' : '❌'}`);
    console.log(`   Multi-sig has EXECUTOR role: ${hasExecutorRole ? '✅' : '❌'}`);

    // Check that deployer has renounced admin (should happen automatically)
    const [deployer] = await ethers.getSigners();
    const deployerHasAdmin = await timelock.hasRole(TIMELOCK_ADMIN_ROLE, deployer.address);
    console.log(`   Deployer has ADMIN role: ${deployerHasAdmin ? '⚠️  Should renounce' : '✅ Renounced'}`);

    return hasProposerRole && hasExecutorRole;
}

/**
 * Save deployment info to file
 */
function saveDeploymentInfo(network, multiSig, timelock, signers) {
    if (!CONFIG.SAVE_DEPLOYMENT) return;

    const deploymentInfo = {
        network: network.name,
        chainId: network.chainId,
        timestamp: new Date().toISOString(),
        deployedBy: signers[0],

        multiSig: {
            address: multiSig.address,
            threshold: CONFIG.MULTISIG_THRESHOLD,
            signers: signers,
            type: "SimpleMultiSig" // Change to "GnosisSafe" in production
        },

        timelock: {
            address: timelock.address,
            minDelay: CONFIG.MIN_DELAY_NORMAL,
            minDelayEmergency: CONFIG.MIN_DELAY_EMERGENCY,
            proposers: [multiSig.address],
            executors: [multiSig.address],
            admin: ethers.ZeroAddress
        },

        nextSteps: [
            "Transfer ownership of SageRegistryV2 to Timelock",
            "Transfer ownership of ERC8004ValidationRegistry to Timelock",
            "Transfer ownership of ERC8004ReputationRegistry to Timelock",
            "Test ownership transfer with multi-sig + timelock flow",
            "Document emergency procedures"
        ]
    };

    // Create deployment directory if it doesn't exist
    const deployDir = path.join(__dirname, "..", CONFIG.DEPLOYMENT_DIR);
    if (!fs.existsSync(deployDir)) {
        fs.mkdirSync(deployDir, { recursive: true });
    }

    // Save to file
    const filename = `governance-${network.name}-${Date.now()}.json`;
    const filepath = path.join(deployDir, filename);
    fs.writeFileSync(filepath, JSON.stringify(deploymentInfo, null, 2));

    console.log(`\n💾 Deployment info saved to: ${filepath}`);

    // Also save latest
    const latestPath = path.join(deployDir, `governance-${network.name}-latest.json`);
    fs.writeFileSync(latestPath, JSON.stringify(deploymentInfo, null, 2));

    return filepath;
}

/**
 * Main deployment function
 */
async function main() {
    console.log("=".repeat(60));
    console.log("SAGE Governance Deployment: Multi-sig + Timelock");
    console.log("=".repeat(60));

    // Get network info
    const network = await ethers.provider.getNetwork();
    console.log(`\n🌐 Network: ${network.name}`);
    console.log(`   Chain ID: ${network.chainId}`);

    // Get deployer
    const [deployer] = await ethers.getSigners();
    console.log(`\n👤 Deployer: ${deployer.address}`);
    const balance = await ethers.provider.getBalance(deployer.address);
    console.log(`   Balance: ${ethers.formatEther(balance)} ETH`);

    // Get signer addresses
    let signers = getSignerAddresses();

    // If no signers configured, use test addresses for local development
    if (!signers) {
        console.log("\n⚠️  No signer addresses configured in .env");
        console.log("   Using test addresses for demonstration");
        console.log("   ⚠️  DO NOT USE IN PRODUCTION!");

        const testSigners = await ethers.getSigners();
        signers = testSigners.slice(0, 5).map(s => s.address);
    }

    console.log("\n👥 Multi-sig Signers:");
    signers.forEach((signer, i) => {
        console.log(`   ${i + 1}. ${signer}`);
    });

    // Step 1: Deploy Multi-sig
    const multiSig = await deployMultiSig(signers, CONFIG.MULTISIG_THRESHOLD);

    // Step 2: Deploy Timelock
    const timelock = await deployTimelock(
        [multiSig.address],  // Only multi-sig can propose
        [multiSig.address],  // Only multi-sig can execute
        CONFIG.MIN_DELAY_NORMAL
    );

    // Step 3: Verify role configuration
    const rolesConfigured = await verifyRoles(timelock.contract, multiSig.address);

    if (!rolesConfigured) {
        console.log("\n❌ Error: Role configuration failed!");
        process.exit(1);
    }

    // Step 4: Save deployment info
    saveDeploymentInfo(network, multiSig, timelock, signers);

    // Final summary
    console.log("\n" + "=".repeat(60));
    console.log("✅ Deployment Complete!");
    console.log("=".repeat(60));
    console.log(`\n📝 Deployment Summary:`);
    console.log(`   Multi-sig: ${multiSig.address}`);
    console.log(`   Timelock: ${timelock.address}`);
    console.log(`   Threshold: ${CONFIG.MULTISIG_THRESHOLD} of ${CONFIG.MULTISIG_SIGNERS_COUNT}`);
    console.log(`   Min Delay: ${CONFIG.MIN_DELAY_NORMAL / 3600} hours`);

    console.log(`\n📋 Next Steps:`);
    console.log(`   1. Verify contracts on Etherscan`);
    console.log(`   2. Transfer ownership of SAGE contracts to Timelock`);
    console.log(`   3. Test admin operations through multi-sig + timelock`);
    console.log(`   4. Document emergency procedures`);
    console.log(`   5. Distribute hardware wallets to signers`);

    console.log(`\n⚠️  Important:`);
    console.log(`   - Keep this deployment info secure`);
    console.log(`   - Verify all signer addresses before mainnet`);
    console.log(`   - Test thoroughly on testnet first`);
    console.log(`   - Set up monitoring and alerts`);

    console.log("\n" + "=".repeat(60) + "\n");
}

// Execute deployment
main()
    .then(() => process.exit(0))
    .catch((error) => {
        console.error("\n❌ Deployment failed:");
        console.error(error);
        process.exit(1);
    });
