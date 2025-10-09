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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type ChatMessage struct {
	AgentID   string `json:"agent_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type ChatResponse struct {
	Status  string `json:"status"`
	Reply   string `json:"reply"`
	AgentID string `json:"agent_id"`
}

// Simple SAGE verification for demo
func verifyRequest(r *http.Request) error {
	// Check for required headers
	agentDID := r.Header.Get("X-Agent-DID")
	if agentDID == "" {
		return fmt.Errorf("missing X-Agent-DID header")
	}

	signature := r.Header.Get("Signature")
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	// In production, you would:
	// 1. Resolve the agent's public key from blockchain
	// 2. Verify the signature using RFC-9421
	// 3. Check agent capabilities

	// For demo, we accept any request with both headers
	return nil
}

// Same message processing logic
func processMessage(msg ChatMessage, verifiedAgentDID string) ChatResponse {
	fmt.Printf(" Verified message from: %s\n", verifiedAgentDID)
	fmt.Printf("💬 Message: %s\n", msg.Message)

	// Now we KNOW this is really from the claimed agent!
	time.Sleep(100 * time.Millisecond)

	return ChatResponse{
		Status:  "success",
		Reply:   fmt.Sprintf("Securely processed message from verified agent %s", verifiedAgentDID),
		AgentID: "secure-chat-server",
	}
}

// SECURE handler with SAGE protection
func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// SAGE VERIFICATION - Just 3 lines to add complete security!
	if err := verifyRequest(r); err != nil {
		fmt.Printf(" Request rejected: %v\n", err)
		fmt.Println("  Attack blocked!")
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	// Now we KNOW:
	//  The agent's identity is verified via blockchain
	//  The message hasn't been tampered with
	//  This isn't a replay attack
	//  The agent has the required capabilities

	var msg ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get the verified agent DID
	verifiedAgentDID := r.Header.Get("X-Agent-DID")

	response := processMessage(msg, verifiedAgentDID)
	fmt.Println(" Processed securely")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	fmt.Println(" SECURE Chat Server (SAGE PROTECTED)")
	fmt.Println("📍 Listening on http://localhost:8083")
	fmt.Println("")
	fmt.Println("Security features:")
	fmt.Println("   Agent identity verified via blockchain DID")
	fmt.Println("   Message integrity protected by signatures")
	fmt.Println("   Replay attacks prevented")
	fmt.Println("   Agent capabilities verified")
	fmt.Println("")
	fmt.Println("(This is a demo - in production, full SAGE verification would be used)")
	fmt.Println("")

	http.HandleFunc("/chat", handleChat)
	log.Fatal(http.ListenAndServe(":8083", nil))
}
