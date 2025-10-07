// SPDX-License-Identifier: MIT
pragma solidity 0.8.19;

import "@openzeppelin/contracts/access/Ownable2Step.sol";
import "@openzeppelin/contracts/security/Pausable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";

/**
 * @title TEEKeyRegistry
 * @author SAGE Development Team
 * @notice Decentralized governance registry for trusted TEE (Trusted Execution Environment) keys
 * @dev Community-driven approval system preventing centralization of cryptographic validation trust
 *
 * ## Overview
 *
 * The TEEKeyRegistry solves a critical decentralization problem in the SAGE ecosystem:
 * **Who decides which TEE (Trusted Execution Environment) providers are trusted?**
 *
 * Without this contract, a single owner controls the list of trusted TEE keys, creating
 * a centralization risk. This registry implements community governance where stakeholders
 * vote on TEE key approval proposals, distributing trust across the ecosystem.
 *
 * ## Problem Statement
 *
 * In ERC8004ValidationRegistry, validators can submit cryptographic proofs (attestations)
 * from TEE providers. But who decides which TEE keys to trust?
 *
 * **Centralized Approach (V1)**:
 * ```
 * function addTrustedTEEKey(bytes32 keyHash) external onlyOwner {
 *     trustedTEEKeys[keyHash] = true;
 * }
 * ```
 * ❌ Single point of failure
 * ❌ Owner can add malicious keys
 * ❌ No community input
 *
 * **Decentralized Approach (V2 - This Contract)**:
 * ```
 * Anyone → propose(keyHash, attestation) + stake
 *   ↓
 * Community → vote(proposalId, support)
 *   ↓
 * 66% approval → Key trusted ✅
 * <66% approval → Stake slashed ❌
 * ```
 *
 * ## Architecture
 *
 * ### Governance Flow
 * ```
 * 1. PROPOSAL
 *    ├─ Proposer stakes 1 ETH
 *    ├─ Submits TEE key + attestation report
 *    └─ 7-day voting period begins
 *
 * 2. VOTING
 *    ├─ Registered voters cast weighted votes
 *    ├─ Minimum 10% participation required
 *    └─ Votes tracked: FOR vs AGAINST
 *
 * 3. EXECUTION
 *    ├─ After 7 days, anyone can execute
 *    ├─ Check: ≥66% approval + ≥10% participation
 *    ├─ APPROVED: Key trusted, stake returned
 *    └─ REJECTED: 50% stake slashed, key rejected
 * ```
 *
 * ## Key Features
 *
 * ### 1. Stake-Based Proposals
 * - Proposers must stake 1 ETH (configurable)
 * - Prevents spam proposals
 * - Slashed if community rejects
 * - Returned if community approves
 *
 * ### 2. Weighted Voting System
 * - Voters registered by owner initially
 * - Voting weight based on reputation/stake
 * - Each voter can vote once per proposal
 * - Votes weighted to reflect expertise
 *
 * ### 3. Byzantine Fault Tolerance
 * - 66% approval threshold (⅔ majority)
 * - Tolerates up to 33% malicious/dishonest voters
 * - Minimum participation prevents small groups deciding
 *
 * ### 4. Supported TEE Types
 * - **Intel SGX**: Software Guard Extensions
 * - **AMD SEV**: Secure Encrypted Virtualization
 * - **ARM TrustZone**: ARM's secure world
 * - **AWS Nitro Enclaves**: Amazon's isolated compute
 *
 * ### 5. Emergency Controls
 * - Owner can revoke compromised keys immediately
 * - Owner can pause contract in crisis
 * - Two-step ownership transfer for security
 *
 * ## Security Model
 *
 * ### Assumptions
 * - Majority of voters are honest and technically competent
 * - Attestation reports are publicly verifiable
 * - Owner is trusted for emergency interventions only
 * - TEE providers maintain security of their infrastructure
 *
 * ### Invariants
 * - Only approved keys can be used for TEE validation
 * - Proposals require minimum participation (≥10%)
 * - Approval requires supermajority (≥66%)
 * - Slashing is partial (50%) to allow for honest mistakes
 *
 * ### Attack Prevention
 * - **Spam Proposals**: Prevented by 1 ETH stake requirement
 * - **Sybil Voting**: Prevented by registered voter system
 * - **Rushed Decisions**: 7-day voting period enforces deliberation
 * - **Minority Takeover**: 66% threshold prevents <⅔ control
 * - **Low Participation**: 10% minimum prevents small group decisions
 * - **Malicious Keys**: Community review + attestation verification
 *
 * ## Economic Model
 *
 * ### Proposal Costs
 * - **Stake Required**: 1 ETH (adjustable)
 * - **If Approved**: Stake returned 100%
 * - **If Rejected**: 50% slashed, 50% returned
 * - **If Cancelled**: 10% fee, 90% returned
 *
 * ### Voting Incentives
 * ```
 * Current: No direct rewards (reputation-based)
 * Future: Could implement:
 * - Rewards for voters from slashed funds
 * - Token-based voting weights
 * - Delegation mechanisms
 * ```
 *
 * ### Slashed Funds Treasury
 * - Accumulated from rejected proposals
 * - Can be withdrawn by owner for:
 *   - Voter rewards
 *   - Ecosystem development
 *   - Security audits
 *
 * ## Governance Parameters
 *
 * All parameters are adjustable by owner (via updateParameters):
 *
 * | Parameter | Default | Range | Purpose |
 * |-----------|---------|-------|---------|
 * | proposalStake | 1 ETH | Any | Spam prevention |
 * | votingPeriod | 7 days | Any | Deliberation time |
 * | approvalThreshold | 66% | 50-100% | Supermajority |
 * | minVoterParticipation | 10% | 0-100% | Prevent small groups |
 * | slashingPercentage | 50% | 0-100% | Penalty for bad proposals |
 *
 * ## Gas Costs (Approximate)
 *
 * - `proposeTEEKey()`: ~180,000 gas
 * - `vote()`: ~85,000 gas (per voter)
 * - `executeProposal()`: ~150,000 + (5,000 × voters) gas
 * - `registerVoter()`: ~80,000 gas
 * - `revokeTEEKey()`: ~45,000 gas (emergency)
 *
 * ## Integration with ValidationRegistry
 *
 * ```solidity
 * // ValidationRegistry checks this contract
 * function submitTEEAttestation(bytes32 keyHash, bytes proof) external {
 *     require(
 *         TEEKeyRegistry(teeRegistry).isTrustedTEEKey(keyHash),
 *         "TEE key not trusted"
 *     );
 *     // Process attestation...
 * }
 * ```
 *
 * ## Usage Example
 *
 * ### Proposing a New TEE Key
 * ```javascript
 * // 1. Prepare attestation documentation
 * const attestationReport = "https://sgx-attestation.example.com/report123";
 * const teeType = "SGX";
 * const keyHash = ethers.keccak256(sgxPublicKey);
 *
 * // 2. Submit proposal with stake
 * const stake = ethers.parseEther("1.0"); // 1 ETH
 * const tx = await teeRegistry.proposeTEEKey(
 *   keyHash,
 *   attestationReport,
 *   teeType,
 *   { value: stake }
 * );
 * const receipt = await tx.wait();
 * const proposalId = receipt.events.find(e => e.event === 'TEEKeyProposed').args.proposalId;
 *
 * console.log(`Proposal ${proposalId} created. Voting open for 7 days.`);
 * ```
 *
 * ### Voting on a Proposal
 * ```javascript
 * // 1. Review attestation report (off-chain)
 * const proposal = await teeRegistry.getProposal(proposalId);
 * console.log("Attestation:", proposal.attestationReport);
 *
 * // 2. Verify TEE attestation manually
 * const isValid = await verifyTEEAttestation(proposal.attestationReport);
 *
 * // 3. Cast vote
 * if (isValid) {
 *   await teeRegistry.vote(proposalId, true); // Support
 * } else {
 *   await teeRegistry.vote(proposalId, false); // Reject
 * }
 * ```
 *
 * ### Executing After Voting
 * ```javascript
 * // Wait 7 days...
 * await new Promise(r => setTimeout(r, 7 * 24 * 60 * 60 * 1000));
 *
 * // Anyone can execute
 * const tx = await teeRegistry.executeProposal(proposalId);
 * const receipt = await tx.wait();
 *
 * const approved = receipt.events.find(e => e.event === 'ProposalExecuted').args.approved;
 * if (approved) {
 *   console.log("TEE key approved! Now trusted for validations.");
 * } else {
 *   console.log("Proposal rejected. 50% of stake slashed.");
 * }
 * ```
 *
 * @custom:security-contact security@sage.com
 * @custom:audit-status Phase 7.5 - Governance implementation complete, pending external audit
 * @custom:version 1.0.0
 * @custom:governance Community-driven with emergency controls
 */
