// Package main demonstrates a basic MCP calculator tool secured with SAGE
package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/rfc9421"
	"github.com/sage-x-project/sage/did"
)

// CalculatorTool represents a simple MCP tool that performs calculations
type CalculatorTool struct {
	name        string
	description string
	sage        *core.Core
	didManager  *did.Manager
}

// ToolRequest represents an MCP tool request
type ToolRequest struct {
	Tool      string                 `json:"tool"`
	Operation string                 `json:"operation"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResponse represents an MCP tool response
type ToolResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// NewCalculatorTool creates a new calculator tool with SAGE security
func NewCalculatorTool() (*CalculatorTool, error) {
	// Initialize SAGE core
	sageCore := core.New()
	if sageCore == nil {
		return nil, fmt.Errorf("failed to initialize SAGE")
	}

	// Initialize DID manager
	didManager := did.NewManager()

	// For this demo, we'll use mock configuration
	// In production, use real contract address and RPC endpoint
	config := &did.RegistryConfig{
		ContractAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f7F1a", // Example address
		RPCEndpoint:     "https://eth-mainnet.example.com",              // Example endpoint
	}
	didManager.Configure(did.ChainEthereum, config)

	return &CalculatorTool{
		name:        "calculator",
		description: "Performs basic arithmetic operations",
		sage:        sageCore,
		didManager:  didManager,
	}, nil
}

// HandleRequest processes incoming tool requests with SAGE verification
func (t *CalculatorTool) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Step 1: Verify SAGE signature
	verifier := rfc9421.NewHTTPVerifier()

	// Extract agent DID from headers
	agentDID := r.Header.Get("X-Agent-DID")
	if agentDID == "" {
		http.Error(w, "Missing X-Agent-DID header", http.StatusBadRequest)
		return
	}

	// Resolve agent's public key from DID
	publicKey, err := t.didManager.ResolvePublicKey(r.Context(), did.AgentDID(agentDID))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve agent DID: %v", err), http.StatusUnauthorized)
		return
	}

	// Verify the request signature
	err = verifier.VerifyRequest(r, publicKey, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusUnauthorized)
		return
	}

	// Step 2: Check agent capabilities
	agentMetadata, err := t.didManager.ResolveAgent(r.Context(), did.AgentDID(agentDID))
	if err != nil {
		http.Error(w, "Failed to get agent metadata", http.StatusInternalServerError)
		return
	}

	// Verify agent has permission to use calculator
	if !t.hasCapability(agentMetadata.Capabilities, "calculator") {
		http.Error(w, "Agent lacks calculator capability", http.StatusForbidden)
		return
	}

	// Step 3: Parse and process the request
	var req ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Execute the calculation
	result, err := t.calculate(req.Operation, req.Arguments)

	var response ToolResponse
	if err != nil {
		response.Error = err.Error()
	} else {
		response.Result = result
	}

	// Step 4: Sign and send the response
	signer := rfc9421.NewHTTPSigner()
	if err := signer.SignResponse(w, response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to sign response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// calculate performs the actual calculation
func (t *CalculatorTool) calculate(operation string, args map[string]interface{}) (float64, error) {
	switch operation {
	case "add":
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return a + b, nil
	case "subtract":
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return a - b, nil
	case "multiply":
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return a * b, nil
	case "divide":
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return a / b, nil
	default:
		return 0, fmt.Errorf("unknown operation: %s", operation)
	}
}

// hasCapability checks if the agent has a specific capability
func (t *CalculatorTool) hasCapability(capabilities map[string]interface{}, capability string) bool {
	if val, ok := capabilities[capability]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

// GetToolDefinition returns the MCP tool definition
func (t *CalculatorTool) GetToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        t.name,
		"description": t.description,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"add", "subtract", "multiply", "divide"},
					"description": "The arithmetic operation to perform",
				},
				"arguments": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{
							"type":        "number",
							"description": "First operand",
						},
						"b": map[string]interface{}{
							"type":        "number",
							"description": "Second operand",
						},
					},
					"required": []string{"a", "b"},
				},
			},
			"required": []string{"operation", "arguments"},
		},
	}
}
