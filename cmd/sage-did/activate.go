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
	"encoding/hex"
	"fmt"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/spf13/cobra"
)

var activateCmd = &cobra.Command{
	Use:   "activate <commit-hash>",
	Short: "Activate agent (Phase 3 of 3)",
	Long: `Activate a registered agent after activation delay.

THREE-PHASE REGISTRATION:
  Phase 1: commit    - Send commitment hash + 0.01 ETH stake
  Phase 2: register  - Reveal commitment after 1-60 minutes
  Phase 3: activate  - Activate agent after 1+ hour (THIS COMMAND)

ACTIVATION DELAY:
  You must wait at least 1 hour after registration before activating.
  This delay prevents rapid spam and Sybil attacks.

STAKE REFUND:
  After successful activation, your 0.01 ETH stake will be refunded.

Example:
  sage-did activate abc123...

  sage-did activate --commit-hash abc123...`,
	Args: cobra.MaximumNArgs(1),
	RunE: runActivate,
}

var (
	activateCommitHash string
	activateRPC        string
	activateContract   string
	activatePrivateKey string
)

func init() {
	rootCmd.AddCommand(activateCmd)

	activateCmd.Flags().StringVar(&activateCommitHash, "commit-hash", "", "Commitment hash (hex)")
	activateCmd.Flags().StringVar(&activateRPC, "rpc", "http://localhost:8545", "Ethereum RPC endpoint")
	activateCmd.Flags().StringVar(&activateContract, "contract", "", "Registry contract address")
	activateCmd.Flags().StringVar(&activatePrivateKey, "private-key", "", "Private key hex")
}

func runActivate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get commit hash from args or flag
	var commitHashHex string
	if len(args) > 0 {
		commitHashHex = args[0]
	} else if activateCommitHash != "" {
		commitHashHex = activateCommitHash
	} else {
		return fmt.Errorf("commit hash required: provide as argument or use --commit-hash")
	}

	// Parse commit hash
	commitHashBytes, err := hex.DecodeString(commitHashHex)
	if err != nil {
		return fmt.Errorf("invalid commit hash: %w", err)
	}
	if len(commitHashBytes) != 32 {
		return fmt.Errorf("commit hash must be 32 bytes (64 hex characters)")
	}

	var commitHash [32]byte
	copy(commitHash[:], commitHashBytes)

	// Load commitment status
	status, err := loadCommitmentStatus(commitHash)
	if err != nil {
		return fmt.Errorf("failed to load commitment: %w (have you run 'commit' and 'register'?)", err)
	}

	// Check phase
	if status.Phase != did.PhaseRegistered {
		return fmt.Errorf("agent must be in 'Registered' phase, currently in: %v", status.Phase)
	}

	// Check activation time
	if time.Now().Before(status.CanActivateAt) {
		waitTime := time.Until(status.CanActivateAt)
		return fmt.Errorf("must wait %v before activation (can activate at %s)",
			waitTime.Round(time.Second),
			status.CanActivateAt.Format(time.RFC3339))
	}

	// Create client
	client, err := createAgentCardClient(activateRPC, activateContract, activatePrivateKey, "")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Activate
	fmt.Println("Activating agent...")
	fmt.Printf("  Agent ID: %x\n", status.AgentID)
	fmt.Println()

	if err := client.ActivateAgent(ctx, status); err != nil {
		return fmt.Errorf("activation failed: %w", err)
	}

	// Update status
	status.Phase = did.PhaseActivated
	if err := saveCommitmentStatus(status); err != nil {
		fmt.Printf("Warning: failed to update commitment status: %v\n", err)
	}

	// Success
	fmt.Println("✓ Activation successful!")
	fmt.Printf("  Agent ID: %x\n", status.AgentID)
	fmt.Println("  Your 0.01 ETH stake will be refunded")
	fmt.Println()
	fmt.Println("Your agent is now fully active and can be queried:")
	fmt.Printf("  sage-did resolve %s\n", status.Params.DID)

	return nil
}
