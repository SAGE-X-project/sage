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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sage-did",
	Short: "SAGE DID CLI - Decentralized Identifier management",
	Long: `SAGE DID CLI provides tools for managing Decentralized Identifiers (DIDs)
and AI agent registration on blockchain for the SAGE project.

This tool supports:
- Agent registration on Ethereum and Solana
- DID resolution and verification
- Agent metadata management
- Agent search and discovery
- Multi-chain DID operations`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Disable default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
