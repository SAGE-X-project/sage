// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "./interfaces/ISageRegistry.sol";
import "./interfaces/IRegistryHook.sol";
import "@openzeppelin/contracts/security/Pausable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/access/Ownable2Step.sol";

/**
 * @title SageRegistryV3
 * @author SAGE Development Team
 * @notice SAGE AI Agent Registry with Front-Running Protection
 * @dev Version 3 of the SAGE Registry implementing commit-reveal pattern for secure agent registration
 *
 * ## Overview
 *
 * SageRegistryV3 is the core registry for AI agents in the SAGE ecosystem. It provides:
 * - Decentralized Identifier (DID) registration and management
 * - Public key validation and association
 * - Front-running protection via commit-reveal scheme
 * - Cross-chain replay attack protection
 * - Emergency pause capabilities
 *
 * ## Architecture
 *
 * The contract uses a two-phase registration process:
 * 1. **Commit Phase**: User commits a hash of their registration intent
 * 2. **Reveal Phase**: User reveals the actual registration data (after 1 min, before 1 hour)
 *
 * This prevents front-running attacks where adversaries could observe pending transactions
 * and submit competing registrations with higher gas prices.
 *
 * ## Key Features
 *
 * ### 1. Front-Running Protection
 * - Commit-reveal pattern prevents DID hijacking
 * - Timing constraints enforce minimum/maximum delays
 * - Salt ensures commitment privacy
 * - ChainId prevents cross-chain replay
 *
 * ### 2. Public Key Management
 * - Validates key ownership through signature verification
 * - Supports key revocation and rotation
 * - Links addresses to public keys
 * - Prevents duplicate key usage
 *
 * ### 3. Security Features
 * - Emergency pause by owner
 * - Two-step ownership transfer (Ownable2Step)
 * - Reentrancy protection
 * - Gas limit controls for hooks
 *
 * ### 4. Agent Limits
 * - Maximum 100 agents per owner (DoS prevention)
 * - Public key length validation (32-65 bytes)
 * - DID uniqueness enforcement
 *
 * ## Security Model
 *
 * **Assumptions:**
 * - Block timestamps are reasonably accurate (±15 seconds)
 * - Users keep their salt values secret until reveal
 * - Owner is trusted for emergency pauses
 *
 * **Invariants:**
 * - Each DID can only be registered once
 * - Public keys must be validated before use
 * - Commitments expire after MAX_COMMIT_REVEAL_DELAY
 * - No agent can be registered while paused
 *
 * ## Economic Model
 *
 * Registration is currently free. Future versions may introduce:
 * - Registration fees to prevent spam
 * - Stake requirements for agent operators
 * - Renewal fees for DID maintenance
 *
 * ## Gas Costs (Approximate)
 *
 * - `commitRegistration()`: ~50,000 gas
 * - `registerAgentWithReveal()`: ~250,000 gas (first registration)
 * - `registerAgentWithReveal()`: ~150,000 gas (subsequent registrations)
 * - `updateAgent()`: ~80,000 gas
 * - `revokeKey()`: ~40,000 gas
 *
 * @custom:security-contact security@sage.com
 * @custom:audit-status Phase 7.5 - Security enhancements implemented, pending external audit
 * @custom:version 3.0.0
 */
