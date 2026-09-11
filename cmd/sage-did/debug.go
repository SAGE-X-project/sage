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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug DID operations",
	Long: `Inspect a DID: parse it into chain and identifier, validate the
identifier for the chain, and optionally resolve it against the registry.`,
	RunE: runDebug,
}

var (
	debugDID          string
	debugResolve      bool
	debugParseOnly    bool
	debugRPCEndpoint  string
	debugContractAddr string
	debugVerbose      bool
)

func init() {
	rootCmd.AddCommand(debugCmd)

	debugCmd.Flags().StringVar(&debugDID, "did", "", "DID to debug (required)")
	debugCmd.Flags().BoolVar(&debugResolve, "resolve", false, "Resolve the DID against the registry")
	debugCmd.Flags().BoolVar(&debugParseOnly, "parse", false, "Only parse the DID")
	debugCmd.Flags().StringVar(&debugRPCEndpoint, "rpc", "", "Blockchain RPC endpoint (default per chain)")
	debugCmd.Flags().StringVar(&debugContractAddr, "contract", "", "Registry contract address (default per chain)")
	debugCmd.Flags().BoolVarP(&debugVerbose, "verbose", "v", false, "Print the resolved metadata as JSON")

	if err := debugCmd.MarkFlagRequired("did"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}
}

func runDebug(cmd *cobra.Command, args []string) error {
	agentDID := did.AgentDID(debugDID)
	fmt.Printf("Debugging DID: %s\n\n", agentDID)

	chain, identifier, err := did.ParseDID(agentDID)
	if err != nil {
		return fmt.Errorf("invalid DID: %w", err)
	}
	fmt.Println("Parsed:")
	fmt.Printf("  Chain:      %s\n", chain)
	fmt.Printf("  Identifier: %s\n", identifier)

	if chain == did.ChainEthereum {
		if common.IsHexAddress(identifier) {
			fmt.Printf("  Address:    %s (checksummed)\n", common.HexToAddress(identifier).Hex())
		} else {
			fmt.Println("  Warning: identifier is not a hex Ethereum address")
		}
	}

	if debugParseOnly || !debugResolve {
		return nil
	}

	config := &did.RegistryConfig{
		Chain:           chain,
		RPCEndpoint:     debugRPCEndpoint,
		ContractAddress: debugContractAddr,
	}
	if config.RPCEndpoint == "" {
		config.RPCEndpoint = getDefaultRPCEndpoint(chain)
	}
	if config.ContractAddress == "" {
		config.ContractAddress = getDefaultContractAddress(chain)
	}

	manager := did.NewManager()
	if err := manager.Configure(chain, config); err != nil {
		return fmt.Errorf("failed to configure DID manager: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("\nResolving via %s (%s)...\n", config.RPCEndpoint, config.ContractAddress)
	start := time.Now()
	metadata, err := manager.ResolveAgent(ctx, agentDID)
	if err != nil {
		return fmt.Errorf("failed to resolve DID: %w", err)
	}
	fmt.Printf("Resolved in %s\n", time.Since(start).Round(time.Millisecond))

	if debugVerbose {
		out, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode metadata: %w", err)
		}
		fmt.Println(string(out))
		return nil
	}
	fmt.Printf("  Name:        %s\n", metadata.Name)
	fmt.Printf("  Endpoint:    %s\n", metadata.Endpoint)
	fmt.Printf("  Owner:       %s\n", metadata.Owner)
	fmt.Printf("  Active:      %v\n", metadata.IsActive)
	fmt.Printf("  Signing key: %v\n", metadata.PublicKey != nil)
	fmt.Printf("  KEM key:     %v\n", metadata.PublicKEMKey != nil)
	fmt.Printf("  Updated:     %s\n", metadata.UpdatedAt.Format(time.RFC3339))
	return nil
}
