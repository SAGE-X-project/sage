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
	"fmt"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/spf13/cobra"
)

var deactivateCmd = &cobra.Command{
	Use:   "deactivate [DID]",
	Short: "Deactivate an AI agent",
	Long: `Deactivate an AI agent on blockchain. This operation marks the agent as inactive
but does not delete it. Only the agent owner can deactivate an agent.`,
	Args: cobra.ExactArgs(1),
	RunE: runDeactivate,
}

var (
	// Deactivate flags
	deactivateKeyFile      string
	deactivateKeyFormat    string
	deactivateStorageDir   string
	deactivateKeyID        string
	deactivateRPCEndpoint  string
	deactivateContractAddr string
	deactivatePrivateKey   string
	deactivateConfirm      bool
)

func init() {
	rootCmd.AddCommand(deactivateCmd)

	// Key source flags
	deactivateCmd.Flags().StringVarP(&deactivateKeyFile, "key", "k", "", "Key file path (JWK or PEM format)")
	deactivateCmd.Flags().StringVar(&deactivateKeyFormat, "key-format", "jwk", "Key file format (jwk, pem)")
	deactivateCmd.Flags().StringVar(&deactivateStorageDir, "storage-dir", "", "Key storage directory")
	deactivateCmd.Flags().StringVar(&deactivateKeyID, "key-id", "", "Key ID in storage")

	// Blockchain connection flags
	deactivateCmd.Flags().StringVar(&deactivateRPCEndpoint, "rpc", "", "Blockchain RPC endpoint")
	deactivateCmd.Flags().StringVar(&deactivateContractAddr, "contract", "", "DID registry contract address")
	deactivateCmd.Flags().StringVar(&deactivatePrivateKey, "private-key", "", "Transaction signer private key")

	// Confirmation flag
	deactivateCmd.Flags().BoolVarP(&deactivateConfirm, "yes", "y", false, "Skip confirmation prompt")
}

func runDeactivate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	agentDID := did.AgentDID(args[0])

	// Parse DID to get chain
	chain, _, err := did.ParseDID(agentDID)
	if err != nil {
		return fmt.Errorf("invalid DID: %w", err)
	}

	// Confirm deactivation
	if !deactivateConfirm {
		fmt.Printf("  Are you sure you want to deactivate agent %s? (y/N): ", agentDID)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Printf("Failed to read input: %v\n", err)
			fmt.Println("Deactivation cancelled")
			return nil
		}
		if response != "y" && response != "Y" {
			fmt.Println("Deactivation cancelled")
			return nil
		}
	}

	// Load key pair
	keyPair, err := loadKeyPairForDeactivate()
	if err != nil {
		return fmt.Errorf("failed to load key pair: %w", err)
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     deactivateRPCEndpoint,
		ContractAddress: deactivateContractAddr,
		PrivateKey:      deactivatePrivateKey,
	}

	if config.RPCEndpoint == "" {
		config.RPCEndpoint = getDefaultRPCEndpoint(chain)
	}
	if config.ContractAddress == "" {
		config.ContractAddress = getDefaultContractAddress(chain)
	}

	// Create DID manager
	manager := did.NewManager()
	if err := manager.Configure(chain, config); err != nil {
		return fmt.Errorf("failed to configure DID manager: %w", err)
	}

	// Deactivate agent
	fmt.Printf("Deactivating agent %s...\n", agentDID)
	if err := manager.DeactivateAgent(ctx, agentDID, keyPair); err != nil {
		return fmt.Errorf("deactivation failed: %w", err)
	}

	fmt.Println(" Agent deactivated successfully!")
	fmt.Println("\nThe agent is now inactive and cannot be used for operations.")
	fmt.Println("The agent data remains on-chain but is marked as deactivated.")

	return nil
}

func loadKeyPairForDeactivate() (crypto.KeyPair, error) {
	// Override global flags with deactivate-specific ones
	if deactivateStorageDir != "" {
		registerStorageDir = deactivateStorageDir
	}
	if deactivateKeyID != "" {
		registerKeyID = deactivateKeyID
	}
	if deactivateKeyFile != "" {
		registerKeyFile = deactivateKeyFile
	}
	if deactivateKeyFormat != "" {
		registerKeyFormat = deactivateKeyFormat
	}

	return loadKeyPair()
}
