// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later

package testutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestEnvironment manages test dependencies
type TestEnvironment struct {
	EthereumRPC     string
	ContractAddress string
	Auth0Available  bool
	SolanaRPC       string
	skipIntegration bool
}

// NewTestEnvironment creates a new test environment
func NewTestEnvironment() *TestEnvironment {
	env := &TestEnvironment{}
	env.detectEnvironment()
	return env
}

// detectEnvironment automatically detects available services
func (e *TestEnvironment) detectEnvironment() {
	// Check Ethereum RPC
	e.EthereumRPC = os.Getenv("ETHEREUM_RPC_URL")
	if e.EthereumRPC == "" {
		e.EthereumRPC = "http://localhost:8545"
	}

	if e.checkService(e.EthereumRPC) {
		e.ContractAddress = os.Getenv("ETHEREUM_CONTRACT_ADDRESS")
		if e.ContractAddress == "" {
			// Use default test contract address
			e.ContractAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
		}
	}

	// Check Auth0 configuration
	e.Auth0Available = e.checkAuth0Config()

	// Check Solana RPC
	e.SolanaRPC = os.Getenv("SOLANA_RPC_URL")
	if e.SolanaRPC == "" {
		e.SolanaRPC = "http://localhost:8899"
	}

	// Check if we should skip integration tests
	if os.Getenv("SKIP_INTEGRATION") == "true" {
		e.skipIntegration = true
	}
}

// checkService checks if a service is available
func (e *TestEnvironment) checkService(url string) bool {
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode < 500
}

// checkAuth0Config checks if Auth0 is configured
func (e *TestEnvironment) checkAuth0Config() bool {
	required := []string{
		"AUTH0_DOMAIN_1",
		"AUTH0_CLIENT_ID_1",
		"AUTH0_CLIENT_SECRET_1",
	}

	for _, env := range required {
		if os.Getenv(env) == "" {
			return false
		}
	}

	return true
}

// RequireEthereum skips the test if Ethereum is not available
func (e *TestEnvironment) RequireEthereum(t *testing.T) {
	t.Helper()

	if !e.checkService(e.EthereumRPC) {
		// Instead of skipping, start a mock Ethereum server
		mockServer := e.StartMockEthereum(t)
		e.EthereumRPC = mockServer.URL
	}
}

// RequireAuth0 ensures Auth0 is available or provides mock
func (e *TestEnvironment) RequireAuth0(t *testing.T) {
	t.Helper()

	if !e.Auth0Available {
		// Setup mock Auth0 environment variables
		e.SetupMockAuth0(t)
		e.Auth0Available = true
	}
}

// StartMockEthereum starts a mock Ethereum server
func (e *TestEnvironment) StartMockEthereum(t *testing.T) *MockEthereumServer {
	t.Helper()

	server := NewMockEthereumServer()
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start mock Ethereum server: %v", err)
	}

	t.Cleanup(func() {
		server.Stop()
	})

	return server
}

// SetupMockAuth0 sets up mock Auth0 configuration
func (e *TestEnvironment) SetupMockAuth0(t *testing.T) {
	t.Helper()

	// Set mock environment variables
	mockEnv := map[string]string{
		"AUTH0_DOMAIN_1":        "mock-domain.auth0.com",
		"AUTH0_CLIENT_ID_1":     "mock-client-id-1",
		"AUTH0_CLIENT_SECRET_1": "mock-client-secret-1",
		"TEST_DID_1":            "did:sage:mock:agent1",
		"IDENTIFIER_1":          "https://api.mock.com/agent1",
		"AUTH0_KEY_ID_1":        "mock-key-1",

		"AUTH0_DOMAIN_2":        "mock-domain.auth0.com",
		"AUTH0_CLIENT_ID_2":     "mock-client-id-2",
		"AUTH0_CLIENT_SECRET_2": "mock-client-secret-2",
		"TEST_DID_2":            "did:sage:mock:agent2",
		"IDENTIFIER_2":          "https://api.mock.com/agent2",
		"AUTH0_KEY_ID_2":        "mock-key-2",
	}

	for key, value := range mockEnv {
		t.Setenv(key, value)
	}
}

// ShouldSkipIntegration returns whether to skip integration tests
func (e *TestEnvironment) ShouldSkipIntegration(t *testing.T) bool {
	if testing.Short() {
		return true
	}

	if e.skipIntegration {
		return true
	}

	// Don't skip if we can provide mocks
	return false
}

// MockEthereumServer is a mock Ethereum JSON-RPC server
type MockEthereumServer struct {
	listener net.Listener
	server   *http.Server
	URL      string
}

// NewMockEthereumServer creates a new mock Ethereum server
func NewMockEthereumServer() *MockEthereumServer {
	return &MockEthereumServer{}
}

// Start starts the mock server
func (m *MockEthereumServer) Start() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	m.listener = listener
	m.URL = fmt.Sprintf("http://%s", listener.Addr())

	mux := http.NewServeMux()
	mux.HandleFunc("/", m.handleRPC)

	m.server = &http.Server{
		Handler: mux,
	}

	go func() { _ = m.server.Serve(listener) }()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Stop stops the mock server
func (m *MockEthereumServer) Stop() {
	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.server.Shutdown(ctx)
	}
}

// handleRPC handles JSON-RPC requests
func (m *MockEthereumServer) handleRPC(w http.ResponseWriter, r *http.Request) {
	// Simple mock response
	response := `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": "0x1"
	}`

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(response))
}

// CreateTestContract creates a mock contract for testing
func (e *TestEnvironment) CreateTestContract(t *testing.T) string {
	t.Helper()

	if e.ContractAddress != "" {
		return e.ContractAddress
	}

	// Return a mock contract address
	return "0x0000000000000000000000000000000000000001"
}

// SetupTestDID sets up a test DID
func (e *TestEnvironment) SetupTestDID(t *testing.T, did string) {
	t.Helper()

	// This would normally register the DID on-chain
	// For testing, we just ensure the environment is ready
	e.RequireEthereum(t)
}
