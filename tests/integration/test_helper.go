//go:build integration
// +build integration

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