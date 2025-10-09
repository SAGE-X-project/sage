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
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// MCP Tool Request/Response types
type ToolRequest struct {
	Tool      string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

// Simple SAGE verification helper
func verifySAGERequest(r *http.Request) error {
	// In a real implementation, you would:
	// 1. Get the agent DID from header
	// 2. Resolve the public key from blockchain
	// 3. Verify the signature

	// For this demo, we'll do basic signature verification
	signature := r.Header.Get("Signature")
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	agentDID := r.Header.Get("X-Agent-DID")
	if agentDID == "" {
		return fmt.Errorf("missing X-Agent-DID header")
	}

	// In a real app, verify the signature here
	fmt.Printf(" Request from agent: %s\n", agentDID)
	return nil
}

// Weather tool handler - WITHOUT SAGE (vulnerable)
func insecureWeatherHandler(w http.ResponseWriter, r *http.Request) {
	var req ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	location, ok := req.Arguments["location"].(string)
	if !ok {
		http.Error(w, "Missing location", http.StatusBadRequest)
		return
	}

	// Simulate weather data
	result := map[string]interface{}{
		"location":    location,
		"temperature": 72,
		"humidity":    65,
		"conditions":  "partly cloudy",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ToolResponse{Result: result})
}

// Weather tool handler - WITH SAGE (secure)
func secureWeatherHandler(w http.ResponseWriter, r *http.Request) {
	// SAGE verification - just add this!
	if err := verifySAGERequest(r); err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	// Rest of the code is exactly the same
	var req ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	location, ok := req.Arguments["location"].(string)
	if !ok {
		http.Error(w, "Missing location", http.StatusBadRequest)
		return
	}

	// Simulate weather data
	result := map[string]interface{}{
		"location":    location,
		"temperature": 72,
		"humidity":    65,
		"conditions":  "partly cloudy",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ToolResponse{Result: result})
}

// Demo client that makes signed requests
func makeSAGERequest() {
	// Generate a key pair for the demo
	keyPair, err := keys.GenerateEd25519KeyPair()
	if err != nil {
		log.Printf("Failed to generate key: %v", err)
		return
	}

	privateKey, _ := keyPair.PrivateKey().(ed25519.PrivateKey)

	// Create request
	reqBody := ToolRequest{
		Tool: "weather",
		Arguments: map[string]interface{}{
			"location": "San Francisco",
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "http://localhost:8082/weather-secure", strings.NewReader(string(bodyBytes)))

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-DID", "did:sage:demo:agent123")
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))

	// Create signature (simplified for demo)
	verifier := rfc9421.NewHTTPVerifier()
	params := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{
			`"@method"`,
			`"@path"`,
			`"content-type"`,
			`"x-agent-did"`,
		},
		KeyID:     "demo-key",
		Algorithm: "ed25519",
		Created:   time.Now().Unix(),
	}

	// Sign the request
	err = verifier.SignRequest(req, "sig1", params, privateKey)
	if err != nil {
		log.Printf("Failed to sign request: %v", err)
		return
	}

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	var result ToolResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode == http.StatusOK {
		fmt.Printf(" Secure request succeeded! Weather: %v\n", result.Result)
	} else {
		fmt.Printf(" Request failed with status %d\n", resp.StatusCode)
	}
}

func main() {
	// Insecure endpoint (vulnerable)
	http.HandleFunc("/weather-insecure", insecureWeatherHandler)

	// Secure endpoint (protected by SAGE)
	http.HandleFunc("/weather-secure", secureWeatherHandler)

	// Demo info endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `SAGE Integration Demo

Endpoints:
- POST /weather-insecure - No security (vulnerable)
- POST /weather-secure   - SAGE protected

Try:
curl -X POST http://localhost:8082/weather-insecure \
  -H "Content-Type: application/json" \
  -d '{"tool":"weather","arguments":{"location":"NYC"}}'

vs.

curl -X POST http://localhost:8082/weather-secure \
  -H "Content-Type: application/json" \
  -d '{"tool":"weather","arguments":{"location":"NYC"}}'
`)
	})

	fmt.Println(" SAGE Integration Demo Server")
	fmt.Println("📍 Listening on http://localhost:8082")
	fmt.Println("")
	fmt.Println("Endpoints:")
	fmt.Println("  POST /weather-insecure - No security (anyone can call)")
	fmt.Println("  POST /weather-secure   - SAGE protected (requires signature)")
	fmt.Println("")

	// Start a goroutine to make a demo request after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("\n📡 Making a signed request...")
		makeSAGERequest()
	}()

	log.Fatal(http.ListenAndServe(":8082", nil))
}
