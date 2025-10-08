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
	"strings"

	"github.com/spf13/cobra"
	"github.com/sage-x-project/sage/did"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents by owner address",
	Long: `List all AI agents owned by a specific blockchain address.
This command retrieves all DIDs associated with the given owner address.`,
	RunE: runList,
}

var (
	// List flags
	listChain         string
	listOwner         string
	listRPCEndpoint   string
	listContractAddr  string
	listOutput        string
	listFormat        string
)

func init() {
	rootCmd.AddCommand(listCmd)

	// Required flags
	listCmd.Flags().StringVarP(&listChain, "chain", "c", "", "Blockchain network (ethereum, solana)")
	listCmd.Flags().StringVar(&listOwner, "owner", "", "Owner address")

	// Optional flags
	listCmd.Flags().StringVar(&listRPCEndpoint, "rpc", "", "Blockchain RPC endpoint")
	listCmd.Flags().StringVar(&listContractAddr, "contract", "", "DID registry contract address")
	listCmd.Flags().StringVarP(&listOutput, "output", "o", "", "Output file path")
	listCmd.Flags().StringVar(&listFormat, "format", "table", "Output format (table, json)")

	// Mark required flags
	listCmd.MarkFlagRequired("chain")
	listCmd.MarkFlagRequired("owner")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse chain
	chain, err := parseChain(listChain)
	if err != nil {
		return err
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     listRPCEndpoint,
		ContractAddress: listContractAddr,
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

	// List agents
	fmt.Printf("Listing agents owned by %s on %s...\n", listOwner, chain)
	agents, err := manager.ListAgentsByOwner(ctx, listOwner)
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	// Format output
	var output string
	switch listFormat {
	case "json":
		data, err := json.MarshalIndent(agents, "", "  ")
		if err != nil {
			return err
		}
		output = string(data)
	case "table":
		output = formatAgentsTable(agents)
	default:
		return fmt.Errorf("unsupported format: %s", listFormat)
	}

	// Write output
	if listOutput != "" {
		if err := os.WriteFile(listOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf(" Agent list saved to %s\n", listOutput)
	} else {
		fmt.Println("\n" + output)
	}

	fmt.Printf("\nTotal agents: %d\n", len(agents))

	return nil
}

func formatAgentsTable(agents []*did.AgentMetadata) string {
	if len(agents) == 0 {
		return "No agents found"
	}

	// Header
	output := "DID                                          | Name                 | Status   | Endpoint\n"
	output += strings.Repeat("-", 100) + "\n"

	// Rows
	for _, agent := range agents {
		status := "Active"
		if !agent.IsActive {
			status = "Inactive"
		}
		
		// Truncate long values
		didStr := string(agent.DID)
		if len(didStr) > 40 {
			didStr = didStr[:37] + "..."
		}
		
		nameStr := agent.Name
		if len(nameStr) > 20 {
			nameStr = nameStr[:17] + "..."
		}
		
		endpointStr := agent.Endpoint
		if len(endpointStr) > 30 {
			endpointStr = endpointStr[:27] + "..."
		}

		output += fmt.Sprintf("%-40s | %-20s | %-8s | %s\n", 
			didStr, nameStr, status, endpointStr)
	}

	return output
}