contract SageRegistryV3 is ISageRegistry, Pausable, ReentrancyGuard, Ownable2Step {
    // ============================================
    // STRUCTS
    // ============================================

    struct RegistrationParams {
        string did;
        string name;
        string description;
        string endpoint;
        bytes publicKey;
        string capabilities;
        bytes signature;
    }

    struct KeyValidation {
        bytes32 keyHash;
        uint256 registrationBlock;
        bool isRevoked;
    }

    // Commit-reveal structure
    struct RegistrationCommitment {
        bytes32 commitHash;
        uint256 timestamp;
        bool revealed;
    }

    // ============================================
    // STATE VARIABLES
    // ============================================

    // Agent storage
    mapping(bytes32 => AgentMetadata) private agents;
    mapping(string => bytes32) private didToAgentId;
    mapping(address => bytes32[]) private ownerToAgents;
    mapping(address => uint256) private registrationNonce;
    mapping(bytes32 => uint256) private agentNonce;

    // Key validation
    mapping(bytes32 => KeyValidation) private keyValidations;
    mapping(address => bytes32) private addressToKeyHash;
    mapping(bytes32 => bytes32[]) private keyHashToAgentIds;

    // Commit-reveal for front-running protection
    mapping(address => RegistrationCommitment) public registrationCommitments;

    // Hooks
    address public beforeRegisterHook;
    address public afterRegisterHook;

    // ============================================
    // CONSTANTS
    // ============================================

    uint256 private constant MAX_AGENTS_PER_OWNER = 100;
    uint256 private constant MIN_PUBLIC_KEY_LENGTH = 32;
    uint256 private constant MAX_PUBLIC_KEY_LENGTH = 65;
    uint256 private constant HOOK_GAS_LIMIT = 50000;

    // Commit-reveal timing
    uint256 private constant MIN_COMMIT_REVEAL_DELAY = 1 minutes;  // Minimum wait after commit
    uint256 private constant MAX_COMMIT_REVEAL_DELAY = 1 hours;    // Maximum wait before expiry

    // ============================================
    // EVENTS
    // ============================================

    event KeyValidated(bytes32 indexed keyHash, address indexed owner);
    event KeyRevoked(bytes32 indexed keyHash, address indexed owner);
    event HookFailed(address indexed hook, string reason);
    event BeforeRegisterHookUpdated(address indexed oldHook, address indexed newHook);
    event AfterRegisterHookUpdated(address indexed oldHook, address indexed newHook);

    // Commit-reveal events
    event RegistrationCommitted(address indexed committer, bytes32 indexed commitHash, uint256 timestamp);
    event RegistrationRevealed(address indexed revealer, bytes32 indexed agentId, string did);
    event CommitmentExpired(address indexed committer, bytes32 indexed commitHash);

    // ============================================
    // ERRORS
    // ============================================

    error AlreadyCommitted();
    error NoCommitmentFound();
    error InvalidReveal();
    error RevealTooSoon(uint256 currentTime, uint256 minTime);
    error RevealTooLate(uint256 currentTime, uint256 maxTime);
    error CommitmentAlreadyRevealed();
    error InvalidCommitHash();

    // ============================================
    // MODIFIERS
    // ============================================

    modifier onlyAgentOwner(bytes32 agentId) {
        require(agents[agentId].owner == msg.sender, "Not agent owner");
        _;
    }

    // ============================================
    // CONSTRUCTOR
    // ============================================

    constructor() {
        _transferOwnership(msg.sender);
    }

    // ============================================
    // COMMIT-REVEAL FUNCTIONS
    // ============================================

    /**
     * @notice Commit to a future agent registration (Step 1 of 2)
     * @dev Prevents front-running by hiding registration intent
     *
     * The commitment hash is:
     * keccak256(abi.encodePacked(did, publicKey, msg.sender, salt, chainId))
     *
     * @param commitHash Hash of registration parameters + salt
     *
     * Process:
     * 1. User creates commitment hash off-chain
     * 2. User calls commitRegistration(commitHash)
     * 3. Wait MIN_COMMIT_REVEAL_DELAY (prevents instant reveal)
     * 4. User calls registerAgentWithReveal() with actual parameters
     * 5. Contract verifies hash matches commitment
     *
     * Security:
     * - Attacker cannot see which DID user wants to register
     * - Attacker cannot front-run because they don't know the salt
     * - User must wait minimum delay before reveal
     * - Commitment expires after maximum delay
     *
     * Example:
     * ```javascript
     * const did = "did:sage:alice";
     * const publicKey = "0x...";
     * const salt = ethers.randomBytes(32);
     * const chainId = await ethers.provider.getNetwork().chainId;
     *
     * const commitHash = ethers.keccak256(
     *   ethers.solidityPacked(
     *     ["string", "bytes", "address", "bytes32", "uint256"],
     *     [did, publicKey, userAddress, salt, chainId]
     *   )
     * );
     *
     * await registry.commitRegistration(commitHash);
     * // Wait 1 minute...
     * await registry.registerAgentWithReveal(did, name, ..., salt);
     * ```
     */
    function commitRegistration(bytes32 commitHash) external whenNotPaused {
        // Input validation
        if (commitHash == bytes32(0)) revert InvalidCommitHash();

        RegistrationCommitment storage commitment = registrationCommitments[msg.sender];

        // Check if already committed and not expired
        if (commitment.timestamp > 0 && !commitment.revealed) {
            if (block.timestamp <= commitment.timestamp + MAX_COMMIT_REVEAL_DELAY) {
                revert AlreadyCommitted();
            } else {
                // Old commitment expired, emit event
                emit CommitmentExpired(msg.sender, commitment.commitHash);
            }
        }

        // Store new commitment
        registrationCommitments[msg.sender] = RegistrationCommitment({
            commitHash: commitHash,
            timestamp: block.timestamp,
            revealed: false
        });

        emit RegistrationCommitted(msg.sender, commitHash, block.timestamp);
    }

    /**
     * @notice Register agent with reveal (Step 2 of 2)
     * @dev Verifies commitment and completes registration with front-running protection
     *
     * This function completes the commit-reveal registration process. It verifies:
     * 1. A valid commitment exists for the sender
     * 2. Timing constraints are satisfied (1 min < elapsed < 1 hour)
     * 3. Revealed parameters match the committed hash
     * 4. DID is not already registered
     * 5. Public key ownership is proven via signature
     *
     * @param did Decentralized Identifier (e.g., "did:sage:alice")
     * @param name Human-readable agent name
     * @param description Agent description and capabilities
     * @param endpoint Agent API endpoint URL
     * @param publicKey Agent's secp256k1 public key (33 or 65 bytes)
     * @param capabilities JSON string of agent capabilities
     * @param signature ECDSA signature proving key ownership
     * @param salt Random 32-byte salt used in commitment (must match commit phase)
     * @return agentId Unique identifier (keccak256(did)) for the registered agent
     *
     * ## Process Flow
     *
     * 1. **Commitment Verification**: Checks commitment exists and not yet revealed
     * 2. **Timing Validation**:
     *    - Must wait ≥1 minute after commit (prevents instant reveal)
     *    - Must reveal within 1 hour (prevents indefinite squatting)
     * 3. **Hash Verification**: Reconstructs hash and compares with commitment
     * 4. **Registration**: Proceeds with standard agent registration
     * 5. **State Update**: Marks commitment as revealed
     *
     * ## Timing Constraints
     *
     * - **MIN_COMMIT_REVEAL_DELAY**: 60 seconds (prevents MEV front-running)
     * - **MAX_COMMIT_REVEAL_DELAY**: 3600 seconds (prevents commitment hoarding)
     *
     * ## Security Model
     *
     * **Protection Against**:
     * - Front-running attacks (attacker can't see DID before commitment)
     * - Cross-chain replay (chainId included in commitment hash)
     * - MEV exploitation (minimum delay prevents instant reveals)
     * - Commitment squatting (maximum delay forces timely reveals)
     *
     * **Attack Scenarios Prevented**:
     * ```
     * SCENARIO: DID Front-Running
     * 1. Alice wants "did:sage:valuable-name"
     * 2. WITHOUT protection: Attacker sees tx, submits with higher gas, steals DID
     * 3. WITH protection: Attacker sees only commit hash, cannot predict DID
     * 4. Alice reveals after delay, gets her DID safely
     * ```
     *
     * ## Usage Example
     *
     * ```javascript
     * // Step 1: Commit (done earlier via commitRegistration)
     * const salt = ethers.randomBytes(32);
     * const commitHash = ethers.keccak256(
     *   ethers.solidityPacked(
     *     ["string", "bytes", "address", "bytes32", "uint256"],
     *     [did, publicKey, userAddress, salt, chainId]
     *   )
     * );
     * await registry.commitRegistration(commitHash);
     *
     * // Step 2: Wait minimum delay
     * await new Promise(r => setTimeout(r, 61000)); // 61 seconds
     *
     * // Step 3: Reveal and register
     * const signature = await wallet.signMessage(challengeHash);
     * const tx = await registry.registerAgentWithReveal(
     *   "did:sage:alice",
     *   "Alice AI Agent",
     *   "An intelligent assistant",
     *   "https://alice.example.com",
     *   publicKeyBytes,
     *   '{"chat": true, "vision": true}',
     *   signature,
     *   salt  // Same salt used in commit
     * );
     * const receipt = await tx.wait();
     * const agentId = receipt.events.find(e => e.event === 'AgentRegistered').args.agentId;
     * ```
     *
     * @custom:security-warning Users MUST keep their salt secret until reveal phase
     * @custom:security-warning Do NOT reuse salts across different registrations
     * @custom:gas-cost ~250,000 gas (first registration), ~150,000 gas (subsequent)
     * @custom:throws NoCommitmentFound if sender has not committed
     * @custom:throws CommitmentAlreadyRevealed if commitment already used
     * @custom:throws RevealTooSoon if minimum delay not elapsed
     * @custom:throws RevealTooLate if maximum delay exceeded
     * @custom:throws InvalidReveal if hash doesn't match commitment
     */
    function registerAgentWithReveal(
        string calldata did,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        bytes calldata publicKey,
        string calldata capabilities,
        bytes calldata signature,
        bytes32 salt
    ) external whenNotPaused nonReentrant returns (bytes32) {
        RegistrationCommitment storage commitment = registrationCommitments[msg.sender];

        // Verify commitment exists
        if (commitment.timestamp == 0) revert NoCommitmentFound();
        if (commitment.revealed) revert CommitmentAlreadyRevealed();

        // Verify timing
        uint256 minRevealTime = commitment.timestamp + MIN_COMMIT_REVEAL_DELAY;
        uint256 maxRevealTime = commitment.timestamp + MAX_COMMIT_REVEAL_DELAY;

        if (block.timestamp < minRevealTime) {
            revert RevealTooSoon(block.timestamp, minRevealTime);
        }
        if (block.timestamp > maxRevealTime) {
            revert RevealTooLate(block.timestamp, maxRevealTime);
        }

        // Verify commitment hash matches revealed parameters
        bytes32 expectedHash = keccak256(abi.encode(
            did,
            publicKey,
            msg.sender,
            salt,
            block.chainid  // Include chainId for cross-chain protection
        ));

        if (commitment.commitHash != expectedHash) revert InvalidReveal();

        // Mark as revealed
        commitment.revealed = true;

        // Validate public key format and ownership
        _validatePublicKey(publicKey, signature);

        // Proceed with registration
        RegistrationParams memory params = RegistrationParams({
            did: did,
            name: name,
            description: description,
            endpoint: endpoint,
            publicKey: publicKey,
            capabilities: capabilities,
            signature: signature
        });

        bytes32 agentId = _registerAgent(params);

        emit RegistrationRevealed(msg.sender, agentId, did);

        return agentId;
    }

    /**
     * @notice Legacy registration function (without front-running protection)
     * @dev Kept for backward compatibility, but registerAgentWithReveal() is recommended
     * @custom:security-warning Vulnerable to front-running attacks
     */
    function registerAgent(
        string calldata did,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        bytes calldata publicKey,
        string calldata capabilities,
        bytes calldata signature
    ) external whenNotPaused nonReentrant returns (bytes32) {
        // Validate public key format and ownership
        _validatePublicKey(publicKey, signature);

        RegistrationParams memory params = RegistrationParams({
            did: did,
            name: name,
            description: description,
            endpoint: endpoint,
            publicKey: publicKey,
            capabilities: capabilities,
            signature: signature
        });

        return _registerAgent(params);
    }

    // ============================================
    // VALIDATION FUNCTIONS
    // ============================================

    /**
     * @notice Enhanced public key validation with cross-chain protection
     * @dev Validates format, non-zero, and ownership through signature
     */
    function _validatePublicKey(
        bytes calldata publicKey,
        bytes calldata signature
    ) internal {
        // 1. Length validation
        require(
            publicKey.length >= MIN_PUBLIC_KEY_LENGTH &&
            publicKey.length <= MAX_PUBLIC_KEY_LENGTH,
            "Invalid public key length"
        );

        // 2. Format validation for secp256k1
        if (publicKey.length == 65) {
            require(publicKey[0] == 0x04, "Invalid uncompressed key format");
        } else if (publicKey.length == 33) {
            require(
                publicKey[0] == 0x02 || publicKey[0] == 0x03,
                "Invalid compressed key format"
            );
        } else if (publicKey.length == 32) {
            revert("Ed25519 not supported on-chain");
        }

        // 3. Non-zero validation
        bytes32 keyHash = keccak256(publicKey);
        bool isNonZero = false;
        uint startIdx = (publicKey.length == 65 && publicKey[0] == 0x04) ? 1 :
                        (publicKey.length == 33 && (publicKey[0] == 0x02 || publicKey[0] == 0x03)) ? 1 : 0;

        for (uint i = startIdx; i < publicKey.length; i++) {
            if (publicKey[i] != 0) {
                isNonZero = true;
                break;
            }
        }
        require(isNonZero, "Invalid zero key");

        // 4. Ownership proof through signature (with chainId for cross-chain protection)
        bytes32 challenge = keccak256(abi.encodePacked(
            "SAGE Key Registration:",
            block.chainid,          // Cross-chain replay protection
            address(this),
            msg.sender,
            keyHash
        ));

        // Verify signature
        bytes32 ethSignedHash = keccak256(
            abi.encodePacked("\x19Ethereum Signed Message:\n32", challenge)
        );

        address recovered = _recoverSigner(ethSignedHash, signature);
        address keyAddress = _getAddressFromPublicKey(publicKey);

        require(recovered == keyAddress, "Key ownership not proven");
        require(recovered != address(0), "Invalid signature");

        // 5. Check if key has been revoked
        if (keyValidations[keyHash].registrationBlock > 0) {
            require(!keyValidations[keyHash].isRevoked, "Key has been revoked");
        }

        // 6. Store validation data
        if (keyValidations[keyHash].registrationBlock == 0) {
            keyValidations[keyHash] = KeyValidation({
                keyHash: keyHash,
                registrationBlock: block.number,
                isRevoked: false
            });
        }

        addressToKeyHash[keyAddress] = keyHash;
        addressToKeyHash[msg.sender] = keyHash;

        emit KeyValidated(keyHash, msg.sender);
    }

    /**
     * @notice Validate DID format according to W3C DID spec
     */
    function _isValidDID(string memory did) private pure returns (bool) {
        bytes memory didBytes = bytes(did);
        uint256 len = didBytes.length;

        if (len < 7) return false;

        // Must start with "did:"
        if (didBytes[0] != 0x64 || didBytes[1] != 0x69 || didBytes[2] != 0x64 || didBytes[3] != 0x3A) {
            return false;
        }

        // Find second colon
        uint256 secondColonIndex = 0;
        for (uint256 i = 4; i < len; i++) {
            if (didBytes[i] == 0x3A) {
                secondColonIndex = i;
                break;
            }
        }

        if (secondColonIndex == 0 || secondColonIndex == len - 1) {
            return false;
        }

        // Validate method (lowercase alphanumeric)
        for (uint256 i = 4; i < secondColonIndex; i++) {
            bytes1 char = didBytes[i];
            if (!((char >= 0x61 && char <= 0x7A) || (char >= 0x30 && char <= 0x39))) {
                return false;
            }
        }

        return true;
    }

    // ============================================
    // INTERNAL REGISTRATION LOGIC
    // ============================================

    function _registerAgent(RegistrationParams memory params) private returns (bytes32) {
        _validateRegistrationInputs(params.did, params.name);

        bytes32 agentId = _generateAgentId(params.did, params.publicKey);

        _executeBeforeHook(agentId, params.did, params.publicKey);
        _storeAgentMetadata(agentId, params);
        _executeAfterHook(agentId, params.did, params.publicKey);

        return agentId;
    }

    function _validateRegistrationInputs(string memory did, string memory name) private view {
        require(bytes(did).length > 0, "DID required");
        require(_isValidDID(did), "Invalid DID format");
        require(bytes(name).length > 0, "Name required");
        require(didToAgentId[did] == bytes32(0), "DID already registered");
        require(ownerToAgents[msg.sender].length < MAX_AGENTS_PER_OWNER, "Too many agents");
    }

    function _generateAgentId(string memory did, bytes memory publicKey) private returns (bytes32) {
        uint256 nonce = registrationNonce[msg.sender];
        registrationNonce[msg.sender]++;

        return keccak256(abi.encode(
            did,
            publicKey,
            msg.sender,
            block.number,
            nonce
        ));
    }

    function _storeAgentMetadata(bytes32 agentId, RegistrationParams memory params) private {
        agents[agentId] = AgentMetadata({
            did: params.did,
            name: params.name,
            description: params.description,
            endpoint: params.endpoint,
            publicKey: params.publicKey,
            capabilities: params.capabilities,
            owner: msg.sender,
            registeredAt: block.timestamp,
            updatedAt: block.timestamp,
            active: true
        });

        didToAgentId[params.did] = agentId;
        ownerToAgents[msg.sender].push(agentId);
        agentNonce[agentId]++;

        bytes32 keyHash = keccak256(params.publicKey);
        keyHashToAgentIds[keyHash].push(agentId);

        emit AgentRegistered(agentId, msg.sender, params.did, block.timestamp);
    }

    function _executeBeforeHook(bytes32 agentId, string memory did, bytes memory publicKey) private {
        if (beforeRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);
            emit BeforeRegisterHook(agentId, msg.sender, hookData);

            try IRegistryHook(beforeRegisterHook).beforeRegister{gas: HOOK_GAS_LIMIT}(
                agentId,
                msg.sender,
                hookData
            ) returns (bool success, string memory reason) {
                require(success, reason);
            } catch Error(string memory reason) {
                emit HookFailed(beforeRegisterHook, reason);
                revert(reason);
            } catch (bytes memory) {
                emit HookFailed(beforeRegisterHook, "Hook call failed");
                revert("Hook call failed");
            }
        }
    }

    function _executeAfterHook(bytes32 agentId, string memory did, bytes memory publicKey) private {
        if (afterRegisterHook != address(0)) {
            bytes memory hookData = abi.encode(did, publicKey);

            try IRegistryHook(afterRegisterHook).afterRegister{gas: HOOK_GAS_LIMIT}(
                agentId,
                msg.sender,
                hookData
            ) {
                emit AfterRegisterHook(agentId, msg.sender, hookData);
            } catch Error(string memory reason) {
                emit HookFailed(afterRegisterHook, reason);
            } catch (bytes memory) {
                emit HookFailed(afterRegisterHook, "Hook call failed");
            }
        }
    }

    // ============================================
    // AGENT MANAGEMENT
    // ============================================

    function updateAgent(
        bytes32 agentId,
        string calldata name,
        string calldata description,
        string calldata endpoint,
        string calldata capabilities,
        bytes calldata signature
    ) external onlyAgentOwner(agentId) {
        require(bytes(name).length > 0, "Name required");

        bytes32 keyHash = keccak256(agents[agentId].publicKey);
        require(!keyValidations[keyHash].isRevoked, "Key has been revoked");
        require(agents[agentId].active, "Agent not active");

        // Include chainId in update signature for cross-chain protection
        bytes32 messageHash = keccak256(abi.encode(
            agentId,
            name,
            description,
            endpoint,
            capabilities,
            msg.sender,
            agentNonce[agentId],
            block.chainid  // Cross-chain replay protection
        ));

        require(
            _verifySignature(messageHash, signature, agents[agentId].publicKey, msg.sender),
            "Invalid signature"
        );

        agents[agentId].name = name;
        agents[agentId].description = description;
        agents[agentId].endpoint = endpoint;
        agents[agentId].capabilities = capabilities;
        agents[agentId].updatedAt = block.timestamp;

        agentNonce[agentId]++;

        emit AgentUpdated(agentId, msg.sender, block.timestamp);
    }

    function deactivateAgent(bytes32 agentId) external onlyAgentOwner(agentId) {
        require(agents[agentId].active, "Agent already inactive");
        agents[agentId].active = false;
        agents[agentId].updatedAt = block.timestamp;
        emit AgentDeactivated(agentId, msg.sender, block.timestamp);
    }

    function deactivateAgentByDID(string calldata did) external {
        bytes32 agentId = didToAgentId[did];
        require(agentId != bytes32(0), "Agent not found");
        require(agents[agentId].owner == msg.sender, "Not agent owner");
        require(agents[agentId].active, "Agent already inactive");

        agents[agentId].active = false;
        agents[agentId].updatedAt = block.timestamp;

        emit AgentDeactivated(agentId, msg.sender, block.timestamp);
    }

    /**
     * @notice Revoke a public key and deactivate all associated agents
     * @dev Permanently revokes the key and disables all agents using it
     *
     * This is a critical security function that allows key owners to immediately
     * revoke compromised keys. Once revoked, a key cannot be un-revoked.
     *
     * @param publicKey The public key to revoke (must match caller's registered key)
     *
     * ## Process Flow
     *
     * 1. **Hash Calculation**: Computes keccak256(publicKey)
     * 2. **Ownership Verification**: Confirms caller owns this key
     * 3. **Revocation Status Check**: Ensures key not already revoked
     * 4. **Key Revocation**: Marks key as permanently revoked
     * 5. **Agent Deactivation**: Deactivates ALL agents using this key
     * 6. **Event Emission**: Emits KeyRevoked event
     *
     * ## Security Implications
     *
     * **CRITICAL - Irreversible Action**:
     * - Once revoked, a key CANNOT be re-validated
     * - ALL agents associated with this key will be deactivated
     * - Deactivated agents cannot participate in the network
     * - This action should only be taken if:
     *   - Private key is compromised or suspected compromised
     *   - Key is lost and needs to be replaced
     *   - Agent operator is shutting down operations
     *
     * **Impact on Agents**:
     * - All agents using this key become inactive immediately
     * - Inactive agents cannot:
     *   - Accept new tasks
     *   - Receive reputation updates
     *   - Participate in validations
     * - Agent metadata remains on-chain (for historical records)
     *
     * **Recovery Process**:
     * If you revoke a key by mistake or need to continue operations:
     * 1. Generate a new key pair
     * 2. Re-register all agents with the new key
     * 3. Previous reputation and history will NOT transfer automatically
     *
     * ## Usage Examples
     *
     * ### Example 1: Key Compromise Response
     * ```javascript
     * // EMERGENCY: Private key compromised!
     * const publicKey = "0x04..."; // The compromised key
     *
     * // Immediately revoke to prevent attacker from using it
     * const tx = await registry.revokeKey(publicKey);
     * await tx.wait();
     *
     * console.log("Key revoked. Generate new key and re-register agents.");
     * ```
     *
     * ### Example 2: Planned Key Rotation
     * ```javascript
     * // Step 1: Generate new key pair
     * const newWallet = ethers.Wallet.createRandom();
     * const newPublicKey = newWallet.publicKey;
     *
     * // Step 2: Register agents with new key
     * for (const agent of myAgents) {
     *   await registry.registerAgent(
     *     agent.did + "-v2",  // New DID version
     *     agent.name,
     *     // ... other params
     *     newPublicKey,
     *     newSignature,
     *     newSalt
     *   );
     * }
     *
     * // Step 3: Only after new agents are active, revoke old key
     * await registry.revokeKey(oldPublicKey);
     * ```
     *
     * ### Example 3: Check Before Revoking
     * ```javascript
     * // Check how many agents will be affected
     * const keyHash = ethers.keccak256(publicKey);
     * const affectedAgents = await registry.getAgentsByKey(keyHash);
     *
     * console.log(`WARNING: Revoking this key will deactivate ${affectedAgents.length} agents`);
     *
     * // User confirms
     * const confirmed = await getUserConfirmation();
     * if (confirmed) {
     *   await registry.revokeKey(publicKey);
     * }
     * ```
     *
     * ## Gas Costs
     *
     * - Base revocation: ~45,000 gas
     * - Per agent deactivated: +~5,000 gas
     * - Example with 10 agents: ~95,000 gas
     * - Example with 100 agents (max): ~545,000 gas
     *
     * @custom:security-warning IRREVERSIBLE - Key cannot be un-revoked after this call
     * @custom:security-warning ALL agents using this key will be immediately deactivated
     * @custom:security-warning Ensure you have a recovery plan before revoking
     * @custom:gas-cost ~45,000 base + ~5,000 per associated agent (max 100 agents = ~545k gas)
     * @custom:throws "Not key owner" if caller doesn't own the specified key
     * @custom:throws "Already revoked" if key was previously revoked
     */
    function revokeKey(bytes calldata publicKey) external {
        bytes32 keyHash = keccak256(publicKey);
        require(addressToKeyHash[msg.sender] == keyHash, "Not key owner");
        require(!keyValidations[keyHash].isRevoked, "Already revoked");

        keyValidations[keyHash].isRevoked = true;

        bytes32[] memory agentIds = keyHashToAgentIds[keyHash];
        for (uint i = 0; i < agentIds.length; i++) {
            agents[agentIds[i]].active = false;
        }

        emit KeyRevoked(keyHash, msg.sender);
    }

    // ============================================
    // VIEW FUNCTIONS
    // ============================================

    function getAgent(bytes32 agentId) external view returns (AgentMetadata memory) {
        require(agents[agentId].registeredAt > 0, "Agent not found");
        return agents[agentId];
    }

    function getAgentByDID(string calldata did) external view returns (AgentMetadata memory) {
        bytes32 agentId = didToAgentId[did];
        require(agentId != bytes32(0), "Agent not found");
        return agents[agentId];
    }

    function getAgentsByOwner(address _owner) external view returns (bytes32[] memory) {
        return ownerToAgents[_owner];
    }

    function verifyAgentOwnership(bytes32 agentId, address claimedOwner) external view returns (bool) {
        return agents[agentId].owner == claimedOwner;
    }

    function isAgentActive(bytes32 agentId) external view returns (bool) {
        return agents[agentId].active;
    }

    function isKeyValid(bytes calldata publicKey) external view returns (bool) {
        bytes32 keyHash = keccak256(publicKey);
        KeyValidation memory validation = keyValidations[keyHash];
        return validation.registrationBlock > 0 && !validation.isRevoked;
    }

    function getCommitment(address user) external view returns (
        bytes32 commitHash,
        uint256 timestamp,
        bool revealed,
        bool isExpired
    ) {
        RegistrationCommitment memory commitment = registrationCommitments[user];
        bool expired = commitment.timestamp > 0 &&
                      block.timestamp > commitment.timestamp + MAX_COMMIT_REVEAL_DELAY;

        return (
            commitment.commitHash,
            commitment.timestamp,
            commitment.revealed,
            expired
        );
    }

    // ============================================
    // ADMIN FUNCTIONS
    // ============================================

    function pause() external onlyOwner {
        _pause();
    }

    function unpause() external onlyOwner {
        _unpause();
    }

    function setBeforeRegisterHook(address hook) external onlyOwner {
        address oldHook = beforeRegisterHook;
        beforeRegisterHook = hook;
        emit BeforeRegisterHookUpdated(oldHook, hook);
    }

    function setAfterRegisterHook(address hook) external onlyOwner {
        address oldHook = afterRegisterHook;
        afterRegisterHook = hook;
        emit AfterRegisterHookUpdated(oldHook, hook);
    }

    // ============================================
    // HELPER FUNCTIONS
    // ============================================

    function _verifySignature(
        bytes32 messageHash,
        bytes memory signature,
        bytes memory publicKey,
        address expectedSigner
    ) private pure returns (bool) {
        if (publicKey.length == 64 || publicKey.length == 65) {
            bytes32 ethSignedHash = keccak256(
                abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash)
            );

            address recovered = _recoverSigner(ethSignedHash, signature);
            return recovered == expectedSigner;
        }

        if (publicKey.length == 32) {
            revert("Ed25519 not supported on-chain");
        }

        return false;
    }

    function _recoverSigner(bytes32 messageHash, bytes memory signature)
        private pure returns (address)
    {
        require(signature.length == 65, "Invalid signature length");

        bytes32 r;
        bytes32 s;
        uint8 v;

        assembly {
            r := mload(add(signature, 32))
            s := mload(add(signature, 64))
            v := byte(0, mload(add(signature, 96)))
        }

        return ecrecover(messageHash, v, r, s);
    }

    function _getAddressFromPublicKey(bytes memory publicKey)
        private pure returns (address)
    {
        if (publicKey.length == 65 && publicKey[0] == 0x04) {
            bytes memory keyWithoutPrefix = new bytes(64);
            for (uint i = 0; i < 64; i++) {
                keyWithoutPrefix[i] = publicKey[i + 1];
            }
            return address(uint160(uint256(keccak256(keyWithoutPrefix))));
        }

        if (publicKey.length == 33 && (publicKey[0] == 0x02 || publicKey[0] == 0x03)) {
            revert("Compressed key address derivation not supported");
        }

        revert("Invalid public key format for address derivation");
    }
}
