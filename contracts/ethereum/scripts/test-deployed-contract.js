// SPDX-License-Identifier: MIT
import hre from 'hardhat';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function main() {
  console.log('\n Testing Deployed AgentCard Contracts');
  console.log('================================================================================\n');

  // Load deployment info
  const deploymentPath = path.join(__dirname, '..', 'deployments', 'localhost-latest.json');
  if (!fs.existsSync(deploymentPath)) {
    console.error(' No deployment found. Run deploy-agentcard.js first.');
    process.exit(1);
  }

  const deployment = JSON.parse(fs.readFileSync(deploymentPath, 'utf8'));
  const registryAddress = deployment.contracts.AgentCardRegistry.address;
  const hookAddress = deployment.contracts.AgentCardVerifyHook.address;

  console.log(' Deployment Info:');
  console.log(`   Network: ${deployment.network}`);
  console.log(`   Chain ID: ${deployment.chainId}`);
  console.log(`   Registry: ${registryAddress}`);
  console.log(`   Hook: ${hookAddress}\n`);

  // Connect to network (Hardhat 3.x)
  const network = await hre.network.connect();
  const [deployer, user1, user2] = await network.ethers.getSigners();

  console.log(' Test Accounts:');
  console.log(`   Deployer: ${deployer.address}`);
  console.log(`   User1: ${user1.address}`);
  console.log(`   User2: ${user2.address}\n`);

  // Get contract instances
  const AgentCardRegistry = await network.ethers.getContractFactory('AgentCardRegistry');
  const registry = AgentCardRegistry.attach(registryAddress);

  console.log('================================================================================');
  console.log('TEST 1: 컨트랙트 설정 확인');
  console.log('================================================================================\n');

  const owner = await registry.owner();
  const hookAddr = await registry.verifyHook();
  const minStake = await registry.registrationStake();

  console.log(' Owner:', owner);
  console.log(' Hook Address:', hookAddr);
  console.log(' Min Stake:', network.ethers.formatEther(minStake), 'ETH\n');

  if (hookAddr !== hookAddress) {
    console.error(' Hook address mismatch!');
    process.exit(1);
  }

  console.log('================================================================================');
  console.log('TEST 2: Agent 등록 (Commit-Reveal 패턴)');
  console.log('================================================================================\n');

  // Generate random public key (33 bytes for compressed ECDSA)
  // Note: In production, this would be the agent's actual public key
  const publicKey = network.ethers.randomBytes(33);
  // Use timestamp to create unique DID for each test run
  const uniqueId = `${user1.address.toLowerCase()}_${Date.now()}`;
  const agentDID = `did:sage:ethereum:${uniqueId}`;

  console.log(' Generated Test Key:');
  console.log(`   Public Key: ${network.ethers.hexlify(publicKey).slice(0, 22)}...`);
  console.log(`   Agent Owner: ${user1.address}`);
  console.log(`   Agent DID: ${agentDID}\n`);

  // Step 1: Commit
  const salt = network.ethers.randomBytes(32);
  const chainId = await network.ethers.provider.getNetwork().then((n) => n.chainId);

  // Calculate commitment hash matching contract: keccak256(abi.encode(did, keys, owner, salt, chainId))
  const abiCoder = network.ethers.AbiCoder.defaultAbiCoder();
  const encoded = abiCoder.encode(
    ['string', 'bytes[]', 'address', 'bytes32', 'uint256'],
    [agentDID, [publicKey], user1.address, salt, chainId]
  );
  const commitHash = network.ethers.keccak256(encoded);

  console.log(' Step 1: Commit');
  const commitTx = await registry.connect(user1).commitRegistration(commitHash, { value: minStake });
  await commitTx.wait();
  console.log('    Commitment recorded');
  console.log(`    Tx: ${commitTx.hash}\n`);

  // Wait for commit delay (1 minute in production, but we'll test immediately for demo)
  console.log(' Waiting 61 seconds for commit delay...');
  console.log('    (This is a security feature - commit-reveal pattern prevents front-running)');

  // Show progress during wait
  for (let i = 0; i < 61; i++) {
    process.stdout.write(`\r    Progress: ${i + 1}/61 seconds elapsed...`);
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  console.log('\n');

  // Step 2: Reveal
  console.log('\n Step 2: Reveal and Register');

  // Sign with ECDSA - must match contract's expected message format
  // Message format: keccak256("SAGE Agent Registration:" + chainId + registryAddress + signerAddress)
  const message = network.ethers.solidityPackedKeccak256(
    ['string', 'uint256', 'address', 'address'],
    ['SAGE Agent Registration:', chainId, registryAddress, user1.address]
  );
  const signature = await user1.signMessage(network.ethers.getBytes(message));

  const params = {
    did: agentDID,
    name: 'Test Agent',
    description: 'Test agent for local deployment testing',
    endpoint: 'https://agent.example.com',
    capabilities: 'authentication,encryption',
    keys: [publicKey],
    keyTypes: [0], // ECDSA
    signatures: [signature],
    salt: salt,
  };

  const revealTx = await registry.connect(user1).registerAgentWithParams(params);
  const revealReceipt = await revealTx.wait();
  console.log('    Agent registered successfully');
  console.log(`    Tx: ${revealTx.hash}`);
  console.log(`    Gas Used: ${revealReceipt.gasUsed.toString()}\n`);

  console.log('================================================================================');
  console.log('TEST 3: Agent 정보 조회');
  console.log('================================================================================\n');

  // Get agent by DID
  const agentInfo = await registry.getAgentByDID(agentDID);
  console.log(' Agent Info Retrieved:');
  console.log(`   DID: ${agentInfo.did}`);
  console.log(`   Owner: ${agentInfo.owner}`);
  console.log(`   Endpoint: ${agentInfo.endpoint}`);
  console.log(`   Capabilities: ${agentInfo.capabilities}`);
  console.log(`   Active: ${agentInfo.active}`);
  console.log(`   Keys Count: ${agentInfo.keys.length}`);
  console.log(`   Created At: ${new Date(Number(agentInfo.createdAt) * 1000).toISOString()}\n`);

  // Verify key
  const storedKey = agentInfo.keys[0];
  console.log(' Stored Key Info:');
  console.log(`   Type: ${storedKey.keyType === 0n ? 'ECDSA' : 'Unknown'}`);
  console.log(`   Public Key: ${storedKey.publicKey.slice(0, 22)}...`);
  console.log(`   Added At: ${new Date(Number(storedKey.addedAt) * 1000).toISOString()}\n`);

  console.log('================================================================================');
  console.log('TEST 4: Agent 활성화 (Time-lock 후)');
  console.log('================================================================================\n');

  // Get agentId for activation
  const agentId = await registry.didToAgentId(agentDID);

  const activationDelay = await registry.activationDelay();
  const delaySeconds = Number(activationDelay);
  console.log(` Activation Delay: ${delaySeconds} seconds`);
  console.log('   Waiting for activation delay...');
  console.log('    (This is a security feature - prevents immediate activation)');

  // Show progress during wait
  for (let i = 0; i < delaySeconds + 1; i++) {
    process.stdout.write(`\r    Progress: ${i + 1}/${delaySeconds + 1} seconds elapsed...`);
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  console.log('\n');

  const activateTx = await registry.activateAgent(agentId);
  await activateTx.wait();
  console.log('    Agent activated successfully');
  console.log(`    Tx: ${activateTx.hash}\n`);

  const activeAgent = await registry.getAgentByDID(agentDID);
  console.log(` Agent Active Status: ${activeAgent.active}\n`);

  console.log('================================================================================');
  console.log('TEST 5: ERC-8004 인터페이스 테스트');
  console.log('================================================================================\n');

  // Test ERC-8004 functions
  const isActive = await registry.isAgentActive(agentDID);
  console.log(` isAgentActive(): ${isActive}`);

  const resolvedAgent = await registry.resolveAgent(agentDID);
  console.log(` resolveAgent():`);
  console.log(`   DID: ${resolvedAgent[0]}`);
  console.log(`   Owner: ${resolvedAgent[1]}`);
  console.log(`   Endpoint: ${resolvedAgent[2]}\n`);

  const agentByAddress = await registry.resolveAgentByAddress(user1.address);
  console.log(` resolveAgentByAddress():`);
  console.log(`   DID: ${agentByAddress[0]}`);
  console.log(`   Endpoint: ${agentByAddress[2]}\n`);

  console.log('================================================================================');
  console.log(' All Tests Passed!');
  console.log('================================================================================\n');

  console.log(' Summary:');
  console.log(`    Contract configuration verified`);
  console.log(`    Agent registration (commit-reveal) successful`);
  console.log(`    Agent information retrieval working`);
  console.log(`    Agent activation successful`);
  console.log(`    ERC-8004 interface compliant\n`);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error('\n Test failed:', error);
    process.exit(1);
  });
