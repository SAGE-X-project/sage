package ethereum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// ErrCommitRevealRequired is returned by AgentCardClient.Register: the
// AgentCardRegistry contract only accepts registrations through the
// commit-reveal flow, which spans several transactions and waiting periods.
var ErrCommitRevealRequired = errors.New("AgentCardRegistry registers agents through commit-reveal: " +
	"use AgentCardClient.CommitRegistration, RegisterAgent and ActivateAgent " +
	"(sage-did commit / register / activate)")

// ErrSearchUnsupported is returned by Search: the registry keeps no index
// that can be queried on-chain.
var ErrSearchUnsupported = errors.New("search requires off-chain indexing")

// AgentCardClient is the Ethereum did.ChainClient: it is both the Registry
// and the Resolver did.Manager installs for did.ChainEthereum.
var _ did.ChainClient = (*AgentCardClient)(nil)

// Register always returns ErrCommitRevealRequired; see CommitRegistration.
func (c *AgentCardClient) Register(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
	return nil, ErrCommitRevealRequired
}

// Resolve retrieves agent metadata by DID.
func (c *AgentCardClient) Resolve(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	return c.GetAgentByDID(ctx, string(agentDID))
}

// ResolvePublicKey retrieves only the signing key of an active agent.
func (c *AgentCardClient) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	metadata, err := c.Resolve(ctx, agentDID)
	if err != nil {
		return nil, err
	}
	if !metadata.IsActive {
		return nil, did.ErrInactiveAgent
	}
	if metadata.PublicKey == nil {
		return nil, did.ErrNoSigningKey
	}
	return metadata.PublicKey, nil
}

// ResolveKEMKey retrieves the raw X25519 KEM key of an active agent, or nil.
func (c *AgentCardClient) ResolveKEMKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	metadata, err := c.Resolve(ctx, agentDID)
	if err != nil {
		return nil, err
	}
	if !metadata.IsActive {
		return nil, did.ErrInactiveAgent
	}
	return metadata.PublicKEMKey, nil
}

// VerifyMetadata checks if the provided metadata matches the on-chain data.
func (c *AgentCardClient) VerifyMetadata(ctx context.Context, agentDID did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
	onChainData, err := c.Resolve(ctx, agentDID)
	if err != nil {
		if errors.Is(err, did.ErrDIDNotFound) {
			return &did.VerificationResult{
				Valid:      false,
				Error:      "DID not found on chain",
				VerifiedAt: time.Now(),
			}, nil
		}
		return nil, err
	}

	valid := true
	var problems []string
	if metadata.Name != onChainData.Name {
		valid = false
		problems = append(problems, fmt.Sprintf("name mismatch: expected %s, got %s", onChainData.Name, metadata.Name))
	}
	if metadata.Description != onChainData.Description {
		valid = false
		problems = append(problems, "description mismatch")
	}
	if metadata.Endpoint != onChainData.Endpoint {
		valid = false
		problems = append(problems, fmt.Sprintf("endpoint mismatch: expected %s, got %s", onChainData.Endpoint, metadata.Endpoint))
	}
	if !compareCapabilities(metadata.Capabilities, onChainData.Capabilities) {
		valid = false
		problems = append(problems, "capabilities mismatch")
	}

	result := &did.VerificationResult{
		Valid:      valid,
		Agent:      onChainData,
		VerifiedAt: time.Now(),
	}
	if !valid {
		result.Error = strings.Join(problems, "; ")
	}
	return result, nil
}

// ListAgentsByOwner retrieves every agent owned by an address. Agents whose
// record cannot be read are skipped.
func (c *AgentCardClient) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
	if !common.IsHexAddress(ownerAddress) {
		return nil, fmt.Errorf("invalid Ethereum address: %s", ownerAddress)
	}
	ids, err := c.contract.GetAgentsByOwner(&bind.CallOpts{Context: ctx}, common.HexToAddress(ownerAddress))
	if err != nil {
		return nil, fmt.Errorf("failed to get agents by owner: %w", err)
	}
	agents := make([]*did.AgentMetadata, 0, len(ids))
	for _, id := range ids {
		metadata, err := c.GetAgent(ctx, id)
		if err != nil {
			continue
		}
		agents = append(agents, metadata)
	}
	return agents, nil
}

// Search always returns ErrSearchUnsupported.
func (c *AgentCardClient) Search(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	return nil, ErrSearchUnsupported
}

// GetRegistrationStatus reads the receipt of a registration transaction.
func (c *AgentCardClient) GetRegistrationStatus(ctx context.Context, txHash string) (*did.RegistrationResult, error) {
	receipt, err := c.client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	if receipt.Status == 0 {
		return nil, fmt.Errorf("transaction failed")
	}
	block, err := c.client.BlockByNumber(ctx, receipt.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}
	blockTime := block.Time()
	const maxInt64 = 1<<63 - 1
	if blockTime > maxInt64 {
		return nil, fmt.Errorf("block timestamp overflow: %d exceeds maximum int64 value", blockTime)
	}
	return &did.RegistrationResult{
		TransactionHash: txHash,
		BlockNumber:     receipt.BlockNumber.Uint64(),
		Timestamp:       time.Unix(int64(blockTime), 0), // #nosec G115 - overflow checked above
		GasUsed:         receipt.GasUsed,
	}, nil
}

// Update changes the endpoint and/or capabilities of an agent. The
// AgentCardRegistry contract cannot change the name or description, so
// those keys are rejected. A field that is not in updates keeps its current
// on-chain value. The transaction is signed by the configured private key;
// keyPair is accepted for interface compatibility and not used.
func (c *AgentCardClient) Update(ctx context.Context, agentDID did.AgentDID, updates map[string]interface{}, keyPair sagecrypto.KeyPair) error {
	for _, key := range []string{"name", "description"} {
		if _, ok := updates[key]; ok {
			return fmt.Errorf("AgentCardRegistry does not allow changing the %s", key)
		}
	}
	endpoint, hasEndpoint := updates["endpoint"].(string)
	capabilities, hasCaps := updates["capabilities"]
	if !hasEndpoint && !hasCaps {
		return fmt.Errorf("no updatable field given (endpoint, capabilities)")
	}

	current, err := c.Resolve(ctx, agentDID)
	if err != nil {
		return err
	}
	if !hasEndpoint {
		endpoint = current.Endpoint
	}
	if !hasCaps {
		capabilities = current.Capabilities
	}
	capBytes, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}
	return c.UpdateAgent(ctx, c.computeAgentID(string(agentDID)), endpoint, string(capBytes))
}

// Deactivate deactivates an agent. The transaction is signed by the
// configured private key; keyPair is accepted for interface compatibility
// and not used.
func (c *AgentCardClient) Deactivate(ctx context.Context, agentDID did.AgentDID, keyPair sagecrypto.KeyPair) error {
	return c.DeactivateAgent(ctx, c.computeAgentID(string(agentDID)))
}

// compareCapabilities reports whether two capability maps are equal.
func compareCapabilities(cap1, cap2 map[string]interface{}) bool {
	if len(cap1) != len(cap2) {
		return false
	}
	json1, err1 := json.Marshal(cap1)
	json2, err2 := json.Marshal(cap2)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(string(json1), string(json2))
}
