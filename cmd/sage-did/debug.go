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
	"github.com/sage-x-project/sage/pkg/agent/did/ethereum"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug DID operations",
	Long: `Debug DID operations and inspect DID documents.

This command provides debugging utilities for:
- Parsing and validating DIDs
- Resolving DID documents
- Checking cache status
- Verifying signatures`,
	RunE: runDebug,
}

var (
	didString       string
	resolveFlag     bool
	parseOnly       bool
	checkCache      bool
	verifySignature bool
	message         string
	signature       string
	verbose         bool
)

func init() {
	rootCmd.AddCommand(debugCmd)

	debugCmd.Flags().StringVar(&didString, "did", "", "DID to debug (required)")
	debugCmd.Flags().BoolVar(&resolveFlag, "resolve", false, "Resolve the DID document")
	debugCmd.Flags().BoolVar(&parseOnly, "parse", false, "Only parse the DID")
	debugCmd.Flags().BoolVar(&checkCache, "cache", false, "Check cache status")
	debugCmd.Flags().BoolVar(&verifySignature, "verify", false, "Verify a signature")
	debugCmd.Flags().StringVar(&message, "message", "", "Message for signature verification")
	debugCmd.Flags().StringVar(&signature, "signature", "", "Signature to verify")
	debugCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	debugCmd.MarkFlagRequired("did")
}

func runDebug(cmd *cobra.Command, args []string) error {
	fmt.Printf(" Debugging DID: %s\n\n", didString)

	// Create resolver
	resolver := ethereum.NewResolverWithCache(100, 5*time.Minute)

	// Parse DID
	fmt.Println(" Parsing DID...")
	parsedDID, err := resolver.ParseDID(didString)
	if err != nil {
		fmt.Printf(" Failed to parse DID: %v\n", err)
		return err
	}

	fmt.Println(" DID parsed successfully:")
	fmt.Printf("  Scheme:  %s\n", parsedDID.Scheme)
	fmt.Printf("  Method:  %s\n", parsedDID.Method)
	fmt.Printf("  Network: %s\n", parsedDID.Network)
	fmt.Printf("  Address: %s\n", parsedDID.Address)

	// Validate Ethereum address
	if !common.IsHexAddress(parsedDID.Address) {
		fmt.Printf("\n  Warning: Address is not a valid Ethereum address\n")
	} else {
		addr := common.HexToAddress(parsedDID.Address)
		fmt.Printf("\n📍 Ethereum Address (checksummed): %s\n", addr.Hex())
	}

	if parseOnly {
		return nil
	}

	// Resolve DID if requested
	if resolveFlag {
		fmt.Println("\n Resolving DID document...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		doc, err := resolver.Resolve(ctx, didString)
		if err != nil {
			fmt.Printf(" Failed to resolve DID: %v\n", err)
			return err
		}

		fmt.Println(" DID document resolved:")

		if verbose {
			// Pretty print JSON
			docJSON, err := json.MarshalIndent(doc, "  ", "  ")
			if err != nil {
				fmt.Printf("  Failed to marshal document: %v\n", err)
			} else {
				fmt.Printf("%s\n", docJSON)
			}
		} else {
			fmt.Printf("  ID:         %s\n", doc.ID)
			fmt.Printf("  Controller: %s\n", doc.Controller)
			fmt.Printf("  Public Key: %s...\n", doc.PublicKey[:20])
			fmt.Printf("  Created:    %s\n", doc.Created.Format(time.RFC3339))
			fmt.Printf("  Updated:    %s\n", doc.Updated.Format(time.RFC3339))
			fmt.Printf("  Revoked:    %v\n", doc.Revoked)
		}
	}

	// Check cache if requested
	if checkCache {
		fmt.Println("\n💾 Checking cache...")

		// Try to resolve from cache (it should be instant if cached)
		start := time.Now()
		ctx := context.Background()
		_, err := resolver.Resolve(ctx, didString)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf(" Cache check failed: %v\n", err)
		} else {
			if duration < 1*time.Millisecond {
				fmt.Printf(" DID is cached (resolution time: %v)\n", duration)
			} else {
				fmt.Printf("ℹ️  DID not in cache (resolution time: %v)\n", duration)
			}
		}
	}

	// Verify signature if requested
	if verifySignature && message != "" && signature != "" {
		fmt.Println("\n Verifying signature...")
		fmt.Printf("  Message:   %s\n", message)
		fmt.Printf("  Signature: %s...\n", signature[:20])

		// This would need the actual implementation
		fmt.Println("  Signature verification not fully implemented in debug mode")
	}

	// Print summary
	if verbose {
		fmt.Println("\n Debug Summary:")
		fmt.Printf("  DID Valid:     %v\n", err == nil)
		fmt.Printf("  Network:       %s\n", parsedDID.Network)
		fmt.Printf("  Address Valid: %v\n", common.IsHexAddress(parsedDID.Address))

		if resolveFlag {
			fmt.Printf("  Resolvable:    true\n")
		}
	}

	return nil
}
