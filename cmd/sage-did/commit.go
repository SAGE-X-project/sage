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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/did/ethereum"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit agent registration (Phase 1 of 3)",
	Long: `Commit to agent registration with commitment hash and stake.

THREE-PHASE REGISTRATION:
  Phase 1: commit    - Send commitment hash + 0.01 ETH stake
  Phase 2: register  - Reveal commitment after 1-60 minutes
  Phase 3: activate  - Activate agent after 1+ hour

ANTI-FRONT-RUNNING PROTECTION:
  The commit-reveal pattern prevents others from front-running your
  registration by hiding the DID and keys until Phase 2.

COMMITMENT STATE:
  The commitment is saved to ~/.sage/commitments/<hash>.json for
  use in Phase 2 (register command).

Example:
  sage-did commit \
    --chain ethereum \
    --name "My Agent" \
    --endpoint https://agent.example.com \
    --key keys/agent.pem`,
	RunE: runCommit,
}

var (
	commitChain        string
	commitName         string
	commitDescription  string
	commitEndpoint     string
	commitCapabilities string
	commitKeyFile      string
	commitRPCEndpoint  string
	commitContractAddr string
	commitPrivateKey   string
)

func init() {
	rootCmd.AddCommand(commitCmd)

	// Required flags
	commitCmd.Flags().StringVarP(&commitChain, "chain", "c", "ethereum", "Blockchain network")
	commitCmd.Flags().StringVarP(&commitName, "name", "n", "", "Agent name (required)")
	commitCmd.Flags().StringVar(&commitEndpoint, "endpoint", "", "Agent API endpoint (required)")
	commitCmd.Flags().StringVar(&commitKeyFile, "key", "", "Private key file (required)")

	// Optional flags
	commitCmd.Flags().StringVarP(&commitDescription, "description", "d", "", "Agent description")
	commitCmd.Flags().StringVar(&commitCapabilities, "capabilities", "", "Agent capabilities (JSON)")
	commitCmd.Flags().StringVar(&commitRPCEndpoint, "rpc", "http://localhost:8545", "Ethereum RPC endpoint")
	commitCmd.Flags().StringVar(&commitContractAddr, "contract", "", "Registry contract address")
	commitCmd.Flags().StringVar(&commitPrivateKey, "private-key", "", "Private key hex (overrides --key)")

	if err := commitCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}
	if err := commitCmd.MarkFlagRequired("endpoint"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}
}

func runCommit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate inputs
	if commitChain != "ethereum" {
		return fmt.Errorf("only ethereum chain is supported currently")
	}

	if commitName == "" || commitEndpoint == "" {
		return fmt.Errorf("name and endpoint are required")
	}

	if commitKeyFile == "" && commitPrivateKey == "" {
		return fmt.Errorf("either --key or --private-key is required")
	}

	// Create AgentCardClient
	client, err := createAgentCardClient(commitRPCEndpoint, commitContractAddr, commitPrivateKey, commitKeyFile)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Prepare registration parameters
	params := &did.RegistrationParams{
		DID:          fmt.Sprintf("did:sage:%s:%s", commitChain, "TBD"), // Will be computed
		Name:         commitName,
		Description:  commitDescription,
		Endpoint:     commitEndpoint,
		Capabilities: commitCapabilities,
		Keys:         [][]byte{}, // TODO: Load from key file
		KeyTypes:     []did.KeyType{},
		Signatures:   [][]byte{},
	}

	// Commit registration
	fmt.Println("Committing registration...")
	fmt.Printf("  Name: %s\n", params.Name)
	fmt.Printf("  Endpoint: %s\n", params.Endpoint)
	fmt.Printf("  Stake: 0.01 ETH\n")
	fmt.Println()

	status, err := client.CommitRegistration(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Save commitment status
	if err := saveCommitmentStatus(status); err != nil {
		return fmt.Errorf("failed to save commitment: %w", err)
	}

	// Print success
	fmt.Println("✓ Commitment successful!")
	fmt.Printf("  Commit Hash: %x\n", status.CommitHash)
	fmt.Printf("  Timestamp: %s\n", status.CommitTimestamp.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("NEXT STEPS:")
	fmt.Println("  1. Wait 1-60 minutes")
	fmt.Printf("  2. Run: sage-did register --commit-hash %x\n", status.CommitHash)
	fmt.Println("  3. Wait 1+ hour after registration")
	fmt.Println("  4. Run: sage-did activate <agent-id>")

	return nil
}

func createAgentCardClient(rpcEndpoint, contractAddr, privateKeyHex, keyFile string) (*ethereum.AgentCardClient, error) {
	// Get private key
	if privateKeyHex == "" && keyFile != "" {
		// TODO: Load from file
		return nil, fmt.Errorf("key file loading not implemented yet")
	}

	config := &did.RegistryConfig{
		RPCEndpoint:     rpcEndpoint,
		ContractAddress: contractAddr,
		PrivateKey:      privateKeyHex,
	}

	return ethereum.NewAgentCardClient(config)
}

func saveCommitmentStatus(status *did.CommitmentStatus) error {
	// Create commitments directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	commitDir := filepath.Join(homeDir, ".sage", "commitments")
	if err := os.MkdirAll(commitDir, 0755); err != nil {
		return err
	}

	// Save to file
	filename := filepath.Join(commitDir, fmt.Sprintf("%x.json", status.CommitHash))
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0600)
}

func loadCommitmentStatus(commitHash [32]byte) (*did.CommitmentStatus, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(homeDir, ".sage", "commitments", fmt.Sprintf("%x.json", commitHash))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("commitment not found: %w", err)
	}

	var status did.CommitmentStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	return &status, nil
}
