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


package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sage-x-project/sage/did"
)

var updateCmd = &cobra.Command{
	Use:   "update [DID]",
	Short: "Update agent metadata",
	Long: `Update the metadata of an existing AI agent on blockchain.
Only the agent owner can update the metadata.`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdate,
}

var (
	// Update flags
	updateName           string
	updateDescription    string
	updateEndpoint       string
	updateCapabilities   string
	updateKeyFile        string
	updateKeyFormat      string
	updateStorageDir     string
	updateKeyID          string
	updateRPCEndpoint    string
	updateContractAddr   string
	updatePrivateKey     string
)

func init() {
	rootCmd.AddCommand(updateCmd)

	// Update fields
	updateCmd.Flags().StringVarP(&updateName, "name", "n", "", "New agent name")
	updateCmd.Flags().StringVarP(&updateDescription, "description", "d", "", "New agent description")
	updateCmd.Flags().StringVar(&updateEndpoint, "endpoint", "", "New agent endpoint")
	updateCmd.Flags().StringVar(&updateCapabilities, "capabilities", "", "New agent capabilities (JSON)")

	// Key source flags
	updateCmd.Flags().StringVarP(&updateKeyFile, "key", "k", "", "Key file path (JWK or PEM format)")
	updateCmd.Flags().StringVar(&updateKeyFormat, "key-format", "jwk", "Key file format (jwk, pem)")
	updateCmd.Flags().StringVar(&updateStorageDir, "storage-dir", "", "Key storage directory")
	updateCmd.Flags().StringVar(&updateKeyID, "key-id", "", "Key ID in storage")

	// Blockchain connection flags
	updateCmd.Flags().StringVar(&updateRPCEndpoint, "rpc", "", "Blockchain RPC endpoint")
	updateCmd.Flags().StringVar(&updateContractAddr, "contract", "", "DID registry contract address")
	updateCmd.Flags().StringVar(&updatePrivateKey, "private-key", "", "Transaction signer private key")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	agentDID := did.AgentDID(args[0])

	// Parse DID to get chain
	chain, _, err := did.ParseDID(agentDID)
	if err != nil {
		return fmt.Errorf("invalid DID: %w", err)
	}

	// Load key pair
	keyPair, err := loadKeyPair()
	if err != nil {
		return fmt.Errorf("failed to load key pair: %w", err)
	}

	// Build updates map
	updates := make(map[string]interface{})
	if updateName != "" {
		updates["name"] = updateName
	}
	if updateDescription != "" {
		updates["description"] = updateDescription
	}
	if updateEndpoint != "" {
		updates["endpoint"] = updateEndpoint
	}
	if updateCapabilities != "" {
		var capabilities map[string]interface{}
		if err := json.Unmarshal([]byte(updateCapabilities), &capabilities); err != nil {
			return fmt.Errorf("invalid capabilities JSON: %w", err)
		}
		updates["capabilities"] = capabilities
	}

	if len(updates) == 0 {
		return fmt.Errorf("no updates specified")
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     updateRPCEndpoint,
		ContractAddress: updateContractAddr,
		PrivateKey:      updatePrivateKey,
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

	// Update agent
	fmt.Printf("Updating agent %s...\n", agentDID)
	if err := manager.UpdateAgent(ctx, agentDID, updates, keyPair); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println(" Agent updated successfully!")
	
	// Show what was updated
	fmt.Println("\nUpdated fields:")
	for key, value := range updates {
		fmt.Printf("  %s: %v\n", key, value)
	}

	return nil
}