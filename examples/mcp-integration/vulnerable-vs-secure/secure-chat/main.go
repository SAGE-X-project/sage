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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
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

// trustedAgents holds the agents this server accepts, loaded from
// SAGE_TRUSTED_AGENTS ("did=hex-ed25519-public-key;did=..."). A production
// server resolves keys from the on-chain DID document instead. Requests from
// any other DID, unsigned requests, and replayed or stale signatures are
// rejected by rfc9421.HTTPVerifier.
var (
	trustedAgents  = loadTrustedAgents(os.Getenv("SAGE_TRUSTED_AGENTS"))
	serverVerifier = rfc9421.NewHTTPVerifier()
)

func loadTrustedAgents(spec string) map[string]ed25519.PublicKey {
	out := map[string]ed25519.PublicKey{}
	for _, entry := range strings.Split(spec, ";") {
		did, hexKey, ok := strings.Cut(strings.TrimSpace(entry), "=")
		if !ok {
			continue
		}
		raw, err := hex.DecodeString(strings.TrimSpace(hexKey))
		if err != nil || len(raw) != ed25519.PublicKeySize {
			log.Printf("ignoring SAGE_TRUSTED_AGENTS entry for %q: not a 32-byte hex Ed25519 key", strings.TrimSpace(did)) // #nosec G706 -- value is quoted, no control characters reach the log
			continue
		}
		out[strings.TrimSpace(did)] = ed25519.PublicKey(raw)
	}
	return out
}

// verifyRequest verifies the RFC 9421 signature and returns the verified DID.
func verifyRequest(r *http.Request) (string, error) {
	inputs, err := rfc9421.ParseSignatureInput(r.Header.Get("Signature-Input"))
	if err != nil || len(inputs) == 0 {
		return "", fmt.Errorf("missing or malformed Signature-Input header")
	}
	var keyID string
	for _, params := range inputs {
		keyID = params.KeyID
		break
	}
	agentDID := rfc9421.KeyIDDID(keyID)
	pub, ok := trustedAgents[agentDID]
	if !ok {
		return "", fmt.Errorf("unknown agent %q", agentDID)
	}
	opts := rfc9421.StrictHTTPVerificationOptions()
	opts.ExpectedDID = agentDID
	if err := serverVerifier.VerifyRequest(r, pub, opts); err != nil {
		return "", err
	}
	return agentDID, nil
}

// Same message processing logic
func processMessage(msg ChatMessage, verifiedAgentDID string) ChatResponse {
	fmt.Printf(" Verified message from: %s\n", verifiedAgentDID)
	fmt.Printf(" Message: %s\n", msg.Message)

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

	// SAGE verification: the request must carry a valid RFC 9421 signature
	// from a trusted agent key covering method, target, authority and body
	// digest, with a fresh, unused nonce.
	verifiedAgentDID, err := verifyRequest(r)
	if err != nil {
		fmt.Printf(" Request rejected: %v\n", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Now we know: the signer holds the key trusted for verifiedAgentDID, the
	// signed parts of the request were not modified, and the signature is
	// neither stale nor replayed. Capabilities are not checked by this demo.

	var msg ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	response := processMessage(msg, verifiedAgentDID)
	fmt.Println(" Processed securely")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func main() {
	fmt.Println(" SECURE Chat Server (SAGE PROTECTED)")
	fmt.Println(" Listening on http://localhost:8083")
	fmt.Println("")
	fmt.Println("Checks performed on every request (rfc9421.HTTPVerifier, strict options):")
	fmt.Println("   RFC 9421 signature from a key in SAGE_TRUSTED_AGENTS")
	fmt.Println("   Covered: @method, @target-uri, @authority, content-digest")
	fmt.Println("   Freshness (5 min) and nonce replay rejection")
	fmt.Println("")
	fmt.Printf("Trusted agents: %d (set SAGE_TRUSTED_AGENTS=\"did=hexpubkey;...\")\n", len(trustedAgents))
	fmt.Println("(Demo: keys come from the environment; production resolves them from the DID document)")
	fmt.Println("")

	http.HandleFunc("/chat", handleChat)

	// Configure HTTP server with timeouts to prevent resource exhaustion
	server := &http.Server{
		Addr:         ":8083",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
