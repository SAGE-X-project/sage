// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// SAGEClient represents an AI agent that can securely call MCP tools
type SAGEClient struct {
	agentDID   string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	httpClient *http.Client
	verifier   *rfc9421.HTTPVerifier
}

// NewSAGEClient creates a new SAGE-enabled client
func NewSAGEClient(agentDID string) (*SAGEClient, error) {
	// Generate or load key pair
	keyPair, err := keys.GenerateEd25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	privateKey, ok := keyPair.PrivateKey().(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid private key type")
	}

	publicKey, ok := keyPair.PublicKey().(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid public key type")
	}

	return &SAGEClient{
		agentDID:   agentDID,
		privateKey: privateKey,
		publicKey:  publicKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		verifier:   rfc9421.NewHTTPVerifier(),
	}, nil
}

// CallTool makes a secure call to an MCP tool
func (c *SAGEClient) CallTool(toolURL string, request interface{}) (interface{}, error) {
	// Serialize request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", toolURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-DID", c.agentDID)
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))

	// Sign the request with SAGE
	params := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{
			`"@method"`,
			`"@path"`,
			`"content-type"`,
			`"content-length"`,
			`"date"`,
			`"x-agent-did"`,
		},
		KeyID:     c.agentDID,
		Algorithm: "ed25519",
		Created:   time.Now().Unix(),
	}

	err = c.verifier.SignRequest(req, "sig1", params, c.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tool returned error: %s - %s", resp.Status, string(responseBody))
	}

	// Verify response signature (if present)
	if resp.Header.Get("Signature") != "" {
		// In a real implementation, we would resolve the tool's public key
		// and verify the response signature
		fmt.Println(" Response signature present (verification would happen here)")
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// Example usage showing how simple it is to add SAGE
func ExampleUsage() {
	// This is all you need to add SAGE security to your AI agent!
	client, err := NewSAGEClient("did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f7F1a")
	if err != nil {
		panic(err)
	}

	// Make a secure tool call
	result, err := client.CallTool("http://localhost:8080/tools/calculator/execute", map[string]interface{}{
		"tool":      "calculator",
		"operation": "add",
		"arguments": map[string]interface{}{
			"a": 10,
			"b": 20,
		},
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}
}
