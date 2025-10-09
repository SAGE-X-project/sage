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
	"time"

	"github.com/spf13/cobra"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [DID]",
	Short: "Verify agent metadata against blockchain",
	Long: `Verify that local agent metadata matches the on-chain data.
This command compares provided metadata with the blockchain record.`,
	Args: cobra.ExactArgs(1),
	RunE: runVerify,
}

var (
	// Verify flags
	verifyMetadataFile   string
	verifyRPCEndpoint    string
	verifyContractAddr   string
)

func init() {
	rootCmd.AddCommand(verifyCmd)

	// Required flags
	verifyCmd.Flags().StringVarP(&verifyMetadataFile, "metadata", "m", "", "Metadata file to verify (JSON)")

	// Optional flags
	verifyCmd.Flags().StringVar(&verifyRPCEndpoint, "rpc", "", "Blockchain RPC endpoint")
	verifyCmd.Flags().StringVar(&verifyContractAddr, "contract", "", "DID registry contract address")

	// Mark required flags
	verifyCmd.MarkFlagRequired("metadata")
}

func runVerify(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	agentDID := did.AgentDID(args[0])

	// Parse DID to get chain
	chain, _, err := did.ParseDID(agentDID)
	if err != nil {
		return fmt.Errorf("invalid DID: %w", err)
	}

	// Load metadata from file
	metadataData, err := os.ReadFile(verifyMetadataFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata did.AgentMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return fmt.Errorf("invalid metadata JSON: %w", err)
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     verifyRPCEndpoint,
		ContractAddress: verifyContractAddr,
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

	// For now, we'll just resolve and compare
	// In a full implementation, we'd add a VerifyMetadata method to the manager
	fmt.Printf("Verifying metadata for %s...\n", agentDID)
	
	// Resolve current on-chain metadata
	onChainMetadata, err := manager.ResolveAgent(ctx, agentDID)
	if err != nil {
		return fmt.Errorf("failed to resolve DID: %w", err)
	}

	// Compare metadata
	valid := true
	errorMsg := ""
	
	if metadata.Name != onChainMetadata.Name {
		valid = false
		errorMsg = fmt.Sprintf("name mismatch: expected %s, got %s", onChainMetadata.Name, metadata.Name)
	}
	
	if metadata.Endpoint != onChainMetadata.Endpoint {
		valid = false
		if errorMsg != "" {
			errorMsg += "; "
		}
		errorMsg += fmt.Sprintf("endpoint mismatch: expected %s, got %s", onChainMetadata.Endpoint, metadata.Endpoint)
	}

	result := &did.VerificationResult{
		Valid:      valid,
		Error:      errorMsg,
		Agent:      onChainMetadata,
		VerifiedAt: time.Now(),
	}
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Display result
	if result.Valid {
		fmt.Println("\n Metadata verification PASSED")
		fmt.Println("The provided metadata matches the on-chain record.")
	} else {
		fmt.Println("\n Metadata verification FAILED")
		fmt.Printf("Error: %s\n", result.Error)
	}

	// Show on-chain data if available
	if result.Agent != nil {
		fmt.Println("\nOn-chain metadata:")
		fmt.Printf("  Name: %s\n", result.Agent.Name)
		fmt.Printf("  Endpoint: %s\n", result.Agent.Endpoint)
		fmt.Printf("  Active: %v\n", result.Agent.IsActive)
		fmt.Printf("  Owner: %s\n", result.Agent.Owner)
	}

	return nil
}
