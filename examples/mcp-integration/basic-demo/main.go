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
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// === MCP Tool Types ===
type ToolRequest struct {
	Tool      string                 `json:"tool"`
	Operation string                 `json:"operation"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// === Calculator Tool ===
type Calculator struct {
	// In production, this would verify against blockchain DIDs
	trustedAgents map[string]ed25519.PublicKey
}

func NewCalculator() *Calculator {
	return &Calculator{
		trustedAgents: make(map[string]ed25519.PublicKey),
	}
}

// AddTrustedAgent registers a trusted agent for demo purposes
func (c *Calculator) AddTrustedAgent(agentDID string, publicKey ed25519.PublicKey) {
	c.trustedAgents[agentDID] = publicKey
}

// VerifyRequest checks SAGE signature on incoming request
func (c *Calculator) VerifyRequest(r *http.Request) error {
	// Get agent DID
	agentDID := r.Header.Get("X-Agent-DID")
	if agentDID == "" {
		return fmt.Errorf("missing X-Agent-DID header")
	}

	// Get public key (in production, resolve from blockchain)
	publicKey, ok := c.trustedAgents[agentDID]
	if !ok {
		return fmt.Errorf("unknown agent: %s", agentDID)
	}

	// Verify signature
	verifier := rfc9421.NewHTTPVerifier()
	return verifier.VerifyRequest(r, publicKey, nil)
}

// HandleRequest processes calculator requests
func (c *Calculator) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// 1. Verify SAGE signature
	if err := c.VerifyRequest(r); err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		fmt.Printf(" Rejected request: %v\n", err)
		return
	}

	agentDID := r.Header.Get("X-Agent-DID")
	fmt.Printf(" Verified request from: %s\n", agentDID)

	// 2. Parse request
	var req ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 3. Perform calculation
	result, err := c.Calculate(req.Operation, req.Arguments)

	// 4. Send response
	resp := ToolResponse{}
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Result = result
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Calculate performs the actual calculation
func (c *Calculator) Calculate(op string, args map[string]interface{}) (interface{}, error) {
	// Get operands - handle both float64 and int
	getNumber := func(key string) (float64, error) {
		val, ok := args[key]
		if !ok {
			return 0, fmt.Errorf("missing argument: %s", key)
		}
		switch v := val.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		default:
			return 0, fmt.Errorf("invalid number type for %s", key)
		}
	}

	a, err := getNumber("a")
	if err != nil {
		return nil, err
	}
	b, err := getNumber("b")
	if err != nil {
		return nil, err
	}

	switch op {
	case "add":
		return a + b, nil
	case "subtract":
		return a - b, nil
	case "multiply":
		return a * b, nil
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return a / b, nil
	default:
		return nil, fmt.Errorf("unknown operation: %s", op)
	}
}

// === Demo Agent ===
type DemoAgent struct {
	did        string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

func NewDemoAgent(name string) (*DemoAgent, error) {
	// Generate key pair
	keyPair, err := keys.GenerateEd25519KeyPair()
	if err != nil {
		return nil, err
	}

	privateKey, _ := keyPair.PrivateKey().(ed25519.PrivateKey)
	publicKey, _ := keyPair.PublicKey().(ed25519.PublicKey)

	return &DemoAgent{
		did:        fmt.Sprintf("did:sage:demo:%s", name),
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// CallTool makes a SAGE-signed request to the tool
func (a *DemoAgent) CallTool(url string, operation string, args map[string]interface{}) error {
	// Create request body
	reqBody := ToolRequest{
		Tool:      "calculator",
		Operation: operation,
		Arguments: args,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-DID", a.did)
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))

	// Sign the request
	verifier := rfc9421.NewHTTPVerifier()
	params := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{
			`"@method"`,
			`"@path"`,
			`"content-type"`,
			`"x-agent-did"`,
			`"date"`,
		},
		KeyID:     a.did,
		Algorithm: "ed25519",
		Created:   time.Now().Unix(),
	}

	err = verifier.SignRequest(req, "sig1", params, a.privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	// Make request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(body))
	}

	var result ToolResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Error != "" {
		fmt.Printf("   Error: %s\n", result.Error)
	} else {
		fmt.Printf("   Result: %v\n", result.Result)
	}

	return nil
}

// === Main ===
func main() {
	// Create calculator tool
	calc := NewCalculator()

	// Create demo agents
	alice, _ := NewDemoAgent("alice")
	bob, _ := NewDemoAgent("bob")
	eve, _ := NewDemoAgent("eve") // Untrusted agent

	// Register trusted agents
	calc.AddTrustedAgent(alice.did, alice.publicKey)
	calc.AddTrustedAgent(bob.did, bob.publicKey)
	// Note: Eve is NOT registered as trusted

	// Set up HTTP handler
	http.HandleFunc("/calculator", calc.HandleRequest)

	// Info endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `SAGE-Secured Calculator Tool

Endpoints:
  POST /calculator - Execute calculation (requires SAGE signature)

Example request:
{
  "tool": "calculator",
  "operation": "add",
  "arguments": {"a": 10, "b": 20}
}

Trusted agents:
  - %s
  - %s

Untrusted agents:
  - %s (will be rejected)
`, alice.did, bob.did, eve.did)
	})

	// Start server
	fmt.Println(" SAGE-Secured Calculator Tool")
	fmt.Println("📍 Listening on http://localhost:8080")
	fmt.Println("")
	fmt.Printf("Trusted agents:\n")
	fmt.Printf("  - %s\n", alice.did)
	fmt.Printf("  - %s\n", bob.did)
	fmt.Printf("\nUntrusted agents:\n")
	fmt.Printf("  - %s\n", eve.did)
	fmt.Println("")

	// Run demo requests after server starts
	go func() {
		time.Sleep(2 * time.Second)

		fmt.Println("\n=== Running Demo Requests ===")

		// Alice's request (trusted)
		fmt.Printf("\n1. Alice (trusted) requests: 10 + 20\n")
		_ = alice.CallTool("http://localhost:8080/calculator", "add", map[string]interface{}{
			"a": 10, "b": 20,
		})

		// Bob's request (trusted)
		fmt.Printf("\n2. Bob (trusted) requests: 100 / 5\n")
		_ = bob.CallTool("http://localhost:8080/calculator", "divide", map[string]interface{}{
			"a": 100, "b": 5,
		})

		// Eve's request (untrusted - will fail)
		fmt.Printf("\n3. Eve (untrusted) requests: 50 * 2\n")
		err := eve.CallTool("http://localhost:8080/calculator", "multiply", map[string]interface{}{
			"a": 50, "b": 2,
		})
		if err != nil {
			fmt.Printf("   Failed as expected: %v\n", err)
		}

		// Invalid request (no signature)
		fmt.Printf("\n4. Anonymous request (no signature)\n")
		resp, err := http.Post("http://localhost:8080/calculator", "application/json",
			strings.NewReader(`{"tool":"calculator","operation":"add","arguments":{"a":1,"b":1}}`))
		if err == nil {
			if resp.StatusCode == http.StatusUnauthorized {
				fmt.Printf("   Rejected as expected: %s\n", resp.Status)
			}
			_ = resp.Body.Close()
		}

		fmt.Println("\n=== Demo Complete ===")
	}()

	log.Fatal(http.ListenAndServe(":8080", nil))
}
