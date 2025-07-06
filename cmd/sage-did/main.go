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