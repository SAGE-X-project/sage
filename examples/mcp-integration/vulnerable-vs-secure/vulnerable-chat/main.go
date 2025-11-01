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

// Simulated message processing
func processMessage(msg ChatMessage) ChatResponse {
	// In a real system, this could execute commands, access databases, etc.
	fmt.Printf("  Received message from: %s\n", msg.AgentID)
	fmt.Printf(" Message: %s\n", msg.Message)

	// Simulate some processing
	time.Sleep(100 * time.Millisecond)

	return ChatResponse{
		Status:  "success",
		Reply:   fmt.Sprintf("Processed message from %s", msg.AgentID),
		AgentID: "chat-server",
	}
}

// VULNERABLE handler - accepts any request!
func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// NO VERIFICATION! Anyone can claim to be any agent!
	// NO SIGNATURE CHECK! Messages can be tampered with!
	// NO REPLAY PROTECTION! Old messages can be resent!

	response := processMessage(msg)
	fmt.Println(" Processed successfully (THIS IS BAD!)")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func main() {
	fmt.Println(" VULNERABLE Chat Server (NO SECURITY)")
	fmt.Println("  DO NOT USE IN PRODUCTION!")
	fmt.Println(" Listening on http://localhost:8082")
	fmt.Println("")
	fmt.Println("Problems with this server:")
	fmt.Println("   No identity verification")
	fmt.Println("   No message integrity checks")
	fmt.Println("   No replay attack protection")
	fmt.Println("   Anyone can impersonate any agent")
	fmt.Println("")

	http.HandleFunc("/chat", handleChat)

	// Configure HTTP server with timeouts to prevent resource exhaustion
	server := &http.Server{
		Addr:         ":8082",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
