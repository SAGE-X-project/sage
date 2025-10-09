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


package integration

import (
	"net/http"
	"os"
	"testing"
	"time"
)

// CheckBlockchainConnection checks if a blockchain is running on the specified URL
func CheckBlockchainConnection(url string) bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SkipIfNoBlockchain skips the test if no blockchain is available
func SkipIfNoBlockchain(t *testing.T) {
	t.Helper()

	// Check environment variable to force skip
	if os.Getenv("SKIP_BLOCKCHAIN_TESTS") == "true" {
		t.Skip("Skipping blockchain tests (SKIP_BLOCKCHAIN_TESTS=true)")
	}

	// Check if blockchain is running
	rpcURL := getEnvOrDefault("SAGE_RPC_URL", "http://localhost:8545")
	if !CheckBlockchainConnection(rpcURL) {
		t.Skipf("Skipping test: blockchain not available at %s", rpcURL)
	}
}

// RequireBlockchain marks the test as requiring blockchain
// It will skip if blockchain is not available instead of failing
func RequireBlockchain(t *testing.T) {
	t.Helper()
	SkipIfNoBlockchain(t)
}
