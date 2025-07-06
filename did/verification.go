package did

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MetadataVerifier provides DID metadata verification and validation
type MetadataVerifier struct {
	resolver Resolver
}

// NewMetadataVerifier creates a new metadata verifier
func NewMetadataVerifier(resolver Resolver) *MetadataVerifier {
	return &MetadataVerifier{
		resolver: resolver,
	}
}

// ValidationOptions contains options for DID validation
type ValidationOptions struct {
	// RequireActiveAgent ensures the agent is active
	RequireActiveAgent bool
	
	// RequiredCapabilities are capabilities the agent must have
	RequiredCapabilities []string
	
	// ValidateEndpoint ensures the endpoint is reachable (future enhancement)
	ValidateEndpoint bool
}

// DefaultValidationOptions returns default validation options
func DefaultValidationOptions() *ValidationOptions {
	return &ValidationOptions{
		RequireActiveAgent: true,
		ValidateEndpoint:   false,
	}
}

// ValidateAgent validates an agent's DID and metadata
func (v *MetadataVerifier) ValidateAgent(ctx context.Context, did AgentDID, opts *ValidationOptions) (*AgentMetadata, error) {
	if opts == nil {
		opts = DefaultValidationOptions()
	}
	
	// Resolve agent metadata
	agent, err := v.resolver.Resolve(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve agent DID: %w", err)
	}
	
	// Check if agent is active
	if opts.RequireActiveAgent && !agent.IsActive {
		return nil, ErrInactiveAgent
	}
	
	// Check required capabilities
	if len(opts.RequiredCapabilities) > 0 {
		if !hasRequiredCapabilities(agent.Capabilities, opts.RequiredCapabilities) {
			return nil, fmt.Errorf("agent missing required capabilities")
		}
	}
	
	// TODO: Add endpoint validation if opts.ValidateEndpoint is true
	
	return agent, nil
}

// VerifyMetadataConsistency checks if provided metadata matches on-chain data
func (v *MetadataVerifier) VerifyMetadataConsistency(ctx context.Context, did AgentDID, providedMetadata *AgentMetadata) (*VerificationResult, error) {
	// Use the resolver's built-in verification
	return v.resolver.VerifyMetadata(ctx, did, providedMetadata)
}

// CheckCapabilities verifies if an agent has specific capabilities
func (v *MetadataVerifier) CheckCapabilities(ctx context.Context, did AgentDID, requiredCapabilities []string) (bool, error) {
	agent, err := v.resolver.Resolve(ctx, did)
	if err != nil {
		return false, fmt.Errorf("failed to resolve agent DID: %w", err)
	}
	
	if !agent.IsActive {
		return false, ErrInactiveAgent
	}
	
	return hasRequiredCapabilities(agent.Capabilities, requiredCapabilities), nil
}

// GetAgentPublicKey retrieves and validates an agent's public key
func (v *MetadataVerifier) GetAgentPublicKey(ctx context.Context, did AgentDID) (interface{}, error) {
	return v.resolver.ResolvePublicKey(ctx, did)
}

// MatchMetadata checks if provided metadata matches expected values
func (v *MetadataVerifier) MatchMetadata(agentMetadata *AgentMetadata, expectedValues map[string]interface{}) error {
	// Check endpoint match
	if endpoint, ok := expectedValues["endpoint"].(string); ok {
		if endpoint != agentMetadata.Endpoint {
			return fmt.Errorf("endpoint mismatch: expected %s, got %s", endpoint, agentMetadata.Endpoint)
		}
	}
	
	// Check name match
	if name, ok := expectedValues["name"].(string); ok {
		if name != agentMetadata.Name {
			return fmt.Errorf("name mismatch: expected %s, got %s", name, agentMetadata.Name)
		}
	}
	
	// Check capabilities match
	if capabilities, ok := expectedValues["capabilities"].(map[string]interface{}); ok {
		for key, expectedValue := range capabilities {
			agentValue, exists := agentMetadata.Capabilities[key]
			if !exists {
				return fmt.Errorf("capability %s not found in agent", key)
			}
			
			// Deep comparison
			if !compareValues(expectedValue, agentValue) {
				return fmt.Errorf("capability %s value mismatch", key)
			}
		}
	}
	
	return nil
}

// ValidateAgentForOperation checks if an agent is valid for a specific operation
func (v *MetadataVerifier) ValidateAgentForOperation(ctx context.Context, did AgentDID, operation string, requiredCapabilities []string) (*ValidationResult, error) {
	result := &ValidationResult{
		OperationType: operation,
		Timestamp:     time.Now(),
	}
	
	// Resolve agent
	agent, err := v.resolver.Resolve(ctx, did)
	if err != nil {
		result.Valid = false
		result.Error = fmt.Sprintf("failed to resolve DID: %v", err)
		return result, nil
	}
	
	result.Agent = agent
	
	// Check if agent is active
	if !agent.IsActive {
		result.Valid = false
		result.Error = "agent is not active"
		return result, nil
	}
	
	// Check capabilities
	if len(requiredCapabilities) > 0 {
		if !hasRequiredCapabilities(agent.Capabilities, requiredCapabilities) {
			result.Valid = false
			result.Error = "agent missing required capabilities"
			result.MissingCapabilities = findMissingCapabilities(agent.Capabilities, requiredCapabilities)
			return result, nil
		}
	}
	
	result.Valid = true
	return result, nil
}

// ValidationResult contains detailed validation results
type ValidationResult struct {
	Valid               bool           `json:"valid"`
	Agent               *AgentMetadata `json:"agent,omitempty"`
	Error               string         `json:"error,omitempty"`
	OperationType       string         `json:"operation_type"`
	MissingCapabilities []string       `json:"missing_capabilities,omitempty"`
	Timestamp           time.Time      `json:"timestamp"`
}

// Helper functions

func hasRequiredCapabilities(agentCaps map[string]interface{}, required []string) bool {
	for _, req := range required {
		value, exists := agentCaps[req]
		if !exists {
			return false
		}
		boolValue, ok := value.(bool)
		if !ok || !boolValue {
			return false
		}
	}
	return true
}

func findMissingCapabilities(agentCaps map[string]interface{}, required []string) []string {
	var missing []string
	for _, req := range required {
		value, exists := agentCaps[req]
		if !exists {
			missing = append(missing, req)
			continue
		}
		boolValue, ok := value.(bool)
		if !ok || !boolValue {
			missing = append(missing, req)
		}
	}
	return missing
}

func compareValues(v1, v2 interface{}) bool {
	// Simple comparison - can be enhanced for deep object comparison
	j1, _ := json.Marshal(v1)
	j2, _ := json.Marshal(v2)
	return string(j1) == string(j2)
}