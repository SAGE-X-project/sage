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

	"github.com/spf13/cobra"
	"github.com/sage-x-project/sage/did"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve [DID]",
	Short: "Resolve an agent DID to retrieve metadata",
	Long: `Resolve a Decentralized Identifier (DID) to retrieve the agent's metadata
from the blockchain, including public key, endpoint, and capabilities.`,
	Args: cobra.ExactArgs(1),
	RunE: runResolve,
}

var (
	// Resolve flags
	resolveRPCEndpoint  string
	resolveContractAddr string
	resolveOutput       string
	resolveFormat       string
)

func init() {
	rootCmd.AddCommand(resolveCmd)

	// Optional flags
	resolveCmd.Flags().StringVar(&resolveRPCEndpoint, "rpc", "", "Blockchain RPC endpoint")
	resolveCmd.Flags().StringVar(&resolveContractAddr, "contract", "", "DID registry contract address")
	resolveCmd.Flags().StringVarP(&resolveOutput, "output", "o", "", "Output file path")
	resolveCmd.Flags().StringVar(&resolveFormat, "format", "json", "Output format (json, text)")
}

func runResolve(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	agentDID := did.AgentDID(args[0])

	// Parse DID to get chain
	chain, _, err := did.ParseDID(agentDID)
	if err != nil {
		return fmt.Errorf("invalid DID: %w", err)
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     resolveRPCEndpoint,
		ContractAddress: resolveContractAddr,
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

	// Resolve agent
	fmt.Printf("Resolving %s...\n", agentDID)
	metadata, err := manager.ResolveAgent(ctx, agentDID)
	if err != nil {
		return fmt.Errorf("failed to resolve DID: %w", err)
	}

	// Format output
	var output string
	switch resolveFormat {
	case "json":
		data, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			return err
		}
		output = string(data)
	case "text":
		output = formatMetadataText(metadata)
	default:
		return fmt.Errorf("unsupported format: %s", resolveFormat)
	}

	// Write output
	if resolveOutput != "" {
		if err := os.WriteFile(resolveOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf(" Metadata saved to %s\n", resolveOutput)
	} else {
		fmt.Println("\n" + output)
	}

	return nil
}

func formatMetadataText(metadata *did.AgentMetadata) string {
	output := fmt.Sprintf("DID: %s\n", metadata.DID)
	output += fmt.Sprintf("Name: %s\n", metadata.Name)
	if metadata.Description != "" {
		output += fmt.Sprintf("Description: %s\n", metadata.Description)
	}
	output += fmt.Sprintf("Endpoint: %s\n", metadata.Endpoint)
	output += fmt.Sprintf("Owner: %s\n", metadata.Owner)
	output += fmt.Sprintf("Active: %v\n", metadata.IsActive)
	output += fmt.Sprintf("Created: %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("Updated: %s\n", metadata.UpdatedAt.Format("2006-01-02 15:04:05"))
	
	if len(metadata.Capabilities) > 0 {
		output += "Capabilities:\n"
		for key, value := range metadata.Capabilities {
			output += fmt.Sprintf("  %s: %v\n", key, value)
		}
	}
	
	return output
}
