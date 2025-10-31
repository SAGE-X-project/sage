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

var registerCmd = &cobra.Command{
	Use:   "register [commit-hash]",
	Short: "Register agent (Phase 2 of 3)",
	Long: `Register a registered agent by revealing commitment (Phase 2 of 3).

THREE-PHASE REGISTRATION:
  Phase 1: commit    - Send commitment hash + 0.01 ETH stake
  Phase 2: register  - Reveal commitment after 1-60 minutes (THIS COMMAND)
  Phase 3: activate  - Activate agent after 1+ hour

TIMING REQUIREMENTS:
  You must wait 1-60 minutes after commit before registering.
  This timing window prevents both front-running and long-term squatting.

COMMITMENT REVEAL:
  This command reveals your DID, keys, and metadata that were committed in Phase 1.
  The contract verifies the revealed data matches your commitment hash.

Example:
  sage-did register abc123...

  sage-did register --commit-hash abc123...

NOTE: Ed25519 keys on Ethereum require off-chain approval by contract owner.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRegister,
}

var (
	// Register flags
	registerCommitHash string
	registerRPC        string
	registerContract   string
	registerPrivateKey string
)

func init() {
	rootCmd.AddCommand(registerCmd)

	registerCmd.Flags().StringVar(&registerCommitHash, "commit-hash", "", "Commitment hash (hex)")
	registerCmd.Flags().StringVar(&registerRPC, "rpc", "http://localhost:8545", "Ethereum RPC endpoint")
	registerCmd.Flags().StringVar(&registerContract, "contract", "", "Registry contract address")
	registerCmd.Flags().StringVar(&registerPrivateKey, "private-key", "", "Private key hex")
}

func runRegister(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get commit hash from args or flag
	var commitHashHex string
	if len(args) > 0 {
		commitHashHex = args[0]
	} else if registerCommitHash != "" {
		commitHashHex = registerCommitHash
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
		return fmt.Errorf("failed to load commitment: %w (have you run 'commit' first?)", err)
	}

	// Check phase
	if status.Phase != did.PhaseCommitted {
		return fmt.Errorf("agent must be in 'Committed' phase, currently in: %v", status.Phase)
	}

	// Check timing (1-60 minutes)
	elapsed := time.Since(status.CommitTimestamp)
	if elapsed < time.Minute {
		waitTime := time.Minute - elapsed
		return fmt.Errorf("must wait %v before registration (committed %v ago)",
			waitTime.Round(time.Second),
			elapsed.Round(time.Second))
	}
	if elapsed > 60*time.Minute {
		return fmt.Errorf("commitment expired (must register within 60 minutes, committed %v ago)",
			elapsed.Round(time.Minute))
	}

	// Create client
	client, err := createAgentCardClient(registerRPC, registerContract, registerPrivateKey, "")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Register (reveal commitment)
	fmt.Println("Registering agent (revealing commitment)...")
	fmt.Printf("  DID: %s\n", status.Params.DID)
	fmt.Printf("  Name: %s\n", status.Params.Name)
	fmt.Printf("  Elapsed: %v\n", elapsed.Round(time.Second))
	fmt.Println()

	updatedStatus, err := client.RegisterAgent(ctx, status)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Update status
	if err := saveCommitmentStatus(updatedStatus); err != nil {
		fmt.Printf("Warning: failed to update commitment status: %v\n", err)
	}

	// Success
	fmt.Println("✓ Registration successful!")
	fmt.Printf("  Agent ID: %x\n", updatedStatus.AgentID)
	fmt.Printf("  Can activate at: %s\n", updatedStatus.CanActivateAt.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("NEXT STEPS:")
	fmt.Println("  1. Wait at least 1 hour")
	fmt.Printf("  2. Run: sage-did activate %x\n", commitHash)

	return nil
}

// Legacy variables (TODO: Remove when other commands are refactored)
var (
	registerStorageDir string
	registerKeyID      string
	registerKeyFile    string
	registerKeyFormat  string
	registerChain      string
)

// Helper functions (shared with other commands)

func getDefaultRPCEndpoint(chain did.Chain) string {
	switch chain {
	case did.ChainEthereum:
		return "https://eth-mainnet.g.alchemy.com/v2/your-api-key"
	case did.ChainSolana:
		return "https://api.mainnet-beta.solana.com"
	default:
		return ""
	}
}

func getDefaultContractAddress(chain did.Chain) string {
	// Default contract addresses for AgentCardRegistry
	// These are placeholder addresses. For production deployments:
	// 1. Use --contract-address flag with the actual deployed address
	// 2. Refer to contracts/DEPLOYED_ADDRESSES.md for network-specific addresses
	// 3. For local testing, use the address from deployment output
	switch chain {
	case did.ChainEthereum:
		// Placeholder - Update after mainnet/testnet deployment
		// See: contracts/DEPLOYED_ADDRESSES.md
		return "0x0000000000000000000000000000000000000000"
	case did.ChainSolana:
		// Placeholder for future Solana support
		return "11111111111111111111111111111111"
	default:
		return ""
	}
}