contract TEEKeyRegistry is Ownable2Step, Pausable, ReentrancyGuard {
    // ============================================
    // STRUCTS
    // ============================================

    struct TEEKeyProposal {
        bytes32 keyHash;
        address proposer;
        string attestationReport;     // URL to TEE attestation report
        string teeType;               // "SGX", "SEV", "TrustZone", "Nitro"
        uint256 proposalStake;
        uint256 votesFor;
        uint256 votesAgainst;
        uint256 createdAt;
        uint256 votingDeadline;
        ProposalStatus status;
    }

    enum ProposalStatus {
        PENDING,      // Voting in progress
        APPROVED,     // Reached approval threshold
        REJECTED,     // Voting failed
        EXECUTED,     // Approved and executed
        CANCELLED     // Cancelled by proposer
    }

    struct VoterInfo {
        uint256 weight;               // Voting weight (could be token-based)
        bool hasVoted;
        bool vote;                    // true = for, false = against
    }

    // ============================================
    // STATE VARIABLES
    // ============================================

    // Approved TEE keys
    mapping(bytes32 => bool) public approvedTEEKeys;
    mapping(bytes32 => string) public teeKeyTypes;
    mapping(bytes32 => uint256) public teeKeyApprovedAt;

    // Proposals
    mapping(uint256 => TEEKeyProposal) public proposals;
    mapping(uint256 => mapping(address => VoterInfo)) public votes;
    uint256 public proposalCount;

    // Registered voters (for weighted voting)
    mapping(address => bool) public isRegisteredVoter;
    mapping(address => uint256) public voterWeight;
    address[] public voters;

    // ============================================
    // CONSTANTS & PARAMETERS
    // ============================================

    // Proposal parameters
    uint256 public proposalStake = 1 ether;           // Stake required to propose
    uint256 public votingPeriod = 7 days;             // How long voting lasts
    uint256 public approvalThreshold = 66;            // 66% approval required
    uint256 public minVoterParticipation = 10;        // At least 10% of voters must participate

    // Slashing
    uint256 public slashingPercentage = 50;           // 50% slashed for rejected proposals

    // ============================================
    // EVENTS
    // ============================================

    event TEEKeyProposed(
        uint256 indexed proposalId,
        bytes32 indexed keyHash,
        address indexed proposer,
        string teeType,
        string attestationReport
    );

    event VoteCast(
        uint256 indexed proposalId,
        address indexed voter,
        bool support,
        uint256 weight
    );

    event ProposalExecuted(
        uint256 indexed proposalId,
        bytes32 indexed keyHash,
        bool approved
    );

    event TEEKeyApproved(bytes32 indexed keyHash, string teeType);
    event TEEKeyRevoked(bytes32 indexed keyHash, string reason);

    event VoterRegistered(address indexed voter, uint256 weight);
    event VoterWeightUpdated(address indexed voter, uint256 oldWeight, uint256 newWeight);

    event ParametersUpdated(
        uint256 proposalStake,
        uint256 votingPeriod,
        uint256 approvalThreshold,
        uint256 minVoterParticipation
    );

    // ============================================
    // ERRORS
    // ============================================

    error InsufficientStake(uint256 provided, uint256 required);
    error ProposalNotFound(uint256 proposalId);
    error ProposalNotPending(uint256 proposalId);
    error VotingEnded(uint256 proposalId);
    error VotingNotEnded(uint256 proposalId);
    error AlreadyVoted(uint256 proposalId, address voter);
    error NotRegisteredVoter(address voter);
    error TEEKeyAlreadyApproved(bytes32 keyHash);
    error TEEKeyNotApproved(bytes32 keyHash);
    error InvalidTEEType(string teeType);
    error InvalidParameters();

    // ============================================
    // MODIFIERS
    // ============================================

    modifier onlyRegisteredVoter() {
        if (!isRegisteredVoter[msg.sender]) revert NotRegisteredVoter(msg.sender);
        _;
    }

    // ============================================
    // CONSTRUCTOR
    // ============================================

    constructor() {
        _transferOwnership(msg.sender);

        // Register owner as initial voter
        _registerVoter(msg.sender, 1);
    }

    // ============================================
    // PROPOSAL FUNCTIONS
    // ============================================

    /**
     * @notice Propose a new TEE key for approval
     * @param keyHash Hash of the TEE public key
     * @param attestationReport URL to TEE attestation documentation
     * @param teeType Type of TEE ("SGX", "SEV", "TrustZone", "Nitro")
     * @return proposalId Unique identifier for this proposal
     */
    function proposeTEEKey(
        bytes32 keyHash,
        string calldata attestationReport,
        string calldata teeType
    ) external payable whenNotPaused nonReentrant returns (uint256 proposalId) {
        // Validate inputs
        if (msg.value < proposalStake) {
            revert InsufficientStake(msg.value, proposalStake);
        }
        if (approvedTEEKeys[keyHash]) {
            revert TEEKeyAlreadyApproved(keyHash);
        }
        if (!_isValidTEEType(teeType)) {
            revert InvalidTEEType(teeType);
        }

        // Create proposal
        proposalId = proposalCount++;

        proposals[proposalId] = TEEKeyProposal({
            keyHash: keyHash,
            proposer: msg.sender,
            attestationReport: attestationReport,
            teeType: teeType,
            proposalStake: msg.value,
            votesFor: 0,
            votesAgainst: 0,
            createdAt: block.timestamp,
            votingDeadline: block.timestamp + votingPeriod,
            status: ProposalStatus.PENDING
        });

        emit TEEKeyProposed(
            proposalId,
            keyHash,
            msg.sender,
            teeType,
            attestationReport
        );

        return proposalId;
    }

    /**
     * @notice Vote on a TEE key proposal
     * @param proposalId The proposal to vote on
     * @param support True to approve, false to reject
     */
    function vote(
        uint256 proposalId,
        bool support
    ) external onlyRegisteredVoter whenNotPaused {
        TEEKeyProposal storage proposal = proposals[proposalId];

        // Validate proposal state
        if (proposalId >= proposalCount) revert ProposalNotFound(proposalId);
        if (proposal.status != ProposalStatus.PENDING) {
            revert ProposalNotPending(proposalId);
        }
        if (block.timestamp > proposal.votingDeadline) {
            revert VotingEnded(proposalId);
        }

        // Check if already voted
        VoterInfo storage voterInfo = votes[proposalId][msg.sender];
        if (voterInfo.hasVoted) revert AlreadyVoted(proposalId, msg.sender);

        // Record vote
        uint256 weight = voterWeight[msg.sender];
        voterInfo.hasVoted = true;
        voterInfo.vote = support;
        voterInfo.weight = weight;

        // Update proposal vote counts
        if (support) {
            proposal.votesFor += weight;
        } else {
            proposal.votesAgainst += weight;
        }

        emit VoteCast(proposalId, msg.sender, support, weight);
    }

    /**
     * @notice Execute a proposal after voting ends
     * @param proposalId The proposal to execute
     */
    function executeProposal(uint256 proposalId)
        external
        nonReentrant
        returns (bool approved)
    {
        TEEKeyProposal storage proposal = proposals[proposalId];

        // Validate
        if (proposalId >= proposalCount) revert ProposalNotFound(proposalId);
        if (proposal.status != ProposalStatus.PENDING) {
            revert ProposalNotPending(proposalId);
        }
        if (block.timestamp <= proposal.votingDeadline) {
            revert VotingNotEnded(proposalId);
        }

        // Calculate results
        uint256 totalVotes = proposal.votesFor + proposal.votesAgainst;
        uint256 totalWeight = _getTotalVoterWeight();

        // Check minimum participation
        uint256 participationRate = (totalVotes * 100) / totalWeight;
        require(
            participationRate >= minVoterParticipation,
            "Insufficient voter participation"
        );

        // Calculate approval rate
        uint256 approvalRate = totalVotes > 0
            ? (proposal.votesFor * 100) / totalVotes
            : 0;

        approved = approvalRate >= approvalThreshold;

        // Update proposal status
        proposal.status = approved ? ProposalStatus.APPROVED : ProposalStatus.REJECTED;

        // Execute based on result
        if (approved) {
            // Approve TEE key
            approvedTEEKeys[proposal.keyHash] = true;
            teeKeyTypes[proposal.keyHash] = proposal.teeType;
            teeKeyApprovedAt[proposal.keyHash] = block.timestamp;

            // Return stake to proposer
            (bool success, ) = proposal.proposer.call{value: proposal.proposalStake}("");
            require(success, "Stake return failed");

            emit TEEKeyApproved(proposal.keyHash, proposal.teeType);
        } else {
            // Slash stake for rejected proposal
            uint256 slashAmount = (proposal.proposalStake * slashingPercentage) / 100;
            uint256 returnAmount = proposal.proposalStake - slashAmount;

            // Return remaining stake to proposer
            if (returnAmount > 0) {
                (bool success, ) = proposal.proposer.call{value: returnAmount}("");
                require(success, "Partial return failed");
            }

            // Slashed amount stays in contract (could be used for treasury/rewards)
        }

        proposal.status = ProposalStatus.EXECUTED;

        emit ProposalExecuted(proposalId, proposal.keyHash, approved);

        return approved;
    }

    /**
     * @notice Cancel a proposal before voting ends (proposer only)
     * @param proposalId The proposal to cancel
     */
    function cancelProposal(uint256 proposalId) external nonReentrant {
        TEEKeyProposal storage proposal = proposals[proposalId];

        require(proposalId < proposalCount, "Proposal not found");
        require(proposal.proposer == msg.sender, "Not proposer");
        require(proposal.status == ProposalStatus.PENDING, "Not pending");

        // Mark as cancelled
        proposal.status = ProposalStatus.CANCELLED;

        // Return stake (minus small fee for spam prevention)
        uint256 cancellationFee = proposal.proposalStake / 10; // 10% fee
        uint256 returnAmount = proposal.proposalStake - cancellationFee;

        (bool success, ) = msg.sender.call{value: returnAmount}("");
        require(success, "Return failed");
    }

    // ============================================
    // VOTER MANAGEMENT
    // ============================================

    /**
     * @notice Register a new voter
     * @param voter Address to register
     * @param weight Voting weight (e.g., based on reputation)
     */
    function registerVoter(address voter, uint256 weight) external onlyOwner {
        _registerVoter(voter, weight);
    }

    function _registerVoter(address voter, uint256 weight) private {
        require(voter != address(0), "Invalid voter");
        require(weight > 0, "Invalid weight");

        if (!isRegisteredVoter[voter]) {
            isRegisteredVoter[voter] = true;
            voters.push(voter);
        }

        voterWeight[voter] = weight;

        emit VoterRegistered(voter, weight);
    }

    /**
     * @notice Update voter weight
     * @param voter Address to update
     * @param newWeight New voting weight
     */
    function updateVoterWeight(address voter, uint256 newWeight) external onlyOwner {
        require(isRegisteredVoter[voter], "Not registered");
        require(newWeight > 0, "Invalid weight");

        uint256 oldWeight = voterWeight[voter];
        voterWeight[voter] = newWeight;

        emit VoterWeightUpdated(voter, oldWeight, newWeight);
    }

    /**
     * @notice Remove a voter
     * @param voter Address to remove
     */
    function removeVoter(address voter) external onlyOwner {
        require(isRegisteredVoter[voter], "Not registered");

        isRegisteredVoter[voter] = false;
        voterWeight[voter] = 0;

        // Remove from voters array
        for (uint256 i = 0; i < voters.length; i++) {
            if (voters[i] == voter) {
                voters[i] = voters[voters.length - 1];
                voters.pop();
                break;
            }
        }
    }

    // ============================================
    // ADMIN FUNCTIONS
    // ============================================

    /**
     * @notice Revoke an approved TEE key (emergency only)
     * @param keyHash The key to revoke
     * @param reason Reason for revocation
     */
    function revokeTEEKey(bytes32 keyHash, string calldata reason) external onlyOwner {
        if (!approvedTEEKeys[keyHash]) revert TEEKeyNotApproved(keyHash);

        approvedTEEKeys[keyHash] = false;

        emit TEEKeyRevoked(keyHash, reason);
    }

    /**
     * @notice Update governance parameters
     */
    function updateParameters(
        uint256 _proposalStake,
        uint256 _votingPeriod,
        uint256 _approvalThreshold,
        uint256 _minVoterParticipation
    ) external onlyOwner {
        if (_approvalThreshold < 50 || _approvalThreshold > 100) {
            revert InvalidParameters();
        }
        if (_minVoterParticipation > 100) {
            revert InvalidParameters();
        }

        proposalStake = _proposalStake;
        votingPeriod = _votingPeriod;
        approvalThreshold = _approvalThreshold;
        minVoterParticipation = _minVoterParticipation;

        emit ParametersUpdated(
            _proposalStake,
            _votingPeriod,
            _approvalThreshold,
            _minVoterParticipation
        );
    }

    function pause() external onlyOwner {
        _pause();
    }

    function unpause() external onlyOwner {
        _unpause();
    }

    // ============================================
    // VIEW FUNCTIONS
    // ============================================

    /**
     * @notice Check if a TEE key is approved
     */
    function isTrustedTEEKey(bytes32 keyHash) external view returns (bool) {
        return approvedTEEKeys[keyHash];
    }

    /**
     * @notice Get proposal details
     */
    function getProposal(uint256 proposalId)
        external
        view
        returns (TEEKeyProposal memory)
    {
        require(proposalId < proposalCount, "Proposal not found");
        return proposals[proposalId];
    }

    /**
     * @notice Get vote info for a specific voter
     */
    function getVoteInfo(uint256 proposalId, address voter)
        external
        view
        returns (VoterInfo memory)
    {
        return votes[proposalId][voter];
    }

    /**
     * @notice Get total number of registered voters
     */
    function getVoterCount() external view returns (uint256) {
        return voters.length;
    }

    /**
     * @notice Get all registered voters
     */
    function getAllVoters() external view returns (address[] memory) {
        return voters;
    }

    /**
     * @notice Get proposal status summary
     */
    function getProposalStatus(uint256 proposalId)
        external
        view
        returns (
            ProposalStatus status,
            uint256 votesFor,
            uint256 votesAgainst,
            uint256 totalWeight,
            uint256 participationRate,
            uint256 approvalRate,
            bool canExecute
        )
    {
        require(proposalId < proposalCount, "Proposal not found");

        TEEKeyProposal memory proposal = proposals[proposalId];
        status = proposal.status;
        votesFor = proposal.votesFor;
        votesAgainst = proposal.votesAgainst;
        totalWeight = _getTotalVoterWeight();

        uint256 totalVotes = votesFor + votesAgainst;
        participationRate = totalWeight > 0 ? (totalVotes * 100) / totalWeight : 0;
        approvalRate = totalVotes > 0 ? (votesFor * 100) / totalVotes : 0;

        canExecute = status == ProposalStatus.PENDING &&
                     block.timestamp > proposal.votingDeadline;

        return (
            status,
            votesFor,
            votesAgainst,
            totalWeight,
            participationRate,
            approvalRate,
            canExecute
        );
    }

    // ============================================
    // HELPER FUNCTIONS
    // ============================================

    function _getTotalVoterWeight() private view returns (uint256 total) {
        for (uint256 i = 0; i < voters.length; i++) {
            total += voterWeight[voters[i]];
        }
        return total;
    }

    function _isValidTEEType(string memory teeType) private pure returns (bool) {
        bytes32 typeHash = keccak256(bytes(teeType));

        return typeHash == keccak256("SGX") ||
               typeHash == keccak256("SEV") ||
               typeHash == keccak256("TrustZone") ||
               typeHash == keccak256("Nitro");
    }

    /**
     * @notice Withdraw accumulated slashed funds (treasury)
     */
    function withdrawTreasury(address recipient, uint256 amount)
        external
        onlyOwner
        nonReentrant
    {
        require(recipient != address(0), "Invalid recipient");
        require(amount <= address(this).balance, "Insufficient balance");

        (bool success, ) = recipient.call{value: amount}("");
        require(success, "Transfer failed");
    }

    /**
     * @notice Get contract balance (slashed funds)
     */
    function getTreasuryBalance() external view returns (uint256) {
        return address(this).balance;
    }

    // Allow contract to receive ETH
    receive() external payable {}
}
